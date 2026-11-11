// Decodifica filtros, IDs e corpos HTTP de funcionários.
package funcionarios

import (
	"errors"
	"io"
	"net/http"
	"net/url"

	"github.com/lucalopezz/agrofin/apps/api/internal/platform/httpjson"
	"github.com/lucalopezz/agrofin/apps/api/internal/platform/uuid"
)

// Cadastro e renomeação compartilham o mesmo contrato de entrada: somente name.
type employeeNameRequest struct {
	Name string `json:"name"`
}

// Retorna ponteiro para bool, pois pode ser null, true ou false. O ponteiro distingue explicitamente false de parâmetro ausente (nil).
func decodeListEmployeesRequest(w http.ResponseWriter, r *http.Request) (*bool, bool) {
	// URL.Query ignora erros de decodificação. Não podemos transformar uma
	// query malformada em ausência de filtro e devolver ambos os estados.
	query, err := url.ParseQuery(r.URL.RawQuery)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, errorResponse{Error: "Parâmetros de consulta inválidos"})
		return nil, false
	}
	values, present := query["active"]
	if !present {
		return nil, true
	}
	// Um valor repetido é ambíguo, mesmo quando os dois valores são iguais.
	if len(values) != 1 || (values[0] != "true" && values[0] != "false") {
		writeJSON(w, http.StatusBadRequest, errorResponse{Error: "Informe active uma única vez com true ou false"})
		return nil, false
	}
	// O ponteiro distingue explicitamente false de parâmetro ausente (nil).
	active := values[0] == "true"
	return &active, true
}

func decodeEmployeeNameRequest(w http.ResponseWriter, r *http.Request) (*employeeNameRequest, bool) {
	input, err := httpjson.Decode[employeeNameRequest](w, r)
	switch {
	case errors.Is(err, httpjson.ErrUnsupportedMediaType):
		writeJSON(w, http.StatusUnsupportedMediaType, errorResponse{Error: "Envie o corpo como application/json"})
		return nil, false
	case errors.Is(err, httpjson.ErrBodyTooLarge):
		writeJSON(w, http.StatusRequestEntityTooLarge, errorResponse{Error: "Corpo da requisição excede 4 KiB"})
		return nil, false
	case err != nil || input == nil:
		writeJSON(w, http.StatusBadRequest, errorResponse{Error: "Envie um único objeto JSON contendo somente name"})
		return nil, false
	}

	return input, true
}

// decodeEmployeeID exige o formato UUID usado nas respostas da API. A validação
// do caminho pertence ao protocolo e evita enviar um UUID malformado ao banco.
func decodeEmployeeID(w http.ResponseWriter, r *http.Request) (string, bool) {
	id, err := uuid.Normalize(r.PathValue("id"))
	if err == nil {
		return id, true
	}
	writeJSON(w, http.StatusBadRequest, errorResponse{Error: "Identificador do funcionário deve ser um UUID válido"})
	return "", false
}

func decodeEmployeeStateChangeRequest(w http.ResponseWriter, r *http.Request) (string, bool) {
	id, ok := decodeEmployeeID(w, r)
	if !ok {
		return "", false
	}
	// Desativação e reativação usam somente o caminho e a sessão. Ler até um byte rejeita
	// qualquer corpo sem carregá-lo inteiro, inclusive com tamanho desconhecido.
	body, err := io.ReadAll(io.LimitReader(r.Body, 1))
	if err != nil || len(body) != 0 {
		writeJSON(w, http.StatusBadRequest, errorResponse{Error: "Envie a alteração de estado sem corpo na requisição"})
		return "", false
	}
	return id, true
}
