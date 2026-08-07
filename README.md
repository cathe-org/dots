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

demo sketch of yaml:

```yaml
methods:
  - md
  - ocaml
  - odoc
config:
  - auto_rebuild:
      - enable: true
        interval: daily
  - email_notifications:
      - enable: true
        warnings: true
        general: false
  - flags:
      - clean: false
        reset: false
repos:
  - url: ""
    branch: main
    build_method: mkdocs
    domain:
      - name: ""
        indexed: true
        linked: []
    config:
      - auto_rebuild: true
        input_dir: ./
        theme: material
        colors:
          - accent: []
            fg: []
            bg: []
```
