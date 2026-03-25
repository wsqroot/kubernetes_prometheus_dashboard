export function parseKubeconfigText(text) {
  const currentContext = (text.match(/current-context:\s*([^\r\n]+)/) || [])[1] || '';
  const sectionLines = text.split(/\r?\n/);
  let mode = '';
  const clusters = [];
  const contexts = [];

  sectionLines.forEach((line) => {
    const trimmed = line.trim();
    if (trimmed === 'clusters:') {
      mode = 'clusters';
      return;
    }
    if (trimmed === 'contexts:') {
      mode = 'contexts';
      return;
    }
    if (/^[A-Za-z-]+:/.test(trimmed) && trimmed !== 'clusters:' && trimmed !== 'contexts:') {
      if (!trimmed.startsWith('- name:')) {
        mode = '';
      }
    }
    if (trimmed.startsWith('- name:')) {
      const value = trimmed.replace('- name:', '').trim();
      if (mode === 'clusters') {
        clusters.push(value);
      }
      if (mode === 'contexts') {
        contexts.push(value);
      }
    }
  });

  return {
    clusters: [...new Set(clusters)],
    contexts: [...new Set(contexts)],
    currentContext: currentContext.trim()
  };
}

export function buildImportedClusters(clusterNames, existingClusters, currentContext) {
  const existingByName = new Map(existingClusters.map((item) => [item.name, item]));
  return clusterNames.map((clusterName, index) => {
    const existing = existingByName.get(clusterName);
    if (existing) {
      return existing;
    }

    const fallbackNamespaces = ['default', 'kube-system', 'monitoring'];
    return {
      id: `imported-${index + 1}`,
      name: clusterName,
      namespaces: fallbackNamespaces,
      stats: [
        { label: '在线节点数', value: '--', trend: '通过 kubeconfig 导入，待接入采集' },
        { label: '活跃命名空间', value: `${fallbackNamespaces.length}`, trend: '默认展示导入命名空间' },
        { label: '待处理告警', value: '--', trend: '待接入监控系统' }
      ],
      devices: [
        { name: '待接入节点清单', desc: '当前仅通过 kubeconfig 导入集群访问配置', status: 'online', text: '待采集' }
      ],
      tasks: [
        { name: 'kubeconfig 导入完成', desc: `上下文 ${currentContext || clusterName}` }
      ]
    };
  });
}
