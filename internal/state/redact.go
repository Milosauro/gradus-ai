package state

import "regexp"

var redactionPatterns = []*regexp.Regexp{
	// GitHub tokens (ghp_, gho_, ghu_, ghs_, ghr_, github_pat_)
	regexp.MustCompile(`\b(?:gh[pousr]_[A-Za-z0-9_]{12,}|github_pat_[A-Za-z0-9_]{16,})\b`),
	// OpenAI / Anthropic / generic sk- secret keys
	regexp.MustCompile(`\bsk-[A-Za-z0-9_-]{12,}\b`),
	// AWS access key IDs
	regexp.MustCompile(`\bAKIA[0-9A-Z]{16}\b`),
	// Bearer tokens
	regexp.MustCompile(`(?i)\bBearer\s+[A-Za-z0-9._~+/=-]{12,}`),
	// JWTs (three base64url segments starting with eyJ)
	regexp.MustCompile(`\beyJ[A-Za-z0-9_-]{8,}\.eyJ[A-Za-z0-9_-]{8,}\.[A-Za-z0-9_-]{8,}\b`),
	// Email addresses
	regexp.MustCompile(`\b[A-Za-z0-9._%+-]+@[A-Za-z0-9.-]+\.[A-Za-z]{2,}\b`),
}

// RedactText replaces secrets, tokens, and email addresses with [REDACTED] (Threat Model T4).
func RedactText(input string) string {
	out := input
	for _, re := range redactionPatterns {
		out = re.ReplaceAllString(out, "[REDACTED]")
	}
	return out
}
