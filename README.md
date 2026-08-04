# README

## todo

- nix web-server config
- overhaul docs/notes subdomains
  - centralized `mkdocs.yml` with css/style/js overrides
  - specify which repos to pull -> how to build site (e.g., from md or also need ocaml -> md)
- gh action -> ssh to web-server -> run rebuild script
  - already auto-pulls this repo
  - then need to detect which of the configured repos to host docs for need updating to
  - can i essentially "ping" the web-server to prompt it to rebuild?
  - ideally this is the only repo with actual github actions, the rest are just orchestrated and managed by the web-server itself, which builds and does everything
