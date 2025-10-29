package message

import (
	"regexp"
	"strings"
)

// ExtractThinkingInfo analyzes a message content to identify thinking patterns
// and returns information about the thinking content
func ExtractThinkingInfo(content string) (hasThinking bool, thinkingTokens int, thinkingContent string) {
	// If content is empty, no thinking
	if content == "" {
		return false, 0, ""
	}

	// Common thinking section markers
	thinkingMarkers := []string{
		"## Thinking:",
		"## Let me think",
		"Let me think about this",
		"Let's think through this",
		"Let me reason through this",
		"Thinking step by step",
		"Reasoning step by step",
		"Step 1:",
		"First, let me analyze",
		"I need to think about",
	}

	// Check for thinking markers in the content
	hasThinking = false
	for _, marker := range thinkingMarkers {
		if strings.Contains(content, marker) {
			hasThinking = true
			break
		}
	}

	// If no thinking detected, return early
	if !hasThinking {
		return false, 0, ""
	}

	// Extract thinking content
	thinkingContent = extractThinkingSection(content)
	thinkingLength := len(thinkingContent)

	return true, thinkingLength, thinkingContent
}

// extractThinkingSection attempts to extract the thinking portion of a response
// This is heuristic-based since different models format thinking differently
func extractThinkingSection(content string) string {
	// Try to find content between thinking markers and conclusion/summary
	// Common patterns:
	// 1. ## Thinking: ... ## Conclusion:
	// 2. Let me think about this... In conclusion,
	// 3. Step 1: ... Step 2: ... Therefore,

	// Pattern 1: Markdown-style thinking sections
	thinkingRegex := regexp.MustCompile(`(?i)(#+\s*Think(ing|s)*:?.*?)(?:#+\s*(Conclusion|Summary|Answer|Therefore)|$)`)
	matches := thinkingRegex.FindStringSubmatch(content)
	if len(matches) > 1 {
		return matches[1]
	}

	// Pattern 2: "Let me think" style followed by conclusion
	let_think_regex := regexp.MustCompile(`(?i)(Let me think.*?)(?:Therefore|In conclusion|To summarize|In summary|So,|Thus,|$)`)
	matches = let_think_regex.FindStringSubmatch(content)
	if len(matches) > 1 {
		return matches[1]
	}

	// Pattern 3: Step-by-step thinking
	step_regex := regexp.MustCompile(`(?i)(Step 1:.*?)(?:In conclusion|To summarize|Therefore|Finally|$)`)
	matches = step_regex.FindStringSubmatch(content)
	if len(matches) > 1 {
		return matches[1]
	}

	// If we can't clearly identify a thinking section, return the first half
	// of the content as a reasonable approximation
	if hasThinkingMarker(content) {
		return content[:len(content)/2]
	}

	// No thinking section found
	return ""
}

// hasThinkingMarker checks if the content has any thinking marker
func hasThinkingMarker(content string) bool {
	markers := []string{
		"think", "reason", "analysis", "step", "examine", "consider",
	}

	for _, marker := range markers {
		if strings.Contains(strings.ToLower(content), marker) {
			return true
		}
	}

	return false
}
