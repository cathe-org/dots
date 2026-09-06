// Command webserver embeds Caddy v2 (the same way xcaddy-built binaries do)
// and adds a "deploy" subcommand for this dots repo's build-and-publish
// workflow. Run it exactly like the caddy CLI, e.g.:
//
//	webserver run --config Caddyfile --adapter caddyfile
//	webserver deploy --config deploy.yaml
package main

import (
	"fmt"

	"github.com/caddyserver/caddy/v2"
	caddycmd "github.com/caddyserver/caddy/v2/cmd"
	"github.com/spf13/cobra"

	// Standard Caddy modules (file_server, tls, reverse_proxy, etc).
	// Add more blank imports here as this binary needs more Caddy modules.
	_ "github.com/caddyserver/caddy/v2/modules/standard"

	// The Caddyfile's `acme_dns cloudflare` directive needs this: it's not
	// part of the standard module set, so plain `caddy` (including the
	// nixpkgs build) can't adapt/run this Caddyfile without it.
	_ "github.com/caddy-dns/cloudflare"

	"webserver/internal/deploy"
)

func init() {
	caddycmd.RegisterCommand(caddycmd.Command{
		Name:  "deploy",
		Usage: "[--config <path>]",
		Short: "Sync configured repos, build their sites, and reload Caddy",
		Long: `
Deploy reads the dots deploy config (deploy.yaml by default), clones or pulls
each configured repo, builds it using the method it declares (e.g. md,
mkdocs, ocaml/odoc), writes the result into the matching /var/www subdomain
directory, and reloads the running Caddy instance so the new content is
served without downtime.
`,
		CobraFunc: func(cmd *cobra.Command) {
			cmd.Flags().StringP("config", "c", "deploy.yaml", "Path to the deploy config YAML")
			cmd.RunE = caddycmd.WrapCommandFuncForCobra(cmdDeploy)
		},
	})
}

func cmdDeploy(fl caddycmd.Flags) (int, error) {
	configPath := fl.String("config")

	if err := deploy.Run(configPath); err != nil {
		return caddy.ExitCodeFailedStartup, fmt.Errorf("deploy: %w", err)
	}
	return caddy.ExitCodeSuccess, nil
}

func main() {
	caddycmd.Main()
}
