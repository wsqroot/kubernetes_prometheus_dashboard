package loadkubeconfig

import (
	"encoding/json"
	"net/http"
	"strconv"
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

func (h *Handler) Import(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, response{Success: false, Message: "method not allowed"})
		return
	}

	file, header, err := r.FormFile("file")
	if err != nil {
		writeJSON(w, http.StatusBadRequest, response{Success: false, Message: "file is required"})
		return
	}
	defer file.Close()

	clusters, err := h.service.Import(r.Context(), header.Filename, file)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, response{Success: false, Message: err.Error()})
		return
	}

	writeJSON(w, http.StatusOK, response{
		Success: true,
		Message: "kubeconfig imported",
		Data:    clusters,
	})
}

func (h *Handler) ListClusters(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeJSON(w, http.StatusMethodNotAllowed, response{Success: false, Message: "method not allowed"})
		return
	}

	clusters, err := h.service.ListClusters(r.Context())
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, response{Success: false, Message: err.Error()})
		return
	}

	writeJSON(w, http.StatusOK, response{
		Success: true,
		Message: "ok",
		Data:    clusters,
	})
}

func (h *Handler) Delete(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodDelete {
		writeJSON(w, http.StatusMethodNotAllowed, response{Success: false, Message: "method not allowed"})
		return
	}

	importIDText := r.URL.Query().Get("import_id")
	if importIDText == "" {
		writeJSON(w, http.StatusBadRequest, response{Success: false, Message: "import_id is required"})
		return
	}

	importID, err := strconv.ParseInt(importIDText, 10, 64)
	if err != nil || importID <= 0 {
		writeJSON(w, http.StatusBadRequest, response{Success: false, Message: "invalid import_id"})
		return
	}

	if err := h.service.DeleteImport(r.Context(), importID); err != nil {
		writeJSON(w, http.StatusBadRequest, response{Success: false, Message: err.Error()})
		return
	}

	writeJSON(w, http.StatusOK, response{
		Success: true,
		Message: "kubeconfig deleted",
	})
}

func writeJSON(w http.ResponseWriter, status int, payload response) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}
