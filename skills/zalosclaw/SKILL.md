---
name: zalosclaw
description: Use this skill when administering, operating, or debugging a ZalosClaw gateway through the ZalosClaw CLI/runtime package. It covers CLI discovery, safe command inspection, gateway health/config diagnostics, agents, skills, MCP/tools, runtime packages, credentials, traces, sessions, channels, providers, cron/jobs, and troubleshooting. Always inspect the live `zalosclaw --help` output first because command availability is version-dependent.
license: Proprietary. Part of ZalosClaw bundled skills.
---

# ZalosClaw Gateway CLI Administration

Use this skill when the user asks you to operate, inspect, administer, or debug a
ZalosClaw gateway through the `zalosclaw` CLI/runtime package.

## Operating Rules

1. Identify the exact binary before running commands.
2. Prefer read-only inspection before mutating actions.
3. Never print, paste, or store API keys, bearer tokens, OAuth tokens, database
   credentials, private keys, cookies, or `.env` contents in chat, issues,
   comments, or logs.
4. Scope every action to the requested gateway, tenant, org, agent, team,
   session, channel, or provider.
5. Ask for explicit confirmation before destructive or security-sensitive
   actions.
6. If command help disagrees with this skill, trust the live `--help` output.

Do not run bare `zalosclaw` unless the user explicitly wants to start the gateway
server. Use `zalosclaw --help`, `zalosclaw version`, or a subcommand for inspection.

## CLI Discovery

Start with read-only discovery:

```bash
command -v zalosclaw
type -a zalosclaw
zalosclaw version
zalosclaw --help
zalosclaw <command> --help
```

If `command -v zalosclaw` is empty on a managed server, also check deployment
runtime paths before concluding the CLI is absent:

```bash
ls -l /var/lib/zalosclaw/data/.runtime/bin/zalosclaw
ls -l /app/data/.runtime/bin/zalosclaw
```

Distinguish the command surface you found:

- Gateway server binary: running `zalosclaw` with no subcommand starts the gateway.
- Operator/admin CLI: subcommands such as `agent`, `skills`, `traces`,
  `sessions`, `providers`, `channels`, `cron`, `config`, `doctor`, `auth`,
  `backup`, `restore`, `migrate`, and `upgrade`.
- Remote operator mode: many admin commands can read `ZALOSCLAW_SERVER`,
  `ZALOSCLAW_GATEWAY_URL`, and `ZALOSCLAW_GATEWAY_TOKEN` from the environment.
  Configure tokens through the shell/session secret manager, not CLI argv or
  pasted text.

Use placeholders in examples:

```bash
zalosclaw traces list
```

## Read-Only Diagnostics

Run safe checks first:

```bash
zalosclaw doctor
zalosclaw config path
zalosclaw config validate
zalosclaw agent list
zalosclaw skills list
zalosclaw sessions list
zalosclaw providers list
zalosclaw channels list
zalosclaw cron list
zalosclaw traces list
```

If a command requires a running gateway, retry with the correct remote target:

```bash
zalosclaw agent list
```

Use `zalosclaw config show` only in a private local terminal when necessary. Treat
the output as sensitive even though the CLI redacts known secret fields; do not
paste it into chats, issues, PRs, or shared logs.

For HTTP-only surfaces, inspect the current API/docs before guessing endpoint
shape. Useful areas include `/health`, `/v1/skills`, `/v1/traces`,
`/v1/packages/runtimes`, `/v1/mcp`, and `/v1/providers` when available.

## Common Workflows

### Agents

Read first:

```bash
zalosclaw agent --help
zalosclaw agent list
```

Before create, update, chat, or delete operations, verify target tenant/org and
agent ID/key. Treat `agent delete` as destructive.

### Skills

Read first:

```bash
zalosclaw skills --help
zalosclaw skills list
zalosclaw skills show <skill>
zalosclaw skills deps status <skill-id-or-path>
zalosclaw skills access get <skill-id>
```

Use dependency scan/check before install:

```bash
zalosclaw skills deps scan <skill-id-or-path>
zalosclaw skills deps check <skill-id-or-path>
```

Mutating skill commands need clear intent and scope:

```bash
zalosclaw skills deps install <skill-id>
zalosclaw skills access set <skill-id> --help
zalosclaw skills grant agent <skill-id> <agent-id>
zalosclaw skills revoke agent <skill-id> <agent-id>
```

System skills are bundled and should not be edited in place. Prefer uploading or
publishing a tenant/custom skill override when customization is required.

### MCP Servers And Tools

First inspect whether this CLI version exposes MCP commands:

```bash
zalosclaw mcp --help
```

If no MCP CLI exists, use the current gateway API/UI/docs for MCP discovery and
permission changes. Do not invent command names. For tool access issues, verify:

- the MCP server is configured and enabled;
- the tool is visible to the requested agent/team;
- permissions or approval rules allow the action;
- gateway logs/traces show the actual failure.

### Runtime Packages

Inspect current runtime/package support before acting:

```bash
zalosclaw packages --help
zalosclaw skills deps status <skill-id-or-path>
```

If there is no packages CLI, use the Packages UI or `/v1/packages/*` API when
available. Install/update/remove package operations are admin-level and may need
approval. Prefer the smallest dependency needed by the selected skill.

### Credentials And Auth

Use read-only status commands first:

```bash
zalosclaw auth status
zalosclaw auth status <provider>
```

For gateway bearer tokens, provider keys, OAuth refresh tokens, and CLI
credentials:

- never echo values;
- do not add them to issues, PRs, chat, or logs;
- prefer environment variables, OS keychain, or gateway credential UI;
- rotate credentials if they were exposed;
- confirm before logout, delete, or replace operations.

### Traces And Failed Runs

Use traces to debug provider errors, tool failures, channel delivery, and stuck
sessions:

```bash
zalosclaw traces --help
zalosclaw traces list --status error
zalosclaw traces get <trace-id> -o json
zalosclaw traces timeline <trace-id>
zalosclaw traces follow --session <session-key>
zalosclaw traces export <trace-id>
```

Redact prompts, tokens, URLs with secrets, headers, and customer data before
posting trace excerpts anywhere public.

### Sessions, Channels, Providers, Cron

Inspect help and list commands:

```bash
zalosclaw sessions --help
zalosclaw channels --help
zalosclaw providers --help
zalosclaw cron --help
```

Treat these as sensitive:

- `sessions delete` and `sessions reset`;
- channel add/delete or credential changes;
- provider add/update/delete and model verification with live credentials;
- cron delete/toggle/run.

## Troubleshooting Playbooks

### Agent Cannot Access A Tool Or MCP

1. Confirm the request is in the intended tenant/org and agent/team.
2. List available skills/tools for that agent context.
3. Check MCP server status/config and permission grants.
4. Inspect the relevant trace for permission, approval, or transport errors.
5. Apply the smallest grant or config change after confirmation.

### Package Install Or Update Failed

1. Check skill dependency status.
2. Inspect runtime availability for Python, Node, system packages, or GitHub
   release installers.
3. Look for pending approval, network, permission, disk, or checksum errors.
4. Retry only after the root cause is clear.

### CLI Credentials Missing Or Expired

1. Run `zalosclaw auth status` or the relevant credential status command.
2. Verify required env vars exist without printing values.
3. Re-authenticate through the approved UI/CLI flow.
4. Re-run the original read-only command before mutating anything.

### Provider Or Model Error

1. Use traces to find the provider, model, status code, and error class.
2. Check `zalosclaw providers list` and provider verification help.
3. Confirm model availability from the current provider config.
4. Avoid changing provider priority or credentials without user approval.

### Skill Not Visible Or Not Granted

1. Run `zalosclaw skills list` and `zalosclaw skills show <skill>`.
2. Inspect access mode and effective access for the target agent/user.
3. Check whether the skill is archived due to missing dependencies.
4. Grant or enable only the requested skill and scope.

### Channel Delivery Failure

1. Identify channel, session, sender, and trace ID.
2. Inspect `zalosclaw channels list` and the failed trace.
3. Check provider/tool errors before blaming the channel.
4. Redact external message IDs and user data when reporting.

## Mutating Command Confirmation

Ask for confirmation before running commands that:

- delete, reset, revoke, restore, force, drop, or migrate down;
- change provider/channel credentials;
- install/update/remove runtime packages;
- grant broad skill/tool access;
- modify tenant/global config;
- trigger cron jobs with external side effects.

Confirmation should include the target, command category, expected effect, and
rollback path if one exists.
