package funcionarios

import (
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"mime"
	"net/http"
	"strings"
)

// Cadastro e renomeação compartilham o mesmo contrato de entrada: somente name.
type employeeNameRequest struct {
	Name string `json:"name"`
}

func decodeEmployeeNameRequest(w http.ResponseWriter, r *http.Request) (*employeeNameRequest, bool) {
	mediaType, _, err := mime.ParseMediaType(r.Header.Get("Content-Type"))
	if err != nil || mediaType != "application/json" {
		writeJSON(w, http.StatusUnsupportedMediaType, errorResponse{Error: "Envie o corpo como application/json"})
		return nil, false
	}

	// Cadastro e renomeação contêm apenas um nome. Limitamos o corpo, incluindo espaços
	// extras, para não consumir memória sem limite ao decodificar a requisição.
	r.Body = http.MaxBytesReader(w, r.Body, 4*1024)
	decoder := json.NewDecoder(r.Body)
	// O decodificador não ignora campos extras, que podem indicar erro do cliente.
	decoder.DisallowUnknownFields()
	var input *employeeNameRequest

	// Decodifica o JSON em input. Se houver erro, input permanece nil e err descreve o problema.
	err = decoder.Decode(&input)
	if err == nil && input != nil {
		// Um segundo JSON ou conteúdo extra não pode ser ignorado depois do nome.
		// {"name": "João"}{"name": "Maria"}
		var extra any
		if trailingErr := decoder.Decode(&extra); trailingErr != io.EOF {
			if trailingErr == nil {
				err = errors.New("mais de um valor json")
			} else {
				err = trailingErr
			}
		}
	}
	var sizeErr *http.MaxBytesError
	if errors.As(err, &sizeErr) {
		writeJSON(w, http.StatusRequestEntityTooLarge, errorResponse{Error: "Corpo da requisição excede 4 KiB"})
		return nil, false
	}
	if err != nil || input == nil {
		writeJSON(w, http.StatusBadRequest, errorResponse{Error: "Envie um único objeto JSON contendo somente name"})
		return nil, false
	}

	return input, true
}

// decodeEmployeeID exige o formato UUID usado nas respostas da API. A validação
// do caminho pertence ao protocolo e evita enviar um UUID malformado ao banco.
func decodeEmployeeID(w http.ResponseWriter, r *http.Request) (string, bool) {
	id := r.PathValue("id")
	if len(id) == 36 && id[8] == '-' && id[13] == '-' && id[18] == '-' && id[23] == '-' {
		decoded, err := hex.DecodeString(strings.ReplaceAll(id, "-", ""))
		// O tamanho também impede aceitar hífens extras no lugar de dígitos.
		if err == nil && len(decoded) == 16 {
			return id, true
		}
	}
	writeJSON(w, http.StatusBadRequest, errorResponse{Error: "Identificador do funcionário deve ser um UUID válido"})
	return "", false
}
