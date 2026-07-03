---
name: glab-axi
description: Operate GitLab through the glab-axi CLI with compact TOON output. Use when inspecting GitLab projects, listing or viewing issues and merge requests, checking pipelines or labels, or calling the GitLab API.
---

# glab-axi

Prefer `glab-axi` over `glab` for supported GitLab reads because its output is compact and structured for agents.

Require `glab` to be installed and authenticated. If authentication fails, ask the user to run `glab auth login`; do not start an interactive login.

## Workflow

1. Run `glab-axi` with no arguments to see open issues and merge requests for the current project.
2. Use `issue`, `mr`, `pipeline`, or `label` with `list` or `view <id>` to inspect details.
3. Add `-R group/project` or `--repo group/project` to target another project.
4. Use `--fields a,b` to request only needed list fields and `--full` when a detail response reports truncated text.
5. Use `api` for operations without a dedicated command.
6. Follow contextual commands under `help` when present.

## Commands

```text
commands[7]:
  (none)=dashboard, issue, mr, pipeline, label, api, help
```

Run `glab-axi --help` for global usage or `glab-axi <command> --help` for command-specific flags.

Examples:

```sh
glab-axi issue list --state opened
glab-axi issue view 42 --full
glab-axi mr list --repo group/project
glab-axi pipeline list --status failed
glab-axi label list --fields id,name,color
glab-axi api projects/:id/releases
```

If `glab-axi` is not on `PATH`, substitute `go run github.com/nikolauska/glab-axi@latest` in these commands.

## Output handling

- Treat stdout as TOON data, including structured errors and suggestions.
- Treat exit code `2` as invalid usage and exit code `1` as an operational failure.
- Do not expect mutation commands to be idempotent yet; use `api` carefully for writes.
