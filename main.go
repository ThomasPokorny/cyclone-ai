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
	client *github.Client
	config *Config
}

// NewCycloneBot creates a new Cyclone bot instance
func NewCycloneBot(config *Config) *CycloneBot {
	ctx := context.Background()
	ts := oauth2.StaticTokenSource(
		&oauth2.Token{AccessToken: config.GitHubToken},
	)
	tc := oauth2.NewClient(ctx, ts)

	return &CycloneBot{
		client: github.NewClient(tc),
		config: config,
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

	// Only process opened or updated PRs
	if payload.Action != "opened" && payload.Action != "synchronize" {
		log.Printf("Ignoring action: %s", payload.Action)
		w.WriteHeader(http.StatusOK)
		return
	}

	log.Printf("Processing PR #%d: %s", payload.PullRequest.GetNumber(), payload.Action)

	// Process the PR in a goroutine to avoid blocking the webhook
	go bot.processPullRequest(payload.Repository, payload.PullRequest)

	w.WriteHeader(http.StatusOK)
}

// processPullRequest handles the main logic for reviewing a PR
func (bot *CycloneBot) processPullRequest(repo *github.Repository, pr *github.PullRequest) {
	ctx := context.Background()

	owner := repo.GetOwner().GetLogin()
	repoName := repo.GetName()
	prNumber := pr.GetNumber()

	log.Printf("Processing PR #%d in %s/%s", prNumber, owner, repoName)

	// Get the PR diff
	diff, err := bot.getPRDiff(ctx, owner, repoName, prNumber)
	if err != nil {
		log.Printf("Error getting PR diff: %v", err)
		return
	}

	// Get AI review (placeholder for now)
	review := bot.getAIReview(diff, pr.GetTitle(), pr.GetBody())

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

// getAIReview generates an AI review of the code (placeholder implementation)
func (bot *CycloneBot) getAIReview(diff, title, body string) ReviewResult {
	claudeReview := bot.callClaudeAPI(diff, title, body)

	// Parse Claude's response into structured feedback
	return bot.parseClaudeResponse(claudeReview, diff)
}

func (bot *CycloneBot) callClaudeAPI(diff, title, body string) string {
	prompt := fmt.Sprintf(`You are Cyclone, an AI code review assistant. Please review this GitHub pull request and provide constructive feedback.

**PR Title:** %s

**PR Description:** %s

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

For any line-specific comments, use this EXACT format:
PR_COMMENT:filename:line_number: [emoji] **[category]**: your comment here

When providing commit-able suggestions, use code blocks with the language specified.


Examples:
PR_COMMENT:main.go:45: 🔍 **nit**: Consider using a more descriptive variable name like 'userCount' instead of 'cnt'
PR_COMMENT:utils.js:123: ⚠️ **issue**: This function needs error handling for the API call
PR_COMMENT:api/handler.py:67: 🚫 **blocking**: 🔒 **security**: Potential SQL injection vulnerability - use parameterized queries

Be constructive, helpful, and focus on actionable feedback.`, title, body, diff)

	reqBody := ClaudeRequest{
		Model:     "claude-3-5-sonnet-20241022",
		MaxTokens: 4000,
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

func (bot *CycloneBot) parseClaudeResponse(claudeText, diff string) ReviewResult {
	lines := strings.Split(claudeText, "\n")
	var summaryLines []string
	var comments []ReviewComment

	// Parse line-specific comments in format "PR_COMMENT:FILE:LINE_NUMBER: comment"
	for _, line := range lines {
		if strings.HasPrefix(line, "PR_COMMENT:") {
			// Remove the PR_COMMENT: prefix
			content := strings.TrimPrefix(line, "PR_COMMENT:")

			// Split into file:line:comment
			parts := strings.SplitN(content, ":", 3)
			if len(parts) >= 3 {
				file := strings.TrimSpace(parts[0])
				lineNumStr := strings.TrimSpace(parts[1])
				comment := strings.TrimSpace(parts[2])

				if lineNum, err := strconv.Atoi(lineNumStr); err == nil {
					comments = append(comments, ReviewComment{
						Path: file,
						Line: lineNum,
						Side: "RIGHT",
						Body: fmt.Sprintf("🌪️ **Cyclone**: %s", comment),
					})
					continue
				}
			}
		}

		// If it's not a PR_COMMENT line, add it to the summary
		summaryLines = append(summaryLines, line)
	}

	summary := strings.Join(summaryLines, "\n")
	if !strings.Contains(summary, "🌪️") {
		summary = "## 🌪️ Cyclone AI Code Review\n\n" + summary
	}

	return ReviewResult{
		Summary:  summary,
		Comments: comments,
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

	// Create the Cyclone bot
	bot := NewCycloneBot(config)

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
