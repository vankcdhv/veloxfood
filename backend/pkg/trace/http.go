package trace

import "net/http"

// InjectHTTPHeader sets X-Trace-Id on h. No-op when traceID is empty.
func InjectHTTPHeader(h http.Header, traceID string) {
	if traceID == "" {
		return
	}
	h.Set(HeaderHTTP, traceID)
}

// ExtractHTTPHeader reads X-Trace-Id from h. Returns "" if absent.
func ExtractHTTPHeader(h http.Header) string {
	return h.Get(HeaderHTTP)
}

type roundTripper struct{ next http.RoundTripper }

func (rt roundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	if id := FromContext(req.Context()); id != "" {
		req = req.Clone(req.Context())
		req.Header.Set(HeaderHTTP, id)
	}
	return rt.next.RoundTrip(req)
}

// RoundTripper wraps next so every outbound request gets X-Trace-Id from its
// context. If next is nil, http.DefaultTransport is used.
//
// Usage:
//
//	client := &http.Client{Transport: trace.RoundTripper(nil)}
func RoundTripper(next http.RoundTripper) http.RoundTripper {
	if next == nil {
		next = http.DefaultTransport
	}
	return roundTripper{next: next}
}
