package config

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
)

// SupabaseClient implements DatabaseClient for Supabase
type SupabaseClient struct {
	url    string
	apiKey string
	client *http.Client
}

// NewSupabaseClient creates a new Supabase client
func NewSupabaseClient(url, apiKey string) *SupabaseClient {
	return &SupabaseClient{
		url:    url,
		apiKey: apiKey,
		client: &http.Client{},
	}
}

// GetInstallationByInstallationID retrieves installation by GitHub installation ID
func (s *SupabaseClient) GetInstallationByInstallationID(ctx context.Context, installationID int64) (*Installation, error) {
	query := fmt.Sprintf("installation_id=eq.%d", installationID)

	req, err := s.buildRequest("GET", "/rest/v1/installation", query, nil)
	if err != nil {
		return nil, err
	}

	resp, err := s.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("installation not found: %d", installationID)
	}

	var installations []Installation
	if err := json.NewDecoder(resp.Body).Decode(&installations); err != nil {
		return nil, err
	}

	if len(installations) == 0 {
		return nil, fmt.Errorf("installation not found: %d", installationID)
	}

	return &installations[0], nil
}

// GetOrganizationByInstallationAndName retrieves organization by installation and name
func (s *SupabaseClient) GetOrganizationByInstallationAndName(ctx context.Context, installationDBID int64, orgName string) ([]Organization, error) {
	query := fmt.Sprintf("installation_id=eq.%d", installationDBID)

	req, err := s.buildRequest("GET", "/rest/v1/organization", query, nil)
	if err != nil {
		return nil, err
	}

	resp, err := s.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("organization not found: %s", orgName)
	}

	var organizations []Organization
	if err := json.NewDecoder(resp.Body).Decode(&organizations); err != nil {
		return nil, err
	}

	if len(organizations) == 0 {
		return nil, fmt.Errorf("organization not found: %s", orgName)
	}

	return organizations, nil
}

// GetRepositoryByOrganizationAndName retrieves repository by organization and name
func (s *SupabaseClient) GetRepositoryByOrganizationAndName(ctx context.Context, organizationID int64, repoName string) (*Repository, error) {
	query := fmt.Sprintf("organization_id=eq.%d&name=eq.%s", organizationID, repoName)

	req, err := s.buildRequest("GET", "/rest/v1/repository", query, nil)
	if err != nil {
		return nil, err
	}

	resp, err := s.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("repository not found: %s", repoName)
	}

	var repositories []Repository
	if err := json.NewDecoder(resp.Body).Decode(&repositories); err != nil {
		return nil, err
	}

	if len(repositories) == 0 {
		return nil, fmt.Errorf("repository not found: %s", repoName)
	}

	return &repositories[0], nil
}

// buildRequest helper method for Supabase API requests
func (s *SupabaseClient) buildRequest(method, path, query string, body interface{}) (*http.Request, error) {
	url := s.url + path
	if query != "" {
		url += "?" + query
	}

	var reqBody []byte
	var err error
	if body != nil {
		reqBody, err = json.Marshal(body)
		if err != nil {
			return nil, err
		}
	}

	req, err := http.NewRequest(method, url, strings.NewReader(string(reqBody)))
	if err != nil {
		return nil, err
	}

	req.Header.Set("apikey", s.apiKey)
	req.Header.Set("Authorization", "Bearer "+s.apiKey)
	req.Header.Set("Content-Type", "application/json")

	return req, nil
}
