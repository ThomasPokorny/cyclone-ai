package config

import (
	"context"
	"fmt"
	"log"
)

type Installation struct {
	ID             int64  `json:"id"`
	InstallationID int64  `json:"installation_id"`
	CreatedAt      string `json:"created_at"`
}

type ConfigProvider interface {
	GetRepositoryConfig(ctx context.Context, orgName, repoName string, installationID int64) (*RepositoryConfig, error)
}

type DatabaseClient interface {
	GetInstallationByInstallationID(ctx context.Context, installationID int64) (*Installation, error)
	GetOrganizationByInstallationAndName(ctx context.Context, installationDBID int64, orgName string) ([]Organization, error)
	GetRepositoryByOrganizationAndName(ctx context.Context, organizationID int64, repoName string) (*Repository, error)
}

type Organization struct {
	ID   int64  `json:"id"`
	Name string `json:"name"`
}

type Repository struct {
	ID           int64  `json:"id"`
	Name         string `json:"name"`
	Precision    string `json:"precision"`
	CustomPrompt string `json:"custom_prompt"`
}

type SupabaseProvider struct {
	client DatabaseClient
}

func NewSupabaseProvider(cfg *Config) (ConfigProvider, error) {
	client := NewSupabaseClient(cfg.SupabaseURL, cfg.SupabaseAPIKey)
	return &SupabaseProvider{
		client: client,
	}, nil
}

func (sp *SupabaseProvider) GetRepositoryConfig(ctx context.Context, orgName, repoName string, installationID int64) (*RepositoryConfig, error) {
	// Step 1: Get installation from database
	installation, err := sp.client.GetInstallationByInstallationID(ctx, installationID)
	if err != nil {
		return nil, fmt.Errorf("installation not found for installation_id %d: %w", installationID, err)
	}

	log.Printf("Found installation %d.",
		installation.InstallationID)

	// Step 2: Get organization from database
	organizations, err := sp.client.GetOrganizationByInstallationAndName(ctx, installation.ID, orgName)
	if err != nil {
		return nil, fmt.Errorf("organization '%s' not found for installation %d: %w", orgName, installationID, err)
	}

	log.Printf("Found organization '%s' for installation %d", organizations[0].Name, installationID)

	// Step 3: Get repository configuration from database
	repository, err := sp.client.GetRepositoryByOrganizationAndName(ctx, organizations[0].ID, repoName)
	if err != nil {
		return nil, fmt.Errorf("repository '%s' not found in organization '%s': %w", repoName, orgName, err)
	}

	log.Printf("Found repository config: %s (precision: %s)", repository.Name, repository.Precision)

	// Step 4: Return actual repository configuration from database
	return &RepositoryConfig{
		Name:         repository.Name,
		Precision:    ReviewPrecision(repository.Precision),
		CustomPrompt: repository.CustomPrompt,
	}, nil
}
