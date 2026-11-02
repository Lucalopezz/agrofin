// Package colheitas contém o domínio dos lançamentos de colheita e suas tarifas.
package colheitas

import "time"

// BoxType classifica a colheita para pagamento, separado da classificação de venda.
type BoxType string

const (
	BoxTypeNormal BoxType = "normal"
	BoxTypeMoagem BoxType = "moagem"
)

// Rate representa uma tarifa em memória. AmountCents não deve ser persistido
// em texto claro;
type Rate struct {
	ID          string
	UserID      string
	BoxType     BoxType
	AmountCents int64

	// As datas representam dias de calendário: início inclusivo e fim exclusivo.
	// ValidUntil nil indica uma vigência sem data final definida.
	ValidFrom  time.Time
	ValidUntil *time.Time
}

// normalizeRate reúne as validações locais para o futuro Service montar a tarifa
// antes de persistir. Não consulta outras tarifas nem decide escopo individual.
func normalizeRate(rate Rate) (Rate, error) {
	switch rate.BoxType {
	case BoxTypeNormal, BoxTypeMoagem:
	default:
		return Rate{}, ErrInvalidBoxType
	}

	if rate.AmountCents <= 0 {
		return Rate{}, ErrInvalidRateAmount
	}
	rate.ValidFrom = rateDate(rate.ValidFrom)
	if rate.ValidFrom.IsZero() {
		return Rate{}, ErrInvalidRatePeriod
	}

	if rate.ValidUntil != nil {
		until := rateDate(*rate.ValidUntil)
		if until.IsZero() || !until.After(rate.ValidFrom) {
			return Rate{}, ErrInvalidRatePeriod
		}
		// Copiar a data evita compartilhar o ponteiro recebido: uma alteração no
		// argumento do chamador não pode mudar depois a vigência que foi validada.
		rate.ValidUntil = &until
	}
	return rate, nil
}

func rateDate(date time.Time) time.Time {
	// Extrair os componentes antes de usar UTC preserva o dia informado. Converter
	// o instante para UTC primeiro poderia deslocar a data para o dia anterior/seguinte.
	year, month, day := date.Date()
	return time.Date(year, month, day, 0, 0, 0, 0, time.UTC)
}
