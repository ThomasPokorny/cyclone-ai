// main.go - Cyclone AI Code Review Tool
package main

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"

	"github.com/google/go-github/v57/github"
	"golang.org/x/oauth2"
)

// Config holds our application configuration
type Config struct {
	GitHubToken   string
	Port          string
	WebhookSecret string
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

// getAIReview generates an AI review of the code (placeholder implementation)
func (bot *CycloneBot) getAIReview(diff, title, body string) ReviewResult {
	// TODO: Integrate with AI API (Claude, OpenAI, etc.)
	// For now, return a placeholder review with some line comments

	lines := strings.Split(diff, "\n")
	addedLines := 0
	removedLines := 0

	// Sample line-specific comments (in real implementation, AI would generate these)
	comments := []ReviewComment{}

	// Find some added lines to comment on (just for demo)
	currentFile := ""
	position := 0
	lineNumber := 0

	for _, line := range lines {
		if strings.HasPrefix(line, "=== ") && strings.HasSuffix(line, " ===") {
			// Extract filename
			currentFile = strings.Trim(line, "=== ")
			continue
		}

		if strings.HasPrefix(line, "@@") {
			// Parse hunk header to get line numbers
			// Format: @@ -old_start,old_count +new_start,new_count @@
			parts := strings.Fields(line)
			if len(parts) >= 3 {
				newRange := parts[2] // +new_start,new_count
				if strings.HasPrefix(newRange, "+") {
					fmt.Sscanf(newRange[1:], "%d", &lineNumber)
				}
			}
			position++
			continue
		}

		if strings.HasPrefix(line, "+") && !strings.HasPrefix(line, "+++") {
			addedLines++

			// Add sample comments for certain patterns (demo purposes)
			if strings.Contains(line, "console.log") && currentFile != "" {
				comments = append(comments, ReviewComment{
					Path: currentFile,
					Line: lineNumber,
					Body: "🌪️ **Cyclone**: Consider removing this console.log before merging to production.",
					Side: "RIGHT",
				})
			}

			if strings.Contains(line, "TODO") && currentFile != "" {
				comments = append(comments, ReviewComment{
					Path: currentFile,
					Line: lineNumber,
					Body: "🌪️ **Cyclone**: TODO found - consider creating a GitHub issue to track this work.",
					Side: "RIGHT",
				})
			}

			lineNumber++
		} else if strings.HasPrefix(line, "-") && !strings.HasPrefix(line, "---") {
			removedLines++
		} else if !strings.HasPrefix(line, "\\") && !strings.HasPrefix(line, "@@") {
			lineNumber++
		}

		position++
	}

	summary := fmt.Sprintf(`## 🌪️ Cyclone AI Code Review

**Summary**: This PR modifies %d lines (+%d, -%d additions/deletions).

**Quick Analysis**:
- Files changed: Multiple files detected
- Line-specific comments: %d
- Overall scope: %s

**Next Steps**: 
- [ ] AI integration pending - will provide detailed feedback once connected
- [ ] Consider adding tests for new functionality
- [ ] Verify documentation is updated

*This is a placeholder review. Full Cyclone AI analysis coming soon!*`,
		addedLines+removedLines, addedLines, removedLines, len(comments),
		func() string {
			if addedLines+removedLines > 100 {
				return "Large change"
			}
			return "Small to medium change"
		}())

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
		GitHubToken:   os.Getenv("GITHUB_TOKEN"),
		Port:          getEnv("PORT", "8080"),
		WebhookSecret: os.Getenv("WEBHOOK_SECRET"),
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
