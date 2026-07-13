# EasyMail Postfix Integration

EasyMail does not handle SMTP inbound connections directly. It relies on **Postfix** as the edge MTA to receive mail from external senders. EasyMail acts as an anti-spam gateway and mail storage backend, providing Webmail and IMAP access for end users.

---

## 1. Postfix + EasyMail Relationship

### 1.1 Role Separation

```
External Internet
      │
      ▼  SMTP (port 25)
┌─────────────┐
│   Postfix   │  MTA — receives mail from external servers,
│  (MTA/SMTP) │         handles SMTP conversation, queue, retry
└──────┬──────┘
       │
       ├──────────────────────────────────────────────────────┐
       │ Milter protocol (port 10026)                         │
       │   → EasyMail anti-spam filter decides:              │
       │     accept / reject / spam / quarantine              │
       │                                                      │
       │ LMTP delivery (port 10027)                           │
       │   → EasyMail stores to disk + SQLite index           │
       │                                                      │
       │ Dovecot protocol proxy (port 10025)                  │
       │   → Domain lookup, mailbox lookup, SASL auth         │
       │                                                      │
       │ SMTP submission (port 587)                           │
       │   → SASL-authenticated users send outbound mail      │
       └──────────────────────────────────────────────────────┘
                               │
                               ▼
                     ┌─────────────────┐
                     │   EasyMail      │  Anti-spam gateway +
                     │                 │  Mail storage
                     │  ┌───────────┐  │
                     │  │ Webmail   │  │  Read/send via browser
                     │  ├───────────┤  │
                     │  │ IMAP      │  │  Read via email clients
                     │  ├───────────┤  │
                     │  │ LMTP      │  │  Receive mail from Postfix
                     │  ├───────────┤  │
                     │  │ Milter    │  │  Anti-spam filtering
                     │  └───────────┘  │
                     └─────────────────┘
```

### 1.2 Service Ports

| Port | Protocol | Direction | Purpose |
|---|---|---|---|
| 25 | SMTP | Inbound | Postfix receives external mail |
| 587 | SMTP (STARTTLS) | Inbound | Postfix receives user-submitted mail |
| 10025 | TCP (Dovecot proxy) | EasyMail → Postfix | Domain/mailbox lookup, SASL auth |
| 10026 | TCP (Milter) | EasyMail → Postfix | Anti-spam filtering decision |
| 10027 | TCP (LMTP) | EasyMail → Postfix | Final mail delivery + storage |
| 443/8080 | HTTPS/HTTP | User → EasyMail | Admin panel, Webmail |
| 143/993 | IMAP/IMAPS | User → EasyMail | Read mail via email clients |

### 1.3 Mail Flow Summary

1. External sender delivers mail to **Postfix** (port 25)
2. During SMTP conversation, Postfix invokes the **EasyMail Milter** (port 10026) at each stage (Connect → Helo → MailFrom → RcptTo → Headers → Body) to obtain anti-spam decisions
3. If the milter rejects the mail, Postfix returns a 5xx SMTP error to the sender
4. If accepted, Postfix delivers the mail to **EasyMail LMTP** (port 10027)
5. EasyMail reads the `X-EasyMail-Filter-Action` header injected by the milter to route the mail to the appropriate folder (Inbox / Spam / Quarantine)
6. Users retrieve mail via **Webmail** or **IMAP**

### 1.4 Config Generation (Server Side)

The EasyMail admin panel generates Postfix configuration through the `PostfixConfigService`
(`easymail/internal/app/management/postfix_config_service.go`). The workflow is:

1. **Parameters**: Stored in the `postfix_configs` database table. Managed params (set by system) are read-only; user-defined params are fully editable.
2. **Variable Resolution**: Parameter values support `${section.field}` variable references that are resolved from EasyMail's runtime configuration:

   | Variable | Resolved Value |
   |---|---|
   | `${dovecot.listen}` | Dovecot proxy listen address |
   | `${lmtp.listen}` | LMTP listen address |
   | `${milter.listen}` | Milter listen address |
   | `${imap.listen}` | IMAP listen address |
   | `${admin.listen}` | Admin HTTP listen address |
   | `${webmail.listen}` | Webmail HTTP listen address |
   | `${dovecot.family}` | Dovecot proxy socket family (tcp/unix) |
   | `${lmtp.family}` | LMTP socket family |
   | `${milter.family}` | Milter socket family |
   | `${imap.family}` | IMAP socket family |
   | `${postfix.host}` | Postfix EasyMail host setting |
   | `${storage.root}` | Mail storage root path |

   When a service listens on `0.0.0.0:N`, the address is automatically resolved to `<postfix.host>:N` so Postfix can reach the correct IP.

3. **Domain Auto-Sync**: The `virtual_mailbox_domains` parameter is automatically populated from EasyMail's active domain list.

4. **Config Template**: Parameters are rendered into the `main.cf` template:

   ```
   # === EasyMail managed config ===
   # Generated at: 2026-07-12T15:04:05Z

   smtpd_milters = inet:192.168.1.1:10026
   virtual_transport = lmtp:inet:192.168.1.1:10027
   ...

   virtual_mailbox_domains = example.com, test.com
   # === End EasyMail managed config ===
   ```

---

## 2. Postfix-Agent

### 2.1 Overview

**Postfix-agent** is a standalone HTTP daemon that runs on each Postfix server. It acts as a bridge between the EasyMail admin panel and the local Postfix instance, enabling remote configuration management and queue operations.

```
Postfix Host (192.168.1.10)         EasyMail Server (192.168.1.1)
┌─────────────────────────┐         ┌──────────────────────────┐
│  postfix-agent (:8081)  │ ◄─────►│  Admin Panel + API       │
│                         │  HTTP   │  /api/v1/admin/postfix/* │
│  Auth: X-Agent-Token    │         │                          │
│                         │         │  ┌──────────────────┐    │
│  ┌───────────────────┐  │         │  │ PostfixConfig    │    │
│  │ Postfix (/etc/    │  │         │  │ Service          │    │
│  │ postfix/main.cf)  │  │         │  │                  │    │
│  │                   │  │         │  │ Generate config  │    │
│  │ postqueue -j      │  │         │  │ Resolve ${vars}  │    │
│  │ postsuper -d/-r   │  │         │  │ Push to agent    │    │
│  │ postfix reload    │  │         │  └──────────────────┘    │
│  └───────────────────┘  │         └──────────────────────────┘
└─────────────────────────┘
```

### 2.2 Deployment

#### Binary

The `postfix-agent` binary is built from `easymail/cmd/postfix-agent/main.go`. It is a single self-contained binary with no external dependencies.

```
# On the Postfix server
./postfix-agent \
  --listen=:8081 \
  --token=ema_xxxxxxxxxxxx \
  --postfix-dir=/etc/postfix \
  --staging-dir=/tmp/easymail-staging \
  --backup-dir=/etc/postfix/backups \
  --log-dir=/var/log/easymail-agent \
  --allowed-ips=192.168.1.0/24,10.0.0.0/8
```

#### Command-Line Flags / Environment Variables

| Flag | Env Variable | Default | Description |
|---|---|---|---|
| `--listen` | `LISTEN_ADDR` | `:8081` | Agent HTTP listen address |
| `--token` | `AGENT_TOKEN` | (required) | Pre-shared auth token |
| `--postfix-dir` | `POSTFIX_DIR` | `/etc/postfix` | Postfix configuration directory |
| `--staging-dir` | `STAGING_DIR` | `/tmp/easymail-staging` | Staging directory for config |
| `--backup-dir` | `BACKUP_DIR` | `/etc/postfix/backups` | Backup directory for rollback |
| `--log-dir` | `LOG_DIR` | `/var/log/easymail-agent` | Session log directory |
| `--allowed-ips` | `ALLOWED_IPS` | (all IPs) | Comma-separated IP/CIDR allowlist |

#### systemd Service

```
[Unit]
Description=EasyMail Postfix Agent
After=network.target postfix.service

[Service]
Type=simple
ExecStart=/usr/local/bin/postfix-agent \
  --token=ema_xxxxxxxxxxxx \
  --allowed-ips=192.168.1.0/24
Restart=always
RestartSec=5

[Install]
WantedBy=multi-user.target
```

### 2.3 Agent API Endpoints

All endpoints validate via `X-Agent-Token` header. Endpoints with the `ipAllowMiddleware` also check the connecting IP against the configured allowlist.

#### Status

```
GET /api/v1/agent/status
```

Returns Postfix process status, config hash, version, and agent uptime.

```json
{
  "postfixRunning": true,
  "configHash": "a1b2c3d4...",
  "lastReloadAt": "",
  "postfixVersion": "3.7.6",
  "agentVersion": "1.0.0",
  "uptime": "12h34m56s"
}
```

| Field | Description |
|---|---|
| `postfixRunning` | Whether Postfix is running (checks systemctl, postfix status, ps) |
| `configHash` | SHA-256 of the current `easymail.cf` in Postfix config dir |
| `postfixVersion` | Output of `postconf -d mail_version` |
| `agentVersion` | Built-in agent version string |
| `uptime` | Agent process uptime |

#### Config Management

```
POST /api/v1/agent/config/push

Request:
{
  "mainCf": "content of main.cf..."
}

Response: {"status": "staged"}
```

Stages the received configuration to the staging directory. Writes `easymail.cf` and a merged `main.cf` for validation. The config is cached in memory for the subsequent apply step.

```
POST /api/v1/agent/config/apply

Response: {"status": "applied", "backup": "20260712_150405"}
```

Applies the staged configuration to Postfix:

1. Rewrite staging files for safety
2. Run `postfix -c <staging> check` to validate the new config
3. Backup current `easymail.cf` to `/etc/postfix/backups/<timestamp>/`
4. Write new `easymail.cf` to Postfix config directory (source-of-truth)
5. Apply each parameter via `postconf -e "<param> = <value>"`
6. Run `postfix reload` to activate the new config
7. Cleanup old backups (keep last 10)

```
POST /api/v1/agent/config/rollback

Response: {"status": "rolled_back", "backup": "20260712_150405"}
```

Rolls back to the most recent backup:

1. Locate the latest backup directory
2. Validate backup config via `postfix -c` check
3. Restore `easymail.cf` from backup
4. Apply backup params via `postconf -e`
5. Run `postfix reload`

#### Queue Management

```
GET /api/v1/agent/queue/list?status=deferred&sender=&recipient=&queueId=&page=1&pageSize=100

Response:
{
  "messages": [
    {
      "queueId": "AABBCCDD",
      "size": 12345,
      "age": "2h",
      "sender": "user@example.com",
      "recipients": ["dest@other.com"],
      "status": "deferred",
      "statusText": "connect to mail.other.com[1.2.3.4]:25: Connection timed out"
    }
  ],
  "total": 150,
  "page": 1,
  "pageSize": 100
}
```

Lists messages in the Postfix queue. Uses `postqueue -j` (JSON output) internally. Supports filtering by status, sender, recipient, and queue ID. The `status` field maps from Postfix queue names:

| Postfix Queue | `status` Value |
|---|---|
| `active` | `active` |
| `deferred` | `deferred` |
| `hold` | `held` |
| `incoming` | `incoming` |

```
GET /api/v1/agent/queue/stats

Response:
{
  "total": 150,
  "active": 10,
  "deferred": 135,
  "held": 5
}
```

Returns aggregate queue statistics.

```
POST /api/v1/agent/queue/delete

Request: {"messageIds": ["AABBCCDD", "EEFFGGHH"]}
Response: {"status": "deleted"}
```

Deletes specified messages from the queue using `postsuper -d <queue_id> [queue_id...]`.

```
POST /api/v1/agent/queue/resend

Request: {"messageIds": ["AABBCCDD"]}
Response: {"status": "resent"}
```

Requeues specified messages using `postsuper -r <queue_id> [queue_id...]`.

```
POST /api/v1/agent/queue/flush

Response: {"status": "flushed"}
```

Triggers immediate retry of all deferred mail using `postqueue -f`.

### 2.4 Agent Configuration State Machine

```
                        ┌─────────┐
                        │  IDLE   │
                        └────┬────┘
                             │
                   POST /config/push
                             │
                             ▼
                        ┌─────────┐
                        │ STAGED  │  Config written to staging dir,
                        └────┬────┘  cached in memory
                             │
                   POST /config/apply
                             │
                             ▼
                  ┌─────────────────────┐
                  │  VALIDATE (postfix  │── failure → error response, stay STAGED
                  │  -c check)          │
                  └──────────┬──────────┘
                             │ success
                             ▼
                  ┌─────────────────────┐
                  │  BACKUP + WRITE     │
                  │  easymail.cf        │
                  └──────────┬──────────┘
                             │
                             ▼
                  ┌─────────────────────┐
                  │  postconf -e        │── failure → error response, config unchanged
                  └──────────┬──────────┘
                             │
                             ▼
                  ┌─────────────────────┐
                  │  postfix reload     │── failure → error, previous config still active
                  └──────────┬──────────┘
                             │ success
                             ▼
                        ┌─────────┐
                        │  IDLE   │
                        └─────────┘
```

Rollback restores the last backup by repeating the same pipeline (validate → write → postconf -e → reload).

### 2.5 Security

- **Token Authentication**: Every request to the agent must include the `X-Agent-Token` header matching the pre-shared token set at startup.
- **IP Allowlist**: Optional CIDR-based access control restricts which EasyMail servers can reach the agent.
- **Session Logging**: All requests (including rejected ones) are logged to `sessions.log` with IP, method, path, status code, and duration.
- **Backup Before Apply**: The agent always creates a timestamped backup of `easymail.cf` before applying changes, enabling safe rollback.

---

## 3. Shell Script Deployment (Without Agent)

For environments where deploying the postfix-agent binary is not feasible, EasyMail can generate a self-contained shell script that applies the same Postfix configuration. The script is served via a dedicated API endpoint and can be piped directly into `sh` on the target Postfix server.

### 3.1 Usage

On the Postfix server, run:

```bash
curl -s http://<easymail-admin>:8080/api/v1/admin/postfix/install-script | sudo sh
```

Or save and inspect first:

```bash
curl -s http://<easymail-admin>:8080/api/v1/admin/postfix/install-script > install.sh
chmod +x install.sh
sudo ./install.sh
```

Environment variables can override defaults:

```bash
# Custom Postfix directory (default: /etc/postfix)
POSTFIX_DIR=/etc/postfix \
BACKUP_DIR=/var/backups/postfix \
curl -s http://<easymail-admin>:8080/api/v1/admin/postfix/install-script | sudo sh
```

### 3.2 API Endpoint

| Method | Path | Description |
|---|---|---|
| GET | `/api/v1/admin/postfix/install-script` | Generate shell script for manual Postfix configuration |

The endpoint returns `Content-Type: text/x-shellscript` with a `Content-Disposition: attachment` header, making it suitable for direct piping.

### 3.3 Script Behavior

The generated script performs the following steps:

1. **Root check**: Verifies the script is run as root
2. **Backup**: Creates a timestamped backup of the current `easymail.cf` in `/etc/postfix/backups/<timestamp>/`
3. **Write config**: Writes the rendered `easymail.cf` to the Postfix configuration directory via a heredoc
4. **Apply parameters**: Runs `postconf -e '<param> = <value>'` for each managed parameter
5. **Validate**: Runs `postfix check` to validate the configuration
6. **Reload**: Runs `postfix reload` to activate the configuration
7. **Cleanup**: Removes old backups beyond the last 10

The script is idempotent — running it multiple times produces the same result. It sets `set -e` so it fails fast on any error.

### 3.4 Script Output Example

```
Backed up current easymail.cf to /etc/postfix/backups/20260712_150405/
Wrote easymail.cf to /etc/postfix/
Validating Postfix configuration...
Configuration validation passed.
Reloading Postfix...
Postfix reloaded successfully.
=== EasyMail Postfix configuration applied successfully ===
Config hash: a1b2c3d4e5f6...
```

### 3.5 Comparison: Agent vs. Shell Script

| Aspect | postfix-agent | Shell script |
|---|---|---|
| **Deployment** | Binary, systemd service | Zero installation (curl pipe) |
| **Config delivery** | HTTP push from admin panel | Pull via curl from Postfix server |
| **Rollback** | Automatic via backup | Manual (restore from backup dir) |
| **Queue management** | Full remote queue ops | Not available |
| **Status monitoring** | Live status polling | Not available |
| **Multi-server** | Push to all agents at once | Each server runs script separately |
| **Security** | Token + IP allowlist | Depends on network access |
| **Best for** | Production deployments | Quick setup, testing, automation |

---

## 4. Admin Panel API Endpoints (Server Side)

The EasyMail admin server exposes these endpoints under `/api/v1/admin/` for Postfix configuration management.

### 4.1 Agent Management

| Method | Path | Description |
|---|---|---|
| GET | `/postfix/agents` | List agents (`?keyword`, `?page`, `?pageSize`) |
| GET | `/postfix/agents/:id` | Get agent details |
| POST | `/postfix/agents` | Create agent |
| PUT | `/postfix/agents/:id` | Update agent |
| DELETE | `/postfix/agents/:id` | Delete agent |

### 4.2 Config Parameters

| Method | Path | Description |
|---|---|---|
| GET | `/postfix/configs` | List config params (`?keyword`, `?page`, `?pageSize`) |
| GET | `/postfix/configs/:id` | Get config param |
| POST | `/postfix/configs` | Create config param |
| PUT | `/postfix/configs/:id` | Update config param |
| DELETE | `/postfix/configs/:id` | Delete config param |

### 4.3 Settings & Variables

| Method | Path | Description |
|---|---|---|
| GET | `/postfix/settings` | Get global Postfix settings (EasyMailHost) |
| PUT | `/postfix/settings` | Update global settings |
| GET | `/postfix/variables` | List available `${section.field}` variables |
| GET | `/postfix/local-ips` | Get local IP addresses |

### 4.4 Config Generation & Delivery

| Method | Path | Description |
|---|---|---|
| GET | `/postfix/preview` | Preview rendered config |
| GET | `/postfix/install-script` | **Generate shell script for curl-pipe deployment** |
| POST | `/postfix/agents/:id/push` | Push config to agent (stage) |
| POST | `/postfix/agents/:id/apply` | Apply config on agent |
| POST | `/postfix/agents/:id/rollback` | Rollback config on agent |
| POST | `/postfix/agents/:id/push-and-apply` | Push and apply in one step |

### 4.5 Status & Logs

| Method | Path | Description |
|---|---|---|
| GET | `/postfix/agents/:id/status` | Check agent live status |
| GET | `/postfix/agents/:id/logs` | List delivery logs |
| GET | `/postfix/status` | Config status summary across all agents |

### 4.6 Queue Management (via Agent)

| Method | Path | Description |
|---|---|---|
| GET | `/postfix/queue/agents/:id` | List queue messages |
| GET | `/postfix/queue/agents/:id/stats` | Queue statistics |
| POST | `/postfix/queue/agents/:id/delete` | Delete messages |
| POST | `/postfix/queue/agents/:id/resend` | Resend messages |
| POST | `/postfix/queue/agents/:id/flush` | Flush queue |

---

## 5. Bootstrap Parameters

On startup with `auto_migrate: true`, EasyMail seeds these default Postfix parameters into the `postfix_configs` table (upsert by `param_name`):

| Param Name | Default Value | Managed |
|---|---|---|
| `smtpd_relay_restrictions` | `permit_mynetworks permit_sasl_authenticated defer_unauth_destination` | ✅ |
| `mailbox_size_limit` | `102400000` (100MB) | ❌ (user-editable) |
| `smtpd_sasl_type` | `dovecot` | ✅ |
| `smtpd_sasl_path` | `inet:${dovecot.listen}` | ✅ |
| `smtpd_sasl_auth_enable` | `yes` | ✅ |
| `smtpd_recipient_restrictions` | `permit_mynetworks, permit_sasl_authenticated, reject_unauth_destination` | ✅ |
| `smtpd_sasl_security_options` | `noanonymous` | ✅ |
| `smtpd_sasl_tls_security_options` | `noanonymous` | ✅ |
| `local_recipient_maps` | (empty — disable check) | ✅ |
| `virtual_mailbox_domains` | `tcp:${dovecot.listen}` | ✅ |
| `virtual_mailbox_maps` | `tcp:${dovecot.listen}` | ✅ |
| `smtpd_milters` | `inet:${milter.listen}` | ✅ |
| `milter_default_action` | `accept` | ✅ |
| `virtual_transport` | `lmtp:inet:${lmtp.listen}` | ✅ |

Managed params are read-only in the admin UI. User-defined params can be freely added, modified, or deleted.

---

## 6. Source Map

| Area | Package/Directory |
|---|---|
| Postfix-agent binary | `easymail/cmd/postfix-agent/main.go` |
| Postfix domain entities | `easymail/internal/domain/management/postfix_agent.go` |
| Postfix config domain entities | `easymail/internal/domain/management/postfix_config.go` |
| Config service (server side) | `easymail/internal/app/management/postfix_config_service.go` |
| Admin HTTP handlers | `easymail/internal/portal/admin/handler/postfix_handler.go` |
| Admin router routes | `easymail/internal/portal/admin/router.go` |
| PO / Repository (MySQL) | `easymail/internal/infrastructure/persistence/mysql/postfix_po.go` |
| Repository implementation | `easymail/internal/infrastructure/persistence/mysql/postfix_repo.go` |
| Bootstrap migration | `easymail/internal/infrastructure/migrate/postfix_bootstrap.go` |
| Postfix config (YAML) | `easymail/pkg/config/config.go` (PostfixConfig struct) |
| Milter protocol server | `easymail/internal/protocol/milter/` |
| Milter handler (adapter) | `easymail/internal/adapter/milter/` |
| LMTP protocol server | `easymail/internal/protocol/lmtp/` |
| LMTP handler (adapter) | `easymail/internal/adapter/lmtp/` |