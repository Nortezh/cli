# ntzh — Nortezh CLI

CLI for the Nortezh deployment platform.

## Install

    go install ./cmd/ntzh

## Auth

    ntzh login                                     # browser-based Google login
    ntzh login --service-account ci@x --key-file k # headless (CI)
    ntzh logout
    ntzh whoami

## Commands

    ntzh project list
    ntzh deployment list   --project <name>
    ntzh deployment get    --project <name> <deployment>
    ntzh deployment deploy --project <name> <deployment> --image <ref>
    ntzh deployment rollback --project <name> <deployment> --to <revision>
    ntzh deployment logs   --project <name> <deployment> [--revision N]

Project is required on every project-scoped command (no stored default).
Use `--output json` for machine-readable output, `--debug` to log HTTP traffic
to stderr (Authorization header is redacted).

## Configuration

    ~/.config/ntzh/config.json       # { "server": "..." }
    ~/.config/ntzh/credentials.json  # 0600, bearer or service_account

Env: `NTZH_SERVER`, `NTZH_PROJECT`, `NTZH_CONFIG_DIR`. Precedence: flag > env > file > default.
