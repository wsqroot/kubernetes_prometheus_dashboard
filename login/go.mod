module login

go 1.22.0

require (
	developer-kubernetes/load_kubeconfig v0.0.0
	developer-kubernetes/namespace v0.0.0
	developer-kubernetes/worker_controller/daemonset v0.0.0
	developer-kubernetes/worker_controller/deployment v0.0.0
	developer-kubernetes/worker_controller/job v0.0.0
	developer-kubernetes/worker_controller/statefulset v0.0.0
	github.com/go-sql-driver/mysql v1.8.1
	gopkg.in/yaml.v3 v3.0.1
)

require filippo.io/edwards25519 v1.1.0 // indirect

replace developer-kubernetes/load_kubeconfig => ../load_kubeconfig

replace developer-kubernetes/namespace => ../namespace

replace developer-kubernetes/worker_controller/daemonset => ../worker_controller/daemonset

replace developer-kubernetes/worker_controller/deployment => ../worker_controller/deployment

replace developer-kubernetes/worker_controller/job => ../worker_controller/job

replace developer-kubernetes/worker_controller/statefulset => ../worker_controller/statefulset
