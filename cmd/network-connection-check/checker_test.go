package main

import (
	"context"
	"errors"
	"net"
	"testing"
)

func TestEvaluateResultReachableTarget(t *testing.T) {
	checker := &Checker{
		connectionTarget: "tcp://example.com:443",
	}

	passed, message := checker.evaluateResult(connectionCheckResult{})
	if !passed {
		t.Fatalf("expected reachable target with no error to pass, got message %q", message)
	}

	dialErr := errors.New("dial tcp: connection refused")
	passed, message = checker.evaluateResult(connectionCheckResult{dialErr: dialErr})
	if passed {
		t.Fatal("expected reachable target with dial error to fail")
	}
	if message != dialErr.Error() {
		t.Fatalf("expected original dial error message %q, got %q", dialErr.Error(), message)
	}
	closeErr := errors.New("close tcp: use of closed network connection")
	passed, message = checker.evaluateResult(connectionCheckResult{closeErr: closeErr})
	if passed {
		t.Fatal("expected reachable target with close error to fail")
	}
	if message != closeErr.Error() {
		t.Fatalf("expected original close error message %q, got %q", closeErr.Error(), message)
	}
}

func TestEvaluateResultUnreachableTargetDialError(t *testing.T) {
	checker := &Checker{
		connectionTarget:  "tcp://169.254.169.254:80",
		targetUnreachable: true,
	}

	passed, message := checker.evaluateResult(connectionCheckResult{dialErr: errors.New("dial tcp: i/o timeout")})
	if !passed {
		t.Fatalf("expected unreachable target with dial error to pass, got message %q", message)
	}
}

func TestEvaluateResultUnreachableTargetPreDialContextError(t *testing.T) {
	checker := &Checker{
		connectionTarget:  "tcp://169.254.169.254:80",
		targetUnreachable: true,
	}

	passed, message := checker.evaluateResult(connectionCheckResult{contextErr: context.DeadlineExceeded})
	if passed {
		t.Fatal("expected pre-dial context error to fail even when target should be unreachable")
	}
	want := "Network connection check did not run before the check context ended: context deadline exceeded"
	if message != want {
		t.Fatalf("expected message %q, got %q", want, message)
	}
}

func TestEvaluateResultUnreachableTargetReachable(t *testing.T) {
	checker := &Checker{
		connectionTarget:  "tcp://169.254.169.254:80",
		targetUnreachable: true,
	}

	passed, message := checker.evaluateResult(connectionCheckResult{})
	if passed {
		t.Fatal("expected unreachable target with no dial error to fail")
	}
	want := "Network connection check determined that tcp://169.254.169.254:80 is UP but expected it to be unreachable"
	if message != want {
		t.Fatalf("expected message %q, got %q", want, message)
	}
}

func TestEvaluateResultUnreachableTargetCloseError(t *testing.T) {
	checker := &Checker{
		connectionTarget:  "tcp://169.254.169.254:80",
		targetUnreachable: true,
	}

	closeErr := errors.New("close tcp: use of closed network connection")
	passed, message := checker.evaluateResult(connectionCheckResult{closeErr: closeErr})
	if passed {
		t.Fatal("expected unreachable target with close error after successful dial to fail")
	}
	want := "Network connection check determined that tcp://169.254.169.254:80 is UP but the connection did not close cleanly: close tcp: use of closed network connection"
	if message != want {
		t.Fatalf("expected message %q, got %q", want, message)
	}
}

func TestDoChecksReachableTarget(t *testing.T) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("failed to start test listener: %v", err)
	}
	defer listener.Close()

	accepted := make(chan struct{})
	go func() {
		conn, acceptErr := listener.Accept()
		if acceptErr == nil {
			conn.Close()
		}
		close(accepted)
	}()

	checker := &Checker{connectionTarget: "tcp://" + listener.Addr().String()}
	result := checker.doChecks(context.Background())
	if result.dialErr != nil {
		t.Fatalf("expected reachable target to have no dial error, got %v", result.dialErr)
	}
	if result.closeErr != nil {
		t.Fatalf("expected reachable target to close cleanly, got %v", result.closeErr)
	}
	<-accepted
}

func TestDoChecksUnreachableTarget(t *testing.T) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("failed to reserve local port: %v", err)
	}
	address := listener.Addr().String()
	listener.Close()

	checker := &Checker{connectionTarget: "tcp://" + address}
	result := checker.doChecks(context.Background())
	if result.dialErr == nil {
		t.Fatal("expected unreachable target to produce a dial error")
	}
	if result.closeErr != nil {
		t.Fatalf("expected unreachable target to have no close error, got %v", result.closeErr)
	}
}

func TestDoChecksPreDialContextError(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	checker := &Checker{connectionTarget: "tcp://127.0.0.1:1"}
	result := checker.doChecks(ctx)
	if result.contextErr == nil {
		t.Fatal("expected canceled context before dialing to produce a context error")
	}
	if result.dialErr != nil {
		t.Fatalf("expected canceled context before dialing to skip dial, got %v", result.dialErr)
	}
	if result.closeErr != nil {
		t.Fatalf("expected canceled context before dialing to have no close error, got %v", result.closeErr)
	}
}

func TestSplitAddress(t *testing.T) {
	tests := []struct {
		name        string
		fullAddress string
		wantNetwork string
		wantAddress string
	}{
		{
			name:        "explicit tcp",
			fullAddress: "tcp://example.com:443",
			wantNetwork: "tcp",
			wantAddress: "example.com:443",
		},
		{
			name:        "explicit udp",
			fullAddress: "udp://8.8.8.8:53",
			wantNetwork: "udp",
			wantAddress: "8.8.8.8:53",
		},
		{
			name:        "default tcp",
			fullAddress: "example.com:443",
			wantNetwork: "tcp",
			wantAddress: "example.com:443",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotNetwork, gotAddress := splitAddress(tt.fullAddress)
			if gotNetwork != tt.wantNetwork {
				t.Fatalf("expected network %q, got %q", tt.wantNetwork, gotNetwork)
			}
			if gotAddress != tt.wantAddress {
				t.Fatalf("expected address %q, got %q", tt.wantAddress, gotAddress)
			}
		})
	}
}
