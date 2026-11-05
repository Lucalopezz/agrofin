package colheitas

import (
	"errors"
	"strconv"
	"time"

	"github.com/lucalopezz/agrofin/apps/api/internal/financialcipher"
)

const (
	ratePurpose            = "agrofin.harvest-rate.amount.v1"
	harvestItemRatePurpose = "agrofin.harvest-entry.applied-rate.v1"
)

// rateCipherPurpose é um marcador de compilação compartilhado pelas cifras de
// tarifa e colheita. Os propósitos em string continuam separados no AAD em execução.
type rateCipherPurpose struct{}

// ProtectedRateAmount mantém o nome conhecido pelo pacote de colheitas, mas usa
// o envelope comum. Assim, os chamadores não precisam conhecer a estrutura interna
// compartilhada nem mudar a forma das colunas de persistência.
type ProtectedRateAmount = financialcipher.Amount[rateCipherPurpose]

// RateCipher aplica contextos próprios às tarifas e às fotografias de colheita.
// A implementação criptográfica comum fica fora de um domínio específico.
type RateCipher struct {
	amounts *financialcipher.Cipher[rateCipherPurpose]
}

// NewRateCipher mantém o construtor histórico de colheitas e delega a validação
// criptográfica ao pacote comum. Ele é útil para chamadores independentes; na
// aplicação, a composição usa NewRateCipherWithKeyring para compartilhar o mesmo
// chaveiro já validado com diárias.
func NewRateCipher(activeVersion string, keys map[string][]byte) (*RateCipher, error) {
	keyring, err := financialcipher.NewKeyring(activeVersion, keys)
	if err != nil {
		return nil, ErrInvalidRateCipherConfig
	}
	return NewRateCipherWithKeyring(keyring)
}

// NewRateCipherWithKeyring injeta o chaveiro preparado pela composição da API.
// A reutilização evita manter duas cópias das chaves e das cifras em memória; cada
// método de colheitas ainda usa propósitos próprios no AAD, separados das diárias.
func NewRateCipherWithKeyring(keyring *financialcipher.Keyring) (*RateCipher, error) {
	amounts, err := financialcipher.NewCipher[rateCipherPurpose](keyring)
	if err != nil {
		return nil, ErrInvalidRateCipherConfig
	}
	return &RateCipher{amounts: amounts}, nil
}

// Encrypt protege uma tarifa geral vinculando os centavos à conta e ao ID da tarifa.
// O ID precisa existir antes da gravação porque participa da autenticação do envelope.
func (c *RateCipher) Encrypt(userID, rateID string, amountCents int64) (ProtectedRateAmount, error) {
	if c == nil || c.amounts == nil {
		return ProtectedRateAmount{}, ErrInvalidRateCipherConfig
	}
	amount, err := c.amounts.Encrypt(financialcipher.Context[rateCipherPurpose]{
		Purpose: ratePurpose, UserID: userID, RecordID: rateID,
	}, amountCents)
	return amount, mapRateCipherError(err)
}

// Decrypt reconstrói o mesmo propósito e contexto usados na criação da tarifa.
// Se a conta, o ID ou os bytes do envelope não corresponderem, o pacote comum rejeita a leitura.
func (c *RateCipher) Decrypt(userID, rateID string, amount ProtectedRateAmount) (int64, error) {
	if c == nil || c.amounts == nil {
		return 0, ErrInvalidRateCipherConfig
	}
	value, err := c.amounts.Decrypt(financialcipher.Context[rateCipherPurpose]{
		Purpose: ratePurpose, UserID: userID, RecordID: rateID,
	}, amount)
	return value, mapRateCipherError(err)
}

// EncryptHarvestItemRate registra a fotografia da tarifa que valeu no lançamento.
// Além do dono e do ID da colheita, autentica funcionário, data, tipo e quantidade:
// mudar qualquer um desses dados sem refazer a cifra invalida a leitura do valor.
func (c *RateCipher) EncryptHarvestItemRate(
	userID, harvestID, employeeID string,
	harvestDate time.Time,
	boxType BoxType,
	quantity int32,
	amountCents int64,
) (ProtectedRateAmount, error) {
	context, err := harvestItemRateContext(userID, harvestID, employeeID, harvestDate, boxType, quantity)
	if err != nil {
		return ProtectedRateAmount{}, err
	}
	if c == nil || c.amounts == nil {
		return ProtectedRateAmount{}, ErrInvalidRateCipherConfig
	}
	amount, err := c.amounts.Encrypt(context, amountCents)
	return amount, mapRateCipherError(err)
}

// DecryptHarvestItemRate exige os mesmos campos usados ao cifrar a fotografia.
// Isso mantém o valor aplicado preso ao item original mesmo se a tarifa geral mudar depois.
func (c *RateCipher) DecryptHarvestItemRate(
	userID, harvestID, employeeID string,
	harvestDate time.Time,
	boxType BoxType,
	quantity int32,
	amount ProtectedRateAmount,
) (int64, error) {
	context, err := harvestItemRateContext(userID, harvestID, employeeID, harvestDate, boxType, quantity)
	if err != nil {
		return 0, err
	}
	if c == nil || c.amounts == nil {
		return 0, ErrInvalidRateCipherConfig
	}
	value, err := c.amounts.Decrypt(context, amount)
	return value, mapRateCipherError(err)
}

func harvestItemRateContext(
	userID, harvestID, employeeID string,
	harvestDate time.Time,
	boxType BoxType,
	quantity int32,
) (financialcipher.Context[rateCipherPurpose], error) {
	// Data e quantidade fazem parte da identidade do item para a cifra. Rejeitá-las
	// aqui evita produzir envelopes que não poderiam ser reconstruídos numa consulta.
	if harvestDate.IsZero() || quantity <= 0 {
		return financialcipher.Context[rateCipherPurpose]{}, ErrInvalidRateCipherContext
	}
	if boxType != BoxTypeNormal && boxType != BoxTypeMoagem {
		return financialcipher.Context[rateCipherPurpose]{}, ErrInvalidBoxType
	}
	// A data é reduzida ao dia de calendário e a quantidade é texto decimal estável;
	// essa representação precisa permanecer igual para preservar a leitura histórica.
	return financialcipher.Context[rateCipherPurpose]{
		Purpose:  harvestItemRatePurpose,
		UserID:   userID,
		RecordID: harvestID,
		Attributes: []string{
			employeeID,
			rateDate(harvestDate).Format("2006-01-02"),
			string(boxType),
			strconv.FormatInt(int64(quantity), 10),
		},
	}, nil
}

func mapRateCipherError(err error) error {
	// O serviço de colheitas conhece erros do próprio domínio, não os detalhes do
	// pacote compartilhado. errors.Is preserva o mapeamento mesmo se o erro for envolvido.
	switch {
	case err == nil:
		return nil
	case errors.Is(err, financialcipher.ErrInvalidConfig):
		return ErrInvalidRateCipherConfig
	case errors.Is(err, financialcipher.ErrInvalidContext):
		return ErrInvalidRateCipherContext
	case errors.Is(err, financialcipher.ErrInvalidAmount):
		return ErrInvalidRateAmount
	case errors.Is(err, financialcipher.ErrInvalidProtected):
		return ErrInvalidProtectedRateAmount
	default:
		return err
	}
}
