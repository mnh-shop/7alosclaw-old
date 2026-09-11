# 7alosclaw — Native Production Deployment (Fedora, rootless Podman)

One harness instance, many tenants inside it. Tenant shell commands execute in
throwaway sandbox containers (mode=all, scope=session). No MinIO, no k8s, no
per-tenant containers.

```
Fedora host
├── zalosclaw (systemd --user service, binary on disk)
│     └── spawns per-session sandbox containers (podman, default: --network none)
├── zalosclaw-postgres (rootless quadlet, published 127.0.0.1:5432 only)
└── /data/tenants/<slug>/   ← per-tenant workspace, XFS project quota
```

## Network model (rootless = isolation for free)

- Sandbox boxes default to `--network none` (no network at all).
- With `ZALOSCLAW_SANDBOX_NETWORK=true`, boxes get internet via pasta (user-space
  NAT). Host **loopback is not routable** into rootless containers, so Postgres on
  `127.0.0.1:5432` is unreachable from boxes even with network on. Do not publish
  Postgres on any other interface.
- Remaining exposure with network on: the host's LAN/VPC IP and the cloud metadata
  service. Harden the metadata service at the cloud level (AWS IMDSv2 hop limit 1,
  or `ip route del 169.254.169.254/32` on Hetzner/equivalent).
- nftables egress rules on a bridge do NOT apply here: rootless Podman has no host
  bridge device. Fine-grained egress policy is the deferred `RestrictedDomains` tier.

## Layout

| File | Destination |
|---|---|
| `config.json5.example` | `/data/7alosclaw/config.json5` |
| `zalosclaw.service` | `~/.config/systemd/user/zalosclaw.service` (as service user) |
| `zalosclaw-postgres.container` | `~/.config/containers/systemd/` (as service user) |
| `host-setup.sh` | run once as root, then finish as service user |
| `../scripts/provision-tenant.sh` | operator workstation or host |
| `../scripts/isolation-smoke-test.sh` | operator workstation or host |

## Bring-up order

```bash
# 1. as root
sudo ./host-setup.sh                    # user, subuids, linger, /data, XFS pquota check
# 2. as the service user (alos)
podman build -t zalosclaw-sandbox:bookworm-slim -f Dockerfile.sandbox .
mkdir -p ~/.config/containers/systemd ~/.config/systemd/user
cp deploy/production/zalosclaw-postgres.container ~/.config/containers/systemd/
#    put the password in a podman secret first:
printf '%s' "$(openssl rand -hex 24)" | podman secret create zalosclaw_pg_password -
systemctl --user daemon-reload && systemctl --user start zalosclaw-postgres.service

# 3. build + install the harness
go build -o /usr/local/bin/zalosclaw .   # or download the release binary
./zalosclaw onboard                       # writes .env.local; run migrations
./zalosclaw migrate up

# 4. install config + service
mkdir -p /data/7alosclaw && cp deploy/production/config.json5.example /data/7alosclaw/config.json5
cp deploy/production/zalosclaw.service ~/.config/systemd/user/
systemctl --user daemon-reload && systemctl --user enable --now zalosclaw.service

# 5. provision tenants + run the acceptance drill
scripts/provision-tenant.sh acme "Acme Corp"
scripts/isolation-smoke-test.sh
```

## Tenant resource profile

| Resource | Mechanism | Default |
|---|---|---|
| Disk | XFS project quota per `/data/tenants/<slug>` | 10G soft / 15G hard |
| RAM (exec boxes) | `--memory-reservation` / `--memory` | 512M soft / 2G hard |
| RAM (harness) | systemd `MemoryMax` | 4G shared |
| Exec time | `ZALOSCLAW_SANDBOX_TIMEOUT_SEC` | 300s |
| CPU | `ZALOSCLAW_SANDBOX_CPUS` | 2 |

## Deferred (do not enable yet)

- `restricted_domains` / `network` fine-grained policy — declared, not enforced.
- bootc atomic host + greenboot rollback — adopt after the deployment is stable.
- Per-tenant container tier, k3s — only at ~20 tenants / multi-host.
