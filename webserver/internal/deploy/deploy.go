// Package deploy syncs the repos listed in deploy.yaml, builds each with
// its declared method, publishes the output where the Caddyfile expects it,
// and reloads Caddy.
package deploy

import (
	"errors"
	"fmt"
	"log"
	"path/filepath"
	"strings"

	"webserver/internal/caddyctl"
	"webserver/internal/config"
)

// Run loads configPath, syncs and builds every configured repo, then
// reloads Caddy so the results are served. A single repo failing to build
// is logged and skipped rather than aborting the whole run; Caddy is still
// reloaded afterward so unaffected repos' updates go live.
func Run(configPath string) error {
	cfg, err := config.Load(configPath)
	if err != nil {
		return err
	}

	var failed []string
	for _, repo := range cfg.Repos {
		if err := deployRepo(cfg, repo); err != nil {
			log.Printf("deploy %s: %v", repo.URL, err)
			failed = append(failed, repo.URL)
		}
	}

	if err := caddyctl.Reload(cfg.CaddyBinary, cfg.Caddyfile); err != nil {
		return fmt.Errorf("reloading caddy: %w", err)
	}

	if len(failed) > 0 {
		return fmt.Errorf("%d repo(s) failed to deploy: %s", len(failed), strings.Join(failed, ", "))
	}
	return nil
}

func deployRepo(cfg *config.Config, repo config.Repo) error {
	name := repo.Name
	if name == "" {
		name = repoDirName(repo.URL)
	}
	srcDir := filepath.Join(cfg.WorkDir, name)

	if err := syncRepo(repo.URL, repo.Branch, srcDir); err != nil {
		return fmt.Errorf("sync: %w", err)
	}

	build, ok := builders[repo.BuildMethod]
	if !ok {
		return fmt.Errorf("no builder registered for build_method %q", repo.BuildMethod)
	}

	if len(repo.Domains) == 0 {
		return errors.New("no domains configured to publish to")
	}

	for _, d := range repo.Domains {
		outDir, err := outputDir(cfg.VarWWW, d.Name)
		if err != nil {
			return fmt.Errorf("domain %q: %w", d.Name, err)
		}
		if err := build(srcDir, outDir, repo.Build); err != nil {
			return fmt.Errorf("build (%s) for %s: %w", repo.BuildMethod, d.Name, err)
		}
	}
	return nil
}

// outputDir maps a hostname to the directory the existing Caddyfile serves
// it from: an apex domain like "cathe.dev" publishes to
// <varWWW>/cathe.dev, and a single-level subdomain like "notes.cathe.dev"
// (matched by the Caddyfile's "*.cathe.dev" block and its {labels.2}
// rewrite) publishes to <varWWW>/cathe.dev/subdomains/notes. Anything else
// can't be routed by that Caddyfile shape, so it's rejected here rather
// than silently writing files nothing will serve.
func outputDir(varWWW, host string) (string, error) {
	labels := strings.Split(host, ".")
	switch len(labels) {
	case 2:
		return filepath.Join(varWWW, host), nil
	case 3:
		apex := labels[1] + "." + labels[2]
		sub := labels[0]
		return filepath.Join(varWWW, apex, "subdomains", sub), nil
	default:
		return "", fmt.Errorf("hostname %q must be an apex domain (e.g. cathe.dev) or a single-level subdomain (e.g. notes.cathe.dev)", host)
	}
}

func repoDirName(url string) string {
	url = strings.TrimSuffix(url, "/")
	url = strings.TrimSuffix(url, ".git")
	parts := strings.Split(url, "/")
	return parts[len(parts)-1]
}
