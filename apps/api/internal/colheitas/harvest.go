package colheitas

import (
	"encoding/hex"
	"strings"
	"time"
)

// CreateHarvestInput contém os dados de produção enviados pelo cliente. O dono
// vem da sessão; nenhuma identidade de conta é aceita no lançamento.
type CreateHarvestInput struct {
	EmployeeID  string
	HarvestDate time.Time
	Items       []HarvestItemInput
}

type HarvestItemInput struct {
	BoxType  BoxType
	Quantity int32
}

// HarvestPeriod delimita uma consulta por datas de calendário, com início e fim inclusivos.
type HarvestPeriod struct {
	StartDate time.Time
	EndDate   time.Time
}

// Harvest é o agregado retornado pelo serviço. Cada tipo tem sua própria
// fotografia da tarifa vigente no dia do lançamento.
type Harvest struct {
	ID          string
	UserID      string
	EmployeeID  string
	HarvestDate time.Time
	Items       []HarvestItem
}

type HarvestItem struct {
	BoxType          BoxType
	Quantity         int32
	AppliedRateCents int64
}

// ProtectedHarvest é a forma pronta para persistência; valores financeiros
// chegam ao Repository somente cifrados.
type ProtectedHarvest struct {
	ID          string
	UserID      string
	EmployeeID  string
	HarvestDate time.Time
	Items       []ProtectedHarvestItem
}

// ProtectedHarvestRecord inclui o nome atual do funcionário para projeções de leitura.
type ProtectedHarvestRecord struct {
	Harvest      ProtectedHarvest
	EmployeeName string
}

// HarvestSummary agrupa caixas e valor gerado pela colheita, antes de pagamentos.
type HarvestSummary struct {
	EmployeeID         string
	EmployeeName       string
	NormalBoxes        int64
	MillingBoxes       int64
	NormalAmountCents  int64
	MillingAmountCents int64
	HarvestAmountCents int64
}

type ProtectedHarvestItem struct {
	BoxType       BoxType
	Quantity      int32
	AppliedAmount ProtectedRateAmount
}

func normalizeHarvestPeriod(period HarvestPeriod) (HarvestPeriod, error) {
	period.StartDate = rateDate(period.StartDate)
	period.EndDate = rateDate(period.EndDate)
	// A data de início e fim devem ser válidas e o fim não pode ser anterior ao início.
	if period.StartDate.IsZero() || period.EndDate.IsZero() || period.EndDate.Before(period.StartDate) {
		return HarvestPeriod{}, ErrInvalidHarvestPeriod
	}
	return period, nil
}

func normalizeHarvest(harvest Harvest) (Harvest, error) {
	employeeID, err := normalizeHarvestEmployeeID(harvest.EmployeeID)
	if err != nil {
		return Harvest{}, err
	}
	harvest.EmployeeID = employeeID
	harvest.HarvestDate = rateDate(harvest.HarvestDate)
	if harvest.HarvestDate.IsZero() {
		return Harvest{}, ErrInvalidHarvestDate
	}
	if len(harvest.Items) == 0 {
		return Harvest{}, ErrInvalidHarvestItems
	}

	// Há somente dois tipos de caixa no domínio. Impedir repetição deixa cada
	// tipo em uma única linha do agregado, conforme a restrição do banco.
	seen := make(map[BoxType]struct{}, len(harvest.Items))
	items := make([]HarvestItem, 0, len(harvest.Items))
	for _, item := range harvest.Items {
		switch item.BoxType {
		case BoxTypeNormal, BoxTypeMoagem:
		default:
			return Harvest{}, ErrInvalidBoxType
		}
		if item.Quantity <= 0 {
			return Harvest{}, ErrInvalidHarvestQuantity
		}
		if _, exists := seen[item.BoxType]; exists {
			return Harvest{}, ErrDuplicateHarvestItem
		}
		seen[item.BoxType] = struct{}{}
		items = append(items, item)
	}
	harvest.Items = items
	return harvest, nil
}

func normalizeHarvestEmployeeID(value string) (string, error) {
	if len(value) != 36 || value[8] != '-' || value[13] != '-' || value[18] != '-' || value[23] != '-' || strings.TrimSpace(value) != value {
		return "", ErrInvalidHarvestEmployee
	}
	decoded, err := hex.DecodeString(strings.ReplaceAll(value, "-", ""))
	if err != nil || len(decoded) != 16 {
		return "", ErrInvalidHarvestEmployee
	}
	// PostgreSQL devolve UUIDs em minúsculas; usar a mesma forma no AAD garante
	// que a cifra continue verificável quando o lançamento for lido depois.
	return strings.ToLower(value), nil
}
