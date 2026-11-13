// Protege preços, descontos e recebimentos com contextos próprios de venda.
package vendas

import (
	"errors"
	"time"

	"github.com/lucalopezz/agrofin/apps/api/internal/financialcipher"
	calendarDate "github.com/lucalopezz/agrofin/apps/api/internal/platform/date"
)

const saleMoneyPurpose = "agrofin.sale.money.v1"

type salePurpose struct{}

type Cipher struct {
	amounts *financialcipher.Cipher[salePurpose]
}

func NewCipher(keyring *financialcipher.Keyring) (*Cipher, error) {
	amounts, err := financialcipher.NewCipher[salePurpose](keyring)
	if err != nil {
		return nil, ErrInvalidSaleCipherConfig
	}
	return &Cipher{amounts: amounts}, nil
}

func (c *Cipher) encrypt(purpose, userID, saleID, recordID string, attrs []string, amount int64) (ProtectedAmount, error) {
	ctx, err := saleMoneyContext(purpose, userID, saleID, recordID, attrs)
	if err != nil {
		return ProtectedAmount{}, err
	}
	if c == nil || c.amounts == nil {
		return ProtectedAmount{}, ErrInvalidSaleCipherConfig
	}
	protected, err := c.amounts.Encrypt(ctx, amount)
	return protected, mapSaleCipherError(err)
}

func (c *Cipher) decrypt(purpose, userID, saleID, recordID string, attrs []string, amount ProtectedAmount) (int64, error) {
	ctx, err := saleMoneyContext(purpose, userID, saleID, recordID, attrs)
	if err != nil {
		return 0, err
	}
	if c == nil || c.amounts == nil {
		return 0, ErrInvalidSaleCipherConfig
	}
	value, err := c.amounts.Decrypt(ctx, amount)
	return value, mapSaleCipherError(err)
}

func saleMoneyContext(purpose, userID, saleID, recordID string, attrs []string) (financialcipher.Context[salePurpose], error) {
	if purpose == "" || userID == "" || saleID == "" || recordID == "" {
		return financialcipher.Context[salePurpose]{}, ErrInvalidSaleCipherContext
	}
	return financialcipher.Context[salePurpose]{Purpose: purpose, UserID: userID, RecordID: recordID, Attributes: attrs}, nil
}

func saleDateAttribute(soldAt time.Time) string {
	return calendarDate.Normalize(soldAt).Format(calendarDate.Layout)
}

func mapSaleCipherError(err error) error {
	switch {
	case err == nil:
		return nil
	case errors.Is(err, financialcipher.ErrInvalidConfig):
		return ErrInvalidSaleCipherConfig
	case errors.Is(err, financialcipher.ErrInvalidContext):
		return ErrInvalidSaleCipherContext
	case errors.Is(err, financialcipher.ErrInvalidAmount):
		return ErrInvalidSalePrice
	case errors.Is(err, financialcipher.ErrInvalidProtected):
		return ErrInvalidProtectedSaleAmount
	default:
		return err
	}
}
