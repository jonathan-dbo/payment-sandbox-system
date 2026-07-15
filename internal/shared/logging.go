// Package shared contains cross-cutting shared helpers.
package shared

import (
	"encoding/json"
	"log"
	"strings"
	"time"
)

func LogEvent(event string, fields map[string]any) {
	payload := map[string]any{
		"event":      event,
		"timestamp":  time.Now().UTC().Format(time.RFC3339),
		"attributes": RedactFields(fields),
	}
	if b, err := json.Marshal(payload); err == nil {
		log.Println(string(b))
	}
}

func RedactFields(fields map[string]any) map[string]any {
	out := map[string]any{}
	for k, v := range fields {
		if shouldRedact(k) {
			out[k] = "[REDACTED]"
			continue
		}
		out[k] = v
	}
	return out
}

func shouldRedact(key string) bool {
	normalized := strings.ToLower(key)
	return strings.Contains(normalized, "password") ||
		strings.Contains(normalized, "token") ||
		strings.Contains(normalized, "secret") ||
		strings.Contains(normalized, "jwt")
}
