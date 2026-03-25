package loadkubeconfig

import (
	"context"
	"crypto/sha1"
	"database/sql"
	"encoding/hex"
	"fmt"
	"io"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

type Service struct {
	db              *sql.DB
	maxUploadSizeMB int64
}

type ImportedCluster struct {
	ID             string    `json:"id"`
	ImportID       int64     `json:"import_id"`
	Name           string    `json:"name"`
	ContextName    string    `json:"context_name"`
	ClusterName    string    `json:"cluster_name"`
	SourceFile     string    `json:"source_file"`
	ImportedAt     time.Time `json:"imported_at"`
	KubeconfigBody string    `json:"-"`
}

type kubeconfigRecord struct {
	ID         int64
	SourceFile string
	Content    string
	CreatedAt  time.Time
}

type kubeconfigFile struct {
	CurrentContext string             `yaml:"current-context"`
	Contexts       []namedKubeContext `yaml:"contexts"`
	Clusters       []namedKubeCluster `yaml:"clusters"`
}

type namedKubeContext struct {
	Name    string      `yaml:"name"`
	Context kubeContext `yaml:"context"`
}

type kubeContext struct {
	Cluster string `yaml:"cluster"`
}

type namedKubeCluster struct {
	Name string `yaml:"name"`
}

func NewService(db *sql.DB, maxUploadSizeMB int64) *Service {
	return &Service{
		db:              db,
		maxUploadSizeMB: maxUploadSizeMB,
	}
}

func (s *Service) EnsureSchema() error {
	const createTableSQL = `
CREATE TABLE IF NOT EXISTS imported_kubeconfig (
  id BIGINT PRIMARY KEY AUTO_INCREMENT,
  source_file VARCHAR(255) NOT NULL,
  content LONGTEXT NOT NULL,
  created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci`

	_, err := s.db.Exec(createTableSQL)
	if err != nil {
		return fmt.Errorf("create imported_kubeconfig table: %w", err)
	}
	return nil
}

func (s *Service) Import(ctx context.Context, originalName string, reader io.Reader) ([]ImportedCluster, error) {
	content, err := io.ReadAll(io.LimitReader(reader, s.maxUploadSizeMB*1024*1024+1))
	if err != nil {
		return nil, fmt.Errorf("read file: %w", err)
	}
	if int64(len(content)) > s.maxUploadSizeMB*1024*1024 {
		return nil, fmt.Errorf("file too large")
	}

	cfg, err := parseKubeconfig(content)
	if err != nil {
		return nil, err
	}

	sourceFile := fmt.Sprintf("%s_%s", time.Now().Format("20060102_150405"), sanitizeFilename(originalName))
	result, err := s.db.ExecContext(ctx, "INSERT INTO imported_kubeconfig (source_file, content) VALUES (?, ?)", sourceFile, string(content))
	if err != nil {
		return nil, fmt.Errorf("save kubeconfig: %w", err)
	}

	importID, err := result.LastInsertId()
	if err != nil {
		return nil, fmt.Errorf("read import id: %w", err)
	}

	record := kubeconfigRecord{
		ID:         importID,
		SourceFile: sourceFile,
		Content:    string(content),
		CreatedAt:  time.Now(),
	}
	return buildImportedClusters(record, cfg), nil
}

func (s *Service) ListClusters(ctx context.Context) ([]ImportedCluster, error) {
	rows, err := s.db.QueryContext(ctx, "SELECT id, source_file, content, created_at FROM imported_kubeconfig ORDER BY created_at DESC, id DESC")
	if err != nil {
		return nil, fmt.Errorf("query kubeconfig records: %w", err)
	}
	defer rows.Close()

	imported := make([]ImportedCluster, 0)
	for rows.Next() {
		var record kubeconfigRecord
		if err := rows.Scan(&record.ID, &record.SourceFile, &record.Content, &record.CreatedAt); err != nil {
			return nil, fmt.Errorf("scan kubeconfig record: %w", err)
		}
		cfg, err := parseKubeconfig([]byte(record.Content))
		if err != nil {
			continue
		}
		imported = append(imported, buildImportedClusters(record, cfg)...)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate kubeconfig records: %w", err)
	}
	return imported, nil
}

func (s *Service) ResolveCluster(ctx context.Context, id string) (*ImportedCluster, error) {
	clusters, err := s.ListClusters(ctx)
	if err != nil {
		return nil, err
	}

	for _, item := range clusters {
		if item.ID == id {
			return &item, nil
		}
	}
	return nil, fmt.Errorf("cluster not found")
}

func (s *Service) DeleteImport(ctx context.Context, importID int64) error {
	result, err := s.db.ExecContext(ctx, "DELETE FROM imported_kubeconfig WHERE id = ?", importID)
	if err != nil {
		return fmt.Errorf("delete kubeconfig: %w", err)
	}
	affected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("delete kubeconfig: %w", err)
	}
	if affected == 0 {
		return fmt.Errorf("kubeconfig not found")
	}
	return nil
}

func parseKubeconfig(content []byte) (*kubeconfigFile, error) {
	var cfg kubeconfigFile
	if err := yaml.Unmarshal(content, &cfg); err != nil {
		return nil, fmt.Errorf("parse kubeconfig: %w", err)
	}
	if len(cfg.Contexts) == 0 {
		return nil, fmt.Errorf("no contexts found in kubeconfig")
	}
	return &cfg, nil
}

func buildImportedClusters(record kubeconfigRecord, cfg *kubeconfigFile) []ImportedCluster {
	clusterNames := make(map[string]string, len(cfg.Clusters))
	for _, item := range cfg.Clusters {
		clusterNames[item.Name] = item.Name
	}

	items := make([]ImportedCluster, 0, len(cfg.Contexts))
	for _, contextItem := range cfg.Contexts {
		clusterName := contextItem.Context.Cluster
		if clusterNames[clusterName] == "" {
			clusterName = contextItem.Name
		}
		if clusterName == "" {
			clusterName = contextItem.Name
		}

		hashSource := fmt.Sprintf("%d::%s", record.ID, contextItem.Name)
		sum := sha1.Sum([]byte(hashSource))
		items = append(items, ImportedCluster{
			ID:             hex.EncodeToString(sum[:8]),
			ImportID:       record.ID,
			Name:           clusterName,
			ContextName:    contextItem.Name,
			ClusterName:    clusterName,
			SourceFile:     record.SourceFile,
			ImportedAt:     record.CreatedAt,
			KubeconfigBody: record.Content,
		})
	}

	sort.Slice(items, func(i, j int) bool {
		return items[i].ContextName < items[j].ContextName
	})
	return items
}

func sanitizeFilename(name string) string {
	name = filepath.Base(name)
	replacer := strings.NewReplacer(" ", "_", "/", "_", "\\", "_", ":", "_")
	name = replacer.Replace(name)
	if name == "" {
		return "kubeconfig.yaml"
	}
	return name
}
