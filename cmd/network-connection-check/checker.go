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

// Run implements the entrypoint for check execution.
func (ncc *Checker) Run(ctx context.Context, cancel context.CancelFunc, client *kubernetes.Clientset) error {
	// Log the start of the check.
	log.Infoln("Running network connection checker")

	// Prepare a completion channel and timeout.
	doneChan := make(chan error)
	runTimeout := time.After(ncc.checkTimeout)

	// Store the client for parity with v2 behavior.
	ncc.client = client

	// Run the check in the background.
	go ncc.runChecksAsync(doneChan)

	// Wait for timeout or completion.
	select {
	case <-ctx.Done():
		log.Infoln("Cancelling check and shutting down due to interrupt.")
		return reportFailure("Cancelling check and shutting down due to interrupt.")
	case <-runTimeout:
		cancel()
		log.Infoln("Cancelling check and shutting down due to timeout.")
		return reportFailure(timeoutErrorMessage)
	case err := <-doneChan:
		cancel()
		if err != nil && !ncc.targetUnreachable {
			return reportFailure(err.Error())
		}
		return reportSuccess()
	}
}

// runChecksAsync runs the connection check and sends the result on the channel.
func (ncc *Checker) runChecksAsync(doneChan chan error) {
	// Execute the checks and forward results.
	err := ncc.doChecks()
	doneChan <- err
}

// doChecks validates the network connection call to the endpoint.
func (ncc *Checker) doChecks() error {
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
	conn, err := dialer.Dial(network, address)
	if err != nil {
		errorMessage := "Network connection check determined that " + ncc.connectionTarget + " is DOWN: " + err.Error()
		log.Errorln(errorMessage)
		return errors.New(errorMessage)
	}

	// Close the connection.
	err = conn.Close()
	if err != nil {
		return errors.New(err.Error())
	}

	return nil
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
