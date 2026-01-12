package main

import (
	"context"

	log "github.com/sirupsen/logrus"

	nodecheck "github.com/kuberhealthy/kuberhealthy/v3/pkg/nodecheck"

	// Required for oidc kubectl testing.
	_ "k8s.io/client-go/plugin/pkg/client/auth/oidc"
)

// main loads configuration and executes the network connection check.
func main() {
	// Enable nodecheck debug output for parity with v2 behavior.
	nodecheck.EnableDebugOutput()

	// Parse configuration from environment variables.
	cfg, err := parseConfig()
	if err != nil {
		reportFailureAndExit(err)
		return
	}

	// Create a Kubernetes client.
	client, err := createKubeClient(cfg.KubeConfigFile)
	if err != nil {
		log.Fatalln("Unable to create kubernetes client", err)
	}

	// Create the checker.
	checker := NewChecker(cfg)

	// Build a timeout context for the check.
	ctx, cancelFunc := context.WithTimeout(context.Background(), cfg.CheckTimeout)

	// Wait for the node to join the worker pool.
	waitForNodeToJoin(ctx)

	// Run the check and report results.
	err = checker.Run(ctx, cancelFunc, client)
	if err != nil {
		log.Errorln("Error running network connection check for:", cfg.ConnectionTarget)
	}
	log.Infoln("Done running network connection check for:", cfg.ConnectionTarget)
}

// NewChecker returns a new network connection checker.
func NewChecker(cfg *CheckConfig) *Checker {
	// Build a checker instance from configuration.
	return &Checker{
		connectionTarget:  cfg.ConnectionTarget,
		targetUnreachable: cfg.TargetUnreachable,
		checkTimeout:      cfg.CheckTimeout,
	}
}

// waitForNodeToJoin waits for the node to join the worker pool.
func waitForNodeToJoin(ctx context.Context) {
	// Check if Kuberhealthy is reachable.
	err := nodecheck.WaitForKuberhealthy(ctx)
	if err != nil {
		log.Errorln("Failed to reach Kuberhealthy:", err.Error())
	}
}
