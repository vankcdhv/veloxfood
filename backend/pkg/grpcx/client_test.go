package grpcx

import (
	"context"
	"errors"
	"testing"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// fakeInvoker returns the scripted errors in order, then nil forever.
func fakeInvoker(script []error) (grpc.UnaryInvoker, *int) {
	calls := 0
	return func(_ context.Context, _ string, _, _ any, _ *grpc.ClientConn, _ ...grpc.CallOption) error {
		defer func() { calls++ }()
		if calls < len(script) {
			return script[calls]
		}
		return nil
	}, &calls
}

func TestRetryUnaryInterceptor_RetriesUnavailableThenSucceeds(t *testing.T) {
	inv, calls := fakeInvoker([]error{
		status.Error(codes.Unavailable, "conn refused"),
		status.Error(codes.Unavailable, "conn refused"),
	})
	err := RetryUnaryInterceptor()(context.Background(), "/svc/M", nil, nil, nil, inv)
	if err != nil {
		t.Fatalf("want success after retries, got %v", err)
	}
	if *calls != 3 {
		t.Errorf("calls: want 3 (1 + 2 retries), got %d", *calls)
	}
}

func TestRetryUnaryInterceptor_DoesNotRetryOtherCodes(t *testing.T) {
	for _, code := range []codes.Code{codes.DeadlineExceeded, codes.NotFound, codes.InvalidArgument} {
		inv, calls := fakeInvoker([]error{status.Error(code, "x")})
		err := RetryUnaryInterceptor()(context.Background(), "/svc/M", nil, nil, nil, inv)
		if status.Code(err) != code {
			t.Errorf("%v: error not passed through, got %v", code, err)
		}
		if *calls != 1 {
			t.Errorf("%v: want exactly 1 call, got %d", code, *calls)
		}
	}
}

func TestRetryUnaryInterceptor_GivesUpAfterMaxAttempts(t *testing.T) {
	inv, calls := fakeInvoker([]error{
		status.Error(codes.Unavailable, "down"),
		status.Error(codes.Unavailable, "down"),
		status.Error(codes.Unavailable, "down"),
	})
	err := RetryUnaryInterceptor()(context.Background(), "/svc/M", nil, nil, nil, inv)
	if status.Code(err) != codes.Unavailable {
		t.Fatalf("want Unavailable after exhausting retries, got %v", err)
	}
	if *calls != 3 {
		t.Errorf("calls: want 3, got %d", *calls)
	}
}

func TestBreakerUnaryInterceptor_OpensAfterFailuresAndFailsFast(t *testing.T) {
	addr := "test-breaker-target:1"
	failing := func(_ context.Context, _ string, _, _ any, _ *grpc.ClientConn, _ ...grpc.CallOption) error {
		return status.Error(codes.Unavailable, "down")
	}
	itc := BreakerUnaryInterceptor(addr)

	// Trip the breaker: ≥5 requests, ≥60% failure.
	for i := 0; i < 6; i++ {
		_ = itc(context.Background(), "/svc/M", nil, nil, nil, failing)
	}

	// Now the circuit is open: the invoker must NOT run.
	ran := false
	spy := func(_ context.Context, _ string, _, _ any, _ *grpc.ClientConn, _ ...grpc.CallOption) error {
		ran = true
		return nil
	}
	err := itc(context.Background(), "/svc/M", nil, nil, nil, spy)
	if status.Code(err) != codes.Unavailable {
		t.Fatalf("open breaker: want Unavailable, got %v", err)
	}
	if ran {
		t.Error("open breaker: invoker must not be called")
	}
}

func TestBreakerUnaryInterceptor_ApplicationErrorsDoNotTrip(t *testing.T) {
	addr := "test-breaker-target:2"
	notFound := func(_ context.Context, _ string, _, _ any, _ *grpc.ClientConn, _ ...grpc.CallOption) error {
		return status.Error(codes.NotFound, "no row")
	}
	itc := BreakerUnaryInterceptor(addr)
	for i := 0; i < 10; i++ {
		_ = itc(context.Background(), "/svc/M", nil, nil, nil, notFound)
	}
	// Circuit stays closed — a healthy service answering NotFound is not down.
	ran := false
	spy := func(_ context.Context, _ string, _, _ any, _ *grpc.ClientConn, _ ...grpc.CallOption) error {
		ran = true
		return nil
	}
	if err := itc(context.Background(), "/svc/M", nil, nil, nil, spy); err != nil {
		t.Fatalf("want pass-through, got %v", err)
	}
	if !ran {
		t.Error("invoker should have been called (breaker closed)")
	}
}

func TestTimeoutUnaryInterceptor_AddsDeadlineWhenAbsent(t *testing.T) {
	var sawDeadline bool
	inv := func(ctx context.Context, _ string, _, _ any, _ *grpc.ClientConn, _ ...grpc.CallOption) error {
		_, sawDeadline = ctx.Deadline()
		return nil
	}
	if err := TimeoutUnaryInterceptor(DefaultTimeout)(context.Background(), "/svc/M", nil, nil, nil, inv); err != nil {
		t.Fatal(err)
	}
	if !sawDeadline {
		t.Error("expected a deadline to be injected")
	}
}

func TestTimeoutUnaryInterceptor_KeepsExistingDeadline(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), DefaultTimeout*10)
	defer cancel()
	want, _ := ctx.Deadline()

	var got error
	inv := func(ctx context.Context, _ string, _, _ any, _ *grpc.ClientConn, _ ...grpc.CallOption) error {
		d, ok := ctx.Deadline()
		if !ok || !d.Equal(want) {
			got = errors.New("deadline was replaced")
		}
		return nil
	}
	if err := TimeoutUnaryInterceptor(DefaultTimeout)(ctx, "/svc/M", nil, nil, nil, inv); err != nil {
		t.Fatal(err)
	}
	if got != nil {
		t.Error(got)
	}
}
