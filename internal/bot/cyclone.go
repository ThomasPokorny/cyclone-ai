package bot

import (
	"context"
	"fmt"
	"log"
	"net/http"

	"github.com/google/go-github/v57/github"

	"cyclone/internal/config"
	"cyclone/internal/review"
)

// CycloneBot handles GitHub operations and AI integration
type CycloneBot struct {
	githubClient *review.GitHubClient
	aiClient     *review.AIClient
	config       *config.Config
	reviewConfig *config.ReviewConfig
}

// New creates a new Cyclone bot instance
func New(cfg *config.Config, reviewCfg *config.ReviewConfig) (*CycloneBot, error) {
	// Initialize GitHub client
	githubClient, err := review.NewGitHubClient(cfg.GitHubToken)
	if err != nil {
		return nil, fmt.Errorf("failed to create GitHub client: %w", err)
	}

	// Initialize AI client
	aiClient := review.NewAIClient(cfg.AnthropicToken, "claude-sonnet-4-20250514")

	return &CycloneBot{
		githubClient: githubClient,
		aiClient:     aiClient,
		config:       cfg,
		reviewConfig: reviewCfg,
	}, nil
}

// SetupRoutes configures HTTP routes for the bot
func (bot *CycloneBot) SetupRoutes() {
	http.HandleFunc("/webhook", bot.handleWebhook)
	http.HandleFunc("/health", bot.healthCheck)
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, "Cyclone AI Code Review Bot\nEndpoints:\n- POST /webhook (GitHub webhooks)\n- GET /health (health check)")
	})
}

// ProcessPullRequest handles the main logic for reviewing a PR
func (bot *CycloneBot) ProcessPullRequest(repo *github.Repository, pr *github.PullRequest) {
	ctx := context.Background()

	owner := repo.GetOwner().GetLogin()
	repoName := repo.GetName()
	prNumber := pr.GetNumber()

	log.Printf("Processing PR #%d in %s/%s", prNumber, owner, repoName)

	// Get repository-specific configuration
	repoConfig := bot.reviewConfig.GetRepositoryConfig(owner, repoName)
	if repoConfig == nil {
		log.Printf("Repository %s/%s not found in configuration - skipping review", owner, repoName)
		return
	}

	// Check PR size before proceeding
	sizeCheck := bot.checkPRSize(pr)
	if !sizeCheck.ShouldReview {
		log.Printf("PR #%d is too large - posting skip message instead of review", prNumber)

		// Post skip message as a regular comment
		if err := bot.githubClient.PostComment(ctx, owner, repoName, prNumber, sizeCheck.SkipMessage); err != nil {
			log.Printf("Error posting skip message: %v", err)
		}
		return
	}

	log.Printf("Using precision: %s for repository: %s", repoConfig.Precision, repoName)

	// Get the PR diff
	diff, err := bot.githubClient.GetPRDiff(ctx, owner, repoName, prNumber)
	if err != nil {
		log.Printf("Error getting PR diff: %v", err)
		return
	}

	// Get AI review with repository-specific configuration
	reviewResult := bot.aiClient.GenerateReview(diff, pr.GetTitle(), pr.GetBody(), repoConfig)

	// Prepend size warning if applicable
	if sizeCheck.WarningMessage != "" {
		reviewResult.Summary = sizeCheck.WarningMessage + reviewResult.Summary
	}

	// Post the review with line-specific comments
	if err := bot.githubClient.PostReview(ctx, owner, repoName, prNumber, reviewResult); err != nil {
		log.Printf("Error posting PR review: %v", err)
		return
	}

	log.Printf("Successfully posted AI review for PR #%d", prNumber)
}

// checkPRSize evaluates if a PR is too large for review
func (bot *CycloneBot) checkPRSize(pr *github.PullRequest) review.PRSizeCheck {
	files := pr.GetChangedFiles()
	additions := pr.GetAdditions()
	deletions := pr.GetDeletions()
	totalChanges := additions + deletions

	// Hard limits - skip review entirely
	if files > config.MAX_FILES_FOR_REVIEW {
		return review.PRSizeCheck{
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

*Happy to review once split into smaller chunks!* 🌪️`, files, config.MAX_FILES_FOR_REVIEW),
		}
	}

	if additions > config.MAX_ADDITIONS_FOR_REVIEW {
		return review.PRSizeCheck{
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

*Ready to provide detailed feedback on smaller PRs!* 🌪️`, additions, config.MAX_ADDITIONS_FOR_REVIEW),
		}
	}

	if totalChanges > config.MAX_TOTAL_CHANGES {
		return review.PRSizeCheck{
			ShouldReview: false,
			SkipMessage: fmt.Sprintf(`## 🌪️ Cyclone Notice

**PR Too Large for Automated Review**

This PR has **%d total changes** (+%d, -%d), exceeding our limit of %d changes.

**Recommendation**: Break this into smaller, focused PRs for better review quality and faster merge times.

*Each PR should tell a focused story about one specific change.* 🌪️`, totalChanges, additions, deletions, config.MAX_TOTAL_CHANGES),
		}
	}

	// Warning thresholds - review but warn
	var warnings []string
	if files > config.WARN_FILES_THRESHOLD {
		warnings = append(warnings, fmt.Sprintf("📁 **%d files changed** (consider < %d)", files, config.WARN_FILES_THRESHOLD))
	}
	if additions > config.WARN_ADDITIONS_THRESHOLD {
		warnings = append(warnings, fmt.Sprintf("📈 **%d lines added** (consider < %d)", additions, config.WARN_ADDITIONS_THRESHOLD))
	}

	var warningMessage string
	if len(warnings) > 0 {
		warningMessage = fmt.Sprintf(`**⚠️ Large PR Warning:**
%s

*Smaller PRs are easier to review thoroughly and merge faster.*

---`, fmt.Sprintf("%s\n", warnings[0]))
		if len(warnings) > 1 {
			warningMessage = fmt.Sprintf(`**⚠️ Large PR Warning:**
%s
%s

*Smaller PRs are easier to review thoroughly and merge faster.*

---`, warnings[0], warnings[1])
		}
	}

	return review.PRSizeCheck{
		ShouldReview:   true,
		WarningMessage: warningMessage,
	}
}

// healthCheck provides a simple health check endpoint
func (bot *CycloneBot) healthCheck(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	fmt.Fprintf(w, "Cyclone AI Code Review Bot is running!")
}
