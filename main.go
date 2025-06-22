// main.go - Cyclone AI Code Review Tool
package main

import (
	"bufio"
	"bytes" // Add this
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv" // Add this
	"strings"
	"time" // Add this

	"github.com/google/go-github/v57/github"
	"golang.org/x/oauth2"
)

// Config holds our application configuration
type Config struct {
	GitHubToken    string
	Port           string
	WebhookSecret  string
	AnthropicToken string
}

// CycloneBot handles GitHub operations and AI integration
type CycloneBot struct {
	client       *github.Client
	config       *Config
	reviewConfig *ReviewConfig
}

// NewCycloneBot creates a new Cyclone bot instance
func NewCycloneBot(config *Config, reviewConfig *ReviewConfig) *CycloneBot {
	ctx := context.Background()
	ts := oauth2.StaticTokenSource(
		&oauth2.Token{AccessToken: config.GitHubToken},
	)
	tc := oauth2.NewClient(ctx, ts)

	return &CycloneBot{
		client:       github.NewClient(tc),
		config:       config,
		reviewConfig: reviewConfig,
	}
}

// WebhookPayload represents the GitHub webhook payload
type WebhookPayload struct {
	Action      string              `json:"action"`
	PullRequest *github.PullRequest `json:"pull_request"`
	Repository  *github.Repository  `json:"repository"`
}

// ReviewComment represents a comment on a specific line
type ReviewComment struct {
	Path string
	Line int
	Body string
	Side string
}

// ReviewResult holds the overall review and line-specific comments
type ReviewResult struct {
	Summary  string
	Comments []ReviewComment
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

// ReviewConfig holds the complete review configuration
type ReviewConfig struct {
	Organizations []OrganizationConfig `json:"organizations"`
}

// handleWebhook processes incoming GitHub webhooks
func (bot *CycloneBot) handleWebhook(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Parse the webhook payload
	var payload WebhookPayload
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		log.Printf("Error decoding webhook payload: %v", err)
		http.Error(w, "Bad request", http.StatusBadRequest)
		return
	}

	// Only process specific actions that warrant a review
	if !bot.shouldTriggerReview(payload.Action, payload.PullRequest) {
		log.Printf("Ignoring action: %s for PR #%d", payload.Action, payload.PullRequest.GetNumber())
		w.WriteHeader(http.StatusOK)
		return
	}

	log.Printf("Processing PR #%d: %s", payload.PullRequest.GetNumber(), payload.Action)

	// Process the PR in a goroutine to avoid blocking the webhook
	go bot.processPullRequest(payload.Repository, payload.PullRequest)

	w.WriteHeader(http.StatusOK)
}

func (bot *CycloneBot) shouldTriggerReview(action string, pr *github.PullRequest) bool {
	// Skip draft PRs entirely
	if pr.GetDraft() {
		return false
	}

	switch action {
	case "opened":
		// Review when PR is first opened (and not draft)
		return true

	case "ready_for_review":
		// Review when PR moves from draft to ready
		return true

	case "synchronize":
		// Only review new commits if PR is not draft and we haven't reviewed recently
		// You might want to add additional logic here to avoid reviewing every commit
		return false // For now, skip synchronize events

	default:
		// Skip all other actions (closed, edited, etc.)
		return false
	}
}

// processPullRequest handles the main logic for reviewing a PR
func (bot *CycloneBot) processPullRequest(repo *github.Repository, pr *github.PullRequest) {
	ctx := context.Background()

	owner := repo.GetOwner().GetLogin()
	repoName := repo.GetName()
	prNumber := pr.GetNumber()

	log.Printf("Processing PR #%d in %s/%s", prNumber, owner, repoName)

	// Get repository-specific configuration
	repoConfig := bot.getRepositoryConfig(owner, repoName)
	if repoConfig == nil {
		log.Printf("Repository %s/%s not found in configuration - skipping review", owner, repoName)
		return
	}

	// Check PR size before proceeding
	sizeCheck := bot.checkPRSize(pr)
	if !sizeCheck.ShouldReview {
		log.Printf("PR #%d is too large - posting skip message instead of review", prNumber)

		// Post skip message as a regular comment
		comment := &github.IssueComment{
			Body: github.String(sizeCheck.SkipMessage),
		}

		if _, _, err := bot.client.Issues.CreateComment(ctx, owner, repoName, prNumber, comment); err != nil {
			log.Printf("Error posting skip message: %v", err)
		}
		return
	}

	log.Printf("Using precision: %s for repository: %s", repoConfig.Precision, repoName)

	// Get the PR diff
	diff, err := bot.getPRDiff(ctx, owner, repoName, prNumber)
	if err != nil {
		log.Printf("Error getting PR diff: %v", err)
		return
	}

	// Get AI review with repository-specific configuration
	review := bot.getAIReviewWithConfig(diff, pr.GetTitle(), pr.GetBody(), repoConfig)

	// Prepend size warning if applicable
	if sizeCheck.WarningMessage != "" {
		review.Summary = sizeCheck.WarningMessage + review.Summary
	}

	// Post the review with line-specific comments
	if err := bot.postPRReview(ctx, owner, repoName, prNumber, review); err != nil {
		log.Printf("Error posting PR review: %v", err)
		return
	}

	log.Printf("Successfully posted AI review for PR #%d", prNumber)
}

// getPRDiff fetches the diff for a pull request
func (bot *CycloneBot) getPRDiff(ctx context.Context, owner, repo string, prNumber int) (string, error) {
	// Get the PR files
	files, _, err := bot.client.PullRequests.ListFiles(ctx, owner, repo, prNumber, nil)
	if err != nil {
		return "", fmt.Errorf("failed to get PR files: %w", err)
	}

	var diffBuilder strings.Builder
	for _, file := range files {
		// Skip binary files and very large files
		if file.GetPatch() == "" || file.GetChanges() > 500 {
			continue
		}

		// Additional check for binary files by file extension
		filename := file.GetFilename()
		if isBinaryFile(filename) {
			continue
		}

		diffBuilder.WriteString(fmt.Sprintf("=== %s ===\n", filename))
		diffBuilder.WriteString(file.GetPatch())
		diffBuilder.WriteString("\n\n")
	}

	return diffBuilder.String(), nil
}

// isBinaryFile checks if a file is likely binary based on its extension
func isBinaryFile(filename string) bool {
	binaryExtensions := []string{
		".png", ".jpg", ".jpeg", ".gif", ".ico", ".svg",
		".pdf", ".zip", ".tar", ".gz", ".bz2", ".xz",
		".exe", ".dll", ".so", ".dylib",
		".woff", ".woff2", ".ttf", ".eot",
		".mp3", ".mp4", ".avi", ".mov",
		".class", ".jar", ".war",
	}

	filename = strings.ToLower(filename)
	for _, ext := range binaryExtensions {
		if strings.HasSuffix(filename, ext) {
			return true
		}
	}
	return false
}

// ClaudeResponse represents the response from Claude API
type ClaudeResponse struct {
	Content []struct {
		Text string `json:"text"`
	} `json:"content"`
}

// ClaudeRequest represents a request to Claude API
type ClaudeRequest struct {
	Model     string `json:"model"`
	MaxTokens int    `json:"max_tokens"`
	Messages  []struct {
		Role    string `json:"role"`
		Content string `json:"content"`
	} `json:"messages"`
}

const (
	// Hard limits for PR review
	MAX_FILES_FOR_REVIEW     = 25   // Skip review if more files changed
	MAX_ADDITIONS_FOR_REVIEW = 800  // Skip review if more lines added
	MAX_TOTAL_CHANGES        = 1200 // Skip review if total changes exceed this

	// Warning thresholds (still review, but warn)
	WARN_FILES_THRESHOLD     = 20
	WARN_ADDITIONS_THRESHOLD = 400
)

type PRSizeCheck struct {
	ShouldReview   bool
	WarningMessage string
	SkipMessage    string
}

// getAIReview generates an AI review of the code (placeholder implementation)
func (bot *CycloneBot) getAIReviewWithConfig(diff, title, body string, repoConfig *RepositoryConfig) ReviewResult {
	claudeReview := bot.callClaudeAPIWithConfig(diff, title, body, repoConfig)
	return bot.parseClaudeResponse(claudeReview, diff)
}

func (bot *CycloneBot) callClaudeAPIWithConfig(diff, title, body string, repoConfig *RepositoryConfig) string {
	prompt := fmt.Sprintf(`You are Cyclone, an AI code review assistant. Please review this GitHub pull request and provide constructive feedback.

**PR Title:** %s

**PR Description:** %s

**Review Precision**: %s
 
**Code Changes:**
%s

Please provide:
1. A brief overall summary of the changes
2. Specific feedback categorized by type and priority
3. End with a short, lighthearted poem (2-4 lines) based on the changes made

**Review Guidelines:**
- Be constructive and actionable - explain the "why" behind suggestions
- Include code examples when suggesting alternatives
- Use collaborative language ("we could" vs "you should")
- Focus on logic correctness, security, maintainability, and team conventions
- Acknowledge good patterns when present

**Comment Categories - Use these prefixes:**
- 🧰 **nit**: Minor style/preference issues, non-blocking
- 💡 **suggestion**: Improvements that would be nice but aren't required
- ⚠️ **issue**: Problems that should be addressed before merging
- 🚫 **blocking**: Critical issues that must be fixed
- ❓ **question**: Seeking clarification about intent or approach

**Focus Areas - Use these prefixes when relevant:**
- 🎨 **style**: Formatting, naming conventions
- ⚡ **perf**: Performance concerns
- 🔒 **security**: Security-related issues
- 📚 **docs**: Documentation needs
- 🧪 **test**: Testing coverage or quality
- 🔧 **refactor**: Code organization improvements

**Response Structure:**
Please structure your response EXACTLY as follows:

SUMMARY: $$
**A warm, engaging summary** with emojis and thoughtful analysis (not just bullet points) including:**
- Brief overall analysis of what this PR accomplishes
- Key changes made 
- Impact assessment (what this means for the codebase)
- Good patterns you noticed (acknowledge positive aspects)
- Any overarching concerns or recommendations
- Use emojis carefully to make it visually appealing (🚀 ✨ 🎯 📈 🔧 etc.). 
$$

POEM: $$
A short, lighthearted poem (2-4 lines) inspired by the changes made formatted in italic.
Make it fun and relevant to the code changes.
$$

For any line-specific comments, use this EXACT format:
PR_COMMENT:filename:line_number: [emoji] **[category]**: $$ 
your comment here (can be multiple lines)
include code examples
end your comment
$$
Examples:
PR_COMMENT:main.go:45: 🔍 **nit**: Consider using a more descriptive variable name like 'userCount' instead of 'cnt'
PR_COMMENT:utils.js:123: ⚠️ **issue**: This function needs error handling for the API call
PR_COMMENT:api/handler.py:67: 🚫 **blocking**: 🔒 **security**: Potential SQL injection vulnerability - use parameterized queries


**IMPORTANT Rules:**
- Use SINGLE line numbers only, NOT ranges like "75-82"
- Always include the colon after **[category]**:
- Always use the $$ delimiters for all sections
- Keep general analysis in SUMMARY, use PR_COMMENT only for specific line feedback
- Include code examples in PR_COMMENT when suggesting alternatives

%s

Be constructive, helpful, and focus on actionable feedback.`, title, body, bot.getPrecisionGuidelines(repoConfig.Precision), diff, repoConfig.CustomPrompt)

	reqBody := ClaudeRequest{
		Model:     "claude-3-5-sonnet-20241022", //make models configureable: claude-sonnet-4-20250514, claude-3-5-sonnet-20241022, claude-3-haiku-20240307
		MaxTokens: 8000,
		Messages: []struct {
			Role    string `json:"role"`
			Content string `json:"content"`
		}{
			{
				Role:    "user",
				Content: prompt,
			},
		},
	}

	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		log.Printf("Error marshaling request: %v", err)
		return "Error generating AI review"
	}

	req, err := http.NewRequest("POST", "https://api.anthropic.com/v1/messages", bytes.NewBuffer(jsonData))
	if err != nil {
		log.Printf("Error creating request: %v", err)
		return "Error generating AI review"
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("x-api-key", bot.config.AnthropicToken)
	req.Header.Set("anthropic-version", "2023-06-01")

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		log.Printf("Error calling Claude API: %v", err)
		return "Error generating AI review"
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		log.Printf("Claude API returned status %d", resp.StatusCode)
		return "Error generating AI review"
	}

	var claudeResp ClaudeResponse
	if err := json.NewDecoder(resp.Body).Decode(&claudeResp); err != nil {
		log.Printf("Error decoding response: %v", err)
		return "Error generating AI review"
	}

	if len(claudeResp.Content) > 0 {
		return claudeResp.Content[0].Text
	}

	return "No response from Claude"
}

func (bot *CycloneBot) checkPRSize(pr *github.PullRequest) PRSizeCheck {
	files := pr.GetChangedFiles()
	additions := pr.GetAdditions()
	deletions := pr.GetDeletions()
	totalChanges := additions + deletions

	// Hard limits - skip review entirely
	if files > MAX_FILES_FOR_REVIEW {
		return PRSizeCheck{
			ShouldReview: false,
			SkipMessage: fmt.Sprintf(`## 🌪️ Cyclone Notice

**PR Too Large for Automated Review**

This PR modifies **%d files**, which exceeds our limit of %d files for automated review.

**Why we skip large PRs:**
- 🎯 **Review Quality**: Large PRs are harder to review thoroughly
- 🧠 **Cognitive Load**: Smaller PRs are easier for humans to understand
- 🐛 **Bug Detection**: Issues are easier to spot in focused changes
- 🚀 **Faster Iteration**: Smaller PRs get merged faster

**Suggestions:**
- Consider breaking this into smaller, focused PRs
- Each PR should ideally change < 15 files and < 400 lines
- Group related changes together (e.g., "Add user authentication", "Update API endpoints")

*Happy to review once split into smaller chunks!* 🌪️`, files, MAX_FILES_FOR_REVIEW),
		}
	}

	if additions > MAX_ADDITIONS_FOR_REVIEW {
		return PRSizeCheck{
			ShouldReview: false,
			SkipMessage: fmt.Sprintf(`## 🌪️ Cyclone Notice

**PR Too Large for Automated Review**

This PR adds **%d lines**, which exceeds our limit of %d lines for automated review.

**Large PRs are challenging because:**
- 🔍 **Review Thoroughness**: Hard to catch all issues in large changes
- ⏱️ **Review Time**: Takes much longer to review properly  
- 🤔 **Context Switching**: Difficult to keep all changes in mind
- 🔄 **Merge Conflicts**: Larger PRs are more likely to conflict

**Best Practices:**
- Aim for PRs with < 400 lines of additions
- Split features into logical, reviewable chunks
- Consider feature flags for large features

*Ready to provide detailed feedback on smaller PRs!* 🌪️`, additions, MAX_ADDITIONS_FOR_REVIEW),
		}
	}

	if totalChanges > MAX_TOTAL_CHANGES {
		return PRSizeCheck{
			ShouldReview: false,
			SkipMessage: fmt.Sprintf(`## 🌪️ Cyclone Notice

**PR Too Large for Automated Review**

This PR has **%d total changes** (+%d, -%d), exceeding our limit of %d changes.

**Recommendation**: Break this into smaller, focused PRs for better review quality and faster merge times.

*Each PR should tell a focused story about one specific change.* 🌪️`, totalChanges, additions, deletions, MAX_TOTAL_CHANGES),
		}
	}

	// Warning thresholds - review but warn
	var warnings []string
	if files > WARN_FILES_THRESHOLD {
		warnings = append(warnings, fmt.Sprintf("📁 **%d files changed** (consider < %d)", files, WARN_FILES_THRESHOLD))
	}
	if additions > WARN_ADDITIONS_THRESHOLD {
		warnings = append(warnings, fmt.Sprintf("📈 **%d lines added** (consider < %d)", additions, WARN_ADDITIONS_THRESHOLD))
	}

	var warningMessage string
	if len(warnings) > 0 {
		warningMessage = fmt.Sprintf(`**⚠️ Large PR Warning:**
%s

*Smaller PRs are easier to review thoroughly and merge faster.*

---`, strings.Join(warnings, "\n"))
	}

	return PRSizeCheck{
		ShouldReview:   true,
		WarningMessage: warningMessage,
	}
}

func (bot *CycloneBot) getPrecisionGuidelines(precision ReviewPrecision) string {
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

func (bot *CycloneBot) parseClaudeResponse(claudeText, diff string) ReviewResult {
	var comments []ReviewComment
	var summary string
	var poem string

	// Extract SUMMARY section
	summary = bot.extractSection(claudeText, "SUMMARY:")

	// Extract POEM section
	poem = bot.extractSection(claudeText, "POEM:")

	// Extract PR_COMMENT sections
	parts := strings.Split(claudeText, "PR_COMMENT:")
	for i := 1; i < len(parts); i++ {
		comment := bot.parsePRCommentBlock(parts[i])
		if comment != nil {
			comments = append(comments, *comment)
		}
	}

	// Combine summary and poem
	finalSummary := summary
	if poem != "" {
		finalSummary += "\n\n---\n\n**And now, a little poem about your changes 🌪️✨**\n" + poem
	}

	// Add Cyclone branding if not present
	finalSummary = "## 🌪️ Cyclone AI Code Review\n\n" + finalSummary

	return ReviewResult{
		Summary:  finalSummary,
		Comments: comments,
	}
}

func (bot *CycloneBot) extractSection(text, sectionHeader string) string {
	// Find the section start
	startIndex := strings.Index(text, sectionHeader)
	if startIndex == -1 {
		return ""
	}

	// Find the $$ delimiter after the section header
	delimStart := strings.Index(text[startIndex:], "$$")
	if delimStart == -1 {
		return ""
	}
	delimStart += startIndex + 2 // Move past the $$

	// Find the closing $$ delimiter
	delimEnd := strings.Index(text[delimStart:], "$$")
	if delimEnd == -1 {
		return ""
	}
	delimEnd += delimStart

	// Extract and clean the content
	content := strings.TrimSpace(text[delimStart:delimEnd])
	return content
}

func (bot *CycloneBot) parsePRCommentBlock(block string) *ReviewComment {
	// Find the content between $$ delimiters
	startDelim := strings.Index(block, "$$")
	if startDelim == -1 {
		return nil
	}

	endDelim := strings.LastIndex(block, "$$")
	if endDelim == -1 || endDelim <= startDelim {
		return nil
	}

	// Extract header (file:line:category: part before $$)
	header := strings.TrimSpace(block[:startDelim])

	// Extract content (between the $$ delimiters)
	content := strings.TrimSpace(block[startDelim+2 : endDelim])

	// Parse header: filename:line_number: emoji **category**:
	parts := strings.SplitN(header, ":", 3)
	if len(parts) < 3 {
		log.Printf("Invalid PR_COMMENT header format: %s", header)
		return nil
	}

	file := strings.TrimSpace(parts[0])
	lineNumStr := strings.TrimSpace(parts[1])
	categoryPart := strings.TrimSpace(parts[2])

	lineNum, err := strconv.Atoi(lineNumStr)
	if err != nil {
		log.Printf("Invalid line number in PR_COMMENT: %s", lineNumStr)
		return nil
	}

	// The categoryPart contains: "emoji **category**:"
	return &ReviewComment{
		Path: file,
		Line: lineNum,
		Side: "RIGHT",
		Body: fmt.Sprintf("%s\n\n%s", categoryPart, content),
	}
}

func (bot *CycloneBot) postPRReview(ctx context.Context, owner, repo string, prNumber int, review ReviewResult) error {
	// Prepare review comments for line-specific feedback
	var reviewComments []*github.DraftReviewComment

	for _, comment := range review.Comments {
		reviewComments = append(reviewComments, &github.DraftReviewComment{
			Path: github.String(comment.Path),
			Line: github.Int(comment.Line),
			Side: github.String(comment.Side),
			Body: github.String(comment.Body),
		})
	}

	// Create the review
	reviewRequest := &github.PullRequestReviewRequest{
		Body:     github.String(review.Summary),
		Event:    github.String("COMMENT"), // Can be COMMENT, APPROVE, or REQUEST_CHANGES
		Comments: reviewComments,
	}

	_, _, err := bot.client.PullRequests.CreateReview(ctx, owner, repo, prNumber, reviewRequest)
	if err != nil {
		return fmt.Errorf("failed to create review: %w", err)
	}

	return nil
}

// healthCheck provides a simple health check endpoint
func (bot *CycloneBot) healthCheck(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	fmt.Fprintf(w, "Cyclone AI Code Review Bot is running!")
}

func main() {
	// Load .env file if it exists
	loadEnvFile(".env")

	// Load configuration from environment variables
	config := &Config{
		GitHubToken:    os.Getenv("GITHUB_TOKEN"),
		Port:           getEnv("PORT", "8080"),
		WebhookSecret:  os.Getenv("WEBHOOK_SECRET"),
		AnthropicToken: os.Getenv("ANTHROPIC_API_KEY"),
	}

	// Add validation for anthropic api key
	if config.AnthropicToken == "" {
		log.Fatal("ANTHROPIC_API_KEY environment variable is required")
	}

	// Validate required configuration
	if config.GitHubToken == "" {
		log.Fatal("GITHUB_TOKEN environment variable is required")
	}

	// Load review configuration from JSON file
	reviewConfig, err := loadReviewConfig("review-config.json")
	if err != nil {
		log.Fatalf("Failed to load review configuration: %v", err)
	}

	log.Printf("Loaded configuration for %d organizations", len(reviewConfig.Organizations))

	// Create the Cyclone bot with review configuration
	bot := NewCycloneBot(config, reviewConfig)

	// Set up HTTP routes
	http.HandleFunc("/webhook", bot.handleWebhook)
	http.HandleFunc("/health", bot.healthCheck)
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, "Cyclone AI Code Review Bot\nEndpoints:\n- POST /webhook (GitHub webhooks)\n- GET /health (health check)")
	})

	// Start the server
	log.Printf("Starting server on port %s", config.Port)
	log.Fatal(http.ListenAndServe(":"+config.Port, nil))
}

func (bot *CycloneBot) getRepositoryConfig(owner, repoName string) *RepositoryConfig {
	// Look through all organizations
	for _, org := range bot.reviewConfig.Organizations {
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
