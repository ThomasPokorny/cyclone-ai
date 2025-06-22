package review

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"

	"cyclone/internal/config"
)

// AIClient handles all AI/Claude API operations
type AIClient struct {
	apiKey string
	model  string
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

// NewAIClient creates a new AI client with the provided API key and model
func NewAIClient(apiKey, model string) *AIClient {
	return &AIClient{
		apiKey: apiKey,
		model:  model,
	}
}

// GenerateReview generates an AI review using Claude with repository-specific configuration
func (ai *AIClient) GenerateReview(diff, title, body string, repoConfig *config.RepositoryConfig) ReviewResult {
	claudeReview := ai.callClaudeAPI(diff, title, body, repoConfig)
	return ai.parseClaudeResponse(claudeReview, diff)
}

// callClaudeAPI makes a request to Claude API with repository-specific configuration
func (ai *AIClient) callClaudeAPI(diff, title, body string, repoConfig *config.RepositoryConfig) string {
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

Be constructive, helpful, and focus on actionable feedback.`, title, body, config.GetPrecisionGuidelines(repoConfig.Precision), diff, repoConfig.CustomPrompt)

	reqBody := ClaudeRequest{
		Model:     ai.model, // configurable: claude-sonnet-4-20250514, claude-3-5-sonnet-20241022, claude-3-haiku-20240307
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
	req.Header.Set("x-api-key", ai.apiKey)
	req.Header.Set("anthropic-version", "2023-06-01")

	client := &http.Client{Timeout: 60 * time.Second}
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
