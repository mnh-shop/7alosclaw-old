# Deployment Guide

## Overview

Production target uses a hybrid deployment:

- ZalosClaw gateway runs as a bare-metal `systemd` service.
- PostgreSQL 18 + pgvector runs in Docker.
- Nginx reverse proxies public HTTP/HTTPS traffic to ZalosClaw on localhost.
- Codex CLI is installed on the host for future agent-controlled CLI work.

Current VPS shape:

| Item | Value |
|---|---|
| Host | Private deployment value |
| SSH | Private deployment value |
| Domain | Private deployment value |
| Gateway listen | `127.0.0.1:18790` |
| Public proxy | Nginx on `80/443` |
| Postgres | Docker container `zalosclaw-postgres` |
| Postgres bind | `127.0.0.1:5432` |

Keep concrete hostnames, IPs, SSH users, SSH ports, and credentials in private operator notes or environment variables outside this public repository.

For command examples below:

```bash
export ZALOSCLAW_HOST=<server-ip-or-hostname>
export ZALOSCLAW_SSH_USER=<ssh-user>
export ZALOSCLAW_SSH_PORT=<ssh-port>
export ZALOSCLAW_DOMAIN=<public-domain>
```

## Server Layout

| Path | Purpose |
|---|---|
| `/opt/zalosclaw/releases/<release>` | Immutable ZalosClaw release directory |
| `/opt/zalosclaw/current` | Symlink to active release |
| `/opt/zalosclaw/shared/docker-compose.postgres.yml` | Postgres compose file |
| `/opt/zalosclaw/shared/postgres.env` | Postgres env file |
| `/opt/zalosclaw/backups/` | Uploaded DB backup files |
| `/etc/zalosclaw/config.json` | Gateway config |
| `/etc/zalosclaw/zalosclaw.env` | Gateway env and secrets |
| `/etc/zalosclaw/r2-backup.env` | R2 backup env and secrets |
| `/var/lib/zalosclaw/data` | ZalosClaw persistent data |
| `/var/lib/zalosclaw/workspace` | Agent workspace |
| `/var/lib/zalosclaw/postgres` | Postgres Docker data |
| `/usr/local/bin/zalosclaw-deploy` | Release switch, upgrade, health-check, rollback (source: [`scripts/zuey/zalosclaw-deploy.sh`](../scripts/zuey/zalosclaw-deploy.sh)) |
| `/usr/local/bin/zalosclaw-issue-ssl` | Certbot wrapper for the deployment domain |
| `/usr/local/bin/zalosclaw-backup-r2` | Postgres dump, R2 upload, retention cleanup |
| `/usr/local/bin/zalosclaw-upgrade-release` | Download and deploy a GitHub Release tarball (source: [`scripts/zuey/zalosclaw-upgrade-release.sh`](../scripts/zuey/zalosclaw-upgrade-release.sh)) |

Secrets are stored only in server env files. Do not copy tokens or database passwords into repo docs.

## Runtime Services

Check status:

```bash
ssh -p "$ZALOSCLAW_SSH_PORT" "$ZALOSCLAW_SSH_USER@$ZALOSCLAW_HOST"
systemctl status zalosclaw --no-pager
systemctl status nginx --no-pager
systemctl status zalosclaw-backup-r2.timer --no-pager
docker ps --filter name=zalosclaw-postgres
curl -fsS http://127.0.0.1:18790/health
```

Expected health:

```json
{"status":"ok","protocol":3}
```

## Initial Deployment Record

Completed on 2026-05-17:

- Installed Docker, Docker Compose v2, Nginx, Certbot, Node.js 22, Codex CLI.
- Added the operator workstation SSH public key to the deployment user.
- Installed Codex CLI. For agent-controlled `codex exec`, authenticate the Linux service user that runs ZalosClaw (`zalosclaw`), not only the SSH operator user.
- Restored the latest private PostgreSQL backup into Docker Postgres.
- Upgraded restored schema from `57` to `65`.
- Started the initial ZalosClaw release by `systemd`.
- Added automated database backup to a private Cloudflare R2 bucket.
- Verified local and Nginx health endpoints.

Current verification snapshot:

```text
zalosclaw_active=active
nginx_active=active
health_local={"status":"ok","protocol":3}
health_nginx={"status":"ok","protocol":3}
docker=zalosclaw-postgres pgvector/pgvector:pg18 healthy
schema=65
codex=codex-cli 0.130.0
```

## Codex CLI Auth For Agents

The gateway runs under the `zalosclaw` Linux user. Codex auth is stored under the
effective home directory, so an SSH-session login such as `codex login
--device-auth` under an operator user only writes that user's `~/.codex/auth.json`.
Agents invoking `codex` through the exec tool use the service user's home:

```text
/var/lib/zalosclaw/.codex/auth.json
```

Verify both contexts when debugging auth:

```bash
codex login status
sudo -u zalosclaw -H codex login status
```

If the operator user is logged in but `zalosclaw` is not, either run device auth as
the service user or copy the operator auth file with strict ownership:

```bash
sudo install -d -o zalosclaw -g zalosclaw -m 700 /var/lib/zalosclaw/.codex
sudo install -o zalosclaw -g zalosclaw -m 600 ~/.codex/auth.json /var/lib/zalosclaw/.codex/auth.json
sudo -u zalosclaw -H codex login status
sudo -u zalosclaw -H sh -lc 'mkdir -p /var/lib/zalosclaw/codex-smoke && cd /var/lib/zalosclaw/codex-smoke && codex exec --skip-git-repo-check --sandbox read-only "Reply with exactly: CODEX_AUTH_OK"'
```

Do not store Codex auth material in repository docs. Treat `auth.json` as a
credential.

## DNS And SSL

Cloudflare DNS record:

| Type | Name | Value | Proxy |
|---|---|---|---|
| `A` | Deployment subdomain | Deployment host IP | Proxied |

SSL was issued with Certbot for the deployment domain on 2026-05-17 and Certbot installed automatic renewal.

Verify HTTPS:

```bash
ssh -p "$ZALOSCLAW_SSH_PORT" "$ZALOSCLAW_SSH_USER@$ZALOSCLAW_HOST"
curl -fsS "https://$ZALOSCLAW_DOMAIN/health"
```

Re-issue manually if needed:

```bash
sudo /usr/local/bin/zalosclaw-issue-ssl
```

## Deploy A New Release

Preferred server-side upgrade flow:

```bash
sudo /usr/local/bin/zalosclaw-upgrade-release --dry-run latest
sudo /usr/local/bin/zalosclaw-upgrade-release latest
sudo /usr/local/bin/zalosclaw-upgrade-release v3.12.0
```

The script downloads the Linux amd64 GitHub Release tarball from `digitopvn/zalosclaw`, follows GitHub release redirects, verifies `CHECKSUMS.sha256` when present, falls back to the GitHub release asset SHA256 digest for beta assets without checksum files, extracts to `/opt/zalosclaw/releases/<tag>`, and calls `zalosclaw-deploy`. When invoked from the running gateway service, it first re-launches itself as a transient `systemd-run` unit so `zalosclaw-deploy` can stop/restart `zalosclaw` without killing the upgrade job.

The HTTP API still accepts only `tag`; it does not accept repo names or custom download URLs.

Remote API trigger is available in builds that include the gateway upgrade endpoint:

```bash
curl -fsS -X POST "https://$ZALOSCLAW_DOMAIN/v1/system/gateway/upgrade" \
  -H "Authorization: Bearer <gateway-token>" \
  -H "X-ZalosClaw-Upgrade-Token: <upgrade-token>" \
  -H "Content-Type: application/json" \
  --data '{"tag":"latest"}'
```

Check status:

```bash
curl -fsS "https://$ZALOSCLAW_DOMAIN/v1/system/gateway/upgrade/status" \
  -H "Authorization: Bearer <gateway-token>" \
  -H "X-ZalosClaw-Upgrade-Token: <upgrade-token>"
```

Keep upgrade tokens in server env files or secret managers. Do not put real tokens in docs.

The remote trigger endpoint fails closed unless `ZALOSCLAW_UPGRADE_TRIGGER_TOKEN` is configured in the gateway environment.

### Automatic Beta Deploy From `dev`

Pushing or merging into `dev` runs `.github/workflows/dev-beta-release.yaml`. After Go/Web checks pass, the workflow creates the next semantic beta tag, publishes the linux amd64 prerelease asset and checksum, then deploys that exact beta tag to the zuey VPS through the gateway upgrade endpoint.

Linux arm64 release assets, full checksums, multi-arch Docker images, and beta Docker aliases continue after the fast zuey deploy path. Release asset completion waits until zuey deploy finishes before using `--clobber`, so the VPS upgrade script cannot race with a release asset refresh while downloading the linux amd64 tarball. Those completion jobs are still required to pass; the workflow remains failed if later artifact completion breaks. Zuey deploy and Docker beta alias promotion both skip stale beta tags so older runs cannot roll back a newer beta.

Required GitHub Actions configuration:

| Name | Type | Value |
|---|---|---|
| `ZUEY_ZALOSCLAW_URL` | Secret | Public gateway URL, for example `https://zalosclaw.zuey.me` |
| `ZUEY_ZALOSCLAW_GATEWAY_TOKEN` | Secret | Gateway bearer token from the server env |
| `ZUEY_ZALOSCLAW_UPGRADE_TOKEN` | Secret | Upgrade trigger token from the server env |
| `ZUEY_ZALOSCLAW_USER_ID` | Variable | Optional owner identity, defaults to `system` |

The deploy job sends:

```bash
POST /v1/system/gateway/upgrade {"tag":"vX.Y.Z-beta.N"}
GET /v1/system/gateway/upgrade/status
GET /health
```

The workflow fails if the upgrade status becomes `failed`, times out, or public health does not return `{"status":"ok"}`.

Manual local-build fallback:

Build locally with embedded web UI:

```bash
cd ui/web
pnpm install --frozen-lockfile
pnpm build
cd ../..
rm -rf internal/webui/dist
mkdir -p internal/webui/dist
cp -r ui/web/dist/* internal/webui/dist/
CGO_ENABLED=0 GOOS=linux GOARCH=amd64 \
  go build -tags embedui \
  -ldflags="-s -w -X github.com/nextlevelbuilder/zalosclaw/cmd.Version=<version>" \
  -o dist/zalosclaw-linux-amd64 .
```

Upload:

```bash
release=<version-or-date>
ssh -p "$ZALOSCLAW_SSH_PORT" "$ZALOSCLAW_SSH_USER@$ZALOSCLAW_HOST" "mkdir -p /opt/zalosclaw/releases/$release/migrations"
scp -P "$ZALOSCLAW_SSH_PORT" dist/zalosclaw-linux-amd64 "$ZALOSCLAW_SSH_USER@$ZALOSCLAW_HOST:/opt/zalosclaw/releases/$release/zalosclaw"
scp -P "$ZALOSCLAW_SSH_PORT" migrations/* "$ZALOSCLAW_SSH_USER@$ZALOSCLAW_HOST:/opt/zalosclaw/releases/$release/migrations/"
ssh -p "$ZALOSCLAW_SSH_PORT" "$ZALOSCLAW_SSH_USER@$ZALOSCLAW_HOST" "chmod +x /opt/zalosclaw/releases/$release/zalosclaw && sudo /usr/local/bin/zalosclaw-deploy /opt/zalosclaw/releases/$release"
```

`zalosclaw-deploy` does:

1. Validate binary and migrations.
2. Switch `/opt/zalosclaw/current`.
3. Run `zalosclaw upgrade`.
4. Restart `zalosclaw`.
5. Poll `/health`.
6. Roll back symlink and restart if health fails.

### Self-loop symlink guard

`zalosclaw-deploy` captures the previous release for rollback via `readlink -f /opt/zalosclaw/current` before swinging the symlink. If `/opt/zalosclaw/current` is ever a self-loop (e.g. `current -> current`) or otherwise unresolvable, `readlink -f` exits non-zero. Combined with `set -euo pipefail`, an unguarded call aborts the script before `ln -sfn` runs, so deploys fail silently and zero rollback target is recorded — observed in production on 2026-05-27 where every `deploy_zuey_beta` CI run failed at this step.

The script swallows the readlink failure (`readlink -f ... 2>/dev/null || true`) and logs a warning when the symlink exists but resolves to empty, then proceeds to overwrite. Rollback is skipped in that case because no valid `previous` is available.

If you ever edit `/usr/local/bin/zalosclaw-deploy` on zuey, preserve this guard:

```bash
previous=""
if [ -L /opt/zalosclaw/current ]; then previous="$(readlink -f /opt/zalosclaw/current 2>/dev/null || true)"; fi
if [ -z "$previous" ] && [ -L /opt/zalosclaw/current ]; then
  echo "warn: /opt/zalosclaw/current is a symlink but readlink -f failed (likely self-loop or broken target); overwriting without rollback target" >&2
fi
```

The canonical source of this script lives in the repo at [`scripts/zuey/zalosclaw-deploy.sh`](../scripts/zuey/zalosclaw-deploy.sh), alongside [`scripts/zuey/zalosclaw-upgrade-release.sh`](../scripts/zuey/zalosclaw-upgrade-release.sh). The VPS copies at `/usr/local/bin/zalosclaw-deploy` and `/usr/local/bin/zalosclaw-upgrade-release` are downstream replicas. CI auto-syncs both on every beta release via the `Sync zuey ops scripts to VPS` step in `.github/workflows/dev-beta-release.yaml`. For manual sync (off-CI):

```bash
scp -P 2233 scripts/zuey/zalosclaw-deploy.sh scripts/zuey/zalosclaw-upgrade-release.sh \
  zuey@82.197.71.246:/tmp/
ssh -p 2233 zuey@82.197.71.246 'bash -s' <<'EOF'
set -euo pipefail
ts=$(date +%Y%m%d-%H%M%S)
for name in zalosclaw-deploy zalosclaw-upgrade-release; do
  if [ -f "/usr/local/bin/$name" ]; then
    sudo cp -p "/usr/local/bin/$name" "/usr/local/bin/${name}.bak-${ts}"
  fi
  sudo install -o root -g root -m 0755 "/tmp/${name}.sh" "/usr/local/bin/$name"
  sudo bash -n "/usr/local/bin/$name" && echo "OK: $name"
done
EOF
```

Always back up the live copy before overwriting (the `cp -p` line above does this).

### CI auto-sync — required GitHub secrets

The `Sync zuey ops scripts to VPS` step in `dev-beta-release.yaml` runs `scp + sudo install` before triggering the gateway upgrade endpoint. It needs the following repository secrets configured under **Settings → Secrets and variables → Actions**:

| Secret | Purpose |
|---|---|
| `ZUEY_SSH_PRIVATE_KEY_B64` | CI-only ed25519/rsa key, **base64-encoded as a single line** (`base64 -w0 < /path/to/key`). Its public key must be appended to `zuey@82.197.71.246:~/.ssh/authorized_keys`. **Do not reuse the operator's personal key.** Base64 avoids the `error in libcrypto` failure caused by GitHub Secrets normalizing newlines inside multi-line PEM blocks. |
| `ZUEY_SUDO_PASS` | Same value as `ZUEY_ZALOSCLAW_SUDO_PASS` in the operator's local `.env`; used by `sudo -S` over the SSH session to install scripts. |

Optional repository **variables** (override defaults if the VPS endpoint changes):

| Variable | Default |
|---|---|
| `ZUEY_SSH_HOST` | `82.197.71.246` |
| `ZUEY_SSH_PORT` | `2233` |
| `ZUEY_SSH_USER` | `zuey` |

To rotate the CI SSH key:

```bash
ssh-keygen -t ed25519 -f /tmp/ci-zuey-key -N '' -C 'gh-actions-deploy-zuey-beta'
# add /tmp/ci-zuey-key.pub to zuey:~/.ssh/authorized_keys (consider restricting to scp+install via `command="..."` forced-command)
# base64-encode the private key on a single line, then paste into the
# ZUEY_SSH_PRIVATE_KEY_B64 secret:
base64 -w0 < /tmp/ci-zuey-key  # macOS: `base64 -i /tmp/ci-zuey-key | tr -d '\n'`
# (copy the single-line output and paste it into the secret value)
# then `shred -u /tmp/ci-zuey-key*` locally
```

## Backup And Restore

Automated backups:

| Item | Value |
|---|---|
| Timer | `zalosclaw-backup-r2.timer` |
| Schedule | Every 6 hours: `00:00`, `06:00`, `12:00`, `18:00` server time |
| Source | Docker Postgres container `zalosclaw-postgres` |
| Format | PostgreSQL custom dump, `pg_dump -Fc` |
| Local directory | `/opt/zalosclaw/backups/` |
| R2 bucket | Private deployment value |
| R2 prefix | Private deployment value |
| Retention | Keep latest 20 backups locally and in R2 |

Check timer and latest logs:

```bash
systemctl list-timers zalosclaw-backup-r2.timer --no-pager
journalctl -u zalosclaw-backup-r2.service -n 80 --no-pager
```

Run a manual backup:

```bash
sudo systemctl start zalosclaw-backup-r2.service
```

Create a database dump on server:

```bash
ts=$(date +%Y%m%d-%H%M%S)
docker exec zalosclaw-postgres pg_dump -U zalosclaw -Fc -d zalosclaw > /opt/zalosclaw/backups/zalosclaw-$ts.dump
```

Restore a dump:

```bash
systemctl stop zalosclaw
docker exec -i zalosclaw-postgres pg_restore -U zalosclaw -d zalosclaw --clean --if-exists --no-owner < /opt/zalosclaw/backups/<file>.dump
sudo /usr/local/bin/zalosclaw-deploy /opt/zalosclaw/current
```

## Operational Notes

- Gateway runs as Linux user `zalosclaw`.
- Host-control exceptions are deployment-specific and must be documented privately, not in this public runbook.
- Workspace restriction settings are deployment-specific and must be reviewed before enabling agent-controlled host operations.
- Postgres is bound only to localhost.
- UFW allows the private SSH port, `80/tcp`, and `443/tcp`.
- Reboot is recommended later because the VPS reports a pending kernel upgrade.

## Troubleshooting

Logs:

```bash
journalctl -u zalosclaw -n 200 --no-pager
docker logs zalosclaw-postgres --tail 100
tail -n 100 /var/log/nginx/error.log
```

Restart:

```bash
systemctl restart zalosclaw
docker compose --env-file /opt/zalosclaw/shared/postgres.env -f /opt/zalosclaw/shared/docker-compose.postgres.yml restart postgres
systemctl reload nginx
```

Unresolved questions: none.
