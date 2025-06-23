package bot

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"io"
	"log"
	"net/http"

	"github.com/google/go-github/v57/github"
)

// WebhookPayload represents the GitHub webhook payload
type WebhookPayload struct {
	Action       string              `json:"action"`
	PullRequest  *github.PullRequest `json:"pull_request"`
	Repository   *github.Repository  `json:"repository"`
	Installation *struct {
		ID int64 `json:"id"`
	} `json:"installation"`
}

// handleWebhook processes incoming GitHub webhooks
func (bot *CycloneBot) handleWebhook(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	body, err := io.ReadAll(r.Body)
	if err != nil {
		log.Printf("Error reading webhook body: %v", err)
		http.Error(w, "Bad request", http.StatusBadRequest)
		return
	}

	if bot.config.GitHubWebhookSecret != "" {
		signature := r.Header.Get("X-Hub-Signature-256")
		if !bot.validateWebhookSignature(body, signature) {
			log.Printf("Invalid webhook signature")
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}
	}

	// Parse the webhook payload
	var payload WebhookPayload
	if err := json.Unmarshal(body, &payload); err != nil {
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

	// Get installation ID
	var installationID int64
	if payload.Installation != nil {
		installationID = payload.Installation.ID
	}

	// Process the PR in a goroutine to avoid blocking the webhook
	go bot.ProcessPullRequest(payload.Repository, payload.PullRequest, installationID)

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

func (bot *CycloneBot) validateWebhookSignature(payload []byte, signature string) bool {
	if signature == "" {
		return false
	}

	// Remove 'sha256=' prefix
	if len(signature) > 7 && signature[:7] == "sha256=" {
		signature = signature[7:]
	}

	mac := hmac.New(sha256.New, []byte(bot.config.GitHubWebhookSecret))
	mac.Write(payload)
	expectedMAC := hex.EncodeToString(mac.Sum(nil))

	return hmac.Equal([]byte(signature), []byte(expectedMAC))
}
