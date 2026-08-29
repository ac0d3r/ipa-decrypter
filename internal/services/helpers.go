package services

import "strings"

func firstNonEmpty(vals ...string) string {
	for _, v := range vals {
		if strings.TrimSpace(v) != "" {
			return strings.TrimSpace(v)
		}
	}
	return ""
}

func errString(err error) string {
	if err == nil {
		return ""
	}
	return err.Error()
}

func looksLikeLicenseRequired(msg string) bool {
	m := strings.ToLower(msg)
	return strings.Contains(m, "license is required") || strings.Contains(m, "license required")
}

func looksLikeLicenseAlreadyOwned(msg string) bool {
	m := strings.ToLower(msg)
	return strings.Contains(m, "license already exists") || strings.Contains(m, "already purchased")
}

func looksLikeAppleUnknownError(msg string) bool {
	m := strings.ToLower(msg)
	return strings.Contains(m, "unknown error") || strings.Contains(m, "an unknown error")
}

// Apple sometimes returns HTTP 204 (no body) for a successful free-app "get".
// ipatool treats it as failure, but license may already be acquired.
func looksLikeAppleHTTP204(msg string) bool {
	return strings.Contains(msg, "HTTP 204")
}
