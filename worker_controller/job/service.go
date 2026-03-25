package job

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

type jobList struct {
	Items []jobItem `json:"items"`
}

type jobItem struct {
	Metadata jobMetadata `json:"metadata"`
	Spec     jobSpec     `json:"spec"`
	Status   jobStatus   `json:"status"`
}

type jobMetadata struct {
	Name string `json:"name"`
}

type jobSpec struct {
	Parallelism *int32         `json:"parallelism"`
	Completions *int32         `json:"completions"`
	Template    jobPodTemplate `json:"template"`
}

type jobPodTemplate struct {
	Spec jobPodSpec `json:"spec"`
}

type jobPodSpec struct {
	Containers []jobContainer `json:"containers"`
}

type jobContainer struct {
	Resources jobResources `json:"resources"`
}

type jobResources struct {
	Requests map[string]string `json:"requests"`
	Limits   map[string]string `json:"limits"`
}

type jobStatus struct {
	Active    int32 `json:"active"`
	Succeeded int32 `json:"succeeded"`
	Failed    int32 `json:"failed"`
	Ready     int32 `json:"ready"`
}

func NewService(kubeconfigService *loadkubeconfig.Service) *Service {
	return &Service{kubeconfigService: kubeconfigService}
}

func (s *Service) List(ctx context.Context, clusterID, namespace string) (*ResourceList, error) {
	cluster, err := s.kubeconfigService.ResolveCluster(ctx, clusterID)
	if err != nil {
		return nil, err
	}

	items, err := s.kubectlJobs(ctx, cluster, namespace)
	if err != nil {
		return nil, fmt.Errorf("list jobs: %w", err)
	}

	rows := make([]ResourceRow, 0, len(items))
	for _, item := range items {
		expected := int32(1)
		if item.Spec.Completions != nil {
			expected = *item.Spec.Completions
		}
		parallelism := int32(1)
		if item.Spec.Parallelism != nil {
			parallelism = *item.Spec.Parallelism
		}

		rows = append(rows, ResourceRow{
			Name:           item.Metadata.Name,
			ObjectName:     item.Metadata.Name,
			Count:          fmt.Sprintf("%d", parallelism),
			Target:         fmt.Sprintf("%d/%d", item.Status.Succeeded, expected),
			Status:         jobStatusText(expected, item.Status.Active, item.Status.Succeeded, item.Status.Failed),
			Runtime:        fmt.Sprintf("活跃 %d / 并行 %d", item.Status.Active, parallelism),
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

func (s *Service) kubectlJobs(ctx context.Context, cluster *loadkubeconfig.ImportedCluster, namespace string) ([]jobItem, error) {
	tempFile, err := os.CreateTemp("", "job-kubeconfig-*.yaml")
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
		"get", "jobs",
		"-n", namespace,
		"-o", "json",
	}

	cmd := exec.CommandContext(ctx, "kubectl", args...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("%s", strings.TrimSpace(string(output)))
	}

	var result jobList
	if err := json.Unmarshal(output, &result); err != nil {
		return nil, fmt.Errorf("parse kubectl response: %w", err)
	}
	return result.Items, nil
}

func jobStatusText(expected, active, succeeded, failed int32) string {
	switch {
	case succeeded >= expected && expected > 0:
		return "已完成"
	case failed > 0 && active == 0:
		return "失败"
	case active > 0:
		return "运行中"
	case succeeded > 0:
		return "部分完成"
	default:
		return "待执行"
	}
}

func formatRequestsLimits(containers []jobContainer) string {
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
