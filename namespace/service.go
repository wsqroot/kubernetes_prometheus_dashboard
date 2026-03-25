package namespace

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"os"
	"os/exec"
	"sort"
	"strconv"
	"strings"

	loadkubeconfig "developer-kubernetes/load_kubeconfig"
)

type Service struct {
	kubeconfigService *loadkubeconfig.Service
}

type ClusterOverview struct {
	ClusterID      string         `json:"cluster_id"`
	ClusterName    string         `json:"cluster_name"`
	ContextName    string         `json:"context_name"`
	SourceFile     string         `json:"source_file"`
	NodeCount      int            `json:"node_count"`
	NodeNames      []string       `json:"node_names"`
	NamespaceCount int            `json:"namespace_count"`
	Namespaces     []string       `json:"namespaces"`
	SectionCounts  map[string]int `json:"section_counts"`
	ResourceCounts map[string]int `json:"resource_counts"`
}

type NamespaceResources struct {
	Namespace string                         `json:"namespace"`
	Sections  map[string]map[string][]string `json:"sections"`
}

type NodeResources struct {
	Items []NodeResourceRow `json:"items"`
	Total int               `json:"total"`
}

type NodeResourceRow struct {
	Name       string            `json:"name"`
	IPAddress  string            `json:"ip_address"`
	Labels     map[string]string `json:"labels"`
	LabelPairs []string          `json:"label_pairs"`
	Count      string            `json:"count"`
	Target     string            `json:"target"`
	Status     string            `json:"status"`
	Runtime    string            `json:"runtime"`
	CPU        string            `json:"cpu"`
	Action     string            `json:"action"`
}

type PodResourceRow struct {
	Name           string            `json:"name"`
	ObjectName     string            `json:"object_name"`
	Labels         map[string]string `json:"labels"`
	LabelPairs     []string          `json:"label_pairs"`
	Count          string            `json:"count"`
	Target         string            `json:"target"`
	Status         string            `json:"status"`
	Runtime        string            `json:"runtime"`
	RequestsLimits string            `json:"requests_limits"`
	NodeName       string            `json:"node_name"`
	Containers     []string          `json:"containers"`
	Action         string            `json:"action"`
}

type PodResourceList struct {
	Namespace string           `json:"namespace"`
	Items     []PodResourceRow `json:"items"`
	Total     int              `json:"total"`
}

type ResourceYAML struct {
	Kind      string `json:"kind"`
	Name      string `json:"name"`
	Namespace string `json:"namespace,omitempty"`
	Content   string `json:"content"`
}

type PodExecResult struct {
	PodName       string `json:"pod_name"`
	ContainerName string `json:"container_name"`
	Command       string `json:"command"`
	Output        string `json:"output"`
	WorkingDir    string `json:"working_dir"`
}

type PodLogResult struct {
	PodName       string `json:"pod_name"`
	ContainerName string `json:"container_name"`
	Output        string `json:"output"`
}

type PodLabelResult struct {
	PodName   string            `json:"pod_name"`
	Namespace string            `json:"namespace"`
	Labels    map[string]string `json:"labels"`
}

type NodeLabelResult struct {
	NodeName string            `json:"node_name"`
	Labels   map[string]string `json:"labels"`
}

type NodeExecResult struct {
	NodeName    string `json:"node_name"`
	Command     string `json:"command"`
	Output      string `json:"output"`
	WorkingDir  string `json:"working_dir"`
	DebugTarget string `json:"debug_target"`
}

type kubectlList struct {
	Items []kubectlItem `json:"items"`
}

type kubectlItem struct {
	Metadata kubectlMetadata `json:"metadata"`
}

type kubectlMetadata struct {
	Name      string            `json:"name"`
	Namespace string            `json:"namespace"`
	Labels    map[string]string `json:"labels"`
}

type nodeList struct {
	Items []nodeItem `json:"items"`
}

type nodeItem struct {
	Metadata kubectlMetadata `json:"metadata"`
	Status   nodeStatus      `json:"status"`
}

type nodeStatus struct {
	Addresses   []nodeAddress     `json:"addresses"`
	Conditions  []nodeCondition   `json:"conditions"`
	Capacity    map[string]string `json:"capacity"`
	Allocatable map[string]string `json:"allocatable"`
}

type nodeAddress struct {
	Type    string `json:"type"`
	Address string `json:"address"`
}

type nodeCondition struct {
	Type   string `json:"type"`
	Status string `json:"status"`
}

type podList struct {
	Items []podItem `json:"items"`
}

type podItem struct {
	Metadata podMetadata `json:"metadata"`
	Spec     podSpec     `json:"spec"`
	Status   podStatus   `json:"status"`
}

type podMetadata struct {
	Name   string            `json:"name"`
	Labels map[string]string `json:"labels"`
}

type podSpec struct {
	NodeName    string                 `json:"nodeName"`
	Containers  []podContainer         `json:"containers"`
	Affinity    map[string]interface{} `json:"affinity"`
	Tolerations []podToleration        `json:"tolerations"`
}

type podContainer struct {
	Name      string       `json:"name"`
	Resources podResources `json:"resources"`
}

type podResources struct {
	Requests map[string]string `json:"requests"`
	Limits   map[string]string `json:"limits"`
}

type podToleration struct {
	Key string `json:"key"`
}

type podStatus struct {
	Phase                 string               `json:"phase"`
	Reason                string               `json:"reason"`
	ContainerStatuses     []podContainerStatus `json:"containerStatuses"`
	InitContainerStatuses []podContainerStatus `json:"initContainerStatuses"`
}

type podContainerStatus struct {
	Ready     bool              `json:"ready"`
	State     podContainerState `json:"state"`
	LastState podContainerState `json:"lastState"`
}

type podContainerState struct {
	Waiting    *podStateWaiting    `json:"waiting"`
	Running    *podStateRunning    `json:"running"`
	Terminated *podStateTerminated `json:"terminated"`
}

type podStateWaiting struct {
	Reason string `json:"reason"`
}

type podStateRunning struct{}

type podStateTerminated struct {
	Reason   string `json:"reason"`
	ExitCode int32  `json:"exitCode"`
}

func NewService(kubeconfigService *loadkubeconfig.Service) *Service {
	return &Service{kubeconfigService: kubeconfigService}
}

func (s *Service) ListNamespaces(ctx context.Context, clusterID string) ([]string, error) {
	cluster, err := s.kubeconfigService.ResolveCluster(ctx, clusterID)
	if err != nil {
		return nil, err
	}

	items, err := s.kubectlNames(ctx, cluster, "", "namespaces")
	if err != nil {
		return nil, fmt.Errorf("list namespaces: %w", err)
	}

	sort.Strings(items)
	return items, nil
}

func (s *Service) GetOverview(ctx context.Context, clusterID string) (*ClusterOverview, error) {
	cluster, err := s.kubeconfigService.ResolveCluster(ctx, clusterID)
	if err != nil {
		return nil, err
	}

	namespaces, err := s.ListNamespaces(ctx, clusterID)
	if err != nil {
		return nil, err
	}

	nodes, _ := s.kubectlNames(ctx, cluster, "", "nodes")
	deployments, _ := s.kubectlNames(ctx, cluster, "", "deployments", "-A")
	statefulsets, _ := s.kubectlNames(ctx, cluster, "", "statefulsets", "-A")
	daemonsets, _ := s.kubectlNames(ctx, cluster, "", "daemonsets", "-A")
	jobs, _ := s.kubectlNames(ctx, cluster, "", "jobs", "-A")
	cronjobs, _ := s.kubectlNames(ctx, cluster, "", "cronjobs", "-A")
	pods, _ := s.kubectlNames(ctx, cluster, "", "pods", "-A")
	services, _ := s.kubectlNames(ctx, cluster, "", "services", "-A")
	ingresses, _ := s.kubectlNames(ctx, cluster, "", "ingresses", "-A")
	pvcs, _ := s.kubectlNames(ctx, cluster, "", "pvc", "-A")
	configmaps, _ := s.kubectlNames(ctx, cluster, "", "configmaps", "-A")
	secrets, _ := s.kubectlNames(ctx, cluster, "", "secrets", "-A")
	serviceAccounts, _ := s.kubectlNames(ctx, cluster, "", "serviceaccounts", "-A")
	networkPolicies, _ := s.kubectlNames(ctx, cluster, "", "networkpolicies", "-A")
	roleBindings, _ := s.kubectlNames(ctx, cluster, "", "rolebindings", "-A")

	return &ClusterOverview{
		ClusterID:      cluster.ID,
		ClusterName:    cluster.Name,
		ContextName:    cluster.ContextName,
		SourceFile:     cluster.SourceFile,
		NodeCount:      len(nodes),
		NodeNames:      nodes,
		NamespaceCount: len(namespaces),
		Namespaces:     namespaces,
		SectionCounts: map[string]int{
			"controller": len(deployments) + len(statefulsets) + len(daemonsets) + len(jobs) + len(cronjobs) + len(pods),
			"node":       len(nodes),
			"network":    len(services) + len(ingresses),
			"storage":    len(pvcs) + len(secrets) + len(configmaps),
			"security":   len(serviceAccounts) + len(networkPolicies) + len(roleBindings),
		},
		ResourceCounts: map[string]int{
			"Deployment":     len(deployments),
			"StatefulSet":    len(statefulsets),
			"DaemonSet":      len(daemonsets),
			"Job":            len(jobs),
			"CronJob":        len(cronjobs),
			"Pod":            len(pods),
			"Node":           len(nodes),
			"Service":        len(services),
			"Ingress":        len(ingresses),
			"PVC":            len(pvcs),
			"ConfigMap":      len(configmaps),
			"Secret":         len(secrets),
			"ServiceAccount": len(serviceAccounts),
			"NetworkPolicy":  len(networkPolicies),
			"RBAC":           len(roleBindings),
		},
	}, nil
}

func (s *Service) GetNamespaceResources(ctx context.Context, clusterID, namespaceName string) (*NamespaceResources, error) {
	cluster, err := s.kubeconfigService.ResolveCluster(ctx, clusterID)
	if err != nil {
		return nil, err
	}

	deployments, _ := s.kubectlNames(ctx, cluster, namespaceName, "deployments")
	statefulsets, _ := s.kubectlNames(ctx, cluster, namespaceName, "statefulsets")
	daemonsets, _ := s.kubectlNames(ctx, cluster, namespaceName, "daemonsets")
	jobs, _ := s.kubectlNames(ctx, cluster, namespaceName, "jobs")
	cronjobs, _ := s.kubectlNames(ctx, cluster, namespaceName, "cronjobs")
	pods, _ := s.kubectlNames(ctx, cluster, namespaceName, "pods")
	services, _ := s.kubectlNames(ctx, cluster, namespaceName, "services")
	ingresses, _ := s.kubectlNames(ctx, cluster, namespaceName, "ingresses")
	pvcs, _ := s.kubectlNames(ctx, cluster, namespaceName, "pvc")
	configmaps, _ := s.kubectlNames(ctx, cluster, namespaceName, "configmaps")
	secrets, _ := s.kubectlNames(ctx, cluster, namespaceName, "secrets")
	serviceAccounts, _ := s.kubectlNames(ctx, cluster, namespaceName, "serviceaccounts")
	networkPolicies, _ := s.kubectlNames(ctx, cluster, namespaceName, "networkpolicies")
	roleBindings, _ := s.kubectlNames(ctx, cluster, namespaceName, "rolebindings")

	return &NamespaceResources{
		Namespace: namespaceName,
		Sections: map[string]map[string][]string{
			"controller": {
				"Deployment":  deployments,
				"StatefulSet": statefulsets,
				"DaemonSet":   daemonsets,
				"Job":         jobs,
				"CronJob":     cronjobs,
				"Pod":         pods,
			},
			"network": {
				"Service": services,
				"Ingress": ingresses,
				"Gateway": []string{},
			},
			"storage": {
				"PVC":            pvcs,
				"Secret":         secrets,
				"ConfigMap":      configmaps,
				"PV":             []string{},
				"StorageClass":   []string{},
				"VolumeSnapshot": []string{},
			},
			"security": {
				"ServiceAccount": serviceAccounts,
				"NetworkPolicy":  networkPolicies,
				"RBAC":           roleBindings,
			},
		},
	}, nil
}

func (s *Service) GetNodeResources(ctx context.Context, clusterID string) (*NodeResources, error) {
	cluster, err := s.kubeconfigService.ResolveCluster(ctx, clusterID)
	if err != nil {
		return nil, err
	}

	nodeOutput, err := s.runKubectl(ctx, cluster, "", false, []string{
		"get", "nodes",
		"-o", "json",
	})
	if err != nil {
		return nil, fmt.Errorf("list nodes: %w", err)
	}

	podOutput, err := s.runKubectl(ctx, cluster, "", false, []string{
		"get", "pods",
		"-A",
		"-o", "json",
	})
	if err != nil {
		return nil, fmt.Errorf("list pods for nodes: %w", err)
	}

	var nodes nodeList
	if err := json.Unmarshal(nodeOutput, &nodes); err != nil {
		return nil, fmt.Errorf("parse node list: %w", err)
	}

	var pods podList
	if err := json.Unmarshal(podOutput, &pods); err != nil {
		return nil, fmt.Errorf("parse node pod list: %w", err)
	}

	type nodeUsage struct {
		podCount        int
		requestMilliCPU int64
		requestMemory   int64
	}

	usage := map[string]*nodeUsage{}
	for _, item := range pods.Items {
		nodeName := strings.TrimSpace(item.Spec.NodeName)
		if nodeName == "" || item.Status.Phase == "Succeeded" || item.Status.Phase == "Failed" {
			continue
		}
		entry := usage[nodeName]
		if entry == nil {
			entry = &nodeUsage{}
			usage[nodeName] = entry
		}
		entry.podCount++
		for _, container := range item.Spec.Containers {
			entry.requestMilliCPU += parseCPUToMilli(container.Resources.Requests["cpu"])
			entry.requestMemory += parseMemoryToBytes(container.Resources.Requests["memory"])
		}
	}

	rows := make([]NodeResourceRow, 0, len(nodes.Items))
	for _, item := range nodes.Items {
		entry := usage[item.Metadata.Name]
		if entry == nil {
			entry = &nodeUsage{}
		}
		totalCPU := parseCPUToMilli(item.Status.Capacity["cpu"])
		allocCPU := parseCPUToMilli(item.Status.Allocatable["cpu"])
		remainingCPU := allocCPU - entry.requestMilliCPU
		if remainingCPU < 0 {
			remainingCPU = 0
		}

		totalMemory := parseMemoryToBytes(item.Status.Capacity["memory"])
		allocMemory := parseMemoryToBytes(item.Status.Allocatable["memory"])
		remainingMemory := allocMemory - entry.requestMemory
		if remainingMemory < 0 {
			remainingMemory = 0
		}

		labels := cloneStringMap(item.Metadata.Labels)
		rows = append(rows, NodeResourceRow{
			Name:       item.Metadata.Name,
			IPAddress:  nodeIPAddress(item.Status.Addresses),
			Labels:     labels,
			LabelPairs: formatLabelPairs(labels),
			Count:      fmt.Sprintf("%d", entry.podCount),
			Target:     "",
			Status:     nodeReadyStatus(item.Status.Conditions),
			Runtime:    fmt.Sprintf("%s/%s", formatBytesToGiB(remainingMemory), formatBytesToGiB(totalMemory)),
			CPU:        fmt.Sprintf("%s/%s", formatMilliCPUToCores(remainingCPU), formatMilliCPUToCores(totalCPU)),
			Action:     "编辑",
		})
	}

	sort.Slice(rows, func(i, j int) bool {
		return rows[i].Name < rows[j].Name
	})

	return &NodeResources{
		Items: rows,
		Total: len(rows),
	}, nil
}

func (s *Service) GetPodResources(ctx context.Context, clusterID, namespaceName string) (*PodResourceList, error) {
	cluster, err := s.kubeconfigService.ResolveCluster(ctx, clusterID)
	if err != nil {
		return nil, err
	}

	output, err := s.runKubectl(ctx, cluster, namespaceName, true, []string{
		"get", "pods",
		"-o", "json",
	})
	if err != nil {
		return nil, fmt.Errorf("list pods: %w", err)
	}

	var result podList
	if err := json.Unmarshal(output, &result); err != nil {
		return nil, fmt.Errorf("parse pod list: %w", err)
	}

	rows := make([]PodResourceRow, 0, len(result.Items))
	for _, item := range result.Items {
		totalContainers := len(item.Spec.Containers)
		readyContainers := podReadyContainers(item.Status.ContainerStatuses)
		containerNames := podContainerNames(item.Spec.Containers)
		labels := cloneStringMap(item.Metadata.Labels)
		rows = append(rows, PodResourceRow{
			Name:           item.Metadata.Name,
			ObjectName:     item.Metadata.Name,
			Labels:         labels,
			LabelPairs:     formatLabelPairs(labels),
			Count:          fmt.Sprintf("%d", totalContainers),
			Target:         formatAffinityToleration(item.Spec.Affinity, item.Spec.Tolerations),
			Status:         podStatusText(item.Status),
			Runtime:        fmt.Sprintf("%d/%d", readyContainers, totalContainers),
			RequestsLimits: formatPodRequestsLimits(item.Spec.Containers),
			NodeName:       fallbackText(item.Spec.NodeName, "-"),
			Containers:     containerNames,
			Action:         "编辑",
		})
	}

	sort.Slice(rows, func(i, j int) bool {
		return rows[i].Name < rows[j].Name
	})

	return &PodResourceList{
		Namespace: namespaceName,
		Items:     rows,
		Total:     len(rows),
	}, nil
}

func (s *Service) UpdatePodLabels(ctx context.Context, clusterID, namespaceName, podName string, desired map[string]string) (*PodLabelResult, error) {
	cluster, err := s.kubeconfigService.ResolveCluster(ctx, clusterID)
	if err != nil {
		return nil, err
	}

	currentPod, err := s.getPod(ctx, cluster, namespaceName, podName)
	if err != nil {
		return nil, err
	}

	current := cloneStringMap(currentPod.Metadata.Labels)
	desired = sanitizeLabels(desired)
	args := []string{"label", "pod", podName}

	changed := false
	for key := range current {
		if _, ok := desired[key]; !ok {
			args = append(args, key+"-")
			changed = true
		}
	}
	for key, value := range desired {
		if current[key] == value {
			continue
		}
		args = append(args, fmt.Sprintf("%s=%s", key, value))
		changed = true
	}

	if changed {
		args = append(args, "--overwrite")
		if _, err := s.runKubectl(ctx, cluster, namespaceName, true, args); err != nil {
			return nil, fmt.Errorf("update pod labels: %w", err)
		}
	}

	return &PodLabelResult{
		PodName:   podName,
		Namespace: namespaceName,
		Labels:    desired,
	}, nil
}

func (s *Service) UpdateNodeLabels(ctx context.Context, clusterID, nodeName string, desired map[string]string) (*NodeLabelResult, error) {
	cluster, err := s.kubeconfigService.ResolveCluster(ctx, clusterID)
	if err != nil {
		return nil, err
	}

	currentNode, err := s.getNode(ctx, cluster, nodeName)
	if err != nil {
		return nil, err
	}

	current := cloneStringMap(currentNode.Metadata.Labels)
	desired = sanitizeLabels(desired)
	args := []string{"label", "node", nodeName}

	changed := false
	for key := range current {
		if _, ok := desired[key]; !ok {
			args = append(args, key+"-")
			changed = true
		}
	}
	for key, value := range desired {
		if current[key] == value {
			continue
		}
		args = append(args, fmt.Sprintf("%s=%s", key, value))
		changed = true
	}

	if changed {
		args = append(args, "--overwrite")
		if _, err := s.runKubectl(ctx, cluster, "", false, args); err != nil {
			return nil, fmt.Errorf("update node labels: %w", err)
		}
	}

	return &NodeLabelResult{
		NodeName: nodeName,
		Labels:   desired,
	}, nil
}

func (s *Service) DeletePod(ctx context.Context, clusterID, namespaceName, podName string) error {
	cluster, err := s.kubeconfigService.ResolveCluster(ctx, clusterID)
	if err != nil {
		return err
	}

	namespaceName = strings.TrimSpace(namespaceName)
	podName = strings.TrimSpace(podName)
	if namespaceName == "" || podName == "" {
		return fmt.Errorf("namespace and pod are required")
	}

	if _, err := s.runKubectl(ctx, cluster, namespaceName, true, []string{
		"delete", "pod", podName,
		"--ignore-not-found=true",
	}); err != nil {
		return fmt.Errorf("delete pod: %w", err)
	}
	return nil
}

func (s *Service) DeleteResource(ctx context.Context, clusterID, kind, name, namespaceName string) error {
	cluster, err := s.kubeconfigService.ResolveCluster(ctx, clusterID)
	if err != nil {
		return err
	}

	kind = strings.TrimSpace(kind)
	name = strings.TrimSpace(name)
	namespaceName = strings.TrimSpace(namespaceName)
	if kind == "" || name == "" {
		return fmt.Errorf("kind and name are required")
	}

	resource, namespaced, err := resourceRef(kind)
	if err != nil {
		return err
	}
	if namespaced && namespaceName == "" {
		return fmt.Errorf("namespace is required for %s", kind)
	}

	if _, err := s.runKubectl(ctx, cluster, namespaceName, namespaced, []string{
		"delete", resource, name,
		"--ignore-not-found=true",
	}); err != nil {
		return fmt.Errorf("delete %s: %w", kind, err)
	}
	return nil
}

func (s *Service) ExecPodCommand(ctx context.Context, clusterID, namespaceName, podName, containerName, command, workingDir string) (*PodExecResult, error) {
	cluster, err := s.kubeconfigService.ResolveCluster(ctx, clusterID)
	if err != nil {
		return nil, err
	}

	namespaceName = strings.TrimSpace(namespaceName)
	podName = strings.TrimSpace(podName)
	containerName = strings.TrimSpace(containerName)
	command = strings.TrimSpace(command)
	workingDir = strings.TrimSpace(workingDir)
	if namespaceName == "" || podName == "" || containerName == "" || command == "" {
		return nil, fmt.Errorf("namespace, pod, container and command are required")
	}

	script := buildExecScript(command, workingDir)

	output, err := s.execWithShell(ctx, cluster, namespaceName, podName, containerName, script)
	if err != nil {
		return nil, fmt.Errorf("exec command: %w", err)
	}

	commandOutput, currentDir := parseExecOutput(string(output))

	return &PodExecResult{
		PodName:       podName,
		ContainerName: containerName,
		Command:       command,
		Output:        commandOutput,
		WorkingDir:    currentDir,
	}, nil
}

func (s *Service) ExecNodeCommand(ctx context.Context, clusterID, nodeName, command, workingDir string) (*NodeExecResult, error) {
	cluster, err := s.kubeconfigService.ResolveCluster(ctx, clusterID)
	if err != nil {
		return nil, err
	}

	nodeName = strings.TrimSpace(nodeName)
	command = strings.TrimSpace(command)
	workingDir = strings.TrimSpace(workingDir)
	if nodeName == "" || command == "" {
		return nil, fmt.Errorf("node and command are required")
	}

	script := buildExecScript(command, workingDir)
	output, err := s.runKubectl(ctx, cluster, "", false, []string{
		"debug", fmt.Sprintf("node/%s", nodeName),
		"--profile=general",
		"--attach=true",
		"--stdin=false",
		"--image=busybox:1.36",
		"--",
		"chroot", "/host", "sh", "-lc", script,
	})
	if err != nil {
		return nil, fmt.Errorf("exec node command: %w", err)
	}

	commandOutput, currentDir := parseExecOutput(stripNodeDebugNoise(string(output)))
	return &NodeExecResult{
		NodeName:    nodeName,
		Command:     command,
		Output:      commandOutput,
		WorkingDir:  currentDir,
		DebugTarget: "busybox:1.36",
	}, nil
}

func (s *Service) GetPodLogs(ctx context.Context, clusterID, namespaceName, podName, containerName string) (*PodLogResult, error) {
	cluster, err := s.kubeconfigService.ResolveCluster(ctx, clusterID)
	if err != nil {
		return nil, err
	}

	namespaceName = strings.TrimSpace(namespaceName)
	podName = strings.TrimSpace(podName)
	containerName = strings.TrimSpace(containerName)
	if namespaceName == "" || podName == "" || containerName == "" {
		return nil, fmt.Errorf("namespace, pod and container are required")
	}

	output, err := s.runKubectl(ctx, cluster, namespaceName, true, []string{
		"logs", podName,
		"-c", containerName,
		"--tail=500",
	})
	if err != nil {
		return nil, fmt.Errorf("get logs: %w", err)
	}

	return &PodLogResult{
		PodName:       podName,
		ContainerName: containerName,
		Output:        string(output),
	}, nil
}

func (s *Service) getPod(ctx context.Context, cluster *loadkubeconfig.ImportedCluster, namespaceName, podName string) (*podItem, error) {
	output, err := s.runKubectl(ctx, cluster, namespaceName, true, []string{
		"get", "pod", podName,
		"-o", "json",
	})
	if err != nil {
		return nil, fmt.Errorf("get pod: %w", err)
	}

	var item podItem
	if err := json.Unmarshal(output, &item); err != nil {
		return nil, fmt.Errorf("parse pod: %w", err)
	}
	return &item, nil
}

func (s *Service) getNode(ctx context.Context, cluster *loadkubeconfig.ImportedCluster, nodeName string) (*nodeItem, error) {
	output, err := s.runKubectl(ctx, cluster, "", false, []string{
		"get", "node", nodeName,
		"-o", "json",
	})
	if err != nil {
		return nil, fmt.Errorf("get node: %w", err)
	}

	var item nodeItem
	if err := json.Unmarshal(output, &item); err != nil {
		return nil, fmt.Errorf("parse node: %w", err)
	}
	return &item, nil
}

func (s *Service) execWithShell(ctx context.Context, cluster *loadkubeconfig.ImportedCluster, namespaceName, podName, containerName, script string) ([]byte, error) {
	shells := []string{"/bin/sh", "/bin/ash", "sh", "ash", "/busybox/sh"}
	var lastErr error
	for _, shell := range shells {
		output, err := s.runKubectl(ctx, cluster, namespaceName, true, []string{
			"exec", podName,
			"-c", containerName,
			"--", shell, "-lc", script,
		})
		if err == nil {
			return output, nil
		}
		lastErr = err
		message := err.Error()
		if strings.Contains(message, "executable file not found") || strings.Contains(message, "no such file or directory") {
			continue
		}
		return nil, err
	}
	return nil, fmt.Errorf("container shell not found, tried /bin/sh, /bin/ash, sh, ash, /busybox/sh: %w", lastErr)
}

func (s *Service) GetResourceYAML(ctx context.Context, clusterID, kind, name, namespaceName string) (*ResourceYAML, error) {
	cluster, err := s.kubeconfigService.ResolveCluster(ctx, clusterID)
	if err != nil {
		return nil, err
	}

	resource, namespaced, err := resourceRef(kind)
	if err != nil {
		return nil, err
	}

	output, err := s.runKubectl(ctx, cluster, namespaceName, namespaced, []string{
		"get", resource, name,
		"-o", "yaml",
	})
	if err != nil {
		return nil, fmt.Errorf("get %s yaml: %w", kind, err)
	}

	return &ResourceYAML{
		Kind:      kind,
		Name:      name,
		Namespace: namespaceName,
		Content:   string(output),
	}, nil
}

func (s *Service) ApplyResourceYAML(ctx context.Context, clusterID, yamlContent string) error {
	cluster, err := s.kubeconfigService.ResolveCluster(ctx, clusterID)
	if err != nil {
		return err
	}

	if strings.TrimSpace(yamlContent) == "" {
		return fmt.Errorf("yaml content is required")
	}

	if _, err := s.runKubectlWithInput(ctx, cluster, []string{
		"apply", "-f", "-",
	}, yamlContent); err != nil {
		return fmt.Errorf("apply yaml: %w", err)
	}
	return nil
}

func (s *Service) kubectlNames(ctx context.Context, cluster *loadkubeconfig.ImportedCluster, namespaceName, resource string, extraArgs ...string) ([]string, error) {
	output, err := s.runKubectl(ctx, cluster, namespaceName, namespaceName != "", append([]string{
		"get", resource,
		"-o", "json",
	}, extraArgs...))
	if err != nil {
		return nil, err
	}

	var result kubectlList
	if err := json.Unmarshal(output, &result); err != nil {
		return nil, fmt.Errorf("parse kubectl response: %w", err)
	}

	names := make([]string, 0, len(result.Items))
	for _, item := range result.Items {
		if item.Metadata.Name == "" {
			continue
		}
		names = append(names, item.Metadata.Name)
	}
	sort.Strings(names)
	return names, nil
}

func (s *Service) runKubectl(ctx context.Context, cluster *loadkubeconfig.ImportedCluster, namespaceName string, namespaced bool, commandArgs []string) ([]byte, error) {
	return s.runKubectlWithInput(ctx, cluster, s.buildKubectlArgs(cluster, namespaceName, namespaced, commandArgs), "")
}

func (s *Service) runKubectlWithInput(ctx context.Context, cluster *loadkubeconfig.ImportedCluster, args []string, stdin string) ([]byte, error) {
	tempFile, err := os.CreateTemp("", "dashboard-kubeconfig-*.yaml")
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

	args = append([]string{
		"--kubeconfig", tempPath,
		"--context", cluster.ContextName,
	}, args...)

	cmd := exec.CommandContext(ctx, "kubectl", args...)
	if stdin != "" {
		cmd.Stdin = strings.NewReader(stdin)
	}
	output, err := cmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("%s", strings.TrimSpace(string(output)))
	}
	return output, nil
}

func (s *Service) buildKubectlArgs(cluster *loadkubeconfig.ImportedCluster, namespaceName string, namespaced bool, commandArgs []string) []string {
	args := make([]string, 0, len(commandArgs)+2)
	if namespaced && namespaceName != "" {
		args = append(args, "-n", namespaceName)
	}
	args = append(args, commandArgs...)
	return args
}

func resourceRef(kind string) (string, bool, error) {
	switch kind {
	case "Namespace":
		return "namespaces", false, nil
	case "Deployment":
		return "deployments", true, nil
	case "StatefulSet":
		return "statefulsets", true, nil
	case "DaemonSet":
		return "daemonsets", true, nil
	case "Job":
		return "jobs", true, nil
	case "CronJob":
		return "cronjobs", true, nil
	case "Pod":
		return "pods", true, nil
	case "Node":
		return "nodes", false, nil
	case "Service":
		return "services", true, nil
	case "Ingress":
		return "ingresses", true, nil
	case "Gateway":
		return "gateways.gateway.networking.k8s.io", true, nil
	case "PVC":
		return "pvc", true, nil
	case "ConfigMap":
		return "configmaps", true, nil
	case "Secret":
		return "secrets", true, nil
	case "ServiceAccount":
		return "serviceaccounts", true, nil
	case "NetworkPolicy":
		return "networkpolicies", true, nil
	case "RBAC":
		return "rolebindings", true, nil
	case "PV":
		return "pv", false, nil
	case "StorageClass":
		return "storageclass", false, nil
	case "VolumeSnapshot":
		return "volumesnapshots.snapshot.storage.k8s.io", true, nil
	default:
		return "", false, fmt.Errorf("unsupported resource kind: %s", kind)
	}
}

func nodeIPAddress(addresses []nodeAddress) string {
	for _, item := range addresses {
		if item.Type == "InternalIP" && item.Address != "" {
			return item.Address
		}
	}
	for _, item := range addresses {
		if item.Type == "ExternalIP" && item.Address != "" {
			return item.Address
		}
	}
	for _, item := range addresses {
		if item.Address != "" {
			return item.Address
		}
	}
	return "-"
}

func nodeReadyStatus(conditions []nodeCondition) string {
	for _, item := range conditions {
		if item.Type == "Ready" {
			if item.Status == "True" {
				return "Ready"
			}
			if item.Status == "False" {
				return "NotReady"
			}
			return item.Status
		}
	}
	return "Unknown"
}

func parseCPUToMilli(value string) int64 {
	value = strings.TrimSpace(value)
	if value == "" {
		return 0
	}
	if strings.HasSuffix(value, "m") {
		value = strings.TrimSuffix(value, "m")
		parsed, _ := strconv.ParseInt(value, 10, 64)
		return parsed
	}
	parsed, err := strconv.ParseFloat(value, 64)
	if err != nil {
		return 0
	}
	return int64(parsed * 1000)
}

func formatMilliCPUToCores(value int64) string {
	if value <= 0 {
		return "0"
	}
	cores := float64(value) / 1000
	if math.Abs(cores-math.Round(cores)) < 0.0001 {
		return fmt.Sprintf("%.0f", cores)
	}
	return strings.TrimRight(strings.TrimRight(fmt.Sprintf("%.2f", cores), "0"), ".")
}

func parseMemoryToBytes(value string) int64 {
	value = strings.TrimSpace(value)
	if value == "" {
		return 0
	}
	units := map[string]float64{
		"Ki": 1024,
		"Mi": 1024 * 1024,
		"Gi": 1024 * 1024 * 1024,
		"Ti": 1024 * 1024 * 1024 * 1024,
		"K":  1000,
		"M":  1000 * 1000,
		"G":  1000 * 1000 * 1000,
		"T":  1000 * 1000 * 1000 * 1000,
	}
	for suffix, factor := range units {
		if strings.HasSuffix(value, suffix) {
			number := strings.TrimSuffix(value, suffix)
			parsed, err := strconv.ParseFloat(number, 64)
			if err != nil {
				return 0
			}
			return int64(parsed * factor)
		}
	}
	parsed, err := strconv.ParseInt(value, 10, 64)
	if err != nil {
		return 0
	}
	return parsed
}

func formatBytesToGiB(value int64) string {
	if value <= 0 {
		return "0Gi"
	}
	gib := float64(value) / float64(1024*1024*1024)
	if gib >= 10 {
		return fmt.Sprintf("%.0fGi", gib)
	}
	text := fmt.Sprintf("%.1f", gib)
	text = strings.TrimRight(strings.TrimRight(text, "0"), ".")
	return text + "Gi"
}

func podStatusText(status podStatus) string {
	if reason := podContainerReason(status.InitContainerStatuses, true); reason != "" {
		return reason
	}
	if reason := podCurrentContainerReason(status.ContainerStatuses, false); reason != "" {
		return reason
	}
	if podHasRunningContainer(status.ContainerStatuses) {
		if status.Phase != "" {
			return status.Phase
		}
		return "Running"
	}
	if status.Reason != "" {
		return status.Reason
	}
	if status.Phase != "" {
		return status.Phase
	}
	if reason := podLastTerminatedReason(status.ContainerStatuses); reason != "" {
		return reason
	}
	return "Unknown"
}

func podContainerReason(statuses []podContainerStatus, isInit bool) string {
	for _, item := range statuses {
		if item.State.Waiting != nil && item.State.Waiting.Reason != "" {
			if isInit && item.State.Waiting.Reason == "PodInitializing" {
				continue
			}
			return item.State.Waiting.Reason
		}
		if item.State.Terminated != nil {
			if item.State.Terminated.Reason != "" {
				return item.State.Terminated.Reason
			}
			if item.State.Terminated.ExitCode != 0 {
				return fmt.Sprintf("ExitCode:%d", item.State.Terminated.ExitCode)
			}
		}
	}
	return ""
}

func podCurrentContainerReason(statuses []podContainerStatus, isInit bool) string {
	return podContainerReason(statuses, isInit)
}

func podHasRunningContainer(statuses []podContainerStatus) bool {
	for _, item := range statuses {
		if item.Ready || item.State.Running != nil {
			return true
		}
	}
	return false
}

func podLastTerminatedReason(statuses []podContainerStatus) string {
	for _, item := range statuses {
		if item.LastState.Terminated != nil && item.LastState.Terminated.Reason != "" {
			return item.LastState.Terminated.Reason
		}
	}
	return ""
}

func podReadyContainers(statuses []podContainerStatus) int {
	ready := 0
	for _, item := range statuses {
		if item.Ready {
			ready++
		}
	}
	return ready
}

func podContainerNames(containers []podContainer) []string {
	names := make([]string, 0, len(containers))
	for _, container := range containers {
		if strings.TrimSpace(container.Name) == "" {
			continue
		}
		names = append(names, container.Name)
	}
	return names
}

func formatPodRequestsLimits(containers []podContainer) string {
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

func fallbackText(value, fallback string) string {
	if strings.TrimSpace(value) == "" {
		return fallback
	}
	return value
}

func buildExecScript(command, workingDir string) string {
	escapedCommand := strings.ReplaceAll(command, "'", "'\"'\"'")
	if workingDir != "" {
		escapedDir := strings.ReplaceAll(workingDir, "'", "'\"'\"'")
		return fmt.Sprintf("cd '%s' 2>/dev/null || cd /; %s; echo; echo __CODEX_PWD__; pwd", escapedDir, escapedCommand)
	}
	return fmt.Sprintf("%s; echo; echo __CODEX_PWD__; pwd", escapedCommand)
}

func parseExecOutput(raw string) (string, string) {
	const marker = "__CODEX_PWD__"
	idx := strings.LastIndex(raw, marker)
	if idx == -1 {
		return strings.TrimRight(raw, "\r\n"), ""
	}

	output := strings.TrimRight(raw[:idx], "\r\n")
	workingDir := strings.TrimSpace(raw[idx+len(marker):])
	return output, workingDir
}

func stripNodeDebugNoise(raw string) string {
	lines := strings.Split(strings.ReplaceAll(raw, "\r\n", "\n"), "\n")
	filtered := make([]string, 0, len(lines))
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "Creating debugging pod ") && strings.Contains(trimmed, " with container ") && strings.Contains(trimmed, " on node ") {
			continue
		}
		filtered = append(filtered, line)
	}
	return strings.Join(filtered, "\n")
}

func cloneStringMap(input map[string]string) map[string]string {
	if len(input) == 0 {
		return map[string]string{}
	}
	output := make(map[string]string, len(input))
	for key, value := range input {
		output[key] = value
	}
	return output
}

func formatLabelPairs(labels map[string]string) []string {
	if len(labels) == 0 {
		return nil
	}
	keys := make([]string, 0, len(labels))
	for key := range labels {
		keys = append(keys, key)
	}
	sort.Strings(keys)

	pairs := make([]string, 0, len(keys))
	for _, key := range keys {
		pairs = append(pairs, fmt.Sprintf("%s=%s", key, labels[key]))
	}
	return pairs
}

func formatAffinityToleration(affinity map[string]interface{}, tolerations []podToleration) string {
	affinityText := "无亲和"
	if len(affinity) > 0 {
		affinityText = "有亲和"
	}
	tolerationText := "无容忍"
	if len(tolerations) > 0 {
		tolerationText = "有容忍"
	}
	return affinityText + " / " + tolerationText
}

func sanitizeLabels(input map[string]string) map[string]string {
	output := map[string]string{}
	for key, value := range input {
		key = strings.TrimSpace(key)
		if key == "" {
			continue
		}
		output[key] = strings.TrimSpace(value)
	}
	return output
}
