---
name: ntzh
description: Use when the user wants to deploy, ship, or release a service to Nortezh, or mentions `ntzh`, `nortezh`, or a Nortezh project/deployment.
---

# ntzh - Nortezh deploy

`ntzh` deploys container images to the [Nortezh](https://nortezh.com) platform.
For anything beyond the deploy recipe below, run `ntzh <subcommand> --help`.

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
