package main

import (
	"context"
	"errors"
	"net"
	"strings"
	"time"

	log "github.com/sirupsen/logrus"
	"k8s.io/client-go/kubernetes"
)

// Checker validates that network connections can be established.
type Checker struct {
	// client is the Kubernetes client for parity with v2 behavior.
	client *kubernetes.Clientset
	// connectionTarget is the network target to dial.
	connectionTarget string
	// targetUnreachable flips expected connectivity logic.
	targetUnreachable bool
	// checkTimeout is the timeout for the check.
	checkTimeout time.Duration
}

type connectionCheckResult struct {
	// contextErr is set when the check context is done before dialing begins.
	contextErr error
	// dialErr is set when the check could not establish a connection.
	dialErr error
	// closeErr is set when the check connected but failed to close cleanly.
	closeErr error
}

// Run implements the entrypoint for check execution.
func (ncc *Checker) Run(ctx context.Context, cancel context.CancelFunc, client *kubernetes.Clientset) error {
	// Log the start of the check.
	log.Infoln("Running network connection checker")

	// Store the client for parity with v2 behavior.
	ncc.client = client

	result := ncc.doChecks(ctx)
	cancel()
	checkPassed, message := ncc.evaluateResult(result)
	if checkPassed {
		return reportSuccess()
	}
	return reportFailure(message)
}

func (ncc *Checker) evaluateResult(result connectionCheckResult) (bool, string) {
	if result.contextErr != nil {
		return false, "Network connection check did not run before the check context ended: " + result.contextErr.Error()
	}

	if ncc.targetUnreachable {
		if result.dialErr != nil {
			return true, ""
		}
		if result.closeErr != nil {
			return false, "Network connection check determined that " + ncc.connectionTarget + " is UP but the connection did not close cleanly: " + result.closeErr.Error()
		}
		return false, "Network connection check determined that " + ncc.connectionTarget + " is UP but expected it to be unreachable"
	}

	if result.dialErr != nil {
		return false, result.dialErr.Error()
	}
	if result.closeErr != nil {
		return false, result.closeErr.Error()
	}
	return true, ""
}

// doChecks validates the network connection call to the endpoint.
func (ncc *Checker) doChecks(ctx context.Context) connectionCheckResult {
	if err := ctx.Err(); err != nil {
		return connectionCheckResult{contextErr: err}
	}

	// Split the network and address for dialing.
	network, address := splitAddress(ncc.connectionTarget)

	// Build a local address for the dialer.
	var localAddr net.Addr
	if network == "udp" {
		localAddr = &net.UDPAddr{IP: net.ParseIP(ncc.connectionTarget)}
	}
	if network != "udp" {
		localAddr = &net.TCPAddr{IP: net.ParseIP(ncc.connectionTarget)}
	}

	// Dial the target with a timeout.
	dialer := net.Dialer{LocalAddr: localAddr, Timeout: ncc.checkTimeout}
	conn, err := dialer.DialContext(ctx, network, address)
	if err != nil {
		errorMessage := "Network connection check determined that " + ncc.connectionTarget + " is DOWN: " + err.Error()
		log.Errorln(errorMessage)
		return connectionCheckResult{dialErr: errors.New(errorMessage)}
	}

	// Close the connection.
	err = conn.Close()
	if err != nil {
		return connectionCheckResult{closeErr: errors.New(err.Error())}
	}

	return connectionCheckResult{}
}

// splitAddress splits a network address into transport protocol and host:port.
func splitAddress(fullAddress string) (network string, address string) {
	// Split the address on the scheme separator.
	split := strings.SplitN(fullAddress, "://", 2)
	if len(split) == 2 {
		return split[0], split[1]
	}

	// Default to TCP when no scheme is provided.
	return "tcp", fullAddress
}
