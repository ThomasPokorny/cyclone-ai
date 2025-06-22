package review

import (
	"fmt"
	"log"
	"strconv"
	"strings"
)

// parseClaudeResponse converts Claude's text response into structured comments
func (ai *AIClient) parseClaudeResponse(claudeText, diff string) ReviewResult {
	var comments []ReviewComment
	var summary string
	var poem string

	// Extract SUMMARY section
	summary = ai.extractSection(claudeText, "SUMMARY:")

	// Extract POEM section
	poem = ai.extractSection(claudeText, "POEM:")

	// Extract PR_COMMENT sections
	parts := strings.Split(claudeText, "PR_COMMENT:")
	for i := 1; i < len(parts); i++ {
		comment := ai.parsePRCommentBlock(parts[i])
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

// extractSection extracts content between $$ delimiters for a given section
func (ai *AIClient) extractSection(text, sectionHeader string) string {
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

// parsePRCommentBlock parses a single PR_COMMENT block into a ReviewComment
func (ai *AIClient) parsePRCommentBlock(block string) *ReviewComment {
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
