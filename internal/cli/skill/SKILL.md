---
name: ntzh
description: Use when the user wants to deploy, list, inspect, or roll back services on the Nortezh platform, or mentions ntzh, nortezh, "ship to nortezh", or a Nortezh project/deployment.
---

# ntzh - Nortezh deployment CLI

`ntzh` is the command-line client for the [Nortezh](https://nortezh.com)
deployment platform. Install: `go install github.com/nortezh/cli/cmd/ntzh@latest`.

## When to invoke

- User says "deploy to nortezh", "ship X", "ntzh deploy/rollback/revisions", or names a Nortezh project/deployment.
- User asks to list projects/deployments on Nortezh.
- A CI script needs to push a built image to a Nortezh deployment.

If you're not sure the project is on Nortezh, run `ntzh whoami` first — it errors loudly if no credentials exist.

## Mental model (read this before running anything)

- **No stored project state.** Every command requires `--project=<name|slug|id>` (or `NTZH_PROJECT`). The CLI calls `project.list` and resolves the identifier on each invocation.
- **Location is a cluster ID.** Deployment-scoped commands need `--location=<id>`. If omitted, the CLI tries to resolve it from `deployment.list`. `NTZH_LOCATION` short-circuits the lookup.
- **Auth is a 7-day bearer.** No refresh. On `UNAUTHORIZED` the CLI prints `not logged in. Run 'ntzh login'` — re-run it.
- **arpc envelope.** Errors print as `<code>: <message>`. Exit code is non-zero on failure.
- **Output:** default is a table; `--output=json` is for scripting (parse with `jq`).
- **Debug:** `--debug` logs request/response to stderr with the Authorization header redacted. Use this when payload shapes look wrong.

## Auth

```sh
ntzh login                                                       # interactive (browser)
ntzh login --service-account=ci@acme.com --key-file=./key.txt    # CI / headless
ntzh whoami
ntzh logout
```

Credentials live at `~/.config/ntzh/credentials.json` (mode 0600).

## Command recipes

### List projects

```sh
ntzh project list
ntzh project list --output=json | jq '.[].no'   # slugs
```

### List deployments in a project

```sh
ntzh deployment list --project=<project>
```

Table columns: `NAME`, `TYPE`, `STATUS`, `LOCATION`, `REPLICAS`, `LAST_DEPLOYED`.

### Inspect one deployment

```sh
ntzh deployment get <deployment> --project=<project> --location=<location>
ntzh deployment get api --project=acme --location=bkk-1 --output=json
```

### Ship a new image

```sh
ntzh deployment deploy <deployment> \
  --project=<project> \
  --image=<image:tag> \
  --location=<location>
```

Example:

```sh
ntzh deployment deploy staging-bo \
  --project=acme \
  --image=ghcr.io/acme/api:v1.2.3 \
  --location=bkk-1
```

The command returns when the backend has accepted the new revision — it does **not** wait for the rollout to finish. Poll with `ntzh deployment get` or `ntzh deployment revisions` if you need completion status.

### Roll back

```sh
ntzh deployment rollback <deployment> \
  --project=<project> \
  --to=<revision> \
  --location=<location>
```

Use `ntzh deployment revisions <name>` first to find the target revision number.

### Revision history

```sh
ntzh deployment revisions <deployment> --project=<project> --location=<location>
```

This returns **revision history** (not pod log lines). For runtime pod logs, fetch the signed `logUrl` from `ntzh deployment get <name>` and stream from there.

## CI pattern

```yaml
# GitHub Actions
- name: Deploy
  env:
    NTZH_PROJECT: acme
    NTZH_LOCATION: bkk-1
    NTZH_KEY: ${{ secrets.NTZH_KEY }}
  run: |
    ntzh login --service-account=ci@acme.com --key-file=<(echo "$NTZH_KEY")
    ntzh deployment deploy api --image=ghcr.io/acme/api:${{ github.sha }}
```

Setting `NTZH_PROJECT` + `NTZH_LOCATION` lets you drop those flags from individual commands.

## Common errors

| Symptom                                            | Fix                                                         |
| -------------------------------------------------- | ----------------------------------------------------------- |
| `not logged in. Run 'ntzh login'`                  | Bearer expired (7-day cap) or never logged in.              |
| `project not found: <x>`                           | `<x>` doesn't match any project's name, slug, or ID.        |
| `--image is required` / `--to <revision> is required` | Missing required flag.                                    |
| Zero-value table cells                             | Run with `--debug` and verify backend response shape.       |

## Don't

- Don't introduce a `default_project` or `project use` concept — stateless project context is intentional.
- Don't strip `--location` from deployment-scoped commands; the auto-detect is a convenience, not the contract.
- Don't expect bearer tokens to refresh. They don't.
