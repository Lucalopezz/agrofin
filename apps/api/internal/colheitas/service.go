package colheitas

import (
	"context"
	"strings"
	"time"
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

// ProtectedRate é a entidade pronta para persistir. Não contém AmountCents:
// o Repository recebe somente o envelope e os metadados já validados.
type ProtectedRate struct {
	ID         string
	UserID     string
	BoxType    BoxType
	Amount     ProtectedRateAmount
	ValidFrom  time.Time
	ValidUntil *time.Time
}

// As interfaces ficam junto do consumidor e não exigem PostgreSQL nos testes.
// O Repository concreto ainda será implementado; ele deverá manter a restrição
// de sobreposição e traduzir a disputa concorrente para ErrRateOverlap.
type rateRepository interface {
	HasRateOverlap(ctx context.Context, period RatePeriod) (bool, error)
	NextRateID(ctx context.Context) (string, error)
	CreateRate(ctx context.Context, rate ProtectedRate) error
}

type rateAmountProtector interface {
	Encrypt(userID, rateID string, amountCents int64) (ProtectedRateAmount, error)
}

// Service coordena o cadastro de tarifas gerais da conta; funcionários da mesma
// conta usam a tarifa vigente do tipo, sem exceções individuais nesta etapa.
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
	// O ID deve existir antes da cifra para autenticar também o registro. A futura
	// implementação obterá gen_random_uuid() do PostgreSQL sem inserir uma tarifa
	// parcial; a gravação usará esse mesmo ID depois de o valor estar protegido.
	rate.ID, err = s.repo.NextRateID(ctx)
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
