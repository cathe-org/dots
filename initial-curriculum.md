# Dots Webserver: Architecture Recap + Implementation Curriculum

## Context

A few months ago you sketched a `deploy.yaml` idea in this repo's README, then built out a real Go module (`webserver/`) embedding Caddy with a custom `deploy` subcommand — but the actual site-building logic was left stubbed, some config fields were never wired up, and you've since forgotten the intent behind parts of the design. The underlying problem this project solves: today, hosting a new personal site means either hand-configuring GitHub Actions + repo secrets for that specific repo, or transferring ownership to a GitHub org — both a hassle you want to eliminate. The goal is a single `dots` repo that is the *only* place with real CI: it declares (via `deploy.yaml`) which of your repos to pull, how to build each into a site, and which subdomain/domain each lands on, while the webserver itself (not GitHub Actions) does the actual cloning/building/publishing.

This session was pure discussion and planning — no code was written. You explicitly want to implement the actual code yourself, using this document as a curriculum, with me as architect/teacher rather than author. This plan is therefore a **decision record + ordered curriculum**, not a code-change plan.

## Current State (as of this session)

- **`flake.nix`**: a local dev shell only (go, gopls, caddy, xcaddy, git on nixpkgs-unstable). No droplet/NixOS/home-manager config yet.
- **`deploy.yaml`** (untracked): defines `repos[]`, each with `url`/`branch`/`build_method` (`md`/`mkdocs`/`ocaml`/`odoc`) and `domain[]` (`name`/`indexed`/`linked`), plus top-level `auto_rebuild`, `email_notifications`, `flags`.
- **`webserver/`** (untracked Go module): a Cobra CLI embedding Caddy v2 + a `deploy` subcommand.
  - `internal/config/config.go` — the schema above.
  - `internal/deploy/git.go` — **implemented**: clone-or-(fetch+hard-reset) per repo.
  - `internal/deploy/builders.go` — **all four build methods (`md`, `mkdocs`, `ocaml`, `odoc`) are stubs** that just return "not implemented" errors.
  - `internal/deploy/deploy.go` — **implemented**: orchestrates sync → build → publish into `/var/www/<domain>/subdomains/<name>`, then reloads Caddy. Apex vs. one-level-subdomain routing already works.
  - `internal/caddyctl/caddyctl.go` — **implemented**: `caddy reload` for zero-downtime reload.
- **`Caddyfile`** (tracked, hand-maintained): per-domain blocks for `cathe.dev`, `cathe.work`, `jpears.me`, each with a wildcard `*.<domain>` block that auto-detects odoc vs. mkdocs vs. plain output by probing the filesystem at request time.
- **`.github/workflows/deploy-dots.yml`** (tracked): on push to `main`, SSHes to the droplet and runs `git pull` — **and nothing else**. It does not yet invoke `webserver deploy` or trigger any rebuild.
- **Confirmed dead config** (parsed from YAML, never read anywhere in the Go code): `Domain.Linked`, `Domain.Indexed`, `AutoRebuild.*`, `EmailNotifications.*`, `Flags.*`, per-repo `Build.AutoRebuild`. These need a decision (implement, repurpose, or delete), not just `linked`.

## Decisions Made This Session

1. **`linked` field** → repurpose as a list of sibling subdomains to cross-link in a shared nav/footer (ties into "one coherent website"). Finalize the exact shape in Module 1.
2. **Droplet Nix strategy** → phased:
   - **Near-term**: install Nix + home-manager on the *current* Ubuntu droplet to pin tool versions (Caddy, mkdocs, OCaml toolchain) declaratively at the package level. This is a real but partial win — home-manager on non-NixOS can't natively own privileged system services, so systemd units are still hand-maintained pointing at Nix-store paths.
   - **Later**: full NixOS migration for true "fully managed" (declarative `services.caddy`, `systemd.timers`, atomic rollback via generations). Do this as a **blue/green swap** — stand up a *new* droplet on NixOS, validate it fully, cut DNS over, decommission the old one — rather than an in-place conversion (`nixos-infect` explicitly warns it can brick a live box; there's no official DigitalOcean NixOS image to fall back on either).
3. **Site builds stay in Go, not Nix derivations.** Nix's job is "what tools exist on the box," not "build this dynamic content." Nix builds are sandboxed with no network access (except fixed-output derivations needing a *pre-known* hash), which conflicts with `syncRepo()`'s floating-branch model (`git fetch` + `reset --hard origin/<branch>`) — you can't know the hash of "whatever's newest on main" in advance. `builders.go` shelling out to `mkdocs build`/`dune build`/a markdown renderer is the right layer for this.
4. **Rebuild trigger chain**, two complementary paths:
   - Extend `deploy-dots.yml`'s SSH script to run `webserver deploy --config deploy.yaml` right after `git pull` — handles anything changed in the dots repo itself, with pass/fail visible in the Action log.
   - Add a **systemd timer** on the droplet implementing `auto_rebuild.interval` — this is the only thing that can catch a push to a *content* repo (`notes`, `cloakaml`, etc.), which has no GitHub Actions of its own, honoring "dots is the only repo with real CI."
   - A webhook listener for instant (non-daily) content-repo rebuilds is a good later addition, not a first step.
5. **Domain scheme** (yours to finalize, but nothing blocks it): `cathe.dev` = dev/technical hub (docs, OCaml/odoc projects — matches the existing odoc-only tier in `*.cathe.dev`), `cathe.work` = professional/CV-facing, `jpears.me` = personal. `cathe.work`/`jpears.me` currently have zero `repos[]` entries — expected gap, not a bug.
6. **Caddyfile stays hand-maintained for now.** At 3 fixed apex domains, adding a new *subdomain* under an existing domain already needs zero Caddyfile edits (the wildcard block's filesystem-probing tiers handle it) — only a new *apex* domain needs a manual edit, and that's a one-time, already-mostly-done cost. Generating the Caddyfile from `deploy.yaml` (replacing filesystem-probing with explicit per-subdomain blocks chosen by `build_method`) is a real correctness improvement — it removes the failure mode where a stale `_html/`or `site/` dir causes Caddy to serve the wrong tier — but it's sequenced as a later, optional module.
7. **GitHub Pages (`username.github.io`) sites**: plan to migrate. You don't control DNS for `github.io` itself (only content published into it, or a custom domain *you* own attached via CNAME), so the realistic path is: migrate content into a droplet subdomain via the proven pipeline, then turn the old `github.io` repo into a redirect stub. Sequenced after the core pipeline is proven on lower-stakes docs subdomains — migrating also means the droplet needs working deploy-key SSH access for whichever second GitHub account owns that repo.
8. **Curriculum depth**: detailed — each module should explain underlying concepts (Go's `os/exec`, what a NixOS module is, systemd timers vs. cron, etc.) as we work through it, not just hand you a task list.

## How We'll Work From Here

You implement; I architect, explain concepts, and review. We'll go through modules roughly in order (each builds on the last, except the ones marked optional/parallel-safe). For each module, when you're ready to start it, bring it up and we'll go deep on the concepts before/while you write the code — this plan captures *what* and *why*, not the code itself.

## Curriculum

**Module 1 — Finalize the `deploy.yaml`/`config.go` schema.**
Decide implement/repurpose/delete for every currently-inert field: `Domain.Linked` (→ cross-link nav, per decision above — nail down exact shape), `Domain.Indexed` (candidate: drive a `noindex` meta tag/header when false), `AutoRebuild.*`, `EmailNotifications.*`, `Flags.*`, per-repo `Build.AutoRebuild`. Also resolve what `ocaml` vs. `odoc` as distinct `build_method`s actually means (or drop one).
Files: `webserver/internal/config/config.go`, `deploy.yaml`.
Concepts: schema design, why "parsed but never read" is tech debt, `config.Load` validation.
Done when: every field is either consumed by code or explicitly documented as reserved; `build_method` values are validated.

**Module 2 — Implement the `md` build method.**
Goal: render a directory of `.md` files to HTML, copy static assets, write to `outDir`. Simplest builder — good first Go/os-exec-adjacent module.
Files: `webserver/internal/deploy/builders.go`, `go.mod` (a markdown lib, e.g. goldmark).
Done when: `webserver deploy` against a scratch markdown repo produces servable HTML.

**Module 3 — Implement `mkdocs`.**
Goal: shell out to `mkdocs build`, publish `site/`. This is also where the README's "centralized mkdocs.yml with css/js overrides" idea plugs in via `BuildOptions.Theme/Colors`.
Files: `builders.go`, `flake.nix` devShell (add mkdocs/mkdocs-material/Python — currently missing entirely).
Concepts: `os/exec` done properly (capturing stderr, exit codes), generating/overriding `mkdocs.yml` from config.
Done when: deploy against a real repo (e.g. `notes`, once `url:` is filled in) produces a `site/` dir Caddy's `@has_site` tier serves.

**Module 4 — Resolve `ocaml`/`odoc`, then implement `odoc`.**
Goal: `dune build @doc`, publish `_build/default/_doc/_html`.
Files: `builders.go`, `config.go` (subdir option for non-root dune projects), `flake.nix` devShell (ocaml/dune/odoc).
Done when: deploy against `cloakaml-docs` serves correctly through the existing `@has_odoc` tier.

**Module 5 — Wire the GitHub Action to trigger a real rebuild.**
Goal: extend `deploy-dots.yml`'s SSH script to run `webserver deploy --config deploy.yaml` after `git pull`, with real exit-code propagation.
Files: `.github/workflows/deploy-dots.yml`.
Concepts: multi-line `appleboy/ssh-action` scripts; explicitly decide how the compiled `webserver` binary gets/stays current on the droplet (rebuild-on-server vs. ship a prebuilt artifact) — this is a prerequisite decision, not a detail.
Done when: a `deploy.yaml` change pushed to main visibly updates the live site via the Action run.

**Module 6 — systemd timer for `auto_rebuild.interval`.**
Goal: the independent scheduled path that catches content-repo pushes with zero GitHub Actions of their own.
Files: new `webserver-deploy.service`/`.timer` units.
Concepts: systemd timers vs. cron, `OnCalendar` syntax, `journalctl`.
Done when: `systemctl list-timers` shows it scheduled; a commit pushed *only* to a content repo shows up live within one interval.

**Module 7 — Nix + home-manager on the current Ubuntu droplet.**
Goal: install Nix (Determinate installer) on the droplet, add a `homeConfigurations` output pinning Caddy/mkdocs/OCaml toolchain/the `webserver` binary; point Module 5/6's systemd units at the resulting store paths.
Files: `flake.nix`, new `home.nix`.
Concepts: standalone (non-NixOS) home-manager, `home-manager switch`, why it can't natively own privileged/boot-time services (`loginctl enable-linger`, `setcap`) — this gap is expected, not a bug to fix here.
Done when: `home-manager switch` supplies every pinned tool from the flake; existing systemd units keep working against the new store paths.

**Module 8 — Full NixOS migration via blue/green droplet swap.**
Goal: the "fully managed" end state. New droplet on NixOS (community image or `nixos-infect` on a *fresh, disposable* box — never the live one); `nixosConfigurations` in `flake.nix` with `services.caddy`, `systemd.timers` replacing Module 6's hand-authored units, firewall/users declared; deploy via `nixos-rebuild --target-host` or `colmena`; validate; cut DNS; decommission old box.
Files: `flake.nix` (`nixosConfigurations` output), `hosts/<hostname>/configuration.nix` + `hardware-configuration.nix`.
Concepts: NixOS modules, generations/rollback, remote deploy tooling, DNS cutover.
Done when: the new droplet serves all domains/subdomains correctly; a config change applies via `nixos-rebuild switch --target-host`; a deliberately broken config rolls back via a previous generation.

**Module 9 (optional) — Webhook listener for instant rebuilds.**
Goal: complement Module 6's daily poll with near-real-time triggers via a plain GitHub repo webhook (not a workflow file) per content repo.
Files: new `internal/webhook` package + `webserver serve-hooks` subcommand, systemd unit, Caddy `reverse_proxy` route.
Concepts: verifying `X-Hub-Signature-256` HMAC, running a persistent listener safely behind Caddy.
Done when: pushing to `notes` triggers a rebuild within seconds, no GH Actions YAML added there.

**Module 10 (optional) — Generate the Caddyfile from `deploy.yaml`.**
Goal: replace filesystem-probing tiers with explicit per-subdomain blocks chosen by `build_method`, generated before each `Reload()` — removes the stale-`_html`/`site`-dir failure mode and gives `Indexed` something concrete to drive (e.g. a `noindex` header).
Files: new `internal/caddygen`, `deploy.go`.
Concepts: `text/template` config generation, `caddy validate` before reload.
Done when: adding a domain/subdomain purely via `deploy.yaml` produces a correct Caddy block with no manual edit, validated with `caddy validate`.

**Module 11 — GitHub Pages consolidation.**
Goal: migrate one `username.github.io` site into a droplet subdomain via the now-proven pipeline; turn the old repo into a redirect stub.
Files: `deploy.yaml` (new entry), the external `github.io` repo, droplet `~/.ssh/config` for the second GitHub identity's deploy key.
Concepts: GitHub Pages custom-domain settings, meta-refresh/redirect pages, multi-account SSH on one host.
Done when: the old `github.io` URL redirects cleanly; the new subdomain serves the migrated content through the standard pipeline.

## Verification (of this planning session)

No code changes were made. The output of this session is this document. To confirm it captures things correctly: re-read the Decisions and Curriculum sections above and flag anything that doesn't match your intent before we start Module 1 — small corrections now are much cheaper than mid-module surprises.
