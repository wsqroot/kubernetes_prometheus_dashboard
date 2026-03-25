package daemonset

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"sort"
	"strings"

	loadkubeconfig "developer-kubernetes/load_kubeconfig"
)

type Service struct {
	kubeconfigService *loadkubeconfig.Service
}

type ResourceRow struct {
	Name           string `json:"name"`
	ObjectName     string `json:"object_name"`
	Count          string `json:"count"`
	Target         string `json:"target"`
	Status         string `json:"status"`
	Runtime        string `json:"runtime"`
	RequestsLimits string `json:"requests_limits"`
	Action         string `json:"action"`
}

type ResourceList struct {
	Namespace string        `json:"namespace"`
	Items     []ResourceRow `json:"items"`
	Total     int           `json:"total"`
}

type daemonSetList struct {
	Items []daemonSetItem `json:"items"`
}

type daemonSetItem struct {
	Metadata daemonSetMetadata `json:"metadata"`
	Spec     daemonSetSpec     `json:"spec"`
	Status   daemonSetStatus   `json:"status"`
}

type daemonSetMetadata struct {
	Name string `json:"name"`
}

type daemonSetSpec struct {
	Template daemonSetPodTemplate `json:"template"`
}

type daemonSetPodTemplate struct {
	Spec daemonSetPodSpec `json:"spec"`
}

type daemonSetPodSpec struct {
	Containers []daemonSetContainer `json:"containers"`
}

type daemonSetContainer struct {
	Resources daemonSetResources `json:"resources"`
}

type daemonSetResources struct {
	Requests map[string]string `json:"requests"`
	Limits   map[string]string `json:"limits"`
}

type daemonSetStatus struct {
	DesiredNumberScheduled int32 `json:"desiredNumberScheduled"`
	CurrentNumberScheduled int32 `json:"currentNumberScheduled"`
	NumberReady            int32 `json:"numberReady"`
	NumberAvailable        int32 `json:"numberAvailable"`
	NumberUnavailable      int32 `json:"numberUnavailable"`
	UpdatedNumberScheduled int32 `json:"updatedNumberScheduled"`
}

func NewService(kubeconfigService *loadkubeconfig.Service) *Service {
	return &Service{kubeconfigService: kubeconfigService}
}

func (s *Service) List(ctx context.Context, clusterID, namespace string) (*ResourceList, error) {
	cluster, err := s.kubeconfigService.ResolveCluster(ctx, clusterID)
	if err != nil {
		return nil, err
	}

	items, err := s.kubectlDaemonSets(ctx, cluster, namespace)
	if err != nil {
		return nil, fmt.Errorf("list daemonsets: %w", err)
	}

	rows := make([]ResourceRow, 0, len(items))
	for _, item := range items {
		rows = append(rows, ResourceRow{
			Name:           item.Metadata.Name,
			ObjectName:     item.Metadata.Name,
			Count:          fmt.Sprintf("%d", item.Status.DesiredNumberScheduled),
			Target:         fmt.Sprintf("%d/%d", item.Status.CurrentNumberScheduled, item.Status.DesiredNumberScheduled),
			Status:         daemonSetStatusText(item.Status.DesiredNumberScheduled, item.Status.NumberReady, item.Status.NumberAvailable, item.Status.NumberUnavailable),
			Runtime:        fmt.Sprintf("%d/%d", item.Status.NumberReady, item.Status.DesiredNumberScheduled),
			RequestsLimits: formatRequestsLimits(item.Spec.Template.Spec.Containers),
			Action:         "编辑",
		})
	}

	sort.Slice(rows, func(i, j int) bool {
		return rows[i].Name < rows[j].Name
	})

	return &ResourceList{
		Namespace: namespace,
		Items:     rows,
		Total:     len(rows),
	}, nil
}

func (s *Service) kubectlDaemonSets(ctx context.Context, cluster *loadkubeconfig.ImportedCluster, namespace string) ([]daemonSetItem, error) {
	tempFile, err := os.CreateTemp("", "daemonset-kubeconfig-*.yaml")
	if err != nil {
		return nil, fmt.Errorf("create temp kubeconfig: %w", err)
	}
	tempPath := tempFile.Name()
	defer os.Remove(tempPath)

	if _, err := tempFile.WriteString(cluster.KubeconfigBody); err != nil {
		tempFile.Close()
		return nil, fmt.Errorf("write temp kubeconfig: %w", err)
	}
	if err := tempFile.Close(); err != nil {
		return nil, fmt.Errorf("close temp kubeconfig: %w", err)
	}

	args := []string{
		"--kubeconfig", tempPath,
		"--context", cluster.ContextName,
		"get", "daemonsets",
		"-n", namespace,
		"-o", "json",
	}

	cmd := exec.CommandContext(ctx, "kubectl", args...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("%s", strings.TrimSpace(string(output)))
	}

	var result daemonSetList
	if err := json.Unmarshal(output, &result); err != nil {
		return nil, fmt.Errorf("parse kubectl response: %w", err)
	}
	return result.Items, nil
}

func daemonSetStatusText(desired, ready, available, unavailable int32) string {
	switch {
	case desired == 0:
		return "已停止"
	case ready >= desired && available >= desired:
		return "运行中"
	case ready > 0 || available > 0:
		return "部分就绪"
	case unavailable > 0:
		return "未就绪"
	default:
		return "待调度"
	}
}

func formatRequestsLimits(containers []daemonSetContainer) string {
	requests := map[string]string{}
	limits := map[string]string{}
	for _, container := range containers {
		for key, value := range container.Resources.Requests {
			requests[key] = value
		}
		for key, value := range container.Resources.Limits {
			limits[key] = value
		}
	}

	requestText := formatResourceMap(requests)
	limitText := formatResourceMap(limits)
	switch {
	case requestText == "-" && limitText == "-":
		return "-"
	case requestText == "-":
		return "限制: " + limitText
	case limitText == "-":
		return "请求: " + requestText
	default:
		return "请求: " + requestText + " / 限制: " + limitText
	}
}

func formatResourceMap(items map[string]string) string {
	if len(items) == 0 {
		return "-"
	}

	keys := make([]string, 0, len(items))
	for key := range items {
		keys = append(keys, key)
	}
	sort.Strings(keys)

	parts := make([]string, 0, len(keys))
	for _, key := range keys {
		parts = append(parts, fmt.Sprintf("%s=%s", key, items[key]))
	}
	return strings.Join(parts, ", ")
}
