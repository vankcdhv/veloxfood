package usecase

import (
	"encoding/json"
	"fmt"
	"math/rand"
	"strings"
	"time"
)

// buildOrderCode produces a human-readable order code like VLX-260601-001,
// combining the day (yymmdd) with that day's order sequence number.
func buildOrderCode(day time.Time, seq int) string {
	return fmt.Sprintf("VLX-%s-%03d", day.Format("060102"), seq)
}

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
