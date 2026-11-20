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
	// Key-value passwords, secrets, and API keys (e.g. password=hunter2, api_key: "...")
	regexp.MustCompile(`(?i)\b(?:password|passwd|pwd|secret|api[_-]?key|access[_-]?token)\s*[:=]\s*["']?[^\s"']+["']?`),
	// IBAN (e.g. IT60X0542811101000000123456)
	regexp.MustCompile(`\b[A-Z]{2}\d{2}(?:\s?[A-Z0-9]){11,30}\b`),
	// Email addresses
	regexp.MustCompile(`\b[A-Za-z0-9._%+-]+@[A-Za-z0-9.-]+\.[A-Za-z]{2,}\b`),
	// Phone numbers (international or local formats such as +39 347 123 4567 or 347-123-4567)
	regexp.MustCompile(`(?:\+\d{1,3}[\s.-]?)?(?:\(\d{2,4}\)|\d{2,4})[\s.-]\d{3,4}[\s.-]\d{3,4}\b`),
}

// RedactText replaces secrets, tokens, and email addresses with [REDACTED] (Threat Model T4).
func RedactText(input string) string {
	out := input
	for _, re := range redactionPatterns {
		out = re.ReplaceAllString(out, "[REDACTED]")
	}
	return out
}
