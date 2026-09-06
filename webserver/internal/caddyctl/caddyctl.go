// Package caddyctl controls a running Caddy instance from outside its
// process, e.g. after internal/deploy publishes new site content.
package caddyctl

import (
	"fmt"
	"os"
	"os/exec"
)

// Reload tells the running Caddy instance (started elsewhere, e.g. via
// `webserver run` or a system caddy service) to re-read caddyfilePath and
// apply any changes. This is a zero-downtime reload driven by Caddy's admin
// API under the hood; it does not require restarting the process.
func Reload(caddyBinary, caddyfilePath string) error {
	cmd := exec.Command(caddyBinary, "reload", "--config", caddyfilePath, "--adapter", "caddyfile")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("%s reload: %w", caddyBinary, err)
	}
	return nil
}
