package trace

import (
	"fmt"

	amqp "github.com/rabbitmq/amqp091-go"
)

// InjectAMQPTable sets x-trace-id on t. The caller is responsible for ensuring
// t is non-nil (Tables are maps; calling on a nil map would panic). No-op when
// traceID is empty.
func InjectAMQPTable(t amqp.Table, traceID string) {
	if traceID == "" || t == nil {
		return
	}
	t[AMQPHeader] = traceID
}

// ExtractAMQPTable returns the x-trace-id value from t. Handles string, []byte,
// and fmt.Stringer — RabbitMQ brokers occasionally round-trip table values as
// []byte instead of string depending on broker version.
func ExtractAMQPTable(t amqp.Table) string {
	if t == nil {
		return ""
	}
	v, ok := t[AMQPHeader]
	if !ok {
		return ""
	}
	switch x := v.(type) {
	case string:
		return x
	case []byte:
		return string(x)
	case fmt.Stringer:
		return x.String()
	default:
		return ""
	}
}
