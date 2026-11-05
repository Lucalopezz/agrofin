package diarias

import "net/http"

func RegisterRoutes(mux *http.ServeMux, handler *Handler, requireSession func(http.Handler) http.Handler) {
	mux.Handle("POST /daily-work", requireSession(http.HandlerFunc(handler.create)))
}
