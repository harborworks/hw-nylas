# Harbor Works Nylas CLI Fork

This repository is the Harbor Works fork of the upstream `nylas/cli` project.

The product goal is to preserve the Nylas CLI command grammar as much as possible while changing the authentication and transport layer:

- The user runs `hw-nylas`, not `nylas`.
- `hw-nylas` should authenticate with Harbor Works credentials and profiles.
- Nylas grants live in the Harbor Works backend.
- Requests should proxy through Harbor Works to Nylas, with minimal Harbor Works-specific transformations.
- Provider-specific power belongs in provider-specific CLIs such as `hw-nylas` and `hw-gh`, not in a generic `hw nylas request` command.

For now, the Go module and internal import paths intentionally remain aligned with upstream. Keeping that surface stable should make upstream rebases and cherry-picks simpler while this fork is still close to the source project.
