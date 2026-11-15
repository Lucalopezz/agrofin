// Define despesas pagas e suas representações protegidas.
package despesas

import (
	"time"

	"github.com/lucalopezz/agrofin/apps/api/internal/financialcipher"
)

type (
	expensePurpose  struct{}
	protectedAmount = financialcipher.Amount[expensePurpose]
)

type Expense struct {
	ID, UserID, Category, Description string
	PaidAt                            time.Time
	AmountCents                       int64
}
type Input struct {
	PaidAt                time.Time
	Category, Description string
	AmountCents           int64
}
type protectedExpense struct {
	Expense Expense
	Amount  protectedAmount
}
