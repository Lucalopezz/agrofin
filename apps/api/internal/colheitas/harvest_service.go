package colheitas

import (
	"context"
	"strings"
	"time"
)

type harvestRepository interface {
	CheckActiveEmployee(ctx context.Context, userID, employeeID string) error
	NextHarvestID(ctx context.Context) (string, error)
	RateForDate(ctx context.Context, userID string, boxType BoxType, date time.Time) (ProtectedRate, error)
	CreateHarvest(ctx context.Context, harvest ProtectedHarvest) error
	ListHarvests(ctx context.Context, userID string, period HarvestPeriod) ([]ProtectedHarvest, error)
}

type harvestRateProtector interface {
	Decrypt(userID, rateID string, amount ProtectedRateAmount) (int64, error)
	EncryptHarvestItemRate(userID, harvestID, employeeID string, harvestDate time.Time, boxType BoxType, quantity int32, amountCents int64) (ProtectedRateAmount, error)
	DecryptHarvestItemRate(userID, harvestID, employeeID string, harvestDate time.Time, boxType BoxType, quantity int32, amount ProtectedRateAmount) (int64, error)
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
	// envelope de um item seja copiado para outro lançamento.
	harvest.ID, err = s.repo.NextHarvestID(ctx)
	if err != nil {
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
	harvests := make([]Harvest, 0, len(protected))
	// Para cada item da lista de itens protegidos, percorre as colheitas
	for _, record := range protected {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		harvest := Harvest{
			ID: record.ID, UserID: userID, EmployeeID: record.EmployeeID,
			HarvestDate: rateDate(record.HarvestDate),
			// Cria uma lista vazia
			Items: make([]HarvestItem, 0, len(record.Items)),
		}
		// Para cada item de colheita da colheita protegida, adiciona à lista de itens do lançamento
		for _, protectedItem := range record.Items {
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
		for index, protectedItem := range record.Items {
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
		harvests = append(harvests, harvest)
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	return harvests, nil
}
