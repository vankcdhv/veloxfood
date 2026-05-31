package usecase

import (
	"encoding/json"
	"fmt"
	"math/rand"
	"strings"
	"time"
)

// strPtr returns a pointer to s, or nil if s is empty.
func strPtr(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}

// parseDate parses a YYYY-MM-DD string into time.Time (UTC).
func parseDate(s string) (time.Time, error) {
	return time.Parse("2006-01-02", s)
}

// generateOrderCode produces a human-readable order code like VLX-20240601-A3F9.
func generateOrderCode() string {
	const chars = "ABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	suffix := make([]byte, 4)
	for i := range suffix {
		suffix[i] = chars[rand.Intn(len(chars))]
	}
	return fmt.Sprintf("VLX-%s-%s", time.Now().UTC().Format("20060102"), string(suffix))
}

// generatePickupPIN returns a random 4-digit PIN string.
func generatePickupPIN() string {
	return fmt.Sprintf("%04d", rand.Intn(10000))
}

// marshalJSON marshals v to json.RawMessage, returning an empty array on error.
func marshalJSON(v any) json.RawMessage {
	b, err := json.Marshal(v)
	if err != nil {
		return json.RawMessage("[]")
	}
	return b
}

// codesJSON converts a []string slice to a json.RawMessage array.
func codesJSON(codes []string) json.RawMessage {
	if len(codes) == 0 {
		return json.RawMessage("[]")
	}
	quoted := make([]string, len(codes))
	for i, c := range codes {
		quoted[i] = `"` + c + `"`
	}
	return json.RawMessage("[" + strings.Join(quoted, ",") + "]")
}
