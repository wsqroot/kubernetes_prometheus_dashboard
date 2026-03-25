const modules = {
  namespace: { title: '命名空间资源对象', types: [], scope: 'namespace' },
  controller: { title: '控制器资源对象', types: ['Deployment', 'StatefulSet', 'DaemonSet', 'Job', 'Pod'], scope: 'namespace' },
  node: { title: '节点资源', types: ['Node'], scope: 'cluster' },
  network: { title: '网络资源对象', types: ['Service', 'Ingress', 'Gateway'], scope: 'namespace' },
  storage: { title: '存储资源对象', types: ['PVC', 'Secret', 'ConfigMap', 'PV', 'StorageClass', 'VolumeSnapshot'], scope: 'namespace' },
  security: { title: '安全资源对象', types: ['ServiceAccount', 'RBAC', 'NetworkPolicy'], scope: 'namespace' }
};

const defaultAPIBase = (() => {
  const protocol = window.location.protocol === 'https:' ? 'https:' : 'http:';
  const hostname = window.location.hostname || '127.0.0.1';
  return `${protocol}//${hostname}:8080`;
})();

const state = {
  apiBase: defaultAPIBase,
  loginAPI: `${defaultAPIBase}/api/login`,
  form: { username: '', password: '' },
  passwordVisible: false,
  errorMessage: '',
  isSubmitting: false,
  isLoggedIn: false,
  userProfile: { username: '', displayName: '' },
  clusters: [],
  selectedClusterId: '',
  clusterLoadVersion: 0,
  namespaces: [],
  selectedNamespace: '',
  currentView: 'cluster',
  activeResourceSection: 'namespace',
  selectedTypes: { controller: 'Deployment', node: 'Node', network: 'Service', storage: 'PVC', security: 'ServiceAccount' },
  clusterOverview: null,
  namespaceResources: { controller: {}, network: {}, storage: {}, security: {} },
  nodeResources: [],
  controllerRows: [],
  podRows: [],
  searchKeyword: '',
  deletingKubeconfig: false,
  deletingResourceKey: '',
  loadingResources: false,
  expandedEditorKey: '',
  editorContext: null,
  editorYAML: '',
  editorLoading: false,
  editorSaving: false,
  editorError: '',
  editorSuccess: '',
  createVisible: false,
  createKind: '',
  createYAML: '',
  createSaving: false,
  createError: '',
  terminalPickerRow: null,
  pickerMode: 'terminal',
  terminalVisible: false,
  terminalContext: null,
  terminalCwd: '',
  terminalLines: [],
  terminalRunning: false,
  terminalError: '',
  logVisible: false,
  logContext: null,
  logLines: [],
  logKeyword: '',
  logLoading: false,
  logError: '',
  labelActionRowKey: '',
  labelActionMode: '',
  labelDraftSourceKey: '',
  labelDraftKey: '',
  labelDraftValue: '',
  labelDeleteSelections: {},
  labelSaving: false,
  labelError: '',
  labelSuccess: ''
};

const app = document.getElementById('app');
const emptySections = () => ({ controller: {}, network: {}, storage: {}, security: {} });
const rowKey = (kind, name, namespace = '') => `${kind}::${namespace}::${name}`;
const cloneMap = (value) => ({ ...(value || {}) });
const currentModule = () => modules[state.activeResourceSection];
const currentTabs = () => currentModule()?.types || [];
const currentType = () => state.activeResourceSection === 'namespace' ? '' : state.selectedTypes[state.activeResourceSection];
const clusterScoped = () => currentModule()?.scope === 'cluster';
const showNodeColumn = () => state.activeResourceSection === 'controller' && currentType() === 'Pod';
const tableColumnCount = () => showNodeColumn() ? 9 : 8;
const countHeader = () => state.activeResourceSection === 'node' ? 'Pod数量' : (state.activeResourceSection === 'controller' && currentType() !== 'Pod' ? '副本数' : '数量');
const targetHeader = () => state.activeResourceSection === 'node' ? 'Label标签' : (state.activeResourceSection === 'controller' && currentType() !== 'Pod' ? '期望状态' : '目标');
const objectHeader = () => state.activeResourceSection === 'node' ? 'IP地址' : (state.activeResourceSection === 'controller' && currentType() === 'Pod' ? 'Label标签' : '对象名称');
const runtimeHeader = () => state.activeResourceSection === 'node' ? '内存（剩余/总计）' : '运行数';
const requestsHeader = () => state.activeResourceSection === 'node' ? 'CPU（剩余/总计）' : '请求/限制';
const userInitial = () => (state.userProfile.displayName || state.userProfile.username || 'U').charAt(0).toUpperCase();
const escapeHTML = (value) => String(value).replaceAll('&', '&amp;').replaceAll('<', '&lt;').replaceAll('>', '&gt;').replaceAll('"', '&quot;').replaceAll("'", '&#39;');
const createResourceKind = () => state.activeResourceSection === 'namespace' ? 'Namespace' : (state.activeResourceSection === 'node' ? '' : currentType());
const createResourceLabel = () => createResourceKind() || currentModule().title;
const creatableKinds = new Set(['Namespace', 'Deployment', 'StatefulSet', 'DaemonSet', 'Job', 'Pod', 'Service', 'Ingress', 'Gateway', 'PVC', 'Secret', 'ConfigMap', 'PV', 'StorageClass', 'VolumeSnapshot', 'ServiceAccount', 'RBAC', 'NetworkPolicy']);
const namespacedKinds = new Set(['Deployment', 'StatefulSet', 'DaemonSet', 'Job', 'Pod', 'Service', 'Ingress', 'Gateway', 'PVC', 'Secret', 'ConfigMap', 'VolumeSnapshot', 'ServiceAccount', 'RBAC', 'NetworkPolicy']);
const isCreatableCurrentView = () => creatableKinds.has(createResourceKind());

function requestJSON(url, options = {}) {
  return fetch(url, options).catch(() => {
    throw new Error(`无法连接后端服务：${url}`);
  }).then(async (response) => {
    const result = await response.json();
    if (!response.ok || result.success !== true) throw new Error(result.message || '请求失败');
    return result.data;
  });
}

function resetCreateModal() { state.createVisible = false; state.createKind = ''; state.createYAML = ''; state.createSaving = false; state.createError = ''; }
function defaultResourceName(kind) { return `demo-${String(kind || 'resource').toLowerCase()}`; }
function namespaceYAMLLine(kind) { return namespacedKinds.has(kind) ? `  namespace: ${state.selectedNamespace || 'default'}\n` : ''; }
function bumpClusterLoadVersion() { state.clusterLoadVersion += 1; return state.clusterLoadVersion; }
function currentClusterLoadVersion() { return state.clusterLoadVersion; }

function buildCreateTemplate(kind) {
  const name = defaultResourceName(kind);
  switch (kind) {
    case 'Namespace':
      return `apiVersion: v1\nkind: Namespace\nmetadata:\n  name: ${name}\n`;
    case 'Deployment':
      return `apiVersion: apps/v1\nkind: Deployment\nmetadata:\n  name: ${name}\n${namespaceYAMLLine(kind)}spec:\n  replicas: 1\n  selector:\n    matchLabels:\n      app: ${name}\n  template:\n    metadata:\n      labels:\n        app: ${name}\n    spec:\n      containers:\n        - name: ${name}\n          image: nginx:stable\n          ports:\n            - containerPort: 80\n`;
    case 'StatefulSet':
      return `apiVersion: apps/v1\nkind: StatefulSet\nmetadata:\n  name: ${name}\n${namespaceYAMLLine(kind)}spec:\n  serviceName: ${name}\n  replicas: 1\n  selector:\n    matchLabels:\n      app: ${name}\n  template:\n    metadata:\n      labels:\n        app: ${name}\n    spec:\n      containers:\n        - name: ${name}\n          image: nginx:stable\n          ports:\n            - containerPort: 80\n`;
    case 'DaemonSet':
      return `apiVersion: apps/v1\nkind: DaemonSet\nmetadata:\n  name: ${name}\n${namespaceYAMLLine(kind)}spec:\n  selector:\n    matchLabels:\n      app: ${name}\n  template:\n    metadata:\n      labels:\n        app: ${name}\n    spec:\n      containers:\n        - name: ${name}\n          image: nginx:stable\n`;
    case 'Job':
      return `apiVersion: batch/v1\nkind: Job\nmetadata:\n  name: ${name}\n${namespaceYAMLLine(kind)}spec:\n  template:\n    spec:\n      restartPolicy: Never\n      containers:\n        - name: ${name}\n          image: busybox:1.36\n          command: ["sh", "-c", "echo hello"]\n`;
    case 'Pod':
      return `apiVersion: v1\nkind: Pod\nmetadata:\n  name: ${name}\n${namespaceYAMLLine(kind)}spec:\n  containers:\n    - name: ${name}\n      image: nginx:stable\n      ports:\n        - containerPort: 80\n`;
    case 'Service':
      return `apiVersion: v1\nkind: Service\nmetadata:\n  name: ${name}\n${namespaceYAMLLine(kind)}spec:\n  selector:\n    app: ${name}\n  ports:\n    - port: 80\n      targetPort: 80\n`;
    case 'Ingress':
      return `apiVersion: networking.k8s.io/v1\nkind: Ingress\nmetadata:\n  name: ${name}\n${namespaceYAMLLine(kind)}spec:\n  rules:\n    - host: example.local\n      http:\n        paths:\n          - path: /\n            pathType: Prefix\n            backend:\n              service:\n                name: ${name}\n                port:\n                  number: 80\n`;
    case 'Gateway':
      return `apiVersion: gateway.networking.k8s.io/v1\nkind: Gateway\nmetadata:\n  name: ${name}\n${namespaceYAMLLine(kind)}spec:\n  gatewayClassName: istio\n  listeners:\n    - name: http\n      protocol: HTTP\n      port: 80\n      allowedRoutes:\n        namespaces:\n          from: Same\n`;
    case 'PVC':
      return `apiVersion: v1\nkind: PersistentVolumeClaim\nmetadata:\n  name: ${name}\n${namespaceYAMLLine(kind)}spec:\n  accessModes:\n    - ReadWriteOnce\n  resources:\n    requests:\n      storage: 1Gi\n`;
    case 'Secret':
      return `apiVersion: v1\nkind: Secret\nmetadata:\n  name: ${name}\n${namespaceYAMLLine(kind)}type: Opaque\nstringData:\n  username: demo\n  password: demo123\n`;
    case 'ConfigMap':
      return `apiVersion: v1\nkind: ConfigMap\nmetadata:\n  name: ${name}\n${namespaceYAMLLine(kind)}data:\n  app.conf: |\n    key=value\n`;
    case 'PV':
      return `apiVersion: v1\nkind: PersistentVolume\nmetadata:\n  name: ${name}\nspec:\n  capacity:\n    storage: 1Gi\n  accessModes:\n    - ReadWriteOnce\n  persistentVolumeReclaimPolicy: Retain\n  hostPath:\n    path: /tmp/${name}\n`;
    case 'StorageClass':
      return `apiVersion: storage.k8s.io/v1\nkind: StorageClass\nmetadata:\n  name: ${name}\nprovisioner: kubernetes.io/no-provisioner\nvolumeBindingMode: WaitForFirstConsumer\n`;
    case 'VolumeSnapshot':
      return `apiVersion: snapshot.storage.k8s.io/v1\nkind: VolumeSnapshot\nmetadata:\n  name: ${name}\n${namespaceYAMLLine(kind)}spec:\n  source:\n    persistentVolumeClaimName: demo-pvc\n`;
    case 'ServiceAccount':
      return `apiVersion: v1\nkind: ServiceAccount\nmetadata:\n  name: ${name}\n${namespaceYAMLLine(kind)}`;
    case 'RBAC':
      return `apiVersion: rbac.authorization.k8s.io/v1\nkind: RoleBinding\nmetadata:\n  name: ${name}\n${namespaceYAMLLine(kind)}subjects:\n  - kind: ServiceAccount\n    name: default\n    namespace: ${state.selectedNamespace || 'default'}\nroleRef:\n  apiGroup: rbac.authorization.k8s.io\n  kind: ClusterRole\n  name: view\n`;
    case 'NetworkPolicy':
      return `apiVersion: networking.k8s.io/v1\nkind: NetworkPolicy\nmetadata:\n  name: ${name}\n${namespaceYAMLLine(kind)}spec:\n  podSelector: {}\n  policyTypes:\n    - Ingress\n`;
    default:
      return '';
  }
}

function filterRows(rows) {
  const keyword = state.searchKeyword.trim().toLowerCase();
  if (!keyword) return rows;
  return rows.filter((row) => JSON.stringify(row).toLowerCase().includes(keyword));
}

function flattenNamespaceRows() {
  return (state.namespaces || []).map((namespace) => ({
    key: rowKey('Namespace', namespace),
    kind: 'Namespace',
    name: namespace,
    labels: namespace,
    count: '-',
    target: '命名空间',
    status: '已加载',
    pods: '-',
    requestsLimits: '-',
    nodeName: '-',
    containers: [],
    labelPairs: [],
    labelMap: {},
    namespace: '',
    action: '删除'
  }));
}

function resourceRows() {
  if (state.activeResourceSection === 'namespace') return filterRows(flattenNamespaceRows());
  if (state.activeResourceSection === 'node') return filterRows(state.nodeResources);
  if (state.activeResourceSection === 'controller' && currentType() === 'Pod') return filterRows(state.podRows);
  if (state.activeResourceSection === 'controller' && ['Deployment', 'StatefulSet', 'DaemonSet', 'Job'].includes(currentType())) return filterRows(state.controllerRows);
  const items = state.namespaceResources[state.activeResourceSection]?.[currentType()] || [];
  return filterRows(items.map((name) => ({ key: rowKey(currentType(), name, state.selectedNamespace), kind: currentType(), name, labels: name, count: '1 个对象', target: `${currentType()}: ${name}`, status: '已加载', pods: '1', requestsLimits: '-', nodeName: '-', containers: [], labelPairs: [], labelMap: {}, namespace: state.selectedNamespace, action: '编辑' })));
}

function overviewCards() {
  if (!state.clusterOverview) return [];
  return [
    { title: '节点数量', value: state.clusterOverview.node_count, section: 'node', desc: '查看节点资源' },
    { title: '命名空间数量', value: state.clusterOverview.namespace_count, section: 'namespace', desc: '查看命名空间汇总' },
    { title: '控制器资源', value: state.clusterOverview.section_counts?.controller || 0, section: 'controller', desc: 'Deployment / StatefulSet / Pod' },
    { title: '网络资源', value: state.clusterOverview.section_counts?.network || 0, section: 'network', desc: 'Service / Ingress / Gateway' },
    { title: '存储资源', value: state.clusterOverview.section_counts?.storage || 0, section: 'storage', desc: 'PVC / Secret / ConfigMap' },
    { title: '安全资源', value: state.clusterOverview.section_counts?.security || 0, section: 'security', desc: 'ServiceAccount / RBAC / NetworkPolicy' }
  ];
}

function resetEditor() { state.expandedEditorKey=''; state.editorContext=null; state.editorYAML=''; state.editorError=''; state.editorSuccess=''; state.editorLoading=false; state.editorSaving=false; }
function resetTerminal() { state.terminalPickerRow=null; state.pickerMode='terminal'; state.terminalVisible=false; state.terminalContext=null; state.terminalCwd=''; state.terminalLines=[]; state.terminalRunning=false; state.terminalError=''; }
function resetLogs() { state.logVisible=false; state.logContext=null; state.logLines=[]; state.logKeyword=''; state.logLoading=false; state.logError=''; }
function resetLabelAction() { state.labelActionRowKey=''; state.labelActionMode=''; state.labelDraftSourceKey=''; state.labelDraftKey=''; state.labelDraftValue=''; state.labelDeleteSelections={}; state.labelSaving=false; state.labelError=''; state.labelSuccess=''; }

async function openCreateModal() {
  const kind = createResourceKind();
  if (!creatableKinds.has(kind)) throw new Error('当前页面不支持创建该资源');
  state.createVisible = true;
  state.createKind = kind;
  state.createYAML = buildCreateTemplate(kind);
  state.createSaving = false;
  state.createError = '';
  render();
}

async function saveCreateResource() {
  if (!state.createVisible || !state.createYAML.trim()) return;
  state.createSaving = true;
  state.createError = '';
  render();
  try {
    await requestJSON(`${state.apiBase}/api/resource/yaml?cluster_id=${encodeURIComponent(state.selectedClusterId)}`, {
      method: 'PUT',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ content: state.createYAML })
    });
    resetCreateModal();
    await refreshCurrentView();
  } catch (error) {
    state.createError = error.message || '创建失败';
  } finally {
    state.createSaving = false;
    render();
  }
}
async function loadClusters(preferredClusterId = '', preferredNamespace = '') {
  const clusters = (await requestJSON(`${state.apiBase}/api/load_kubeconfig/clusters`)) || [];
  state.clusters = clusters;
  if (!clusters.length) {
    state.selectedClusterId = ''; state.clusterOverview = null; state.namespaces = []; state.namespaceResources = emptySections(); state.nodeResources = []; state.controllerRows = []; state.podRows = []; render(); return;
  }
  const found = clusters.find((item) => item.id === preferredClusterId);
  state.selectedClusterId = found ? found.id : clusters[0].id;
  await loadClusterData(preferredNamespace, state.selectedClusterId, bumpClusterLoadVersion());
}

async function loadClusterData(preferredNamespace = '', clusterId = state.selectedClusterId, loadVersion = currentClusterLoadVersion()) {
  if (!clusterId) return;
  const overview = await requestJSON(`${state.apiBase}/api/namespace/overview?cluster_id=${encodeURIComponent(clusterId)}`);
  const namespaceData = await requestJSON(`${state.apiBase}/api/namespace/list?cluster_id=${encodeURIComponent(clusterId)}`);
  if (clusterId !== state.selectedClusterId || loadVersion !== currentClusterLoadVersion()) return;
  state.clusterOverview = overview;
  state.namespaces = namespaceData.items || [];
  state.selectedNamespace = state.namespaces.includes(preferredNamespace) ? preferredNamespace : (state.namespaces[0] || '');
  window.localStorage.setItem('dashboard_cluster', clusterId);
  if (state.selectedNamespace) window.localStorage.setItem('dashboard_namespace', state.selectedNamespace);
  resetEditor(); resetTerminal(); resetLogs(); resetLabelAction();
  await refreshCurrentView(loadVersion);
}

async function refreshNamespaceList(preferredNamespace = state.selectedNamespace, clusterId = state.selectedClusterId, loadVersion = currentClusterLoadVersion()) {
  if (!clusterId) return;
  const namespaceData = await requestJSON(`${state.apiBase}/api/namespace/list?cluster_id=${encodeURIComponent(clusterId)}`);
  if (clusterId !== state.selectedClusterId || loadVersion !== currentClusterLoadVersion()) return;
  state.namespaces = namespaceData.items || [];
  state.selectedNamespace = state.namespaces.includes(preferredNamespace) ? preferredNamespace : (state.namespaces[0] || '');
  if (state.selectedNamespace) window.localStorage.setItem('dashboard_namespace', state.selectedNamespace);
  render();
}
async function loadNamespaceResources(loadVersion = currentClusterLoadVersion()) { if (!state.selectedClusterId || !state.selectedNamespace) return; const clusterId = state.selectedClusterId; const namespace = state.selectedNamespace; state.loadingResources = true; render(); try { const data = await requestJSON(`${state.apiBase}/api/namespace/resources?cluster_id=${encodeURIComponent(clusterId)}&namespace=${encodeURIComponent(namespace)}`); if (clusterId !== state.selectedClusterId || namespace !== state.selectedNamespace || loadVersion !== currentClusterLoadVersion()) return; state.namespaceResources = data.sections || emptySections(); } finally { state.loadingResources = false; render(); } }
async function loadNodeResources(loadVersion = currentClusterLoadVersion()) { if (!state.selectedClusterId) return; const clusterId = state.selectedClusterId; state.loadingResources = true; render(); try { const data = await requestJSON(`${state.apiBase}/api/namespace/nodes?cluster_id=${encodeURIComponent(clusterId)}`); if (clusterId !== state.selectedClusterId || loadVersion !== currentClusterLoadVersion()) return; state.nodeResources = (data.items || []).map((item) => ({ key: rowKey('Node', item.name, ''), kind: 'Node', name: item.name, labels: item.ip_address || '-', count: item.count || '0', target: '', status: item.status || '-', pods: item.runtime || '-', requestsLimits: item.cpu || '-', nodeName: '-', containers: [], labelPairs: item.label_pairs || [], labelMap: item.labels || {}, namespace: '', action: item.action || '编辑' })); } finally { state.loadingResources = false; render(); } }
async function loadControllerResources(loadVersion = currentClusterLoadVersion()) { if (!state.selectedClusterId || !state.selectedNamespace) return; const clusterId = state.selectedClusterId; const namespace = state.selectedNamespace; const type = currentType(); const endpointMap = { Deployment: 'deployments', StatefulSet: 'statefulsets', DaemonSet: 'daemonsets', Job: 'jobs' }; const endpoint = endpointMap[type]; if (!endpoint) { state.controllerRows = []; render(); return; } state.loadingResources = true; render(); try { const data = await requestJSON(`${state.apiBase}/api/controller/${endpoint}?cluster_id=${encodeURIComponent(clusterId)}&namespace=${encodeURIComponent(namespace)}`); if (clusterId !== state.selectedClusterId || namespace !== state.selectedNamespace || type !== currentType() || loadVersion !== currentClusterLoadVersion()) return; state.controllerRows = (data.items || []).map((item) => ({ key: rowKey(type, item.name, namespace), kind: type, name: item.name, labels: item.object_name, count: item.count, target: item.target, status: item.status, pods: item.runtime, requestsLimits: item.requests_limits || '-', nodeName: '-', containers: [], labelPairs: [], labelMap: {}, namespace, action: item.action || '编辑' })); } finally { state.loadingResources = false; render(); } }
async function loadPodResources(loadVersion = currentClusterLoadVersion()) { if (!state.selectedClusterId || !state.selectedNamespace) return; const clusterId = state.selectedClusterId; const namespace = state.selectedNamespace; state.loadingResources = true; render(); try { const data = await requestJSON(`${state.apiBase}/api/controller/pods?cluster_id=${encodeURIComponent(clusterId)}&namespace=${encodeURIComponent(namespace)}`); if (clusterId !== state.selectedClusterId || namespace !== state.selectedNamespace || loadVersion !== currentClusterLoadVersion()) return; state.podRows = (data.items || []).map((item) => ({ key: rowKey('Pod', item.name, namespace), kind: 'Pod', name: item.name, labels: item.object_name, count: item.count, target: item.target, status: item.status, pods: item.runtime, requestsLimits: item.requests_limits || '-', nodeName: item.node_name || '-', containers: item.containers || [], labelPairs: item.label_pairs || [], labelMap: item.labels || {}, namespace, action: item.action || '编辑' })); } finally { state.loadingResources = false; render(); } }
async function refreshCurrentView(loadVersion = currentClusterLoadVersion()) {
  if (!state.selectedClusterId) return;
  if (state.activeResourceSection === 'namespace') return refreshNamespaceList(state.selectedNamespace, state.selectedClusterId, loadVersion);
  if (state.activeResourceSection === 'node') return loadNodeResources(loadVersion);
  if (state.activeResourceSection === 'controller' && currentType() === 'Pod') return loadPodResources(loadVersion);
  if (state.activeResourceSection === 'controller' && ['Deployment','StatefulSet','DaemonSet','Job'].includes(currentType())) return loadControllerResources(loadVersion);
  if (state.selectedNamespace) return loadNamespaceResources(loadVersion);
}

function highlightLogLine(line) {
  const keyword = state.logKeyword.trim();
  const escaped = escapeHTML(line);
  if (!keyword) return escaped;
  const lowerLine = line.toLowerCase();
  const lowerKeyword = keyword.toLowerCase();
  let start = 0; let out = '';
  while (start < line.length) {
    const idx = lowerLine.indexOf(lowerKeyword, start);
    if (idx === -1) { out += escapeHTML(line.slice(start)); break; }
    out += escapeHTML(line.slice(start, idx));
    out += `<span class="log-highlight">${escapeHTML(line.slice(idx, idx + keyword.length))}</span>`;
    start = idx + keyword.length;
  }
  return out;
}

function labelRequestConfig(row, labels) { return row.kind === 'Node' ? { url: `${state.apiBase}/api/node/labels?cluster_id=${encodeURIComponent(state.selectedClusterId)}`, body: { node_name: row.name, labels } } : { url: `${state.apiBase}/api/controller/pod-labels?cluster_id=${encodeURIComponent(state.selectedClusterId)}`, body: { namespace: row.namespace, pod_name: row.name, labels } }; }
async function refreshRowData(row) { if (row.kind === 'Node') return loadNodeResources(); if (row.kind === 'Pod') return loadPodResources(); return refreshCurrentView(); }
function findRowByKey(key) { return resourceRows().find((row) => row.key === key) || state.podRows.find((row) => row.key === key) || state.nodeResources.find((row) => row.key === key) || state.controllerRows.find((row) => row.key === key); }

async function submitLabelAction(row) {
  state.labelSaving = true; state.labelError = ''; state.labelSuccess = ''; render();
  try {
    const current = cloneMap(row.labelMap); let labels = {};
    if (state.labelActionMode === 'add') { const key = state.labelDraftKey.trim(); if (!key) throw new Error('标签 Key 不能为空'); current[key] = state.labelDraftValue.trim(); labels = current; }
    else if (state.labelActionMode === 'edit') { const sourceKey = state.labelDraftSourceKey.trim(); const newKey = state.labelDraftKey.trim(); if (!sourceKey || !newKey) throw new Error('请选择要修改的标签并填写新的 Key'); delete current[sourceKey]; current[newKey] = state.labelDraftValue.trim(); labels = current; }
    else if (state.labelActionMode === 'delete') { Object.entries(current).forEach(([key, value]) => { if (!state.labelDeleteSelections[key]) labels[key] = value; }); }
    else throw new Error('未选择标签操作');
    const request = labelRequestConfig(row, labels);
    await requestJSON(request.url, { method: 'PUT', headers: { 'Content-Type': 'application/json' }, body: JSON.stringify(request.body) });
    await refreshRowData(row);
    state.labelActionRowKey = row.key;
    state.labelActionMode = '';
    state.labelDraftSourceKey = '';
    state.labelDraftKey = '';
    state.labelDraftValue = '';
    state.labelDeleteSelections = {};
    state.labelError = '';
    state.labelSuccess = '标签已保存';
  } catch (error) { state.labelError = error.message || '标签保存失败'; }
  finally { state.labelSaving = false; render(); }
}

async function toggleEditor(row) {
  if (state.expandedEditorKey === row.key) { resetEditor(); render(); return; }
  state.expandedEditorKey = row.key; state.editorContext = { key: row.key, kind: row.kind, name: row.name, namespace: row.namespace || '' }; state.editorYAML = ''; state.editorError = ''; state.editorSuccess = ''; state.editorLoading = true; render();
  try {
    const params = new URLSearchParams({ cluster_id: state.selectedClusterId, kind: row.kind, name: row.name });
    if (row.namespace) params.set('namespace', row.namespace);
    const data = await requestJSON(`${state.apiBase}/api/resource/yaml?${params.toString()}`);
    state.editorYAML = data.content || '';
  } catch (error) { state.editorError = error.message || '加载 YAML 失败'; }
  finally { state.editorLoading = false; render(); }
}

async function saveEditor() {
  if (!state.editorContext) return;
  state.editorSaving = true; state.editorError = ''; state.editorSuccess = ''; render();
  try { await requestJSON(`${state.apiBase}/api/resource/yaml?cluster_id=${encodeURIComponent(state.selectedClusterId)}`, { method: 'PUT', headers: { 'Content-Type': 'application/json' }, body: JSON.stringify({ content: state.editorYAML }) }); state.editorSuccess = '修改成功'; await refreshCurrentView(); }
  catch (error) { state.editorError = error.message || '保存失败'; }
  finally { state.editorSaving = false; render(); }
}

async function deleteAnyResource(row) {
  if (!window.confirm(`确认删除 ${row.kind} ${row.name} 吗？`)) return;
  state.deletingResourceKey = row.key; render();
  try {
    await requestJSON(`${state.apiBase}/api/resource/delete?cluster_id=${encodeURIComponent(state.selectedClusterId)}`, { method: 'DELETE', headers: { 'Content-Type': 'application/json' }, body: JSON.stringify({ kind: row.kind, name: row.name, namespace: row.namespace || '' }) });
    if (row.kind === 'Namespace') {
      state.namespaces = state.namespaces.filter((item) => item !== row.name);
      if (state.selectedNamespace === row.name) state.selectedNamespace = state.namespaces[0] || '';
    }
    resetEditor(); resetTerminal(); resetLogs(); resetLabelAction();
    await refreshCurrentView();
  }
  finally { state.deletingResourceKey = ''; render(); }
}
function renderLabelControls(row) {
  if (!(row.kind === 'Pod' || row.kind === 'Node')) return row.kind === 'Node' ? '-' : escapeHTML(row.labels || '-');
  const labels = row.kind === 'Pod' ? row.labelPairs : [];
  const pairs = row.labelPairs || [];
  const tags = pairs.length ? `<div class="label-tag-list">${pairs.map((label) => state.labelActionRowKey === row.key && state.labelActionMode === 'delete' ? `<label class="label-delete-item"><input type="checkbox" data-action="toggle-label-delete" data-row-key="${row.key}" data-label-key="${escapeHTML(label.split('=')[0])}" ${state.labelDeleteSelections[label.split('=')[0]] ? 'checked' : ''} /><span class="label-tag">${escapeHTML(label)}</span></label>` : `<span class="label-tag">${escapeHTML(label)}</span>`).join('')}</div>` : '<span>-</span>';
  const editor = state.labelActionRowKey === row.key ? `<div class="label-inline-editor">${state.labelActionMode === 'add' ? `<div class="label-editor-row"><input data-action="label-draft-key" class="toolbar-search label-input" type="text" placeholder="标签 Key" value="${escapeHTML(state.labelDraftKey)}" /><input data-action="label-draft-value" class="toolbar-search label-input" type="text" placeholder="标签 Value" value="${escapeHTML(state.labelDraftValue)}" /></div>` : ''}${state.labelActionMode === 'edit' ? `<div class="label-editor-stack"><select data-action="label-source-key" class="toolbar-select label-action-select"><option value="">选择要修改的标签</option>${pairs.map((label) => { const key = label.split('=')[0]; return `<option value="${escapeHTML(key)}" ${state.labelDraftSourceKey === key ? 'selected' : ''}>${escapeHTML(label)}</option>`; }).join('')}</select><div class="label-editor-row"><input data-action="label-draft-key" class="toolbar-search label-input" type="text" placeholder="新的标签 Key" value="${escapeHTML(state.labelDraftKey)}" /><input data-action="label-draft-value" class="toolbar-search label-input" type="text" placeholder="新的标签 Value" value="${escapeHTML(state.labelDraftValue)}" /></div></div>` : ''}${state.labelActionMode === 'delete' ? '<div class="label-delete-tip">勾选需要删除的标签后点击删除</div>' : ''}<div class="yaml-editor-feedback">${state.labelError ? `<span class="yaml-editor-error">${escapeHTML(state.labelError)}</span>` : ''}${state.labelSuccess ? `<span class="yaml-editor-success">${escapeHTML(state.labelSuccess)}</span>` : ''}</div><div class="yaml-editor-actions"><button type="button" class="toolbar-btn primary" data-action="submit-label" data-row-key="${row.key}">${state.labelSaving ? '保存中...' : (state.labelActionMode === 'delete' ? '删除' : '保存')}</button><button type="button" class="toolbar-btn" data-action="reset-label">关闭</button></div></div>` : '';
  return `${tags}<div class="label-action-toolbar"><select class="toolbar-select label-action-select" data-action="label-mode" data-row-key="${row.key}"><option value="">标签操作</option><option value="add">增加</option><option value="delete">删除</option><option value="edit">修改</option></select></div>${editor}`;
}

function renderRows() {
  const rows = resourceRows();
  if (!rows.length) return `<tr><td colspan="${tableColumnCount()}" class="table-empty">${state.loadingResources ? '资源加载中...' : '没有匹配的资源'}</td></tr>`;
  return rows.map((row) => {
    const actions = row.kind === 'Namespace'
      ? `<a href="#" class="table-action danger" data-action="delete-resource" data-row-key="${row.key}">${state.deletingResourceKey === row.key ? '删除中...' : '删除'}</a>`
      : `${row.kind === 'Node' ? `<a href="#" class="table-action" data-action="node-login" data-row-key="${row.key}">登录</a>` : ''}${row.kind === 'Pod' ? `<a href="#" class="table-action" data-action="pod-login" data-row-key="${row.key}">登录</a><a href="#" class="table-action" data-action="pod-logs" data-row-key="${row.key}">日志</a>` : ''}<a href="#" class="table-action danger" data-action="delete-resource" data-row-key="${row.key}">${state.deletingResourceKey === row.key ? '删除中...' : '删除'}</a><a href="#" class="table-action" data-action="toggle-editor" data-row-key="${row.key}">${state.expandedEditorKey === row.key ? '收起' : '编辑'}</a>`;
    const targetCell = row.kind === 'Node' ? renderLabelControls(row) : escapeHTML(row.target || '-');
    const objectCell = row.kind === 'Pod' ? renderLabelControls(row) : escapeHTML(row.labels || '-');
    const expanded = state.expandedEditorKey === row.key ? `<tr class="resource-expand-row"><td colspan="${tableColumnCount()}"><div class="resource-expand-box"><div class="resource-expand-title">${escapeHTML(row.kind)} / ${escapeHTML(row.name)}</div>${state.editorLoading ? '<div class="yaml-editor-loading">YAML 加载中...</div>' : `<div class="yaml-editor-shell"><textarea id="yaml-editor" class="yaml-editor" spellcheck="false">${escapeHTML(state.editorYAML)}</textarea><div class="yaml-editor-feedback">${state.editorError ? `<span class="yaml-editor-error">${escapeHTML(state.editorError)}</span>` : ''}${state.editorSuccess ? `<span class="yaml-editor-success">${escapeHTML(state.editorSuccess)}</span>` : ''}</div><div class="yaml-editor-actions"><button type="button" class="toolbar-btn primary" data-action="save-editor">${state.editorSaving ? '保存中...' : '保存 YAML'}</button><button type="button" class="toolbar-btn" data-action="reset-editor">关闭</button></div></div>`}</div></td></tr>` : '';
    return `<tr><td class="resource-name-cell">${escapeHTML(row.name)}</td><td>${objectCell}</td><td>${escapeHTML(row.count)}</td><td>${targetCell}</td><td>${escapeHTML(row.status || '-')}</td><td>${escapeHTML(row.pods || '-')}</td>${showNodeColumn() ? `<td>${escapeHTML(row.nodeName || '-')}</td>` : ''}<td>${escapeHTML(row.requestsLimits || '-')}</td><td>${actions}</td></tr>${expanded}`;
  }).join('');
}

function renderClusterView() { return `<div class="cluster-overview"><div class="cluster-overview-hero"><div><h2>${escapeHTML(state.currentCluster?.name || state.currentCluster?.cluster_name || state.currentCluster?.id || state.currentCluster?.source_file || state.currentCluster?.context_name || state.currentCluster?.displayName || state.currentCluster?.title || state.currentCluster?.label || state.currentCluster?.cluster || state.currentCluster?.clusterName || state.currentCluster?.context || state.currentCluster?.source || state.currentCluster?.source_file || state.currentCluster?.name || '暂无集群')}</h2><p>概览卡片和命名空间列表都支持点击进入。</p></div><div class="cluster-overview-actions"><button type="button" class="toolbar-btn primary" data-action="set-section" data-section="namespace">命名空间</button><button type="button" class="toolbar-btn" data-action="set-section" data-section="controller">控制器</button><button type="button" class="toolbar-btn" data-action="set-section" data-section="node">节点</button></div></div><div class="cluster-summary-grid">${overviewCards().map((item) => `<article class="cluster-summary-card compact clickable-card" data-action="set-section" data-section="${item.section}"><div class="stat-label">${escapeHTML(item.title)}</div><div class="stat-value">${escapeHTML(item.value)}</div><div class="cluster-summary-desc">${escapeHTML(item.desc)}</div></article>`).join('')}</div><div class="cluster-panel-grid"><article class="cluster-panel"><h3>已导入配置</h3><div class="cluster-panel-list"><div class="cluster-panel-item"><div><div class="cluster-panel-name">${escapeHTML(state.clusterOverview?.cluster_name || '-')}</div><div class="cluster-panel-desc">上下文：${escapeHTML(state.clusterOverview?.context_name || '-')}</div></div><span class="cluster-panel-action">${escapeHTML(state.currentCluster?.source_file || '-')}</span></div></div></article><article class="cluster-panel"><h3>命名空间列表</h3><div class="cluster-panel-list">${state.namespaces.map((ns) => `<div class="cluster-panel-item clickable" data-action="open-namespace" data-namespace="${escapeHTML(ns)}"><div><div class="cluster-panel-name">${escapeHTML(ns)}</div><div class="cluster-panel-desc">打开该命名空间下的全部资源对象</div></div><span class="cluster-panel-action">打开</span></div>`).join('')}</div></article></div></div>`; }

function renderResourceView() { return `<div class="resource-page"><div class="resource-breadcrumb">${escapeHTML(state.currentCluster?.name || '')} / ${escapeHTML(clusterScoped() || state.activeResourceSection === 'namespace' ? '集群级' : state.selectedNamespace)} / ${escapeHTML(currentModule().title)}</div>${currentTabs().length ? `<div class="resource-tabs">${currentTabs().map((tab) => `<button type="button" class="resource-tab ${currentType() === tab ? 'active' : ''}" data-action="select-type" data-section="${state.activeResourceSection}" data-type="${tab}">${tab}</button>`).join('')}</div>` : ''}<div class="resource-toolbar"><div class="toolbar-actions">${isCreatableCurrentView() ? `<button type="button" class="toolbar-btn primary" data-action="open-create">创建${escapeHTML(createResourceLabel())}</button>` : ''}</div><div class="toolbar-filters">${!clusterScoped() && state.activeResourceSection !== 'namespace' ? `<select id="namespace-select" class="toolbar-select">${state.namespaces.map((ns) => `<option value="${escapeHTML(ns)}" ${state.selectedNamespace === ns ? 'selected' : ''}>${escapeHTML(ns)}</option>`).join('')}</select>` : ''}<input id="search-input" class="toolbar-search" type="text" placeholder="搜索" value="${escapeHTML(state.searchKeyword)}" /></div></div><div class="resource-table-shell"><table class="resource-table"><thead><tr><th>名称</th><th>${objectHeader()}</th><th>${countHeader()}</th><th>${targetHeader()}</th><th>状态</th><th>${runtimeHeader()}</th>${showNodeColumn() ? '<th>节点</th>' : ''}<th>${requestsHeader()}</th><th>操作</th></tr></thead><tbody>${renderRows()}</tbody></table></div></div>`; }

function renderCreateModal() { if (!state.createVisible) return ''; return `<div class="terminal-modal-backdrop"><div class="terminal-modal label-editor-modal"><div class="terminal-modal-header"><div><div class="terminal-title">创建 ${escapeHTML(state.createKind)}</div><div class="terminal-subtitle">${escapeHTML(clusterScoped() ? '集群级资源' : `命名空间 ${state.selectedNamespace || '-'}`)}</div></div><button type="button" class="toolbar-btn" data-action="close-create">关闭</button></div><div class="resource-expand-box"><div class="yaml-editor-shell"><textarea id="create-yaml-editor" class="yaml-editor" spellcheck="false">${escapeHTML(state.createYAML)}</textarea><div class="yaml-editor-feedback">${state.createError ? `<span class="yaml-editor-error">${escapeHTML(state.createError)}</span>` : ''}</div><div class="yaml-editor-actions"><button type="button" class="toolbar-btn primary" data-action="save-create">${state.createSaving ? '创建中...' : '创建资源'}</button><button type="button" class="toolbar-btn" data-action="close-create">取消</button></div></div></div></div></div>`; }

function renderTerminalPicker() { if (!state.terminalPickerRow) return ''; return `<div class="terminal-modal-backdrop"><div class="terminal-picker"><div class="terminal-picker-title">${state.pickerMode === 'logs' ? '选择要查看日志的容器' : '选择要进入的容器'}</div><div class="terminal-picker-subtitle">${escapeHTML(state.terminalPickerRow.name)}</div><div class="terminal-picker-list">${state.terminalPickerRow.containers.map((container) => `<button type="button" class="terminal-picker-btn" data-action="pick-container" data-container="${escapeHTML(container)}">${escapeHTML(container)}</button>`).join('')}</div><div class="yaml-editor-actions"><button type="button" class="toolbar-btn" data-action="close-picker">关闭</button></div></div></div>`; }
function renderTerminalModal() { if (!state.terminalVisible || !state.terminalContext) return ''; return `<div class="terminal-modal-backdrop"><div class="terminal-modal"><div class="terminal-modal-header"><div><div class="terminal-title">${escapeHTML(state.terminalContext.mode === 'node' ? state.terminalContext.nodeName : state.terminalContext.podName)}</div><div class="terminal-subtitle">${escapeHTML(state.terminalContext.mode === 'node' ? '节点终端' : `${state.terminalContext.namespace} / ${state.terminalContext.container}`)}</div></div><button type="button" class="toolbar-btn" data-action="close-terminal">关闭</button></div><div class="terminal-screen">${state.terminalLines.map((line) => `<div class="terminal-line">${escapeHTML(line)}</div>`).join('')}${state.terminalError ? `<div class="terminal-line terminal-error">${escapeHTML(state.terminalError)}</div>` : ''}<div class="terminal-command-line"><span class="terminal-prompt">${escapeHTML(state.terminalCwd || '~')} $</span><div id="terminal-command" class="terminal-command" contenteditable="true" spellcheck="false"></div></div></div></div></div>`; }
function renderLogsModal() { if (!state.logVisible || !state.logContext) return ''; const lines = state.logKeyword.trim() ? state.logLines.filter((line) => line.toLowerCase().includes(state.logKeyword.trim().toLowerCase())) : state.logLines; return `<div class="terminal-modal-backdrop"><div class="terminal-modal"><div class="terminal-modal-header"><div><div class="terminal-title">容器日志</div><div class="terminal-subtitle">${escapeHTML(`${state.logContext.namespace} / ${state.logContext.podName} / ${state.logContext.container}`)}</div></div><button type="button" class="toolbar-btn" data-action="close-logs">关闭</button></div><div class="log-toolbar"><input id="log-keyword-input" class="toolbar-search log-search" type="text" placeholder="搜索日志关键字" value="${escapeHTML(state.logKeyword)}" /></div><div class="terminal-screen">${state.logLoading ? '<div class="terminal-line">日志加载中...</div>' : state.logError ? `<div class="terminal-line terminal-error">${escapeHTML(state.logError)}</div>` : !lines.length ? '<div class="terminal-line">没有匹配的日志</div>' : lines.map((line) => `<div class="terminal-line">${highlightLogLine(line)}</div>`).join('')}</div></div></div>`; }
function renderLogin() { app.innerHTML = `<div class="login-shell"><div class="login-wrap"><div class="login-card"><div class="login-visual"></div><div class="login-form"><div class="brand"><div class="brand-badge"><span class="a"></span><span class="b"></span><span class="c"></span></div><h1>Kubernetes 多集群平台</h1></div><div class="field"><input id="username-input" type="text" placeholder="用户名" value="${escapeHTML(state.form.username)}" /></div><div class="field"><input id="password-input" type="${state.passwordVisible ? 'text' : 'password'}" placeholder="密码" value="${escapeHTML(state.form.password)}" /><span class="toggle-visibility" data-action="toggle-password">${state.passwordVisible ? '隐藏' : '显示'}</span></div><div class="error-tip">${escapeHTML(state.errorMessage)}</div><button class="login-btn" ${state.isSubmitting ? 'disabled' : ''} data-action="submit-login">${state.isSubmitting ? '登录中...' : '登录'}</button></div></div></div></div>`; }
function renderDashboard() { const current = state.clusters.find((item) => item.id === state.selectedClusterId) || {}; state.currentCluster = current; app.innerHTML = `<div class="dashboard-shell"><div class="dashboard"><aside class="sidebar"><div class="brand"><div class="brand-badge"><span class="a"></span><span class="b"></span><span class="c"></span></div><div class="cluster-switcher"><label class="cluster-label">集群</label><select id="cluster-select" class="cluster-select">${state.clusters.map((cluster) => `<option value="${escapeHTML(cluster.id)}" ${state.selectedClusterId === cluster.id ? 'selected' : ''}>${escapeHTML(cluster.name)}</option>`).join('')}</select></div></div>${Object.entries(modules).map(([key, value]) => `<div class="nav-group ${state.currentView === 'resource' && state.activeResourceSection === key ? 'open' : ''}"><div class="nav-item ${state.currentView === 'resource' && state.activeResourceSection === key ? 'active' : ''}" data-action="set-section" data-section="${key}">${escapeHTML(value.title)}</div>${state.currentView === 'resource' && state.activeResourceSection === key && currentTabs().length ? `<div class="nav-subpanel"><div class="nav-chip-list">${currentTabs().map((type) => `<button type="button" class="nav-chip ${currentType() === type ? 'selected' : ''}" data-action="select-type" data-section="${key}" data-type="${type}">${type}</button>`).join('')}</div></div>` : ''}</div>`).join('')}</aside><main class="main-panel"><section class="workspace-shell"><div class="workspace-header"><div class="workspace-context"><button type="button" class="context-back-btn" data-action="go-cluster">返回 ${escapeHTML(current.name || '')}</button>${state.currentView === 'resource' ? `<span>/ ${escapeHTML(clusterScoped() ? '集群级' : state.selectedNamespace)} / ${escapeHTML(currentModule().title)}</span>` : ''}<span class="context-status">运行中</span></div><div class="workspace-actions"><button type="button" class="toolbar-btn" data-action="refresh-dashboard">刷新</button><div class="user-panel"><div class="avatar">${escapeHTML(userInitial())}</div><button class="logout-btn" data-action="logout">退出登录</button></div></div></div>${state.currentView === 'cluster' ? renderClusterView() : renderResourceView()}</section></main></div>${renderCreateModal()}${renderTerminalPicker()}${renderTerminalModal()}${renderLogsModal()}</div>`; }
function render() { if (!state.isLoggedIn) renderLogin(); else renderDashboard(); bindEvents(); bindCreateEditor(); }

function splitLines(text) {
  return String(text || '').replaceAll('\r\n', '\n').split('\n');
}

function pushTerminalOutput(command, output, cwd) {
  if (command) state.terminalLines.push(`${state.terminalCwd || '~'} $ ${command}`);
  splitLines(output).forEach((line) => state.terminalLines.push(line));
  state.terminalCwd = cwd || state.terminalCwd || '~';
}

function bindCreateEditor() {
  const createEditor = document.getElementById('create-yaml-editor');
  if (createEditor) {
    createEditor.oninput = (event) => {
      state.createYAML = event.target.value;
    };
  }
}

function flashMessage(message) {
  if (!message) return;
  window.setTimeout(() => {
    if (state.editorSuccess === message) {
      state.editorSuccess = '';
      render();
    }
    if (state.labelSuccess === message) {
      state.labelSuccess = '';
      render();
    }
  }, 1800);
}

function openResourceSection(section) {
  state.currentView = 'resource';
  state.activeResourceSection = section;
  resetCreateModal();
  resetEditor();
  resetTerminal();
  resetLogs();
  resetLabelAction();
  render();
  refreshCurrentView().catch((error) => window.alert(error.message || '加载资源失败'));
}

async function submitLogin() {
  state.isSubmitting = true;
  state.errorMessage = '';
  render();
  try {
    const data = await requestJSON(state.loginAPI, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({
        username: state.form.username.trim(),
        password: state.form.password.trim()
      })
    });
    state.isLoggedIn = true;
    state.userProfile = {
      username: data.username || state.form.username.trim(),
      displayName: data.username || state.form.username.trim()
    };
    window.localStorage.setItem('dashboard_user', JSON.stringify(state.userProfile));
    render();
    await loadClusters(window.localStorage.getItem('dashboard_cluster') || '', window.localStorage.getItem('dashboard_namespace') || '');
  } catch (error) {
    state.errorMessage = error.message || '登录失败';
    render();
  } finally {
    state.isSubmitting = false;
    render();
  }
}

function logout() {
  state.isLoggedIn = false;
  state.userProfile = { username: '', displayName: '' };
  state.clusters = [];
  state.selectedClusterId = '';
  state.namespaces = [];
  state.selectedNamespace = '';
  state.clusterOverview = null;
  state.namespaceResources = emptySections();
  state.nodeResources = [];
  state.controllerRows = [];
  state.podRows = [];
  state.currentView = 'cluster';
  resetCreateModal();
  resetEditor();
  resetTerminal();
  resetLogs();
  resetLabelAction();
  window.localStorage.removeItem('dashboard_user');
  render();
}

async function importKubeconfig(file) {
  const formData = new FormData();
  formData.append('file', file);
  const response = await fetch(`${state.apiBase}/api/load_kubeconfig/import`, {
    method: 'POST',
    body: formData
  });
  const result = await response.json();
  if (!response.ok || result.success !== true) throw new Error(result.message || '导入失败');
  const imported = result.data || [];
  const preferredClusterId = imported[0]?.id || state.selectedClusterId;
  await loadClusters(preferredClusterId, state.selectedNamespace);
  state.currentView = 'cluster';
  render();
}

async function deleteCurrentKubeconfig() {
  const current = state.clusters.find((item) => item.id === state.selectedClusterId);
  if (!current?.import_id) throw new Error('当前没有可删除的配置');
  if (!window.confirm(`确认删除配置 ${current.source_file || current.name} 吗？`)) return;
  state.deletingKubeconfig = true;
  render();
  try {
    await requestJSON(`${state.apiBase}/api/load_kubeconfig/delete?import_id=${encodeURIComponent(current.import_id)}`, {
      method: 'DELETE'
    });
    await loadClusters('', '');
    state.currentView = 'cluster';
  } finally {
    state.deletingKubeconfig = false;
    render();
  }
}

async function openNodeLogin(row) {
  resetTerminal();
  state.terminalVisible = true;
  state.terminalContext = { mode: 'node', nodeName: row.name };
  state.terminalCwd = '~';
  state.terminalLines = [`已连接节点 ${row.name}`];
  render();
  await executeTerminalCommand('pwd');
}

async function startPodTerminal(row, containerName = '') {
  const containers = row.containers || [];
  if (!containerName && containers.length > 1) {
    state.terminalPickerRow = row;
    state.pickerMode = 'terminal';
    render();
    return;
  }
  resetTerminal();
  state.terminalVisible = true;
  state.terminalContext = {
    mode: 'pod',
    namespace: row.namespace,
    podName: row.name,
    container: containerName || containers[0] || ''
  };
  state.terminalCwd = '~';
  state.terminalLines = [`已连接容器 ${state.terminalContext.container || row.name}`];
  render();
  await executeTerminalCommand('pwd');
}

async function startLogs(row, containerName = '') {
  const containers = row.containers || [];
  if (!containerName && containers.length > 1) {
    state.terminalPickerRow = row;
    state.pickerMode = 'logs';
    render();
    return;
  }
  resetLogs();
  state.logVisible = true;
  state.logLoading = true;
  state.logContext = {
    namespace: row.namespace,
    podName: row.name,
    container: containerName || containers[0] || ''
  };
  render();
  try {
    const data = await requestJSON(`${state.apiBase}/api/controller/pod-logs?cluster_id=${encodeURIComponent(state.selectedClusterId)}`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(state.logContext)
    });
    state.logLines = splitLines(data.output || '');
  } catch (error) {
    state.logError = error.message || '日志获取失败';
  } finally {
    state.logLoading = false;
    render();
  }
}

async function executeTerminalCommand(command) {
  const content = command.trim();
  if (!content || !state.terminalContext || state.terminalRunning) return;
  state.terminalRunning = true;
  state.terminalError = '';
  render();
  try {
    let data;
    if (state.terminalContext.mode === 'node') {
      data = await requestJSON(`${state.apiBase}/api/node/exec?cluster_id=${encodeURIComponent(state.selectedClusterId)}`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({
          node_name: state.terminalContext.nodeName,
          command: content,
          working_dir: state.terminalCwd
        })
      });
    } else {
      data = await requestJSON(`${state.apiBase}/api/controller/pod-exec?cluster_id=${encodeURIComponent(state.selectedClusterId)}`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({
          namespace: state.terminalContext.namespace,
          pod_name: state.terminalContext.podName,
          container: state.terminalContext.container,
          command: content,
          working_dir: state.terminalCwd
        })
      });
    }
    pushTerminalOutput(content, data.output || '', data.working_dir || state.terminalCwd);
  } catch (error) {
    state.terminalError = error.message || '命令执行失败';
    state.terminalLines.push(`${state.terminalCwd || '~'} $ ${content}`);
  } finally {
    state.terminalRunning = false;
    render();
    const terminalInput = document.getElementById('terminal-command');
    if (terminalInput) terminalInput.focus();
  }
}

function bindEvents() {
  const usernameInput = document.getElementById('username-input');
  if (usernameInput) {
    usernameInput.oninput = (event) => { state.form.username = event.target.value; };
    usernameInput.onkeydown = (event) => {
      if (event.key === 'Enter') submitLogin();
    };
  }

  const passwordInput = document.getElementById('password-input');
  if (passwordInput) {
    passwordInput.oninput = (event) => { state.form.password = event.target.value; };
    passwordInput.onkeydown = (event) => {
      if (event.key === 'Enter') submitLogin();
    };
  }

  const clusterSelect = document.getElementById('cluster-select');
  if (clusterSelect) {
    clusterSelect.onchange = async (event) => {
      state.selectedClusterId = event.target.value;
      const loadVersion = bumpClusterLoadVersion();
      state.clusterOverview = null;
      state.namespaces = [];
      state.namespaceResources = emptySections();
      state.nodeResources = [];
      state.controllerRows = [];
      state.podRows = [];
      state.selectedNamespace = '';
      render();
      await loadClusterData('', state.selectedClusterId, loadVersion);
    };
  }

  const namespaceSelect = document.getElementById('namespace-select');
  if (namespaceSelect) {
    namespaceSelect.onchange = async (event) => {
      state.selectedNamespace = event.target.value;
      window.localStorage.setItem('dashboard_namespace', state.selectedNamespace);
      await refreshCurrentView(bumpClusterLoadVersion());
    };
  }

  const searchInput = document.getElementById('search-input');
  if (searchInput) {
    searchInput.oninput = (event) => {
      state.searchKeyword = event.target.value;
      render();
    };
  }

  const yamlEditor = document.getElementById('yaml-editor');
  if (yamlEditor) {
    yamlEditor.oninput = (event) => {
      state.editorYAML = event.target.value;
    };
  }

  const logKeywordInput = document.getElementById('log-keyword-input');
  if (logKeywordInput) {
    logKeywordInput.oninput = (event) => {
      state.logKeyword = event.target.value;
      render();
    };
  }

  const terminalCommand = document.getElementById('terminal-command');
  if (terminalCommand) {
    terminalCommand.focus();
    terminalCommand.onkeydown = (event) => {
      if (event.key !== 'Enter' || event.shiftKey) return;
      event.preventDefault();
      const command = terminalCommand.textContent || '';
      terminalCommand.textContent = '';
      executeTerminalCommand(command);
    };
  }

  const kubeconfigUpload = document.getElementById('kubeconfig-upload');
  if (kubeconfigUpload) {
    kubeconfigUpload.onchange = async (event) => {
      const file = event.target.files?.[0];
      event.target.value = '';
      if (!file) return;
      try {
        await importKubeconfig(file);
      } catch (error) {
        window.alert(error.message || '导入失败');
      }
    };
  }

  document.onclick = async (event) => {
    const actionEl = event.target.closest('[data-action]');
    if (!actionEl) return;
    event.preventDefault();
    const action = actionEl.dataset.action;
    const rowKeyValue = actionEl.dataset.rowKey || '';
    const row = rowKeyValue ? findRowByKey(rowKeyValue) : null;

    try {
      if (action === 'toggle-password') state.passwordVisible = !state.passwordVisible;
      if (action === 'submit-login') await submitLogin();
      if (action === 'logout') logout();
      if (action === 'refresh-dashboard') await loadClusterData(state.selectedNamespace, state.selectedClusterId, bumpClusterLoadVersion());
      if (action === 'open-create') await openCreateModal();
      if (action === 'save-create') await saveCreateResource();
      if (action === 'close-create') resetCreateModal();
      if (action === 'go-cluster') { state.currentView = 'cluster'; resetCreateModal(); resetEditor(); resetTerminal(); resetLogs(); resetLabelAction(); }
      if (action === 'set-section') openResourceSection(actionEl.dataset.section);
      if (action === 'select-type') {
        state.activeResourceSection = actionEl.dataset.section;
        state.selectedTypes[state.activeResourceSection] = actionEl.dataset.type;
        state.currentView = 'resource';
        resetCreateModal();
        resetEditor();
        resetTerminal();
        resetLogs();
        resetLabelAction();
        render();
        await refreshCurrentView();
      }
      if (action === 'open-namespace') {
        state.selectedNamespace = actionEl.dataset.namespace;
        window.localStorage.setItem('dashboard_namespace', state.selectedNamespace);
        openResourceSection('namespace');
      }
      if (action === 'toggle-editor' && row) await toggleEditor(row);
      if (action === 'save-editor') {
        await saveEditor();
        if (state.editorSuccess) flashMessage(state.editorSuccess);
      }
      if (action === 'reset-editor') { resetEditor(); }
      if (action === 'delete-resource' && row) await deleteAnyResource(row);
      if (action === 'node-login' && row) await openNodeLogin(row);
      if (action === 'pod-login' && row) await startPodTerminal(row);
      if (action === 'pod-logs' && row) await startLogs(row);
      if (action === 'close-picker') { state.terminalPickerRow = null; }
      if (action === 'pick-container' && state.terminalPickerRow) {
        const picked = actionEl.dataset.container || '';
        const pickerRow = state.terminalPickerRow;
        state.terminalPickerRow = null;
        if (state.pickerMode === 'logs') await startLogs(pickerRow, picked);
        else await startPodTerminal(pickerRow, picked);
      }
      if (action === 'close-terminal') resetTerminal();
      if (action === 'close-logs') resetLogs();
      if (action === 'submit-label' && row) {
        await submitLabelAction(row);
        if (state.labelSuccess) flashMessage(state.labelSuccess);
      }
      if (action === 'reset-label') resetLabelAction();
      if (action === 'toggle-label-delete') {
        state.labelDeleteSelections[actionEl.dataset.labelKey] = actionEl.checked;
      }
      render();
    } catch (error) {
      window.alert(error.message || '操作失败');
      render();
    }
  };

  document.onchange = (event) => {
    const actionEl = event.target.closest('[data-action]');
    if (!actionEl) return;
    const action = actionEl.dataset.action;
    const rowKeyValue = actionEl.dataset.rowKey || '';
    const row = rowKeyValue ? findRowByKey(rowKeyValue) : null;

    if (action === 'label-mode' && row) {
      state.labelActionRowKey = row.key;
      state.labelActionMode = event.target.value;
      state.labelDraftSourceKey = '';
      state.labelDraftKey = '';
      state.labelDraftValue = '';
      state.labelDeleteSelections = {};
      state.labelError = '';
      state.labelSuccess = '';
      render();
    }
    if (action === 'label-source-key' && row) {
      state.labelDraftSourceKey = event.target.value;
      const sourceValue = row.labelMap?.[state.labelDraftSourceKey] || '';
      state.labelDraftKey = state.labelDraftSourceKey;
      state.labelDraftValue = sourceValue;
      render();
    }
  };

  document.oninput = (event) => {
    const actionEl = event.target.closest('[data-action]');
    if (!actionEl) return;
    const action = actionEl.dataset.action;
    if (action === 'label-draft-key') state.labelDraftKey = event.target.value;
    if (action === 'label-draft-value') state.labelDraftValue = event.target.value;
  };
}

async function initApp() {
  const savedUser = window.localStorage.getItem('dashboard_user');
  if (savedUser) {
    try {
      state.userProfile = JSON.parse(savedUser);
      state.isLoggedIn = true;
    } catch (_) {
      window.localStorage.removeItem('dashboard_user');
    }
  }
  state.selectedClusterId = window.localStorage.getItem('dashboard_cluster') || '';
  state.selectedNamespace = window.localStorage.getItem('dashboard_namespace') || '';
  render();
  if (!state.isLoggedIn) return;
  try {
    await loadClusters(state.selectedClusterId, state.selectedNamespace);
  } catch (error) {
    state.errorMessage = error.message || '初始化失败';
    logout();
  }
}

initApp();
