package main

import (
	"os"

	"github.com/kuberhealthy/kuberhealthy/v3/pkg/checkclient"
	log "github.com/sirupsen/logrus"
)

// reportSuccess reports success to Kuberhealthy.
func reportSuccess() error {
	// Report success to Kuberhealthy.
	err := checkclient.ReportSuccess()
	if err != nil {
		log.Infoln("Error reporting success to Kuberhealthy servers:", err)
		return err
	}

	log.Infoln("Successfully reported success to Kuberhealthy servers")
	return nil
}

// reportFailure reports failure to Kuberhealthy.
func reportFailure(errorMessage string) error {
	// Report failure to Kuberhealthy.
	err := checkclient.ReportFailure([]string{errorMessage})
	if err != nil {
		log.Infoln("Error reporting failure to Kuberhealthy servers:", err)
		return err
	}

	log.Infoln("Successfully reported failure to Kuberhealthy servers")
	return nil
}

// reportFailureAndExit reports failure and exits the process.
func reportFailureAndExit(err error) {
	// Log the error locally.
	log.Errorln(err)

	// Report the failure to Kuberhealthy.
	reportErr := checkclient.ReportFailure([]string{err.Error()})
	if reportErr != nil {
		log.Fatalln("error when reporting to kuberhealthy:", reportErr.Error())
	}

	// Exit after reporting failure.
	os.Exit(0)
}
