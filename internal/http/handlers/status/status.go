package status

import (
	"encoding/json"
	"net/http"

	"ss-api/internal/app"
)

type Handler struct {
	app *app.App
}

func New(appInstance *app.App) http.HandlerFunc {
	h := Handler{app: appInstance}
	return h.handle
}

func (h Handler) handle(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeJSONError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	if path := r.URL.Path; path != "/stella" && path != "/stella/" {
		writeJSONError(w, http.StatusNotFound, "not found")
		return
	}

	response := struct {
		Status    int      `json:"status"`
		Uptime    int64    `json:"uptime"`
		Endpoints []string `json:"endpoints"`
	}{
		Status:    http.StatusOK,
		Uptime:    h.app.StartTime().Unix(),
		Endpoints: h.app.Endpoints(),
	}

	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	_ = json.NewEncoder(w).Encode(response)
}

func writeJSONError(w http.ResponseWriter, status int, message string) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(map[string]string{
		"error": message,
	})
}
