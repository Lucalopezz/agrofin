// Monta respostas HTTP de tarifas, lançamentos e resumos de colheita.
package colheitas

import (
	"context"
	"encoding/json"
	"errors"
	"log"
	"net/http"

	calendarDate "github.com/lucalopezz/agrofin/apps/api/internal/platform/date"
)

type errorResponse struct {
	Error string `json:"error"`
}

type rateResponse struct {
	ID          string  `json:"id"`
	BoxType     BoxType `json:"box_type"`
	AmountCents int64   `json:"amount_cents"`
	ValidFrom   string  `json:"valid_from"`
	ValidUntil  *string `json:"valid_until"`
}

type harvestResponse struct {
	ID          string                `json:"id"`
	EmployeeID  string                `json:"employee_id"`
	HarvestDate string                `json:"harvest_date"`
	Items       []harvestItemResponse `json:"items"`
}

type harvestItemResponse struct {
	BoxType          BoxType `json:"box_type"`
	Quantity         int32   `json:"quantity"`
	AppliedRateCents int64   `json:"applied_rate_cents"`
}

type harvestSummaryResponse struct {
	Employee           harvestSummaryEmployeeResponse `json:"employee"`
	NormalBoxes        int64                          `json:"normal_boxes"`
	MillingBoxes       int64                          `json:"milling_boxes"`
	NormalAmountCents  int64                          `json:"normal_amount_cents"`
	MillingAmountCents int64                          `json:"milling_amount_cents"`
	HarvestAmountCents int64                          `json:"harvest_amount_cents"`
}

type harvestSummaryEmployeeResponse struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

func writeCreateRateResponse(w http.ResponseWriter, rate Rate, err error) {
	switch {
	case errors.Is(err, ErrInvalidBoxType):
		writeJSON(w, http.StatusBadRequest, errorResponse{Error: "O campo box_type deve ser normal ou moagem"})
	case errors.Is(err, ErrInvalidRateAmount):
		writeJSON(w, http.StatusBadRequest, errorResponse{Error: "O campo amount_cents deve ser maior que zero"})
	case errors.Is(err, ErrInvalidRatePeriod):
		writeJSON(w, http.StatusBadRequest, errorResponse{Error: "A vigência da tarifa é inválida"})
	case errors.Is(err, ErrRateOverlap):
		writeJSON(w, http.StatusConflict, errorResponse{Error: "Já existe uma tarifa desse tipo com vigência sobreposta"})
	case errors.Is(err, context.DeadlineExceeded):
		writeJSON(w, http.StatusGatewayTimeout, errorResponse{Error: "Tempo para cadastrar tarifa esgotado"})
	case errors.Is(err, context.Canceled):
		writeJSON(w, http.StatusRequestTimeout, errorResponse{Error: "Requisição cancelada"})
	case err != nil:
		// Não registrar o erro do driver nem dados da tarifa reduz exposição em logs.
		log.Print("erro ao persistir tarifa protegida")
		writeJSON(w, http.StatusInternalServerError, errorResponse{Error: "Não foi possível cadastrar a tarifa"})
	default:
		writeJSON(w, http.StatusCreated, newRateResponse(rate))
	}
}

func writeListRatesResponse(w http.ResponseWriter, rates []Rate, err error) {
	switch {
	case errors.Is(err, context.DeadlineExceeded):
		writeJSON(w, http.StatusGatewayTimeout, errorResponse{Error: "Tempo para listar tarifas esgotado"})
	case errors.Is(err, context.Canceled):
		writeJSON(w, http.StatusRequestTimeout, errorResponse{Error: "Requisição cancelada"})
	case err != nil:
		// O erro pode conter metadados do banco ou detalhes da falha de autenticação da cifra.
		log.Print("erro ao listar tarifas protegidas")
		writeJSON(w, http.StatusInternalServerError, errorResponse{Error: "Não foi possível listar as tarifas"})
	default:
		response := make([]rateResponse, 0, len(rates))
		for _, rate := range rates {
			response = append(response, newRateResponse(rate))
		}
		writeJSON(w, http.StatusOK, response)
	}
}

func writeCreateHarvestResponse(w http.ResponseWriter, harvest Harvest, err error) {
	switch {
	case errors.Is(err, ErrInvalidHarvestEmployee):
		writeJSON(w, http.StatusBadRequest, errorResponse{Error: "O campo employee_id deve ser um UUID válido"})
	case errors.Is(err, ErrInvalidHarvestOwner), errors.Is(err, ErrInvalidHarvestDate), errors.Is(err, ErrInvalidHarvestItems),
		errors.Is(err, ErrInvalidHarvestQuantity), errors.Is(err, ErrDuplicateHarvestItem),
		errors.Is(err, ErrInvalidBoxType):
		writeJSON(w, http.StatusBadRequest, errorResponse{Error: "Confira funcionário, data, tipos e quantidades do lançamento"})
	case errors.Is(err, ErrRateNotFound):
		writeJSON(w, http.StatusConflict, errorResponse{Error: "Não há tarifa vigente para um dos tipos de caixa na data informada"})
	case errors.Is(err, ErrEmployeeNotFound):
		writeJSON(w, http.StatusNotFound, errorResponse{Error: "Funcionário não encontrado"})
	case errors.Is(err, ErrEmployeeInactive):
		writeJSON(w, http.StatusConflict, errorResponse{Error: "Não é possível registrar colheita para funcionário inativo"})
	case errors.Is(err, context.DeadlineExceeded):
		writeJSON(w, http.StatusGatewayTimeout, errorResponse{Error: "Tempo para registrar colheita esgotado"})
	case errors.Is(err, context.Canceled):
		writeJSON(w, http.StatusRequestTimeout, errorResponse{Error: "Requisição cancelada"})
	case err != nil:
		// O erro pode conter dados do banco ou detalhe de cifra; só registramos o fluxo.
		log.Print("erro ao persistir lançamento de colheita")
		writeJSON(w, http.StatusInternalServerError, errorResponse{Error: "Não foi possível registrar a colheita"})
	default:
		response := newHarvestResponse(harvest)
		writeJSON(w, http.StatusCreated, response)
	}
}

func writeListHarvestEntriesResponse(w http.ResponseWriter, harvests []Harvest, err error) {
	switch {
	case errors.Is(err, ErrInvalidHarvestPeriod):
		writeJSON(w, http.StatusBadRequest, errorResponse{Error: "end_date deve ser igual ou posterior a start_date"})
	case errors.Is(err, context.DeadlineExceeded):
		writeJSON(w, http.StatusGatewayTimeout, errorResponse{Error: "Tempo para consultar colheitas esgotado"})
	case errors.Is(err, context.Canceled):
		writeJSON(w, http.StatusRequestTimeout, errorResponse{Error: "Requisição cancelada"})
	case err != nil:
		// Um envelope adulterado ou falha de leitura invalida a consulta inteira;
		// não devolvemos uma lista parcial nem detalhes financeiros nos logs.
		log.Print("erro ao listar lançamentos de colheita protegidos")
		writeJSON(w, http.StatusInternalServerError, errorResponse{Error: "Não foi possível consultar as colheitas"})
	default:
		response := make([]harvestResponse, 0, len(harvests))
		for _, harvest := range harvests {
			response = append(response, newHarvestResponse(harvest))
		}
		writeJSON(w, http.StatusOK, response)
	}
}

func writeHarvestSummaryResponse(w http.ResponseWriter, summaries []HarvestSummary, err error) {
	switch {
	case errors.Is(err, ErrInvalidHarvestPeriod):
		writeJSON(w, http.StatusBadRequest, errorResponse{Error: "end_date deve ser igual ou posterior a start_date"})
	case errors.Is(err, context.DeadlineExceeded):
		writeJSON(w, http.StatusGatewayTimeout, errorResponse{Error: "Tempo para resumir colheitas esgotado"})
	case errors.Is(err, context.Canceled):
		writeJSON(w, http.StatusRequestTimeout, errorResponse{Error: "Requisição cancelada"})
	case err != nil:
		// O resumo depende da autenticação das tarifas aplicadas; uma falha invalida a lista inteira.
		log.Print("erro ao resumir lançamentos de colheita protegidos")
		writeJSON(w, http.StatusInternalServerError, errorResponse{Error: "Não foi possível resumir as colheitas"})
	default:
		response := make([]harvestSummaryResponse, 0, len(summaries))
		for _, summary := range summaries {
			response = append(response, harvestSummaryResponse{
				Employee: harvestSummaryEmployeeResponse{
					ID: summary.EmployeeID, Name: summary.EmployeeName,
				},
				NormalBoxes: summary.NormalBoxes, MillingBoxes: summary.MillingBoxes,
				NormalAmountCents:  summary.NormalAmountCents,
				MillingAmountCents: summary.MillingAmountCents,
				HarvestAmountCents: summary.HarvestAmountCents,
			})
		}
		writeJSON(w, http.StatusOK, response)
	}
}

func newRateResponse(rate Rate) rateResponse {
	response := rateResponse{
		ID: rate.ID, BoxType: rate.BoxType, AmountCents: rate.AmountCents,
		ValidFrom: rate.ValidFrom.Format(calendarDate.Layout),
	}
	if rate.ValidUntil != nil {
		validUntil := rate.ValidUntil.Format(calendarDate.Layout)
		response.ValidUntil = &validUntil
	}
	return response
}

func newHarvestResponse(harvest Harvest) harvestResponse {
	items := make([]harvestItemResponse, 0, len(harvest.Items))
	for _, item := range harvest.Items {
		items = append(items, harvestItemResponse{
			BoxType: item.BoxType, Quantity: item.Quantity,
			AppliedRateCents: item.AppliedRateCents,
		})
	}
	return harvestResponse{
		ID: harvest.ID, EmployeeID: harvest.EmployeeID,
		HarvestDate: harvest.HarvestDate.Format(calendarDate.Layout), Items: items,
	}
}

func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	// Respostas de tarifas e lançamentos contêm dados financeiros; não os armazenamos em cache.
	w.Header().Set("Cache-Control", "no-store")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(value); err != nil {
		// O status já foi enviado; uma segunda resposta produziria JSON inválido.
		log.Print("erro ao escrever resposta de tarifas")
	}
}
