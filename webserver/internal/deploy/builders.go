package deploy

import (
	"fmt"

	"webserver/internal/config"
)

// BuildFunc builds the repo checked out at srcDir and writes the published
// site into outDir, which the caller has already mapped to the
// /var/www/<domain>[/subdomains/<label>] path the Caddyfile expects.
type BuildFunc func(srcDir, outDir string, opts config.BuildOptions) error

var builders = map[config.Method]BuildFunc{
	config.MethodMarkdown: buildMarkdown,
	config.MethodMkdocs:   buildMkdocs,
	config.MethodOCaml:    buildOCaml,
	config.MethodOdoc:     buildOdoc,
}

// buildMarkdown renders a plain directory of markdown files to static HTML.
func buildMarkdown(srcDir, outDir string, opts config.BuildOptions) error {
	return fmt.Errorf("build_method %q not implemented yet (srcDir=%s outDir=%s)", config.MethodMarkdown, srcDir, outDir)
}

// buildMkdocs runs `mkdocs build` against srcDir/mkdocs.yml and publishes
// the generated site/ to outDir.
func buildMkdocs(srcDir, outDir string, opts config.BuildOptions) error {
	return fmt.Errorf("build_method %q not implemented yet (srcDir=%s outDir=%s)", config.MethodMkdocs, srcDir, outDir)
}

// buildOCaml runs `dune build` against srcDir.
func buildOCaml(srcDir, outDir string, opts config.BuildOptions) error {
	return fmt.Errorf("build_method %q not implemented yet (srcDir=%s outDir=%s)", config.MethodOCaml, srcDir, outDir)
}

// buildOdoc runs `dune build @doc` and publishes _build/default/_doc/_html
// to outDir.
func buildOdoc(srcDir, outDir string, opts config.BuildOptions) error {
	return fmt.Errorf("build_method %q not implemented yet (srcDir=%s outDir=%s)", config.MethodOdoc, srcDir, outDir)
}
