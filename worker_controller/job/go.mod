module developer-kubernetes/worker_controller/job

go 1.22.0

require developer-kubernetes/load_kubeconfig v0.0.0

require gopkg.in/yaml.v3 v3.0.1 // indirect

replace developer-kubernetes/load_kubeconfig => ../../load_kubeconfig
