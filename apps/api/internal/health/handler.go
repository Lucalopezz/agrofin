package health

import (
	"encoding/json"
	"log"
	"net/http"
)

type healthResponse struct {
	Status string `json:"status"`
}

func handleHealth(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	// Este endpoint confirma que o processo HTTP responde; não consulta o banco
	// nem garante que os demais fluxos estejam saudáveis.
	if err := json.NewEncoder(w).Encode(healthResponse{Status: "ok"}); err != nil {
		log.Print("erro ao escrever resposta de health check")
	}
}
