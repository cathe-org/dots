// Package config loads this dots repo's deploy.yaml: the list of site repos
// to sync, how to build each one, and where its output gets published.
package config

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

// Method names a build pipeline. It is a plain string (rather than a closed
// enum) so new methods can be added in YAML; internal/deploy still needs a
// matching builder registered in Go before a method actually works.
type Method string

const (
	// Markdown files to static-site HTML
	MethodMarkdown Method = "md"
	// OCaml odoc to generate static-site HTML docs
	MethodOdoc Method = "odoc"
	// Markdown documentation to static-site HTML
	MethodMkdocs Method = "mkdocs"
	// Convert *.ml files to *.md and use MethodMarkdown
	// (useful if actual code is to be displayed)
	MethodOCaml Method = "ocaml"
)

// Config is the root of deploy.yaml.
type Config struct {
	// CaddyBinary is the caddy executable used to reload the running
	// instance after a deploy. Defaults to "caddy" (resolved via PATH).
	CaddyBinary string `yaml:"caddy_binary"`
	// Caddyfile is the config Caddy was (or will be) started with.
	Caddyfile string `yaml:"caddyfile"`
	// VarWWW is the root directory static output gets published under,
	// matching the layout the Caddyfile expects: <var_www>/<domain>/subdomains/<label>/...
	VarWWW string `yaml:"var_www"`
	// WorkDir is where repos get cloned/pulled before building.
	WorkDir string `yaml:"work_dir"`

	AutoRebuild        AutoRebuildConfig        `yaml:"auto_rebuild"`
	EmailNotifications EmailNotificationsConfig `yaml:"email_notifications"`
	Flags              FlagsConfig              `yaml:"flags"`

	Repos []Repo `yaml:"repos"`
}

type AutoRebuildConfig struct {
	Enable   bool   `yaml:"enable"`
	Interval string `yaml:"interval"`
}

type EmailNotificationsConfig struct {
	Enable   bool `yaml:"enable"`
	Warnings bool `yaml:"warnings"`
	General  bool `yaml:"general"`
}

type FlagsConfig struct {
	Clean bool `yaml:"clean"`
	Reset bool `yaml:"reset"`
}

// Repo describes one site source and how to publish it.
type Repo struct {
	Name        string       `yaml:"name"`
	URL         string       `yaml:"url"`
	Branch      string       `yaml:"branch"`
	BuildMethod Method       `yaml:"build_method"`
	Domains     []Domain     `yaml:"domain"`
	Build       BuildOptions `yaml:"config"`
}

// Domain is one hostname a repo's build output is published under, e.g.
// "notes.cathe.dev". The apex ("cathe.dev") and subdomain label ("notes")
// are derived from it when computing the /var/www output path.
type Domain struct {
	Name string `yaml:"name"`
	// If subdomain is shown on domain index page
	Indexed bool `yaml:"indexed"`
	// List of Domain Links
	Linked []DomainLink `yaml:"linked"`
}

// Inter-domain links
type DomainLink struct {
	// Name of domain to link to
	LinkTo string `yaml:"link_to"`
	// Placeholer in site to replace with real link
	LinkPlaceholder string `yaml:"link_placeholder"`
	// Path under linked domain
	Path []string `yaml:"path"`
}

// BuildOptions are passed to whichever builder BuildMethod selects; not
// every field applies to every method.
type BuildOptions struct {
	AutoRebuild bool        `yaml:"auto_rebuild"`
	InputDir    string      `yaml:"input_dir"`
	Theme       string      `yaml:"theme"`
	Colors      ColorScheme `yaml:"colors"`
}

type ColorScheme struct {
	Accent []string `yaml:"accent"`
	FG     []string `yaml:"fg"`
	BG     []string `yaml:"bg"`
}

// Load reads and validates deploy.yaml at path, filling in defaults.
func Load(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("reading %s: %w", path, err)
	}

	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("parsing %s: %w", path, err)
	}

	if cfg.CaddyBinary == "" {
		cfg.CaddyBinary = "caddy"
	}
	if cfg.Caddyfile == "" {
		cfg.Caddyfile = "Caddyfile"
	}
	if cfg.VarWWW == "" {
		cfg.VarWWW = "/var/www"
	}
	if cfg.WorkDir == "" {
		cfg.WorkDir = "work"
	}

	for i, repo := range cfg.Repos {
		if repo.URL == "" {
			return nil, fmt.Errorf("repos[%d]: url is required", i)
		}
		if repo.Branch == "" {
			cfg.Repos[i].Branch = "main"
		}
		if repo.BuildMethod == "" {
			return nil, fmt.Errorf("repos[%d] (%s): build_method is required", i, repo.URL)
		}
	}

	return &cfg, nil
}
