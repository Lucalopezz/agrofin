// Adapta a cifra autenticada ao valor e à data de cada diária.
package diarias

import (
	"errors"
	"time"

	"github.com/lucalopezz/agrofin/apps/api/internal/financialcipher"
	calendarDate "github.com/lucalopezz/agrofin/apps/api/internal/platform/date"
)

const dailyWorkAmountPurpose = "agrofin.daily-work.amount.v1"

// dailyWorkPurpose é um marcador de compilação, sem valor em tempo de execução.
// Ele impede que um Amount ou Context de colheitas seja passado à cifra de diária.
type dailyWorkPurpose struct{}

// Cipher adapta o cifrador comum ao vocabulário e aos erros do domínio de diárias.
// A implementação AES-GCM é compartilhada, mas o propósito impede que o envelope
// seja aceito como tarifa ou valor aplicado de uma caixa.
type Cipher struct {
	amounts *financialcipher.Cipher[dailyWorkPurpose]
}

// NewCipher recebe o chaveiro já validado pela composição da aplicação. A diária
// não cria nem carrega uma cópia própria das chaves, o que mantém um único ponto
// de configuração para os domínios financeiros.
func NewCipher(keyring *financialcipher.Keyring) (*Cipher, error) {
	amounts, err := financialcipher.NewCipher[dailyWorkPurpose](keyring)
	if err != nil {
		return nil, ErrInvalidDailyWorkCipherConfig
	}
	return &Cipher{amounts: amounts}, nil
}

// EncryptDailyWorkAmount protege os centavos e autentica os dados que definem a
// diária. A finalidade própria separa esta cifra das tarifas; conta, lançamento,
// funcionário e data impedem copiar o envelope para outro registro ou contexto.
func (c *Cipher) EncryptDailyWorkAmount(
	userID, dailyWorkID, employeeID string,
	workDate time.Time,
	amountCents int64,
) (ProtectedAmount, error) {
	context, err := dailyWorkAmountContext(userID, dailyWorkID, employeeID, workDate)
	if err != nil {
		return ProtectedAmount{}, err
	}
	if c == nil || c.amounts == nil {
		return ProtectedAmount{}, ErrInvalidDailyWorkCipherConfig
	}
	amount, err := c.amounts.Encrypt(context, amountCents)
	return amount, mapDailyWorkCipherError(err)
}

// DecryptDailyWorkAmount reconstrói o contexto de criação e só recupera os centavos
// quando o envelope pertence exatamente à mesma conta, diária, funcionário e data.
// O resumo por período chama este método depois de filtrar e ler os envelopes no Repository.
func (c *Cipher) DecryptDailyWorkAmount(
	userID, dailyWorkID, employeeID string,
	workDate time.Time,
	amount ProtectedAmount,
) (int64, error) {
	context, err := dailyWorkAmountContext(userID, dailyWorkID, employeeID, workDate)
	if err != nil {
		return 0, err
	}
	if c == nil || c.amounts == nil {
		return 0, ErrInvalidDailyWorkCipherConfig
	}
	value, err := c.amounts.Decrypt(context, amount)
	return value, mapDailyWorkCipherError(err)
}

func dailyWorkAmountContext(userID, dailyWorkID, employeeID string, workDate time.Time) (financialcipher.Context[dailyWorkPurpose], error) {
	workDate = calendarDate.Normalize(workDate)
	if workDate.IsZero() {
		return financialcipher.Context[dailyWorkPurpose]{}, ErrInvalidDailyWorkCipherContext
	}
	// O banco guarda data sem horário. Normalizar para meia-noite UTC garante que
	// uma leitura SQL reconstrua a mesma cadeia autenticada independentemente do fuso.
	return financialcipher.Context[dailyWorkPurpose]{
		Purpose:  dailyWorkAmountPurpose,
		UserID:   userID,
		RecordID: dailyWorkID,
		Attributes: []string{
			employeeID,
			workDate.Format(calendarDate.Layout),
		},
	}, nil
}

func mapDailyWorkCipherError(err error) error {
	// O restante do domínio traduz falhas usando seus próprios erros. O mapeamento
	// evita que handlers e serviços de diárias dependam do pacote de infraestrutura;
	// errors.Is também reconhece erros encapsulados com %w.
	switch {
	case err == nil:
		return nil
	case errors.Is(err, financialcipher.ErrInvalidConfig):
		return ErrInvalidDailyWorkCipherConfig
	case errors.Is(err, financialcipher.ErrInvalidContext):
		return ErrInvalidDailyWorkCipherContext
	case errors.Is(err, financialcipher.ErrInvalidAmount):
		return ErrInvalidDailyWorkAmount
	case errors.Is(err, financialcipher.ErrInvalidProtected):
		return ErrInvalidProtectedDailyWorkAmount
	default:
		return err
	}
}
