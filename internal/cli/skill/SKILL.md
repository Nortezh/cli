---
name: ntzh
description: Use when the user wants to deploy, ship, or release a service to Nortezh, or mentions `ntzh`, `nortezh`, or a Nortezh project/deployment.
---

# ntzh - Nortezh deploy

`ntzh` deploys container images to the [Nortezh](https://nortezh.com) platform.
For anything beyond the deploy recipe below, run `ntzh <subcommand> --help`.

## Before you deploy

If the user didn't give you all the values, **discover them — don't guess**:

1. **Project** — `ntzh project list`. Match by name/slug from the user's prompt; if ambiguous, ask. `NTZH_PROJECT` / `--project` from the user always wins.
2. **Deployment name** — `ntzh deployment list --project=<p>`. This is the first positional arg of `deploy`, **not** the project. Often shaped like `<service>-<env>` (e.g. `api-prod`).
3. **Image** — take from the user's message, the latest pushed tag, or ask. Never invent a tag.
4. **Location** — **omit `--location`** unless the user specified one; the CLI auto-resolves it from `deployment.list`. Set `NTZH_LOCATION` to skip the lookup in CI.

Confirm the target before a prod deploy:

```sh
ntzh deployment get <name> --project=<p>
```

## Deploy recipe

```sh
ntzh deployment deploy <deployment> \
  --project=<project> \
  --image=<image:tag> \
  --location=<location>
```

Concrete example:

```sh
ntzh deployment deploy staging-bo \
  --project=acme \
  --image=ghcr.io/acme/api:v1.2.3 \
  --location=bkk-1
```

- `<project>` accepts the project name, slug, or ID.
- `--location` is the cluster ID; if omitted the CLI auto-detects it from `deployment.list`. Set `NTZH_LOCATION` to skip the lookup.
- `NTZH_PROJECT` defaults `--project` so you can drop it in CI.

Returns when the backend accepts the new revision — does **not** wait for rollout. Poll with `ntzh deployment get` or `ntzh deployment revisions`.

## Auth (when needed)

```sh
ntzh login                                                    # interactive (7-day bearer)
ntzh login --service-account=<email> --key-file=<path>        # CI
```

If a command prints `not logged in. Run 'ntzh login'`, re-run login.

## Discover more

Anything else — list, get, rollback, revisions, output formats — discover via:

```sh
ntzh --help
ntzh deployment --help
ntzh deployment <subcommand> --help
```
