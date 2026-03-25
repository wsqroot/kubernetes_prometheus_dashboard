package deployment

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

type deploymentList struct {
	Items []deploymentItem `json:"items"`
}

type deploymentItem struct {
	Metadata deploymentMetadata `json:"metadata"`
	Spec     deploymentSpec     `json:"spec"`
	Status   deploymentStatus   `json:"status"`
}

type deploymentMetadata struct {
	Name string `json:"name"`
}

type deploymentSpec struct {
	Replicas *int32           `json:"replicas"`
	Template deploymentPodTpl `json:"template"`
}

type deploymentPodTpl struct {
	Spec deploymentPodSpec `json:"spec"`
}

type deploymentPodSpec struct {
	Containers []deploymentContainer `json:"containers"`
}

type deploymentContainer struct {
	Resources deploymentResources `json:"resources"`
}

type deploymentResources struct {
	Requests map[string]string `json:"requests"`
	Limits   map[string]string `json:"limits"`
}

type deploymentStatus struct {
	Replicas          int32 `json:"replicas"`
	UpdatedReplicas   int32 `json:"updatedReplicas"`
	ReadyReplicas     int32 `json:"readyReplicas"`
	AvailableReplicas int32 `json:"availableReplicas"`
	Unavailable       int32 `json:"unavailableReplicas"`
}

func NewService(kubeconfigService *loadkubeconfig.Service) *Service {
	return &Service{kubeconfigService: kubeconfigService}
}

func (s *Service) List(ctx context.Context, clusterID, namespace string) (*ResourceList, error) {
	cluster, err := s.kubeconfigService.ResolveCluster(ctx, clusterID)
	if err != nil {
		return nil, err
	}

	items, err := s.kubectlDeployments(ctx, cluster, namespace)
	if err != nil {
		return nil, fmt.Errorf("list deployments: %w", err)
	}

	rows := make([]ResourceRow, 0, len(items))
	for _, item := range items {
		desired := int32(1)
		if item.Spec.Replicas != nil {
			desired = *item.Spec.Replicas
		} else if item.Status.Replicas > 0 {
			desired = item.Status.Replicas
		}

		rows = append(rows, ResourceRow{
			Name:           item.Metadata.Name,
			ObjectName:     item.Metadata.Name,
			Count:          fmt.Sprintf("%d", desired),
			Target:         fmt.Sprintf("%d/%d", item.Status.AvailableReplicas, desired),
			Status:         deploymentStatusText(desired, item.Status.ReadyReplicas, item.Status.AvailableReplicas, item.Status.Unavailable),
			Runtime:        fmt.Sprintf("%d/%d", item.Status.ReadyReplicas, desired),
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

func (s *Service) kubectlDeployments(ctx context.Context, cluster *loadkubeconfig.ImportedCluster, namespace string) ([]deploymentItem, error) {
	tempFile, err := os.CreateTemp("", "deployment-kubeconfig-*.yaml")
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
		"get", "deployments",
		"-n", namespace,
		"-o", "json",
	}

	cmd := exec.CommandContext(ctx, "kubectl", args...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("%s", strings.TrimSpace(string(output)))
	}

	var result deploymentList
	if err := json.Unmarshal(output, &result); err != nil {
		return nil, fmt.Errorf("parse kubectl response: %w", err)
	}
	return result.Items, nil
}

func deploymentStatusText(desired, ready, available, unavailable int32) string {
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

func formatRequestsLimits(containers []deploymentContainer) string {
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
