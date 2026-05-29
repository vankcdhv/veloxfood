package trace

import (
	"testing"

	amqp "github.com/rabbitmq/amqp091-go"
)

func TestInjectExtractAMQPTable_String(t *testing.T) {
	tbl := amqp.Table{}
	InjectAMQPTable(tbl, "a-trace")
	if got := ExtractAMQPTable(tbl); got != "a-trace" {
		t.Fatalf("ExtractAMQPTable = %q, want a-trace", got)
	}
}

func TestExtractAMQPTable_BytesValue(t *testing.T) {
	tbl := amqp.Table{AMQPHeader: []byte("from-bytes")}
	if got := ExtractAMQPTable(tbl); got != "from-bytes" {
		t.Fatalf("ExtractAMQPTable = %q, want from-bytes", got)
	}
}

func TestExtractAMQPTable_Missing(t *testing.T) {
	if got := ExtractAMQPTable(amqp.Table{}); got != "" {
		t.Fatalf("ExtractAMQPTable on empty table = %q, want empty", got)
	}
	if got := ExtractAMQPTable(nil); got != "" {
		t.Fatalf("ExtractAMQPTable(nil) = %q, want empty", got)
	}
}

func TestInjectAMQPTable_EmptyAndNilNoOp(t *testing.T) {
	tbl := amqp.Table{}
	InjectAMQPTable(tbl, "")
	if len(tbl) != 0 {
		t.Fatalf("expected empty table, got %v", tbl)
	}
	// nil table must not panic
	InjectAMQPTable(nil, "anything")
}
