package state

import "regexp"

var redactionPatterns = []*regexp.Regexp{
	// PEM private key blocks (multiline)
	regexp.MustCompile(`-----BEGIN [A-Z0-9 ]*PRIVATE KEY-----[\s\S]*?-----END [A-Z0-9 ]*PRIVATE KEY-----`),
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
	// Italian fiscal code (Codice Fiscale, 16 chars)
	regexp.MustCompile(`\b[A-Z]{6}\d{2}[A-Z]\d{2}[A-Z]\d{3}[A-Z]\b`),
	// Email addresses
	regexp.MustCompile(`\b[A-Za-z0-9._%+-]+@[A-Za-z0-9.-]+\.[A-Za-z]{2,}\b`),
	// International phone numbers (e.g. +39 333 1234567, +39 347 123 4567, +1-555-123-4567)
	regexp.MustCompile(`\+\d{1,3}[\s.-]?\d{2,4}[\s.-]?\d{3,4}(?:[\s.-]?\d{3,4})\b`),
	// Dashed or parenthesized local phone numbers (avoids false positives on plain space-separated numbers)
	regexp.MustCompile(`(?:\(\d{2,4}\)[\s.-]?\d{3,4}[-.]\d{3,4}|\b\d{3}[-.]\d{3}[-.]\d{4})\b`),
}

// RedactText replaces secrets, tokens, keys, and PII with [REDACTED] (Threat Model T4).
func RedactText(input string) string {
	out := input
	for _, re := range redactionPatterns {
		out = re.ReplaceAllString(out, "[REDACTED]")
	}
	return out
}
