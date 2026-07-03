# glab-axi

Agent-oriented GitLab CLI written in Go. It wraps the authenticated `glab` CLI, keeps data as JSON internally, and emits compact TOON on stdout.

## Install

Requires `glab` to be installed and authenticated.

```sh
go install github.com/nikolauska/glab-axi@latest
```

## Use

```sh
glab-axi
glab-axi issue list --state opened
glab-axi issue view 42
glab-axi mr list --repo group/project
glab-axi pipeline list --status failed
glab-axi label list --fields id,name,color
glab-axi api projects/:id/releases
```

The initial command surface intentionally covers read-heavy agent workflows: dashboard, issue, merge request, pipeline, label, and raw API access. Use `api` for GitLab operations not yet given a dedicated command.

Errors are structured on stdout. Diagnostics from `glab` are condensed, and exit codes are `0` for success, `1` for operational errors, and `2` for usage errors.

## Develop

```sh
make test
make lint
make build
```

## Agent skill

The repository includes an installable skill at [`skills/glab-axi`](skills/glab-axi/SKILL.md). It teaches compatible agents to prefer `glab-axi` for GitLab work and documents the supported command surface.
