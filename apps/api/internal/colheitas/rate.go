// Package colheitas define tarifas gerais e suas vigências.
package colheitas

import (
	"time"
)

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
