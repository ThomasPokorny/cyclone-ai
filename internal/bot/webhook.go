package bot

import (
	"encoding/json"
	"log"
	"net/http"

	"github.com/google/go-github/v57/github"
)

// WebhookPayload represents the GitHub webhook payload
type WebhookPayload struct {
	Action      string              `json:"action"`
	PullRequest *github.PullRequest `json:"pull_request"`
	Repository  *github.Repository  `json:"repository"`
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
	go bot.ProcessPullRequest(payload.Repository, payload.PullRequest)

	w.WriteHeader(http.StatusOK)
}

// shouldTriggerReview determines if we should review this PR based on action and state
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
