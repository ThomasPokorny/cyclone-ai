package config

import (
	"bufio"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"strings"
)

// Load loads both application and review configurations
func Load() (*Config, *ReviewConfig, error) {
	// Load .env file if it exists
	loadEnvFile(".env")

	// Load application configuration from environment variables
	cfg := &Config{
		GitHubToken:    os.Getenv("GITHUB_TOKEN"),
		Port:           getEnv("PORT", "8080"),
		WebhookSecret:  os.Getenv("WEBHOOK_SECRET"),
		AnthropicToken: os.Getenv("ANTHROPIC_API_KEY"),
	}

	// Validate required configuration
	if cfg.GitHubToken == "" {
		return nil, nil, fmt.Errorf("GITHUB_TOKEN environment variable is required")
	}

	if cfg.AnthropicToken == "" {
		return nil, nil, fmt.Errorf("ANTHROPIC_API_KEY environment variable is required")
	}

	// Load review configuration from JSON file
	reviewCfg, err := loadReviewConfig("review-config.json")
	if err != nil {
		return nil, nil, fmt.Errorf("failed to load review configuration: %w", err)
	}

	log.Printf("Loaded configuration for %d organizations", len(reviewCfg.Organizations))

	return cfg, reviewCfg, nil
}

// GetRepositoryConfig finds the configuration for a specific repository
// Returns nil if repository should be ignored (not in config)
func (rc *ReviewConfig) GetRepositoryConfig(owner, repoName string) *RepositoryConfig {
	// Look through all organizations
	for _, org := range rc.Organizations {
		// Match by organization name
		if org.Name == owner {
			// Look for specific repository config
			for _, repo := range org.Repositories {
				if repo.Name == repoName {
					return &repo
				}
			}

			// Look for a wildcard/default repository config
			for _, repo := range org.Repositories {
				if repo.Name == "*" || repo.Name == "default" {
					return &repo
				}
			}
		}
	}

	// Return nil if repository not found - this means ignore it
	return nil
}

// GetPrecisionGuidelines returns review guidelines based on precision level
func GetPrecisionGuidelines(precision ReviewPrecision) string {
	switch precision {
	case PrecisionMinor:
		return `**Review Focus (Minor Precision):**
- Focus primarily on critical bugs and security issues
- Skip most style and formatting comments
- Be lenient with minor code quality issues
- Emphasize 🚫 **blocking** and ⚠️ **issue** categories`

	case PrecisionStrict:
		return `**Review Focus (Strict Precision):**
- Review all aspects including style, performance, and maintainability
- Be thorough with naming conventions and code organization
- Suggest improvements for readability and best practices
- Use all categories including 🧰 **nit** and 💡 **suggestion**
- Consider long-term maintainability and team standards`

	default: // PrecisionMedium
		return `**Review Focus (Medium Precision):**
- Balance between thoroughness and practicality
- Focus on significant issues while noting important style concerns
- Emphasize security, bugs, and maintainability
- Use ⚠️ **issue**, 💡 **suggestion**, and 🧰 **nit** categories appropriately`
	}
}

// loadReviewConfig loads review configuration from a JSON file
func loadReviewConfig(filename string) (*ReviewConfig, error) {
	file, err := os.Open(filename)
	if err != nil {
		return nil, fmt.Errorf("failed to open config file %s: %w", filename, err)
	}
	defer file.Close()

	var config ReviewConfig
	decoder := json.NewDecoder(file)
	if err := decoder.Decode(&config); err != nil {
		return nil, fmt.Errorf("failed to parse config file %s: %w", filename, err)
	}

	return &config, nil
}

// loadEnvFile loads environment variables from a file
func loadEnvFile(filename string) {
	file, err := os.Open(filename)
	if err != nil {
		// .env file is optional, so just return if it doesn't exist
		return
	}
	defer file.Close()

	var lines []string
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		lines = append(lines, scanner.Text())
	}

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		parts := strings.SplitN(line, "=", 2)
		if len(parts) == 2 {
			key := strings.TrimSpace(parts[0])
			value := strings.TrimSpace(parts[1])
			// Remove quotes if present
			if len(value) >= 2 && ((value[0] == '"' && value[len(value)-1] == '"') ||
				(value[0] == '\'' && value[len(value)-1] == '\'')) {
				value = value[1 : len(value)-1]
			}
			os.Setenv(key, value)
		}
	}
}

// getEnv gets an environment variable with a default fallback
func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
