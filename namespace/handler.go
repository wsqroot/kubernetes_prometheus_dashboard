package namespace

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

type applyYAMLRequest struct {
	Content string `json:"content"`
}

type podExecRequest struct {
	Namespace  string `json:"namespace"`
	PodName    string `json:"pod_name"`
	Container  string `json:"container"`
	Command    string `json:"command"`
	WorkingDir string `json:"working_dir"`
}

type podLogsRequest struct {
	Namespace string `json:"namespace"`
	PodName   string `json:"pod_name"`
	Container string `json:"container"`
}

type podLabelsRequest struct {
	Namespace string            `json:"namespace"`
	PodName   string            `json:"pod_name"`
	Labels    map[string]string `json:"labels"`
}

type podDeleteRequest struct {
	Namespace string `json:"namespace"`
	PodName   string `json:"pod_name"`
}

type nodeLabelsRequest struct {
	NodeName string            `json:"node_name"`
	Labels   map[string]string `json:"labels"`
}

type nodeExecRequest struct {
	NodeName   string `json:"node_name"`
	Command    string `json:"command"`
	WorkingDir string `json:"working_dir"`
}

type resourceDeleteRequest struct {
	Kind      string `json:"kind"`
	Name      string `json:"name"`
	Namespace string `json:"namespace"`
}

func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

func (h *Handler) Overview(w http.ResponseWriter, r *http.Request) {
	clusterID := r.URL.Query().Get("cluster_id")
	if clusterID == "" {
		writeJSON(w, http.StatusBadRequest, response{Success: false, Message: "cluster_id is required"})
		return
	}

	overview, err := h.service.GetOverview(r.Context(), clusterID)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, response{Success: false, Message: err.Error()})
		return
	}

	writeJSON(w, http.StatusOK, response{Success: true, Message: "ok", Data: overview})
}

func (h *Handler) ListNamespaces(w http.ResponseWriter, r *http.Request) {
	clusterID := r.URL.Query().Get("cluster_id")
	if clusterID == "" {
		writeJSON(w, http.StatusBadRequest, response{Success: false, Message: "cluster_id is required"})
		return
	}

	namespaces, err := h.service.ListNamespaces(r.Context(), clusterID)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, response{Success: false, Message: err.Error()})
		return
	}

	writeJSON(w, http.StatusOK, response{
		Success: true,
		Message: "ok",
		Data: map[string]interface{}{
			"items": namespaces,
			"total": len(namespaces),
		},
	})
}

func (h *Handler) Resources(w http.ResponseWriter, r *http.Request) {
	clusterID := r.URL.Query().Get("cluster_id")
	namespace := r.URL.Query().Get("namespace")
	if clusterID == "" || namespace == "" {
		writeJSON(w, http.StatusBadRequest, response{Success: false, Message: "cluster_id and namespace are required"})
		return
	}

	resources, err := h.service.GetNamespaceResources(r.Context(), clusterID, namespace)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, response{Success: false, Message: err.Error()})
		return
	}

	writeJSON(w, http.StatusOK, response{Success: true, Message: "ok", Data: resources})
}

func (h *Handler) Nodes(w http.ResponseWriter, r *http.Request) {
	clusterID := r.URL.Query().Get("cluster_id")
	if clusterID == "" {
		writeJSON(w, http.StatusBadRequest, response{Success: false, Message: "cluster_id is required"})
		return
	}

	nodes, err := h.service.GetNodeResources(r.Context(), clusterID)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, response{Success: false, Message: err.Error()})
		return
	}

	writeJSON(w, http.StatusOK, response{Success: true, Message: "ok", Data: nodes})
}

func (h *Handler) Pods(w http.ResponseWriter, r *http.Request) {
	clusterID := r.URL.Query().Get("cluster_id")
	namespace := r.URL.Query().Get("namespace")
	if clusterID == "" || namespace == "" {
		writeJSON(w, http.StatusBadRequest, response{Success: false, Message: "cluster_id and namespace are required"})
		return
	}

	pods, err := h.service.GetPodResources(r.Context(), clusterID, namespace)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, response{Success: false, Message: err.Error()})
		return
	}

	writeJSON(w, http.StatusOK, response{Success: true, Message: "ok", Data: pods})
}

func (h *Handler) PodExec(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, response{Success: false, Message: "method not allowed"})
		return
	}

	clusterID := r.URL.Query().Get("cluster_id")
	if clusterID == "" {
		writeJSON(w, http.StatusBadRequest, response{Success: false, Message: "cluster_id is required"})
		return
	}

	var req podExecRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, response{Success: false, Message: "invalid request body"})
		return
	}

	result, err := h.service.ExecPodCommand(r.Context(), clusterID, req.Namespace, req.PodName, req.Container, req.Command, req.WorkingDir)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, response{Success: false, Message: err.Error()})
		return
	}

	writeJSON(w, http.StatusOK, response{Success: true, Message: "ok", Data: result})
}

func (h *Handler) PodLogs(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, response{Success: false, Message: "method not allowed"})
		return
	}

	clusterID := r.URL.Query().Get("cluster_id")
	if clusterID == "" {
		writeJSON(w, http.StatusBadRequest, response{Success: false, Message: "cluster_id is required"})
		return
	}

	var req podLogsRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, response{Success: false, Message: "invalid request body"})
		return
	}

	result, err := h.service.GetPodLogs(r.Context(), clusterID, req.Namespace, req.PodName, req.Container)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, response{Success: false, Message: err.Error()})
		return
	}

	writeJSON(w, http.StatusOK, response{Success: true, Message: "ok", Data: result})
}

func (h *Handler) PodLabels(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPut {
		writeJSON(w, http.StatusMethodNotAllowed, response{Success: false, Message: "method not allowed"})
		return
	}

	clusterID := r.URL.Query().Get("cluster_id")
	if clusterID == "" {
		writeJSON(w, http.StatusBadRequest, response{Success: false, Message: "cluster_id is required"})
		return
	}

	var req podLabelsRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, response{Success: false, Message: "invalid request body"})
		return
	}

	result, err := h.service.UpdatePodLabels(r.Context(), clusterID, req.Namespace, req.PodName, req.Labels)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, response{Success: false, Message: err.Error()})
		return
	}

	writeJSON(w, http.StatusOK, response{Success: true, Message: "ok", Data: result})
}

func (h *Handler) NodeLabels(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPut {
		writeJSON(w, http.StatusMethodNotAllowed, response{Success: false, Message: "method not allowed"})
		return
	}

	clusterID := r.URL.Query().Get("cluster_id")
	if clusterID == "" {
		writeJSON(w, http.StatusBadRequest, response{Success: false, Message: "cluster_id is required"})
		return
	}

	var req nodeLabelsRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, response{Success: false, Message: "invalid request body"})
		return
	}

	result, err := h.service.UpdateNodeLabels(r.Context(), clusterID, req.NodeName, req.Labels)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, response{Success: false, Message: err.Error()})
		return
	}

	writeJSON(w, http.StatusOK, response{Success: true, Message: "ok", Data: result})
}

func (h *Handler) NodeExec(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, response{Success: false, Message: "method not allowed"})
		return
	}

	clusterID := r.URL.Query().Get("cluster_id")
	if clusterID == "" {
		writeJSON(w, http.StatusBadRequest, response{Success: false, Message: "cluster_id is required"})
		return
	}

	var req nodeExecRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, response{Success: false, Message: "invalid request body"})
		return
	}

	result, err := h.service.ExecNodeCommand(r.Context(), clusterID, req.NodeName, req.Command, req.WorkingDir)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, response{Success: false, Message: err.Error()})
		return
	}

	writeJSON(w, http.StatusOK, response{Success: true, Message: "ok", Data: result})
}

func (h *Handler) PodDelete(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodDelete {
		writeJSON(w, http.StatusMethodNotAllowed, response{Success: false, Message: "method not allowed"})
		return
	}

	clusterID := r.URL.Query().Get("cluster_id")
	if clusterID == "" {
		writeJSON(w, http.StatusBadRequest, response{Success: false, Message: "cluster_id is required"})
		return
	}

	var req podDeleteRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, response{Success: false, Message: "invalid request body"})
		return
	}

	if err := h.service.DeletePod(r.Context(), clusterID, req.Namespace, req.PodName); err != nil {
		writeJSON(w, http.StatusBadRequest, response{Success: false, Message: err.Error()})
		return
	}

	writeJSON(w, http.StatusOK, response{Success: true, Message: "deleted"})
}

func (h *Handler) ResourceDelete(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodDelete {
		writeJSON(w, http.StatusMethodNotAllowed, response{Success: false, Message: "method not allowed"})
		return
	}

	clusterID := r.URL.Query().Get("cluster_id")
	if clusterID == "" {
		writeJSON(w, http.StatusBadRequest, response{Success: false, Message: "cluster_id is required"})
		return
	}

	var req resourceDeleteRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, response{Success: false, Message: "invalid request body"})
		return
	}

	if err := h.service.DeleteResource(r.Context(), clusterID, req.Kind, req.Name, req.Namespace); err != nil {
		writeJSON(w, http.StatusBadRequest, response{Success: false, Message: err.Error()})
		return
	}

	writeJSON(w, http.StatusOK, response{Success: true, Message: "deleted"})
}

func (h *Handler) ResourceYAML(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		clusterID := r.URL.Query().Get("cluster_id")
		kind := r.URL.Query().Get("kind")
		name := r.URL.Query().Get("name")
		namespace := r.URL.Query().Get("namespace")
		if clusterID == "" || kind == "" || name == "" {
			writeJSON(w, http.StatusBadRequest, response{Success: false, Message: "cluster_id, kind and name are required"})
			return
		}

		resource, err := h.service.GetResourceYAML(r.Context(), clusterID, kind, name, namespace)
		if err != nil {
			writeJSON(w, http.StatusBadRequest, response{Success: false, Message: err.Error()})
			return
		}

		writeJSON(w, http.StatusOK, response{Success: true, Message: "ok", Data: resource})
	case http.MethodPut:
		clusterID := r.URL.Query().Get("cluster_id")
		if clusterID == "" {
			writeJSON(w, http.StatusBadRequest, response{Success: false, Message: "cluster_id is required"})
			return
		}

		var req applyYAMLRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeJSON(w, http.StatusBadRequest, response{Success: false, Message: "invalid request body"})
			return
		}

		if err := h.service.ApplyResourceYAML(r.Context(), clusterID, req.Content); err != nil {
			writeJSON(w, http.StatusBadRequest, response{Success: false, Message: err.Error()})
			return
		}

		writeJSON(w, http.StatusOK, response{Success: true, Message: "applied"})
	default:
		writeJSON(w, http.StatusMethodNotAllowed, response{Success: false, Message: "method not allowed"})
	}
}

func writeJSON(w http.ResponseWriter, status int, payload response) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}
