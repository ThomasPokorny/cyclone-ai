package config

// Config holds our application configuration
type Config struct {
	GitHubToken    string
	Port           string
	WebhookSecret  string
	AnthropicToken string
}

// ReviewPrecision defines how strict the review should be
type ReviewPrecision string

const (
	PrecisionMinor  ReviewPrecision = "minor"
	PrecisionMedium ReviewPrecision = "medium"
	PrecisionStrict ReviewPrecision = "strict"
)

// RepositoryConfig holds configuration for a specific repository
type RepositoryConfig struct {
	Name         string          `json:"name"`
	Precision    ReviewPrecision `json:"precision"`
	CustomPrompt string          `json:"custom_prompt"`
}

// OrganizationConfig holds configuration for an entire organization
type OrganizationConfig struct {
	Name         string             `json:"name"`
	Repositories []RepositoryConfig `json:"repositories"`
}
type ReviewConfig struct {
	Organizations []OrganizationConfig `json:"organizations"`
}

// Constants for PR size limits
const (
	// Hard limits for PR review
	MAX_FILES_FOR_REVIEW     = 25   // Skip review if more files changed
	MAX_ADDITIONS_FOR_REVIEW = 800  // Skip review if more lines added
	MAX_TOTAL_CHANGES        = 1200 // Skip review if total changes exceed this

	// Warning thresholds (still review, but warn)
	WARN_FILES_THRESHOLD     = 20
	WARN_ADDITIONS_THRESHOLD = 400
)
