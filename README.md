# ntzh — Nortezh CLI

Command-line client for the Nortezh deployment platform.

## Install

    go install ./cmd/ntzh

Or build locally:

    make build      # produces ./ntzh

## Auth

    ntzh login                                       # browser-based Google login
    ntzh login --service-account ci@acme --key-file k # headless (CI)
    ntzh logout
    ntzh whoami

Bearer credentials expire after 7 days and there is **no refresh** — re-run
`ntzh login` when prompted. Service-account credentials don't expire.

## Commands

    ntzh project list

    ntzh deployment list      --project <name|slug>
    ntzh deployment get       --project <name|slug> <deployment> [--location <id>]
    ntzh deployment deploy    --project <name|slug> <deployment> --image <ref>
    ntzh deployment rollback  --project <name|slug> <deployment> --to <revision>
    ntzh deployment revisions --project <name|slug> <deployment>

`--project` accepts a project name, slug (the `no` field), or internal ID.
It is required on every project-scoped command — there is no stored default.

`--location` (cluster ID) is auto-detected via `deployment.list` when omitted.
Set `NTZH_LOCATION` to skip the lookup.

### Output

`--output table` (default) prints human-readable tables.
`--output json` prints the raw structured response — use this for scripting.

`--debug` logs HTTP request/response to stderr (Authorization header redacted).

Deployment list columns: `NAME`, `TYPE`, `STATUS`, `LOCATION`, `REPLICAS`,
`LAST_DEPLOYED`.

## Configuration

    ~/.config/ntzh/config.json       # { "server": "..." }
    ~/.config/ntzh/credentials.json  # 0600, bearer or service_account

Env: `NTZH_SERVER`, `NTZH_PROJECT`, `NTZH_LOCATION`, `NTZH_CONFIG_DIR`.
Precedence: flag > env > file > default.

## Development

    make test       # go test ./...
    make build      # builds ./ntzh
    make lint       # golangci-lint run
