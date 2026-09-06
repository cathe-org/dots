# Dots Webserver: Architecture Recap + Implementation Curriculum

## Context

A few months ago you sketched a `deploy.yaml` idea in this repo's README, then built out a real Go module (`webserver/`) embedding Caddy with a custom `deploy` subcommand — but the actual site-building logic was left stubbed, some config fields were never wired up, and you'd forgotten the intent behind parts of the design. The underlying problem this project solves: today, hosting a new personal site means either hand-configuring GitHub Actions + repo secrets for that specific repo, or transferring ownership to a GitHub org — both a hassle you want to eliminate. The goal is a single `dots` repo that is the *only* place with real CI: it declares (via `deploy.yaml`) which of your repos to pull, how to build each into a site, and which subdomain/domain each lands on, while the webserver itself (not GitHub Actions) does the actual cloning/building/publishing.

You explicitly want to implement the actual code yourself, using this document as a curriculum, with me as architect/teacher rather than author. This plan is a **decision record + ordered curriculum**, not a code-change plan.

**Session 2 update:** since the first planning pass, you've filled in `config.go` with real intent comments and a genuinely new `DomainLink` design (replacing the old placeholder `linked: []string`), and dropped three real GitHub Actions workflows + a helper script into `_example_yamls/` — these are the actual pipelines you're hoping to subsume, so they're now the ground truth for Modules on builders. One resolved mystery worth noting: `_example_yamls/cloakaml-deploy-docs.yml` has an abandoned, commented-out "Fix links" step that dies mid-`sed` command — a hand-rolled attempt to fix cross-references in odoc output after moving directories around. That's almost certainly what the original forgotten `linked` field existed to solve, and your new `DomainLink{LinkTo, LinkPlaceholder, Path}` struct is the proper version of that same fix.

## Current State (as of Session 2)

- **`flake.nix`**: a local dev shell only (go, gopls, caddy, xcaddy, git on nixpkgs-unstable). No droplet/NixOS/home-manager config yet, and no mkdocs/Python/OCaml toolchain yet either.
- **`deploy.yaml`** (untracked): defines `repos[]`, each with `url`/`branch`/`build_method` (`md`/`mkdocs`/`ocaml`/`odoc`) and `domain[]` (`name`/`indexed`/`linked`), plus top-level `auto_rebuild`, `email_notifications`, `flags`.
- **`webserver/internal/config/config.go`** — now has real doc comments (your own). Current schema:
  - `Domain{Name, Indexed, Linked []DomainLink}` — `Indexed` = "shown on domain index page"; `Linked` is now `[]DomainLink{LinkTo, LinkPlaceholder, Path []string}` — a build-time find/replace mechanism (find `LinkPlaceholder` in built output, replace with a real URL built from `LinkTo` + `Path`).
  - `Method` constants: `md` (plain markdown), `mkdocs`, `odoc` (OCaml odoc → HTML docs), `ocaml` ("convert `.ml` files to `.md`, then use `MethodMarkdown`" — **superseded by Session 2 decision below**: it now chains into `mkdocs`, not raw markdown; the code comment needs updating to match).
  - `AutoRebuild{Enable,Interval}`, `EmailNotifications{Enable,Warnings,General}`, `Flags{Clean,Reset}` — still no intent comments, still unconsumed anywhere in the Go code. Still open (Module 1).
- **`webserver/internal/deploy/`**:
  - `git.go` — **implemented**: clone-or-(fetch+hard-reset) per repo.
  - `builders.go` — **all four build methods are stubs** returning "not implemented" errors.
  - `deploy.go` — **implemented**: orchestrates sync → build → publish into `/var/www/<domain>/subdomains/<name>`, then reloads Caddy. Apex vs. one-level-subdomain routing already works. No `DomainLink` rewriting logic yet.
  - `caddyctl.go` — **implemented**: `caddy reload`.
- **`Caddyfile`** (tracked, hand-maintained): per-domain blocks for `cathe.dev`, `cathe.work`, `jpears.me`; `*.cathe.dev` auto-detects odoc vs. mkdocs vs. plain output by probing the filesystem at request time.
- **`.github/workflows/deploy-dots.yml`** (tracked): on push to `main`, SSHes to the droplet and runs `git pull` only — no rebuild trigger yet.
- **`_example_yamls/`** (untracked) — the real workflows you're replacing:
  - `cloakaml-deploy-docs.yml`: OCaml project → `dune build @doc` → odoc HTML, SCP'd to the server. Maps to `odoc`.
  - `demos-deploy-docs-md.yml`: plain `docs/` folder → `mkdocs build` (theme: shadcn) → SCP'd, with a subdomain derived by stripping a `-docs` suffix off the repo name, and an explicit safety guard (`if [ -z "$SITE" ] || [ -z "$SUB" ]; then exit 1; fi` before `rm -rf "$TARGET"`). Maps to `mkdocs`.
  - `demos-deploy-docs.yml`: `.ml` files → `generate_docs.py` → generated `.md` → `mkdocs build` (theme: material) → SCP'd. Maps to `ocaml` (which per Session 2's decision, chains into `mkdocs`).
  - `generate_docs.py`: a working, self-contained Python script (regex-based) that parses OCaml doc comments (`(** ... *)`, odoc heading/label/bold/italic/inline-code syntax) into Markdown. This is the actual logic behind the `ocaml` method's first stage.

## Decisions

1. **`Linked`/`DomainLink`**: your own design (find `LinkPlaceholder` in built HTML, replace with a URL derived from `LinkTo`+`Path`) is the resolution — no further guessing needed. Implementation (where/how the substitution runs) is its own module (new Module 6 below).
2. **Droplet Nix strategy** → phased: near-term, Nix + home-manager on the *current* Ubuntu droplet to pin tool versions (Caddy, mkdocs, OCaml toolchain, Python) — a real but partial win, since home-manager on non-NixOS can't natively own privileged system services, so systemd units stay hand-maintained pointing at Nix-store paths. Later, full NixOS for true "fully managed" (declarative `services.caddy`, `systemd.timers`, atomic rollback) — via a **blue/green swap** (new droplet, validate, cut DNS, decommission old), not an in-place `nixos-infect` run against the live box.
3. **Site builds stay in Go (`builders.go`), not Nix derivations.** Nix's job is "what tools exist on the box" (mirrors `_example_yamls`' setup steps — `ocaml/setup-ocaml`, `actions/setup-python` — becoming Module 9/10's Nix provisioning, not something re-run on every deploy). Nix builds are sandboxed with no network access except for fixed-output derivations needing a pre-known hash, which conflicts with `syncRepo()`'s floating-branch model. `builders.go` shelling out to `mkdocs build`/`dune build @doc`/`generate_docs.py` — exactly what the example workflows already do — is the right layer.
4. **Rebuild trigger chain**: extend `deploy-dots.yml` to run `webserver deploy` after `git pull` (catches dots-repo changes); add a systemd timer implementing `auto_rebuild.interval` (catches content-repo pushes, which have no GitHub Actions of their own); a webhook listener is a later addition.
5. **Domain scheme** (yours to finalize): `cathe.dev` = dev/technical hub, `cathe.work` = professional/CV-facing, `jpears.me` = personal. Nothing in the code blocks this.
6. **Caddyfile stays hand-maintained for now** — adding a subdomain under an existing domain needs zero Caddyfile edits already; generating it from `deploy.yaml` is a later, optional correctness improvement (removes a stale-directory failure mode, gives `Indexed` something to drive).
7. **GitHub Pages sites**: plan to migrate — content into a droplet subdomain via the proven pipeline, old repo becomes a redirect stub. Sequenced after the core pipeline is proven.
8. **`ocaml` build method chains into `mkdocs`, not plain `md`** (Session 2) — matches what `demos-deploy-docs.yml` actually does today (generate `.md`, then `mkdocs build` with the material theme), overriding the older `config.go` comment. Update that comment when you touch this method.
9. **`generate_docs.py` stays Python for now, shelled out to from the `ocaml` builder** (Session 2) — treated exactly like `mkdocs`/`dune` are: a pinned external tool invoked via `os/exec`, not ported. Porting its OCaml-doc-comment parser to native Go is deliberately deferred to its own optional later module, so you can get the pipeline working end-to-end first.
10. **Curriculum depth**: detailed — explain underlying concepts as we go, not just task lists.

## Porting Guide: GitHub Actions Step → Webserver Equivalent

A general pattern, valid for all three example workflows, worth internalizing before Modules 2–5:

| GH Actions step | Webserver equivalent |
|---|---|
| `actions/checkout@v4` | `git.go`'s `syncRepo()` — already implemented, runs once per repo per `webserver deploy` invocation |
| Toolchain setup (`ocaml/setup-ocaml`, `actions/setup-python`) | **Not per-deploy.** Becomes Module 9/10's job — Nix provisions the toolchain once on the droplet; `builders.go` assumes `dune`/`mkdocs`/`python3` are already on `$PATH` |
| Project-level dependency install (`opam install . --deps-only`, `pip install mkdocs-material`) | Stays inside the relevant `BuildFunc` in `builders.go`, run each deploy via `os/exec` — this genuinely varies per repo/commit, unlike the toolchain itself, so it can't be pushed onto Nix the same way |
| Build command (`dune build @doc`, `mkdocs build`, `python generate_docs.py`) | The core of each `BuildFunc` — same command, run via `os/exec.Command` with `Dir` set to the cloned repo path |
| "Create subdomain directory" + the empty-var safety guard before `rm -rf` | Port this guard explicitly into `deploy.go`'s publish step — never `rm -rf` a computed path without first asserting it's non-empty and under `var_www` |
| SCP/copy step to the server | Becomes a local file copy (`os.CopyFS` or manual walk) from the build's output dir straight into `outputDir()` — no network hop needed since build and publish now happen on the same box |
| Repo-name subdomain derivation (e.g. stripping a `-docs` suffix, lowercasing) | **Eliminated.** `deploy.yaml`'s `domain[].name` states the subdomain explicitly — this is exactly the kind of hand-rolled logic the declarative config is meant to replace |

## How We'll Work From Here

You implement; I architect, explain concepts, and review. Work through modules roughly in order (each builds on the last, except ones marked optional). Bring a module up when you're ready to start it and we'll go deep on the concepts while you write the code.

## Curriculum

**Module 1 — Finish the `deploy.yaml`/`config.go` schema.**
Mostly done — `Linked`→`DomainLink` and `Indexed` now have real intent. Remaining: decide implement/repurpose/delete for `AutoRebuild.*`, `EmailNotifications.*`, `Flags.*`, per-repo `Build.AutoRebuild`; update the `ocaml` doc comment to say "chain into mkdocs" (Decision 8); add `config.Load` validation for the new `DomainLink` shape (e.g. `LinkTo` must reference a domain that actually exists somewhere in `deploy.yaml`).
Files: `config.go`, `deploy.yaml`.
Done when: every field is consumed by code or explicitly documented as reserved; `DomainLink.LinkTo` references are validated at load time.

**Module 2 — Implement the `md` build method.**
Goal: render a directory of `.md` files to HTML, copy static assets, write to `outDir`. No direct `_example_yamls` reference (none of your three workflows use raw markdown-without-mkdocs) — it's still the simplest possible builder and a good first exercise, and is what a very lightweight future repo could use.
Files: `builders.go`, `go.mod` (a markdown lib, e.g. goldmark).
Done when: `webserver deploy` against a scratch markdown repo produces servable HTML.

**Module 3 — Implement `mkdocs`.**
Reference: `_example_yamls/demos-deploy-docs-md.yml` — port its `mkdocs build` step, its "ensure `mkdocs.yml` exists, else generate a minimal one" step (via `BuildOptions.Theme`), and critically its **empty-var safety guard** before touching the target directory.
Files: `builders.go`, `flake.nix` devShell (add mkdocs/mkdocs-material/mkdocs-shadcn/Python — currently missing).
Concepts: `os/exec` done properly (stderr capture, exit codes), generating/overriding `mkdocs.yml` from config, the porting guide's safety-guard row.
Done when: deploy against a real repo produces a `site/` dir Caddy's `@has_site` tier serves.

**Module 4 — Implement `odoc`.**
Reference: `_example_yamls/cloakaml-deploy-docs.yml` — `dune build @doc`, publish `_build/default/_doc/_html`. Ignore its commented-out "Extract html dir"/"Fix links" steps — that problem is now Module 6's job, done properly.
Files: `builders.go`, `config.go` (subdir option for non-root dune projects), `flake.nix` devShell (ocaml/dune/odoc).
Done when: deploy against `cloakaml-docs` serves correctly through the existing `@has_odoc` tier.

**Module 5 — Implement `ocaml` (chains into `mkdocs`).**
Reference: `_example_yamls/demos-deploy-docs.yml` + `generate_docs.py`. Per Decisions 8–9: shell out to `generate_docs.py` (via `os/exec`, Nix-pinned Python) to produce `.md` from `.ml` sources, then hand off directly into Module 3's `mkdocs` `BuildFunc` rather than duplicating build logic.
Files: `builders.go`, `flake.nix` devShell (python3), `_example_yamls/generate_docs.py` copied into the repo proper (e.g. `webserver/scripts/generate_docs.py`) so it ships with the deploy tool instead of living in a throwaway examples folder.
Done when: deploy against a repo with `.ml` sources produces the same kind of `site/` output as Module 3, via the ocaml pre-processing step.

**Module 6 — Implement `DomainLink` rewriting.**
Goal: after a build (any method) but before/during publish, walk the output files and replace each configured `LinkPlaceholder` with the real URL built from `LinkTo` + `Path`. This is the properly-implemented version of the abandoned `sed`-based "Fix links" step in `cloakaml-deploy-docs.yml`.
Files: `deploy.go` (or a new `internal/deploy/links.go`).
Concepts: safe text-file walking (skip binary assets), deciding whether replacement happens in-place on published files or on a working copy pre-publish.
Done when: a placeholder string in a build's output is correctly replaced with a real cross-domain link after `webserver deploy` runs.

**Module 7 — Wire the GitHub Action to trigger a real rebuild.**
Goal: extend `deploy-dots.yml`'s SSH script to run `webserver deploy --config deploy.yaml` after `git pull`, with real exit-code propagation.
Files: `.github/workflows/deploy-dots.yml`.
Concepts: multi-line `appleboy/ssh-action` scripts; decide how the compiled `webserver` binary gets/stays current on the droplet (rebuild-on-server vs. ship a prebuilt artifact).
Done when: a `deploy.yaml` change pushed to main visibly updates the live site via the Action run.

**Module 8 — systemd timer for `auto_rebuild.interval`.**
Goal: the independent scheduled path catching content-repo pushes with zero GitHub Actions of their own.
Files: new `webserver-deploy.service`/`.timer` units.
Concepts: systemd timers vs. cron, `OnCalendar` syntax, `journalctl`.
Done when: `systemctl list-timers` shows it scheduled; a commit pushed only to a content repo shows up live within one interval.

**Module 9 — Nix + home-manager on the current Ubuntu droplet.**
Goal: install Nix (Determinate installer), add a `homeConfigurations` output pinning Caddy/mkdocs/OCaml toolchain/Python/the `webserver` binary; point Modules 7/8's systemd units at the resulting store paths.
Files: `flake.nix`, new `home.nix`.
Concepts: standalone (non-NixOS) home-manager, why it can't natively own privileged/boot-time services (`loginctl enable-linger`, `setcap`).
Done when: `home-manager switch` supplies every pinned tool from the flake; existing systemd units keep working against the new store paths.

**Module 10 — Full NixOS migration via blue/green droplet swap.**
Goal: the "fully managed" end state. New droplet on NixOS (community image or `nixos-infect` on a fresh, disposable box); `nixosConfigurations` with `services.caddy`, `systemd.timers` replacing Module 8's hand-authored units, firewall/users declared; deploy via `nixos-rebuild --target-host` or `colmena`; validate; cut DNS; decommission old box.
Files: `flake.nix` (`nixosConfigurations`), `hosts/<hostname>/configuration.nix` + `hardware-configuration.nix`.
Done when: the new droplet serves all domains/subdomains correctly; a config change applies via `nixos-rebuild switch --target-host`; a broken config rolls back via generation.

**Module 11 (optional) — Webhook listener for instant rebuilds.**
Goal: complement Module 8's daily poll with near-real-time triggers via a plain GitHub repo webhook (not a workflow file) per content repo.
Files: new `internal/webhook` package + `webserver serve-hooks` subcommand, systemd unit, Caddy `reverse_proxy` route.
Done when: pushing to `notes` triggers a rebuild within seconds, no GH Actions YAML added there.

**Module 12 (optional) — Generate the Caddyfile from `deploy.yaml`.**
Goal: replace filesystem-probing tiers with explicit per-subdomain blocks chosen by `build_method`, generated before each `Reload()` — removes the stale-directory failure mode and gives `Indexed` something concrete to drive (e.g. a `noindex` header).
Files: new `internal/caddygen`, `deploy.go`.
Done when: adding a domain/subdomain purely via `deploy.yaml` produces a correct Caddy block with no manual edit, validated with `caddy validate`.

**Module 13 — GitHub Pages consolidation.**
Goal: migrate one `username.github.io` site into a droplet subdomain via the now-proven pipeline; turn the old repo into a redirect stub.
Files: `deploy.yaml` (new entry), the external `github.io` repo, droplet `~/.ssh/config` for the second GitHub identity's deploy key.
Done when: the old `github.io` URL redirects cleanly; the new subdomain serves the migrated content through the standard pipeline.

**Module 14 (optional, low priority) — Port `generate_docs.py` to native Go.**
Goal: per Decision 9, replace the shelled-out Python script with a native Go implementation of the same OCaml-doc-comment-to-Markdown parsing (doc comments, odoc headings/labels/bold/italic/inline-code). A genuine learning exercise in Go string/regex handling — not required for the pipeline to work.
Files: new `webserver/internal/deploy/ocamldoc.go` (or similar), replacing the `os/exec` call added in Module 5.
Done when: output byte-for-byte (or close) matches what the Python script currently produces, and the Python dependency is dropped from the devShell.

## Verification

No code changes were made this session. To confirm this update captures things correctly: check the `ocaml`→`mkdocs` decision and the Module 5/6/14 split against your own mental model before starting — these are new since the last pass and worth a sanity check before you're deep into Module 3.
