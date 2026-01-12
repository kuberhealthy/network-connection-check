package main

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/kuberhealthy/kuberhealthy/v3/pkg/checkclient"
	log "github.com/sirupsen/logrus"
)

// CheckConfig stores configuration for the network connection check.
type CheckConfig struct {
	// KubeConfigFile is the optional kubeconfig path.
	KubeConfigFile string
	// CheckTimeout is the timeout for the overall check.
	CheckTimeout time.Duration
	// ConnectionTarget is the network target to dial.
	ConnectionTarget string
	// TargetUnreachable reports success on connection failure when true.
	TargetUnreachable bool
	// CheckNamespace stores the current namespace if detected.
	CheckNamespace string
}

const (
	// defaultCheckTimeout is the fallback timeout for the check.
	defaultCheckTimeout = 20 * time.Second
	// timeoutErrorMessage is emitted when the check times out.
	timeoutErrorMessage = "Failed to complete network connection check in time! Timeout was reached."
)

// parseConfig reads environment variables and builds a CheckConfig.
func parseConfig() (*CheckConfig, error) {
	// Start with default timeout.
	checkTimeout := defaultCheckTimeout

	// Override timeout using the Kuberhealthy deadline when available.
	timeDeadline, err := checkclient.GetDeadline()
	if err != nil {
		log.Infoln("There was an issue getting the check deadline:", err.Error())
	}
	checkTimeout = timeDeadline.Sub(time.Now().Add(time.Second * 5))
	log.Infoln("Check time limit set to:", checkTimeout)

	// Load the connection target.
	connectionTarget := os.Getenv("CONNECTION_TARGET")
	if len(connectionTarget) == 0 {
		return nil, fmt.Errorf("CONNECTION_TARGET environment variable has not been set")
	}

	// Parse the target unreachable flag.
	targetUnreachable := false
	unreachableEnv := os.Getenv("CONNECTION_TARGET_UNREACHABLE")
	if len(unreachableEnv) != 0 {
		parsed, parseErr := strconv.ParseBool(unreachableEnv)
		if parseErr != nil {
			return nil, fmt.Errorf("CONNECTION_TARGET_UNREACHABLE could not be parsed: %w", parseErr)
		}
		targetUnreachable = parsed
	}

	// Read the namespace from the service account file when present.
	checkNamespace := readNamespace()
	if len(checkNamespace) != 0 {
		log.Infoln("Found pod namespace:", checkNamespace)
	}

	// Assemble configuration.
	cfg := &CheckConfig{}
	cfg.KubeConfigFile = os.Getenv("KUBECONFIG")
	cfg.CheckTimeout = checkTimeout
	cfg.ConnectionTarget = connectionTarget
	cfg.TargetUnreachable = targetUnreachable
	cfg.CheckNamespace = checkNamespace

	return cfg, nil
}

// readNamespace loads the namespace file when running in cluster.
func readNamespace() string {
	// Read the namespace file from the service account mount.
	data, err := os.ReadFile("/var/run/secrets/kubernetes.io/serviceaccount/namespace")
	if err != nil {
		log.Warnln("Failed to open namespace file:", err.Error())
		return ""
	}

	// Trim whitespace from the file content.
	namespace := strings.TrimSpace(string(data))
	return namespace
}
