package daemonset

import (
	"encoding/json"
	"net/http"
)

type Handler struct {
	service *Service
}

type response struct {
	Success bool        `json:"success"`
	Message string      `json:"message"`
	Data    interface{} `json:"data,omitempty"`
}

func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	clusterID := r.URL.Query().Get("cluster_id")
	namespace := r.URL.Query().Get("namespace")
	if clusterID == "" || namespace == "" {
		writeJSON(w, http.StatusBadRequest, response{Success: false, Message: "cluster_id and namespace are required"})
		return
	}

	items, err := h.service.List(r.Context(), clusterID, namespace)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, response{Success: false, Message: err.Error()})
		return
	}

	writeJSON(w, http.StatusOK, response{Success: true, Message: "ok", Data: items})
}

func writeJSON(w http.ResponseWriter, status int, payload response) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}
