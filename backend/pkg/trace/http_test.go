package trace

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestInjectExtractHTTPHeader(t *testing.T) {
	h := http.Header{}
	InjectHTTPHeader(h, "trace-xyz")
	if got := ExtractHTTPHeader(h); got != "trace-xyz" {
		t.Fatalf("ExtractHTTPHeader = %q, want trace-xyz", got)
	}
}

func TestInjectHTTPHeader_EmptyNoOp(t *testing.T) {
	h := http.Header{}
	InjectHTTPHeader(h, "")
	if got := ExtractHTTPHeader(h); got != "" {
		t.Fatalf("ExtractHTTPHeader after empty inject = %q, want empty", got)
	}
}

func TestRoundTripper_InjectsTraceIDFromContext(t *testing.T) {
	var got string
	srv := httptest.NewServer(http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
		got = r.Header.Get(HeaderHTTP)
	}))
	defer srv.Close()

	client := &http.Client{Transport: RoundTripper(nil)}
	req, _ := http.NewRequestWithContext(
		WithTraceID(context.Background(), "rt-trace"),
		http.MethodGet, srv.URL, nil,
	)
	resp, err := client.Do(req)
	if err != nil {
		t.Fatalf("client.Do: %v", err)
	}
	resp.Body.Close()

	if got != "rt-trace" {
		t.Fatalf("server saw header %q, want rt-trace", got)
	}
}

func TestRoundTripper_OmitsHeaderWhenNoTrace(t *testing.T) {
	var got string
	srv := httptest.NewServer(http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
		got = r.Header.Get(HeaderHTTP)
	}))
	defer srv.Close()

	client := &http.Client{Transport: RoundTripper(nil)}
	req, _ := http.NewRequest(http.MethodGet, srv.URL, nil)
	resp, err := client.Do(req)
	if err != nil {
		t.Fatalf("client.Do: %v", err)
	}
	resp.Body.Close()

	if got != "" {
		t.Fatalf("expected no header, got %q", got)
	}
}
