---
name: ntzh
description: Use when the user wants to deploy, ship, or release a service to Nortezh, or mentions `ntzh`, `nortezh`, or a Nortezh project/deployment.
---

# ntzh — Nortezh deploy

Thin CLI over the Nortezh arpc API. **This file is the complete command reference** — every subcommand and flag is listed below. Use it instead of running `ntzh help` / `ntzh <cmd> --help`, and never invent flags that aren't here. If something you need genuinely isn't documented, ask the user rather than guessing a command.

## Global flags & environment

Available on every command:

- `--project <name|slug>` — project, required for all project-scoped commands (or `NTZH_PROJECT`).
- `--server <url>` — API server (or `NTZH_SERVER`; else config file).
- `--output toon|table|json` — `toon` (default, compact/agent-friendly), `table` (aligned columns), or `json` (raw structured). Lists end with a `count:` line; empty lists print `<resource>: 0 found`.
- `--debug` — log HTTP request/response to stderr (Authorization redacted).

Output & errors go to **stdout**. On failure the process exits non-zero and prints a structured `error:` line (plus a `help:` line with the fix when known) to stdout — read it instead of stderr.

Env vars: `NTZH_SERVER`, `NTZH_PROJECT`, `NTZH_LOCATION` (skips location lookup), `NTZH_CONFIG_DIR`. Precedence: **flag > env > config file > default**.

## Discover before you act — don't invent identifiers

If the user didn't supply a value, resolve it; never fabricate one:

- **Project** → `ntzh project list` (matches name, slug, or ID). `--project` / `NTZH_PROJECT` from the user wins.
- **Deployment name** → `ntzh deployment list --project=<p>`. This is the **first positional arg** of get/deploy/rollback/revisions — **not** the project.
- **Image** → from the user, the latest pushed tag, or ask. Never invent a tag.
- **Location** → omit `--location`; the CLI resolves it from `deployment list`. Only pass it if the user did, or when a command **requires** it (see below).

Confirm prod targets first: `ntzh deployment get <name> --project=<p>`.

## When a lookup fails, help — don't just relay the error

The CLI errors hard on bad identifiers (`project not found: <x>`, `could not resolve location for deployment <x>`). Don't stop there — recover:

1. **Re-list and match.** Re-run the relevant `list` and compare the user's value against the real names/slugs/IDs. A typo, casing, or partial name usually has one obvious match — use it.
2. **Guess the closest, then confirm.** If exactly one candidate is a near match, proceed with it and say which one you picked. If several are plausible, show the shortlist and ask. If none match, surface the full list rather than the raw error.
3. **Walk the chain.** A "location" failure usually means the deployment name itself is wrong — re-check it against `deployment list` before retrying. Resolve the project first, then the deployment, then the route/domain.

Never invent an image tag or a brand-new identifier — but a value the user *meant* and mistyped is fair to guess from the list.

## Auth

```sh
ntzh login                                                   # browser, 7-day bearer
ntzh login --service-account=<email> --key-file=<path>       # CI; --key-file=- reads stdin
ntzh login --service-account=<email> --key=<inline-key>      # inline key (not with --key-file)
ntzh whoami                                                  # print current account email
ntzh logout                                                  # delete stored credentials
```

On `not logged in`, re-run `ntzh login`. There is no refresh — a 401 means re-login.

## Projects

```sh
ntzh project list                       # the only project subcommand
```

## Deployments

A deployment has a `name` (e.g. `api`, `web`) and numbered revisions.

```sh
ntzh deployment list   --project=<p>
ntzh deployment get    <name> --project=<p>            # --location auto-resolved
ntzh deployment create <name> --project=<p> --location=<l> --image=<img:tag> [opts]
ntzh deployment deploy <name> --project=<p> --image=<img:tag> [opts]   # --location auto
ntzh deployment rollback  <name> --project=<p> --to=<revision>         # --location auto
ntzh deployment revisions <name> --project=<p>                         # alias: logs
```

### Update env / image / scaling — use `deploy`

`deploy` creates a new revision and patches fields in the same revision. Omitted flags = unchanged (never send a zero value — it would overwrite real state).

```sh
ntzh deployment deploy api --project=<p> --image=img:v2 \
  --set-env DB_URL=postgres://... --set-env DEBUG=true \   # merge (repeatable)
  --remove-env STALE_FLAG \                                # delete a key (repeatable)
  --port 8080 --protocol http --internal=false \
  --min-replica 1 --max-replica 3
```

`--image` is **required** on `deploy`. To change only env/scaling, still pass the current image (from `deployment get`). Returns when the revision is accepted — does **not** wait for rollout. Poll with `deployment get` or `deployment revisions`.

`deployment get <name>` shows current env (TOON keys prefixed `env.`, table rows prefixed `ENV:`, or the `env` object under `--output json`).

### `create` vs `deploy` — flag names differ ⚠️

- **`create`** (new deployment): requires `--image` **and** `--location`. Env flag is **`--env KEY=VALUE`** (repeatable). Has `--type WebService|Worker|CronJob|InternalTCPService` (default WebService) and `--schedule "<cron>"` (required for `CronJob`).
- **`deploy`** (new revision of existing): requires `--image`; `--location` auto-resolves. Env flags are **`--set-env` / `--remove-env`** (NOT `--env`).

`create` has no `--set-env`; `deploy` has no `--env`/`--type`/`--schedule`. Both share `--port`, `--protocol`, `--internal`, `--min-replica`, `--max-replica`.

```sh
ntzh deployment create api     --project=<p> --location=<l> --image=img:v1 --port=8080 --min-replica=1 --max-replica=3
ntzh deployment create worker  --project=<p> --location=<l> --image=img:v1 --type=Worker
ntzh deployment create nightly --project=<p> --location=<l> --image=img:v1 --type=CronJob --schedule="0 2 * * *"
```

`rollback --to <n>` must be a positive revision number from `deployment revisions`. `revisions` returns **revision history** (not pod log lines); for runtime logs fetch the signed `logUrl` from `deployment get`.

## Domains

A domain is tied to one cluster (`--location`). Domain is a **positional** arg.

```sh
ntzh domain list   --project=<p>
ntzh domain get    <domain> --project=<p>                       # verification / DNS hints
ntzh domain create <domain> --project=<p> --location=<l> [--wildcard] [--cdn]
ntzh domain delete <domain> --project=<p>
```

`--location` is **required** on `domain create`. `--wildcard` for `*.acme.com`; `--cdn` enables the paid CDN.

## Routes

A route binds `(domain, path)` to a web-service deployment. The domain must already be registered (`domain create`). Domain and path are **flags here** (unlike domain commands).

```sh
ntzh route list   --project=<p> [--search=<substr>]
ntzh route get    --project=<p> --domain=<d> --path=/
ntzh route create --project=<p> --domain=<d> --path=/ --target=<deployment-name> [--rewrite-path=/$1] [--skip-domain-verify]
ntzh route delete --project=<p> --domain=<d> --path=/
```

`--domain`, `--path`, `--target` are **required** on `create`. `--target` takes a deployment **name** — the CLI prepends `deployment://` automatically. `--location` auto-resolves from the target deployment if omitted. `--path` must start with `/`.
