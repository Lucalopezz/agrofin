package colheitas

import (
	"context"
	"math"
	"sort"
	"strings"
	"time"

	calendarDate "github.com/lucalopezz/agrofin/apps/api/internal/platform/date"
	"github.com/lucalopezz/agrofin/apps/api/internal/platform/uuid"
)

type harvestRepository interface {
	CheckActiveEmployee(ctx context.Context, userID, employeeID string) error
	RateForDate(ctx context.Context, userID string, boxType BoxType, date time.Time) (ProtectedRate, error)
	CreateHarvest(ctx context.Context, harvest ProtectedHarvest) error
	ListHarvests(ctx context.Context, userID string, period HarvestPeriod) ([]ProtectedHarvestRecord, error)
}

type harvestRateProtector interface {
	Decrypt(userID, rateID string, amount ProtectedRateAmount) (int64, error)
	EncryptHarvestItemRate(userID, harvestID, employeeID string, harvestDate time.Time, boxType BoxType, quantity int32, amountCents int64) (ProtectedRateAmount, error)
	DecryptHarvestItemRate(userID, harvestID, employeeID string, harvestDate time.Time, boxType BoxType, quantity int32, amount ProtectedRateAmount) (int64, error)
}

// harvestReadRecord combina o agregado com um dado atual do cadastro somente
// para montar projeções; o nome não passa a fazer parte da entidade Harvest.
type harvestReadRecord struct {
	Harvest      Harvest
	EmployeeName string
}

// HarvestService monta e persiste um lançamento completo. A tarifa é lida
// pela data do evento e cifrada de novo como fotografia imutável do item.
type HarvestService struct {
	repo      harvestRepository
	protector harvestRateProtector
}

func NewHarvestService(repo harvestRepository, protector harvestRateProtector) *HarvestService {
	return &HarvestService{repo: repo, protector: protector}
}

func (s *HarvestService) Create(ctx context.Context, userID string, input CreateHarvestInput) (Harvest, error) {
	if err := ctx.Err(); err != nil {
		return Harvest{}, err
	}
	if userID == "" || strings.TrimSpace(userID) != userID {
		return Harvest{}, ErrInvalidHarvestOwner
	}
	items := make([]HarvestItem, 0, len(input.Items))
	// Para cada item, a tarifa vigente na data do lançamento é lida e decifrada. O
	// valor é re-cifrado com o ID do lançamento, do funcionário e da data, para
	for _, item := range input.Items {
		items = append(items, HarvestItem{BoxType: item.BoxType, Quantity: item.Quantity})
	}
	harvest, err := normalizeHarvest(Harvest{
		UserID: userID, EmployeeID: input.EmployeeID,
		HarvestDate: input.HarvestDate, Items: items,
	})
	if err != nil {
		return Harvest{}, err
	}
	// A checagem antecipada evita procurar tarifas para um ID alheio ou inativo.
	// O Repository repete a verificação sob bloqueio na gravação para fechar a corrida.
	if err := s.repo.CheckActiveEmployee(ctx, userID, harvest.EmployeeID); err != nil {
		return Harvest{}, err
	}

	// O ID precisa existir antes da cifra: ele compõe o AAD e impede que o
	// envelope de um item seja copiado para outro lançamento. Como UUID é
	// independente do estado do banco, gerá-lo localmente evita uma consulta extra.
	harvest.ID, err = uuid.New()
	if err != nil {
		return Harvest{}, err
	}
	if err := ctx.Err(); err != nil {
		return Harvest{}, err
	}
	protected := ProtectedHarvest{
		ID: harvest.ID, UserID: userID, EmployeeID: harvest.EmployeeID,
		HarvestDate: harvest.HarvestDate,
		Items:       make([]ProtectedHarvestItem, 0, len(harvest.Items)),
	}
	for index := range harvest.Items {
		if err := ctx.Err(); err != nil {
			return Harvest{}, err
		}
		item := &harvest.Items[index]
		rate, err := s.repo.RateForDate(ctx, userID, item.BoxType, harvest.HarvestDate)
		if err != nil {
			return Harvest{}, err
		}
		amountCents, err := s.protector.Decrypt(userID, rate.ID, rate.Amount)
		if err != nil {
			return Harvest{}, err
		}
		item.AppliedRateCents = amountCents
		appliedAmount, err := s.protector.EncryptHarvestItemRate(
			userID, harvest.ID, harvest.EmployeeID, harvest.HarvestDate,
			item.BoxType, item.Quantity, amountCents,
		)
		if err != nil {
			return Harvest{}, err
		}
		protected.Items = append(protected.Items, ProtectedHarvestItem{
			BoxType: item.BoxType, Quantity: item.Quantity, AppliedAmount: appliedAmount,
		})
	}
	if err := ctx.Err(); err != nil {
		return Harvest{}, err
	}
	// A persistência revalida e bloqueia o funcionário na mesma transação que
	// grava cabeçalho e itens, para uma desativação concorrente não passar no meio.
	if err := s.repo.CreateHarvest(ctx, protected); err != nil {
		return Harvest{}, err
	}
	return harvest, nil
}

// List valida o período, lê apenas dados do gerente autenticado e só então
// descriptografa as tarifas aplicadas para montar os agregados de resposta.
func (s *HarvestService) List(ctx context.Context, userID string, period HarvestPeriod) ([]Harvest, error) {
	records, err := s.listRecords(ctx, userID, period)
	if err != nil {
		return nil, err
	}
	harvests := make([]Harvest, 0, len(records))
	for _, record := range records {
		harvests = append(harvests, record.Harvest)
	}
	return harvests, nil
}

func (s *HarvestService) listRecords(ctx context.Context, userID string, period HarvestPeriod) ([]harvestReadRecord, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if userID == "" || strings.TrimSpace(userID) != userID {
		return nil, ErrInvalidHarvestOwner
	}

	// Validações de domínio
	period, err := normalizeHarvestPeriod(period)
	if err != nil {
		return nil, err
	}

	// Busca do repo dados criptografados
	protected, err := s.repo.ListHarvests(ctx, userID, period)
	if err != nil {
		return nil, err
	}
	results := make([]harvestReadRecord, 0, len(protected))
	// Para cada item da lista de itens protegidos, percorre as colheitas
	for _, record := range protected {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		protectedHarvest := record.Harvest
		harvest := Harvest{
			ID: protectedHarvest.ID, UserID: userID,
			EmployeeID:  protectedHarvest.EmployeeID,
			HarvestDate: calendarDate.Normalize(protectedHarvest.HarvestDate),
			// Cria uma lista vazia
			Items: make([]HarvestItem, 0, len(protectedHarvest.Items)),
		}
		// Para cada item de colheita da colheita protegida, adiciona à lista de itens do lançamento
		for _, protectedItem := range protectedHarvest.Items {
			harvest.Items = append(harvest.Items, HarvestItem{
				BoxType: protectedItem.BoxType, Quantity: protectedItem.Quantity,
			})
		}
		// Validar antes de abrir envelopes evita tratar metadados fora do domínio
		// como contexto confiável para a autenticação da tarifa aplicada.
		harvest, err = normalizeHarvest(harvest)
		if err != nil {
			return nil, err
		}
		// Para cada item de colheita montado, descriptografa o amount
		for index, protectedItem := range protectedHarvest.Items {
			if err := ctx.Err(); err != nil {
				return nil, err
			}
			amountCents, err := s.protector.DecryptHarvestItemRate(
				userID, harvest.ID, harvest.EmployeeID, harvest.HarvestDate,
				harvest.Items[index].BoxType, harvest.Items[index].Quantity, protectedItem.AppliedAmount,
			)
			if err != nil {
				return nil, err
			}
			// Atualiza o item com o valor descriptografado
			harvest.Items[index].AppliedRateCents = amountCents
		}
		// Adiciona o lançamento completo à lista de resultados
		results = append(results, harvestReadRecord{Harvest: harvest, EmployeeName: record.EmployeeName})
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	return results, nil
}

// Summarize agrupa os lançamentos autorizados por funcionário. O valor soma
// quantidade vezes a tarifa fotografada em cada item, mesmo se ela variar.
func (s *HarvestService) Summarize(ctx context.Context, userID string, period HarvestPeriod) ([]HarvestSummary, error) {
	records, err := s.listRecords(ctx, userID, period)
	if err != nil {
		return nil, err
	}
	// Praticamente a mesma lógica da função acima,
	// mas aqui faz a conta por funcionário
	byEmployee := make(map[string]HarvestSummary)
	for _, record := range records {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		harvest := record.Harvest
		summary, exists := byEmployee[harvest.EmployeeID]
		if !exists {
			summary = HarvestSummary{EmployeeID: harvest.EmployeeID, EmployeeName: record.EmployeeName}
		}
		for _, item := range harvest.Items {
			if err := ctx.Err(); err != nil {
				return nil, err
			}
			quantity := int64(item.Quantity)
			if quantity <= 0 {
				return nil, ErrInvalidHarvestQuantity
			}
			if item.AppliedRateCents <= 0 {
				return nil, ErrInvalidProtectedRateAmount
			}
			// Evitar overflow antes da multiplicação mantém os centavos corretos em int64.
			if item.AppliedRateCents > math.MaxInt64/quantity {
				return nil, ErrHarvestSummaryOverflow
			}
			amount := item.AppliedRateCents * quantity
			var err error
			switch item.BoxType {
			case BoxTypeNormal:
				summary.NormalBoxes, err = addHarvestSummaryValue(summary.NormalBoxes, quantity)
				if err == nil {
					summary.NormalAmountCents, err = addHarvestSummaryValue(summary.NormalAmountCents, amount)
				}
			case BoxTypeMoagem:
				summary.MillingBoxes, err = addHarvestSummaryValue(summary.MillingBoxes, quantity)
				if err == nil {
					summary.MillingAmountCents, err = addHarvestSummaryValue(summary.MillingAmountCents, amount)
				}
			default:
				return nil, ErrInvalidBoxType
			}
			if err != nil {
				return nil, err
			}
			summary.HarvestAmountCents, err = addHarvestSummaryValue(summary.HarvestAmountCents, amount)
			if err != nil {
				return nil, err
			}
		}
		byEmployee[harvest.EmployeeID] = summary
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	summaries := make([]HarvestSummary, 0, len(byEmployee))
	for _, summary := range byEmployee {
		summaries = append(summaries, summary)
	}
	// Map iteration não garante ordem; ordenar por nome e ID deixa a resposta estável.
	sort.Slice(summaries, func(i, j int) bool {
		if summaries[i].EmployeeName != summaries[j].EmployeeName {
			return summaries[i].EmployeeName < summaries[j].EmployeeName
		}
		return summaries[i].EmployeeID < summaries[j].EmployeeID
	})
	return summaries, nil
}

func addHarvestSummaryValue(current, addition int64) (int64, error) {
	if current < 0 || addition < 0 || current > math.MaxInt64-addition {
		return 0, ErrHarvestSummaryOverflow
	}
	return current + addition, nil
}
