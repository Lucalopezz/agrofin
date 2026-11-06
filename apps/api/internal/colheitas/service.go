package colheitas

import (
	"context"
	"strings"
	"time"

	"github.com/lucalopezz/agrofin/apps/api/internal/platform/uuid"
)

// CreateRateInput contém somente os dados da tarifa geral por tipo. O gerente
// vem do chamador confiável e o ID será gerado pela persistência, nunca pelo JSON.
type CreateRateInput struct {
	BoxType     BoxType
	AmountCents int64
	ValidFrom   time.Time
	ValidUntil  *time.Time
}

// RatePeriod delimita a consulta de sobreposição sem enviar valores financeiros.
type RatePeriod struct {
	UserID     string
	BoxType    BoxType
	ValidFrom  time.Time
	ValidUntil *time.Time
}

// ProtectedRate é a representação protegida usada na fronteira do Repository.
// Não contém AmountCents: persistência e leitura trafegam somente envelope e metadados.
type ProtectedRate struct {
	ID         string
	UserID     string
	BoxType    BoxType
	Amount     ProtectedRateAmount
	ValidFrom  time.Time
	ValidUntil *time.Time
}

// As interfaces ficam junto do consumidor e não exigem PostgreSQL nos testes.
// O Repository concreto mantém a restrição de sobreposição no PostgreSQL e
// traduz a disputa concorrente para ErrRateOverlap.
type rateRepository interface {
	HasRateOverlap(ctx context.Context, period RatePeriod) (bool, error)
	CreateRate(ctx context.Context, rate ProtectedRate) error
	ListRates(ctx context.Context, userID string) ([]ProtectedRate, error)
}

type rateAmountProtector interface {
	Encrypt(userID, rateID string, amountCents int64) (ProtectedRateAmount, error)
	Decrypt(userID, rateID string, amount ProtectedRateAmount) (int64, error)
}

// Service coordena o cadastro e a consulta de tarifas gerais da conta;
// funcionários da mesma conta usam a tarifa vigente do tipo, sem exceções individuais.
type Service struct {
	repo      rateRepository
	protector rateAmountProtector
}

// NewService recebe as dependências já configuradas, sem abrir conexão ou carregar
// chaves. Cada operação recebe o contexto do seu próprio chamador.
func NewService(repo rateRepository, protector rateAmountProtector) *Service {
	return &Service{repo: repo, protector: protector}
}

// CreateRate valida, verifica vigência e protege o valor antes de pedir a gravação.
// O retorno em memória contém os centavos, mas a persistência recebe ProtectedRate.
func (s *Service) CreateRate(ctx context.Context, userID string, input CreateRateInput) (Rate, error) {
	if err := ctx.Err(); err != nil {
		return Rate{}, err
	}
	if userID == "" || strings.TrimSpace(userID) != userID {
		return Rate{}, ErrInvalidRateOwner
	}
	rate, err := normalizeRate(Rate{
		UserID: userID, BoxType: input.BoxType, AmountCents: input.AmountCents,
		ValidFrom: input.ValidFrom, ValidUntil: input.ValidUntil,
	})
	if err != nil {
		return Rate{}, err
	}
	overlaps, err := s.repo.HasRateOverlap(ctx, RatePeriod{
		UserID: rate.UserID, BoxType: rate.BoxType,
		ValidFrom: rate.ValidFrom, ValidUntil: rate.ValidUntil,
	})
	if err != nil {
		return Rate{}, err
	}
	if overlaps {
		return Rate{}, ErrRateOverlap
	}
	// O ID é criado na aplicação porque a cifra precisa autenticá-lo antes da
	// primeira gravação; gerar esse valor não exige uma consulta ao PostgreSQL.
	rate.ID, err = uuid.New()
	if err != nil {
		return Rate{}, err
	}
	if err := ctx.Err(); err != nil {
		return Rate{}, err
	}
	amount, err := s.protector.Encrypt(rate.UserID, rate.ID, rate.AmountCents)
	if err != nil {
		return Rate{}, err
	}
	// A cifra é local e não recebe contexto. Se a requisição terminou durante
	// esse trabalho, não iniciamos uma escrita com um contexto já cancelado.
	if err := ctx.Err(); err != nil {
		return Rate{}, err
	}
	err = s.repo.CreateRate(ctx, ProtectedRate{
		ID: rate.ID, UserID: rate.UserID, BoxType: rate.BoxType, Amount: amount,
		ValidFrom: rate.ValidFrom, ValidUntil: rate.ValidUntil,
	})
	if err != nil {
		return Rate{}, err
	}
	return rate, nil
}

// ListRates recupera somente a conta pedida e devolve a lista inteira ou erro.
// Se qualquer envelope falhar na autenticação, descarta o resultado parcial.
func (s *Service) ListRates(ctx context.Context, userID string) ([]Rate, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if userID == "" || strings.TrimSpace(userID) != userID {
		return nil, ErrInvalidRateOwner
	}
	protectedRates, err := s.repo.ListRates(ctx, userID)
	if err != nil {
		return nil, err
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	rates := make([]Rate, 0, len(protectedRates))
	for _, protected := range protectedRates {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		// AAD usa o dono autenticado, não um valor que veio da linha do banco.
		amountCents, err := s.protector.Decrypt(userID, protected.ID, protected.Amount)
		if err != nil {
			return nil, err
		}
		rates = append(rates, Rate{
			ID: protected.ID, UserID: userID, BoxType: protected.BoxType,
			AmountCents: amountCents, ValidFrom: protected.ValidFrom,
			ValidUntil: protected.ValidUntil,
		})
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	return rates, nil
}
