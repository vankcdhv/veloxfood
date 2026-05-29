package trace

import "github.com/segmentio/kafka-go"

// InjectKafkaHeaders returns headers with an x-trace-id entry. If a header
// with the same key already exists it is replaced (not duplicated). When
// traceID is empty the input is returned unchanged.
func InjectKafkaHeaders(headers []kafka.Header, traceID string) []kafka.Header {
	if traceID == "" {
		return headers
	}
	out := headers[:0:0]
	for _, h := range headers {
		if h.Key == KafkaHeader {
			continue
		}
		out = append(out, h)
	}
	return append(out, kafka.Header{Key: KafkaHeader, Value: []byte(traceID)})
}

// ExtractKafkaHeaders returns the first x-trace-id value, "" if absent.
func ExtractKafkaHeaders(headers []kafka.Header) string {
	for _, h := range headers {
		if h.Key == KafkaHeader {
			return string(h.Value)
		}
	}
	return ""
}
