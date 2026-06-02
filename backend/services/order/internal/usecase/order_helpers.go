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

// parseCloseTime parses a "HH:MM" close-time string and returns a time.Time
// for that moment today (local clock). Returns zero time + false on bad input.
func parseCloseTime(hhMM string) (time.Time, bool) {
	var h, m int
	if _, err := fmt.Sscanf(hhMM, "%d:%d", &h, &m); err != nil {
		return time.Time{}, false
	}
	now := time.Now()
	t := time.Date(now.Year(), now.Month(), now.Day(), h, m, 0, 0, now.Location())
	return t, true
}

// isSameLocalDay reports whether t is on the same calendar day as now (local).
func isSameLocalDay(t time.Time) bool {
	now := time.Now()
	y1, mo1, d1 := t.In(now.Location()).Date()
	y2, mo2, d2 := now.Date()
	return y1 == y2 && mo1 == mo2 && d1 == d2
}
