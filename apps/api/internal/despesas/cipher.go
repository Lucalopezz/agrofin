// Protege e recupera valores de despesas com contexto autenticado.
package despesas

import (
	"github.com/lucalopezz/agrofin/apps/api/internal/financialcipher"
	calendarDate "github.com/lucalopezz/agrofin/apps/api/internal/platform/date"
)

// Cipher adapta a cifra financeira compartilhada ao contexto de uma despesa.
type Cipher struct {
	amounts *financialcipher.Cipher[expensePurpose]
}

func NewCipher(keys *financialcipher.Keyring) (*Cipher, error) {
	amounts, err := financialcipher.NewCipher[expensePurpose](keys)
	if err != nil {
		return nil, err
	}
	return &Cipher{amounts: amounts}, nil
}

// O contexto precisa ser idêntico na gravação e na leitura para que dados já
// cifrados continuem legíveis e não possam ser movidos para outra despesa.
func expenseAmountContext(e Expense) financialcipher.Context[expensePurpose] {
	return financialcipher.Context[expensePurpose]{
		Purpose: "agrofin.expense.amount.v1", UserID: e.UserID, RecordID: e.ID,
		Attributes: []string{e.PaidAt.Format(calendarDate.Layout), e.Category},
	}
}

func (c *Cipher) protect(e Expense) (protectedExpense, error) {
	// A data e a categoria também são autenticadas: uma edição gera novo envelope.
	amount, err := c.amounts.Encrypt(expenseAmountContext(e), e.AmountCents)
	if err != nil {
		return protectedExpense{}, err
	}
	// O repositório só recebe o envelope; o valor em claro não atravessa essa fronteira.
	e.AmountCents = 0
	return protectedExpense{Expense: e, Amount: amount}, nil
}

func (c *Cipher) open(p protectedExpense) (Expense, error) {
	e := p.Expense
	// O banco pode devolver o instante em outro fuso; a data foi gravada em UTC.
	e.PaidAt = e.PaidAt.UTC()
	amount, err := c.amounts.Decrypt(expenseAmountContext(e), p.Amount)
	if err != nil {
		return Expense{}, err
	}
	e.AmountCents = amount
	return e, nil
}
