package audit

import (
	"encoding/json"
	"testing"
)

// TestRedactSensitive_RemovesPasswordKeys verifies that sensitive fields
// are stripped from payload maps before JSON serialization.
func TestRedactSensitive_RemovesPasswordKeys(t *testing.T) {
	cases := []struct {
		name    string
		payload map[string]any
		banned  []string
		allowed []string
	}{
		{
			name: "password stripped",
			payload: map[string]any{
				"user_id":  "u1",
				"password": "secret123",
				"email":    "a@b.com",
			},
			banned:  []string{"password"},
			allowed: []string{"user_id", "email"},
		},
		{
			name: "token and code stripped",
			payload: map[string]any{
				"token":   "rawtoken",
				"code":    "123456",
				"action":  "reset",
			},
			banned:  []string{"token", "code"},
			allowed: []string{"action"},
		},
		{
			name: "password_hash stripped",
			payload: map[string]any{
				"password_hash": "$2a$10$...",
				"status":        "active",
			},
			banned:  []string{"password_hash"},
			allowed: []string{"status"},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			result := redactSensitive(tc.payload)
			b, err := json.Marshal(result)
			if err != nil {
				t.Fatalf("marshal failed: %v", err)
			}
			s := string(b)
			for _, key := range tc.banned {
				if containsKey(result.(map[string]any), key) {
					t.Errorf("expected key %q to be redacted, but found in result: %s", key, s)
				}
			}
			for _, key := range tc.allowed {
				if !containsKey(result.(map[string]any), key) {
					t.Errorf("expected key %q to be present, but missing from result: %s", key, s)
				}
			}
		})
	}
}

// TestRedactSensitive_StringMap verifies map[string]string variant.
func TestRedactSensitive_StringMap(t *testing.T) {
	payload := map[string]string{
		"reason":   "test",
		"password": "should be gone",
		"token":    "also gone",
	}
	result := redactSensitive(payload)
	m, ok := result.(map[string]string)
	if !ok {
		t.Fatalf("expected map[string]string, got %T", result)
	}
	if _, has := m["password"]; has {
		t.Error("password should be redacted from map[string]string")
	}
	if _, has := m["token"]; has {
		t.Error("token should be redacted from map[string]string")
	}
	if m["reason"] != "test" {
		t.Error("reason should be preserved")
	}
}

// TestRedactSensitive_NonMap passes through unchanged.
func TestRedactSensitive_NonMap_PassThrough(t *testing.T) {
	payload := "just a string"
	result := redactSensitive(payload)
	if result != payload {
		t.Errorf("expected passthrough, got %v", result)
	}
}

func containsKey(m map[string]any, key string) bool {
	_, ok := m[key]
	return ok
}
