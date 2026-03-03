package system

import (
	"fmt"
	"os/exec"
)

// RestartSingbox restarts the sing-box service using OpenRC
func RestartSingbox() error {
	cmd := exec.Command("rc-service", "sing-box", "restart")
	return runCommandWithLog("rc-service sing-box restart", cmd)
}

// StartSingbox starts the sing-box service using OpenRC
func StartSingbox() error {
	cmd := exec.Command("rc-service", "sing-box", "start")
	return runCommandWithLog("rc-service sing-box start", cmd)
}

// StopSingbox stops the sing-box service using OpenRC
func StopSingbox() error {
	cmd := exec.Command("rc-service", "sing-box", "stop")
	return runCommandWithLog("rc-service sing-box stop", cmd)
}

// IsSingboxRunning checks if the sing-box service is active using OpenRC
func IsSingboxRunning() bool {
	err := exec.Command("rc-service", "sing-box", "status").Run()
	return err == nil
}

func runCommandWithLog(name string, cmd *exec.Cmd) error {
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("%s failed: %v, output: %s", name, err, string(out))
	}
	return nil
}
