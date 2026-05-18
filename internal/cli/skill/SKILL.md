---
name: ntzh
description: Use when the user wants to deploy, ship, or release a service to Nortezh, or mentions `ntzh`, `nortezh`, or a Nortezh project/deployment.
---

# ntzh — Nortezh deploy

Thin CLI over the Nortezh arpc API. Run `ntzh <cmd> --help` for anything past the recipe.

## Discover before you deploy

Never guess. If the user didn't supply a value, look it up:

- **Project** → `ntzh project list` (matches name, slug, or ID). `NTZH_PROJECT` / `--project` from the user wins.
- **Deployment name** → `ntzh deployment list --project=<p>`. This is the first positional arg of `deploy`, **not** the project.
- **Image** → from the user, latest pushed tag, or ask. Never invent.
- **Location** → omit `--location`; the CLI resolves it. Only pass it if the user did.

Confirm prod targets first: `ntzh deployment get <name> --project=<p>`.

## Deploy

```sh
ntzh deployment deploy <name> --project=<p> --image=<image:tag>
```

Returns when the revision is accepted — does not wait for rollout. Poll with `ntzh deployment get` or `ntzh deployment revisions`.

## Routes & domains

A route binds `(domain, path)` to a web-service deployment. The domain must first be registered on the project.

```sh
ntzh domain list   --project=<p>
ntzh domain create --project=<p> --location=<l> <domain>           # add --wildcard / --cdn as needed
ntzh route  list   --project=<p>
ntzh route  create --project=<p> --domain=<d> --path=/ --target=<deployment-name>
ntzh route  delete --project=<p> --domain=<d> --path=/
```

`--location` on `route create` is auto-resolved from the target deployment if omitted. The CLI prepends `deployment://` to `--target` for you.

## Auth

```sh
ntzh login                                              # 7-day bearer
ntzh login --service-account=<email> --key-file=<path>  # CI
```

On `not logged in`, re-run `ntzh login`.
