package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	loadkubeconfig "developer-kubernetes/load_kubeconfig"
	namespaceapi "developer-kubernetes/namespace"
	daemonsetapi "developer-kubernetes/worker_controller/daemonset"
	deploymentapi "developer-kubernetes/worker_controller/deployment"
	jobapi "developer-kubernetes/worker_controller/job"
	statefulsetapi "developer-kubernetes/worker_controller/statefulset"
	"login/internal/config"
	"login/internal/database"
	"login/internal/handler"
	"login/internal/repository"
	"login/internal/service"
)

func main() {
	cfgPath := os.Getenv("LOGIN_CONFIG")
	if cfgPath == "" {
		cfgPath = "configs/config.yaml"
	}

	cfg, err := config.Load(cfgPath)
	if err != nil {
		log.Fatalf("load config: %v", err)
	}

	db, err := database.NewMySQL(cfg.MySQL)
	if err != nil {
		log.Fatalf("connect mysql: %v", err)
	}
	defer db.Close()

	if err := database.EnsureDatabase(cfg.KubeconfigMySQL.AdminDSN, cfg.KubeconfigMySQL.Database); err != nil {
		log.Fatalf("ensure kubeconfig database: %v", err)
	}

	kubeconfigDB, err := database.NewMySQL(config.MySQLConfig{
		DSN:             cfg.KubeconfigMySQL.DSN,
		MaxOpenConns:    cfg.KubeconfigMySQL.MaxOpenConns,
		MaxIdleConns:    cfg.KubeconfigMySQL.MaxIdleConns,
		ConnMaxLifetime: cfg.KubeconfigMySQL.ConnMaxLifetime,
	})
	if err != nil {
		log.Fatalf("connect kubeconfig mysql: %v", err)
	}
	defer kubeconfigDB.Close()

	repo := repository.NewUserRepository(db, cfg.Auth.UserLookupSQL)
	authService := service.NewAuthService(repo, cfg.Auth.PasswordMode)
	authHandler := handler.NewAuthHandler(authService)
	kubeconfigService := loadkubeconfig.NewService(kubeconfigDB, cfg.Kubeconfig.MaxUploadSizeMB)
	if err := kubeconfigService.EnsureSchema(); err != nil {
		log.Fatalf("ensure kubeconfig schema: %v", err)
	}
	kubeconfigHandler := loadkubeconfig.NewHandler(kubeconfigService)
	namespaceHandler := namespaceapi.NewHandler(namespaceapi.NewService(kubeconfigService))
	daemonSetHandler := daemonsetapi.NewHandler(daemonsetapi.NewService(kubeconfigService))
	deploymentHandler := deploymentapi.NewHandler(deploymentapi.NewService(kubeconfigService))
	jobHandler := jobapi.NewHandler(jobapi.NewService(kubeconfigService))
	statefulSetHandler := statefulsetapi.NewHandler(statefulsetapi.NewService(kubeconfigService))

	mux := http.NewServeMux()
	mux.HandleFunc("/healthz", authHandler.Healthz)
	mux.HandleFunc("/api/login", authHandler.Login)
	mux.HandleFunc("/api/load_kubeconfig/import", kubeconfigHandler.Import)
	mux.HandleFunc("/api/load_kubeconfig/clusters", kubeconfigHandler.ListClusters)
	mux.HandleFunc("/api/load_kubeconfig/delete", kubeconfigHandler.Delete)
	mux.HandleFunc("/api/namespace/overview", namespaceHandler.Overview)
	mux.HandleFunc("/api/namespace/list", namespaceHandler.ListNamespaces)
	mux.HandleFunc("/api/namespace/resources", namespaceHandler.Resources)
	mux.HandleFunc("/api/namespace/nodes", namespaceHandler.Nodes)
	mux.HandleFunc("/api/resource/yaml", namespaceHandler.ResourceYAML)
	mux.HandleFunc("/api/resource/delete", namespaceHandler.ResourceDelete)
	mux.HandleFunc("/api/controller/pods", namespaceHandler.Pods)
	mux.HandleFunc("/api/controller/pod-exec", namespaceHandler.PodExec)
	mux.HandleFunc("/api/controller/pod-logs", namespaceHandler.PodLogs)
	mux.HandleFunc("/api/controller/pod-labels", namespaceHandler.PodLabels)
	mux.HandleFunc("/api/node/labels", namespaceHandler.NodeLabels)
	mux.HandleFunc("/api/node/exec", namespaceHandler.NodeExec)
	mux.HandleFunc("/api/controller/pod-delete", namespaceHandler.PodDelete)
	mux.HandleFunc("/api/controller/daemonsets", daemonSetHandler.List)
	mux.HandleFunc("/api/controller/deployments", deploymentHandler.List)
	mux.HandleFunc("/api/controller/jobs", jobHandler.List)
	mux.HandleFunc("/api/controller/statefulsets", statefulSetHandler.List)

	server := &http.Server{
		Addr:         cfg.Server.Addr,
		Handler:      handler.WithCORS(mux, cfg.CORS.AllowedOrigins),
		ReadTimeout:  cfg.Server.ReadTimeout,
		WriteTimeout: cfg.Server.WriteTimeout,
		IdleTimeout:  cfg.Server.IdleTimeout,
	}

	go func() {
		log.Printf("login server listening on %s", cfg.Server.Addr)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("listen server: %v", err)
		}
	}()

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)
	<-stop

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := server.Shutdown(ctx); err != nil {
		log.Printf("shutdown server: %v", err)
	}
}
