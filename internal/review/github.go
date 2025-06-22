package review

import (
	"context"
	"fmt"
	"strings"

	"github.com/google/go-github/v57/github"
	"golang.org/x/oauth2"
)

// GitHubClient handles all GitHub API operations
type GitHubClient struct {
	client *github.Client
}

// NewGitHubClient creates a new GitHub client with the provided token
func NewGitHubClient(token string) (*GitHubClient, error) {
	ctx := context.Background()
	ts := oauth2.StaticTokenSource(
		&oauth2.Token{AccessToken: token},
	)
	tc := oauth2.NewClient(ctx, ts)

	return &GitHubClient{
		client: github.NewClient(tc),
	}, nil
}

// GetPRDiff fetches the diff for a pull request
func (g *GitHubClient) GetPRDiff(ctx context.Context, owner, repo string, prNumber int) (string, error) {
	// Get the PR files
	files, _, err := g.client.PullRequests.ListFiles(ctx, owner, repo, prNumber, nil)
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

// PostReview posts a complete PR review with line-specific comments
func (g *GitHubClient) PostReview(ctx context.Context, owner, repo string, prNumber int, review ReviewResult) error {
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

	_, _, err := g.client.PullRequests.CreateReview(ctx, owner, repo, prNumber, reviewRequest)
	if err != nil {
		return fmt.Errorf("failed to create review: %w", err)
	}

	return nil
}

// PostComment posts a simple comment to a PR (used for skip messages)
func (g *GitHubClient) PostComment(ctx context.Context, owner, repo string, prNumber int, body string) error {
	comment := &github.IssueComment{
		Body: github.String(body),
	}

	_, _, err := g.client.Issues.CreateComment(ctx, owner, repo, prNumber, comment)
	if err != nil {
		return fmt.Errorf("failed to create comment: %w", err)
	}

	return nil
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
