# EasyMail Postfix 集成

EasyMail 不直接处理 SMTP 入站连接。它依赖 **Postfix** 作为边缘 MTA 从外部发件人接收邮件。EasyMail 充当反垃圾网关和邮件存储后端，为最终用户提供 Webmail 和 IMAP 访问。

---

## 1. Postfix + EasyMail 关系

### 1.1 角色分离

```
外部互联网
      │
      ▼  SMTP (端口 25)
┌─────────────┐
│   Postfix   │  MTA — 从外部服务器接收邮件，
│  (MTA/SMTP) │         处理 SMTP 会话、队列、重试
└──────┬──────┘
       │
       ├──────────────────────────────────────────────────────┐
       │ Milter 协议 (端口 10026)                              │
       │   → EasyMail 反垃圾过滤器决策：                       │
       │     accept / reject / spam / quarantine              │
       │                                                      │
       │ LMTP 投递 (端口 10027)                               │
       │   → EasyMail 存储到磁盘 + SQLite 索引                │
       │                                                      │
       │ Dovecot 协议代理 (端口 10025)                         │
       │   → 域名查询、邮箱查询、SASL 认证                    │
       │                                                      │
       │ SMTP 提交 (端口 587)                                 │
       │   → 经 SASL 认证的用户发送外发邮件                   │
       └──────────────────────────────────────────────────────┘
                               │
                               ▼
                     ┌─────────────────┐
                     │   EasyMail      │  反垃圾网关 +
                     │                 │  邮件存储
                     │  ┌───────────┐  │
                     │  │ Webmail   │  │  通过浏览器读/发邮件
                     │  ├───────────┤  │
                     │  │ IMAP      │  │  通过邮件客户端读邮件
                     │  ├───────────┤  │
                     │  │ LMTP      │  │  从 Postfix 接收邮件
                     │  ├───────────┤  │
                     │  │ Milter    │  │  反垃圾过滤
                     │  └───────────┘  │
                     └─────────────────┘
```

### 1.2 服务端口

| 端口 | 协议 | 方向 | 用途 |
|---|---|---|---|
| 25 | SMTP | 入站 | Postfix 接收外部邮件 |
| 587 | SMTP (STARTTLS) | 入站 | Postfix 接收用户提交的邮件 |
| 10025 | TCP (Dovecot 代理) | EasyMail → Postfix | 域名/邮箱查询、SASL 认证 |
| 10026 | TCP (Milter) | EasyMail → Postfix | 反垃圾过滤决策 |
| 10027 | TCP (LMTP) | EasyMail → Postfix | 最终邮件投递 + 存储 |
| 443/8080 | HTTPS/HTTP | 用户 → EasyMail | 管理面板、Webmail |
| 143/993 | IMAP/IMAPS | 用户 → EasyMail | 通过邮件客户端读邮件 |

### 1.3 邮件流程总结

1. 外部发件人将邮件投递到 **Postfix**（端口 25）
2. 在 SMTP 会话期间，Postfix 在每个阶段（Connect → Helo → MailFrom → RcptTo → Headers → Body）调用 **EasyMail Milter**（端口 10026）以获取反垃圾决策
3. 如果 milter 拒绝邮件，Postfix 向发件人返回 5xx SMTP 错误
4. 如果被接受，Postfix 将邮件投递到 **EasyMail LMTP**（端口 10027）
5. EasyMail 读取 milter 注入的 `X-EasyMail-Filter-Action` 头部，将邮件路由到相应的文件夹（收件箱 / 垃圾邮件 / 隔离）
6. 用户通过 **Webmail** 或 **IMAP** 获取邮件

### 1.4 配置生成（服务端）

EasyMail 管理面板通过 `PostfixConfigService`（`easymail/internal/app/management/postfix_config_service.go`）生成 Postfix 配置。工作流程如下：

1. **参数**：存储在 `postfix_configs` 数据库表中。系统设置的管理参数为只读；用户定义的参数完全可编辑。
2. **变量解析**：参数值支持 `${section.field}` 变量引用，从 EasyMail 的运行配置中解析：

   | 变量 | 解析后的值 |
   |---|---|
   | `${dovecot.listen}` | Dovecot 代理监听地址 |
   | `${lmtp.listen}` | LMTP 监听地址 |
   | `${milter.listen}` | Milter 监听地址 |
   | `${imap.listen}` | IMAP 监听地址 |
   | `${admin.listen}` | 管理 HTTP 监听地址 |
   | `${webmail.listen}` | Webmail HTTP 监听地址 |
   | `${dovecot.family}` | Dovecot 代理套接字类型（tcp/unix） |
   | `${lmtp.family}` | LMTP 套接字类型 |
   | `${milter.family}` | Milter 套接字类型 |
   | `${imap.family}` | IMAP 套接字类型 |
   | `${postfix.host}` | Postfix EasyMail 主机设置 |
   | `${storage.root}` | 邮件存储根路径 |

   当服务监听 `0.0.0.0:N` 时，地址会自动解析为 `<postfix.host>:N`，以便 Postfix 能够访问正确的 IP。

3. **域名自动同步**：`virtual_mailbox_domains` 参数自动从 EasyMail 的活跃域名列表填充。
4. **配置模板**：参数被渲染到 `main.cf` 模板中：

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

### 2.1 概述

**Postfix-agent** 是一个独立的 HTTP 守护进程，运行在每个 Postfix 服务器上。它在 EasyMail 管理面板和本地 Postfix 实例之间充当桥梁，实现远程配置管理和队列操作。

```
Postfix 主机 (192.168.1.10)         EasyMail 服务器 (192.168.1.1)
┌─────────────────────────┐         ┌──────────────────────────┐
│  postfix-agent (:8081)  │ ◄─────►│  管理面板 + API          │
│                         │  HTTP   │  /api/v1/admin/postfix/* │
│  认证: X-Agent-Token    │         │                          │
│                         │         │  ┌──────────────────┐    │
│  ┌───────────────────┐  │         │  │ PostfixConfig    │    │
│  │ Postfix (/etc/    │  │         │  │ Service          │    │
│  │ postfix/main.cf)  │  │         │  │                  │    │
│  │                   │  │         │  │ 生成配置         │    │
│  │ postqueue -j      │  │         │  │ 解析 ${vars}     │    │
│  │ postsuper -d/-r   │  │         │  │ 推送到 agent     │    │
│  │ postfix reload    │  │         │  └──────────────────┘    │
│  └───────────────────┘  │         └──────────────────────────┘
└─────────────────────────┘
```

### 2.2 部署

#### 二进制文件

`postfix-agent` 二进制文件从 `easymail/cmd/postfix-agent/main.go` 构建。它是一个独立的二进制文件，没有外部依赖。

```
# 在 Postfix 服务器上
./postfix-agent \
  --listen=:8081 \
  --token=ema_xxxxxxxxxxxx \
  --postfix-dir=/etc/postfix \
  --staging-dir=/tmp/easymail-staging \
  --backup-dir=/etc/postfix/backups \
  --log-dir=/var/log/easymail-agent \
  --allowed-ips=192.168.1.0/24,10.0.0.0/8
```

#### 命令行标志 / 环境变量

| 标志 | 环境变量 | 默认值 | 描述 |
|---|---|---|---|
| `--listen` | `LISTEN_ADDR` | `:8081` | Agent HTTP 监听地址 |
| `--token` | `AGENT_TOKEN` | (必需) | 预共享认证令牌 |
| `--postfix-dir` | `POSTFIX_DIR` | `/etc/postfix` | Postfix 配置目录 |
| `--staging-dir` | `STAGING_DIR` | `/tmp/easymail-staging` | 配置的暂存目录 |
| `--backup-dir` | `BACKUP_DIR` | `/etc/postfix/backups` | 回滚的备份目录 |
| `--log-dir` | `LOG_DIR` | `/var/log/easymail-agent` | 会话日志目录 |
| `--allowed-ips` | `ALLOWED_IPS` | (所有 IP) | 逗号分隔的 IP/CIDR 白名单 |

#### systemd 服务

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

### 2.3 Agent API 端点

所有端点通过 `X-Agent-Token` 头部验证。带有 `ipAllowMiddleware` 的端点还会检查连接 IP 是否在配置的白名单中。

#### 状态

```
GET /api/v1/agent/status
```

返回 Postfix 进程状态、配置哈希、版本和 agent 运行时间。

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

| 字段 | 描述 |
|---|---|
| `postfixRunning` | Postfix 是否正在运行（检查 systemctl、postfix status、ps） |
| `configHash` | Postfix 配置目录中当前 `easymail.cf` 的 SHA-256 值 |
| `postfixVersion` | `postconf -d mail_version` 的输出 |
| `agentVersion` | 内置的 agent 版本字符串 |
| `uptime` | Agent 进程运行时间 |

#### 配置管理

```
POST /api/v1/agent/config/push

请求:
{
  "mainCf": "main.cf 的内容..."
}

响应: {"status": "staged"}
```

将接收到的配置暂存到暂存目录。写入 `easymail.cf` 和合并后的 `main.cf` 用于验证。配置缓存在内存中，供后续的 apply 步骤使用。

```
POST /api/v1/agent/config/apply

响应: {"status": "applied", "backup": "20260712_150405"}
```

将暂存的配置应用到 Postfix：

1. 重新写入暂存文件以确保安全
2. 运行 `postfix -c <staging> check` 验证新配置
3. 备份当前 `easymail.cf` 到 `/etc/postfix/backups/<timestamp>/`
4. 将新的 `easymail.cf` 写入 Postfix 配置目录（事实来源）
5. 通过 `postconf -e "<param> = <value>"` 应用每个参数
6. 运行 `postfix reload` 激活新配置
7. 清理旧备份（保留最近 10 个）

```
POST /api/v1/agent/config/rollback

响应: {"status": "rolled_back", "backup": "20260712_150405"}
```

回滚到最近的备份：

1. 找到最新的备份目录
2. 通过 `postfix -c` 检查验证备份配置
3. 从备份还原 `easymail.cf`
4. 通过 `postconf -e` 应用备份参数
5. 运行 `postfix reload`

#### 队列管理

```
GET /api/v1/agent/queue/list?status=deferred&sender=&recipient=&queueId=&page=1&pageSize=100

响应:
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

列出 Postfix 队列中的消息。内部使用 `postqueue -j`（JSON 输出）。支持按状态、发件人、收件人和队列 ID 过滤。`status` 字段从 Postfix 队列名称映射：

| Postfix 队列 | `status` 值 |
|---|---|
| `active` | `active` |
| `deferred` | `deferred` |
| `hold` | `held` |
| `incoming` | `incoming` |

```
GET /api/v1/agent/queue/stats

响应:
{
  "total": 150,
  "active": 10,
  "deferred": 135,
  "held": 5
}
```

返回聚合队列统计信息。

```
POST /api/v1/agent/queue/delete

请求: {"messageIds": ["AABBCCDD", "EEFFGGHH"]}
响应: {"status": "deleted"}
```

使用 `postsuper -d <queue_id> [queue_id...]` 从队列中删除指定消息。

```
POST /api/v1/agent/queue/resend

请求: {"messageIds": ["AABBCCDD"]}
响应: {"status": "resent"}
```

使用 `postsuper -r <queue_id> [queue_id...]` 重新排队指定消息。

```
POST /api/v1/agent/queue/flush

响应: {"status": "flushed"}
```

使用 `postqueue -f` 触发所有延迟邮件的立即重试。

### 2.4 Agent 配置状态机

```
                        ┌─────────┐
                        │  IDLE   │
                        └────┬────┘
                             │
                   POST /config/push
                             │
                             ▼
                        ┌─────────┐
                        │ STAGED  │  配置已写入暂存目录，
                        └────┬────┘  缓存在内存中
                             │
                   POST /config/apply
                             │
                             ▼
                  ┌─────────────────────┐
                  │  验证 (postfix      │── 失败 → 错误响应，保持 STAGED
                  │  -c check)          │
                  └──────────┬──────────┘
                             │ 成功
                             ▼
                  ┌─────────────────────┐
                  │  备份 + 写入        │
                  │  easymail.cf        │
                  └──────────┬──────────┘
                             │
                             ▼
                  ┌─────────────────────┐
                  │  postconf -e        │── 失败 → 错误响应，配置不变
                  └──────────┬──────────┘
                             │
                             ▼
                  ┌─────────────────────┐
                  │  postfix reload     │── 失败 → 错误，之前的配置仍有效
                  └──────────┬──────────┘
                             │ 成功
                             ▼
                        ┌─────────┐
                        │  IDLE   │
                        └─────────┘
```

回滚通过重复相同的管道（验证 → 写入 → postconf -e → 重新加载）来恢复最后的备份。

### 2.5 安全性

- **令牌认证**：对 agent 的每个请求都必须包含与启动时设置的预共享令牌匹配的 `X-Agent-Token` 头部。
- **IP 白名单**：可选的基于 CIDR 的访问控制，限制哪些 EasyMail 服务器可以访问 agent。
- **会话日志记录**：所有请求（包括被拒绝的）都会记录到 `sessions.log`，包含 IP、方法、路径、状态码和持续时间。
- **应用前备份**：Agent 在应用更改之前始终创建 `easymail.cf` 的带时间戳备份，从而实现安全的回滚。

---

## 3. Shell 脚本部署（无 Agent）

对于无法部署 postfix-agent 二进制文件的环境，EasyMail 可以生成一个自包含的 shell 脚本，应用相同的 Postfix 配置。该脚本通过专用的 API 端点提供，可以在目标 Postfix 服务器上直接通过管道传给 `sh`。

### 3.1 使用方法

在 Postfix 服务器上运行：

```bash
curl -s http://<easymail-admin>:8080/api/v1/admin/postfix/install-script | sudo sh
```

或者先保存并检查：

```bash
curl -s http://<easymail-admin>:8080/api/v1/admin/postfix/install-script > install.sh
chmod +x install.sh
sudo ./install.sh
```

环境变量可以覆盖默认值：

```bash
# 自定义 Postfix 目录（默认：/etc/postfix）
POSTFIX_DIR=/etc/postfix \
BACKUP_DIR=/var/backups/postfix \
curl -s http://<easymail-admin>:8080/api/v1/admin/postfix/install-script | sudo sh
```

### 3.2 API 端点

| 方法 | 路径 | 描述 |
|---|---|---|
| GET | `/api/v1/admin/postfix/install-script` | 生成手动配置 Postfix 的 shell 脚本 |

端点返回 `Content-Type: text/x-shellscript` 并带有 `Content-Disposition: attachment` 头部，适合直接管道传输。

### 3.3 脚本行为

生成的脚本执行以下步骤：

1. **Root 检查**：验证脚本以 root 身份运行
2. **备份**：在 `/etc/postfix/backups/<timestamp>/` 中创建当前 `easymail.cf` 的带时间戳备份
3. **写入配置**：通过 heredoc 将渲染后的 `easymail.cf` 写入 Postfix 配置目录
4. **应用参数**：为每个管理参数运行 `postconf -e '<param> = <value>'`
5. **验证**：运行 `postfix check` 验证配置
6. **重新加载**：运行 `postfix reload` 激活配置
7. **清理**：删除超过最近 10 个的旧备份

该脚本是幂等的——多次运行产生相同的结果。它设置了 `set -e`，因此任何错误都会快速失败。

### 3.4 脚本输出示例

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

### 3.5 对比：Agent vs. Shell 脚本

| 方面 | postfix-agent | Shell 脚本 |
|---|---|---|
| **部署** | 二进制文件、systemd 服务 | 零安装（curl 管道） |
| **配置交付** | 管理面板 HTTP 推送 | 从 Postfix 服务器通过 curl 拉取 |
| **回滚** | 自动备份 | 手动（从备份目录恢复） |
| **队列管理** | 完整的远程队列操作 | 不可用 |
| **状态监控** | 实时状态轮询 | 不可用 |
| **多服务器** | 同时推送到所有 agent | 每个服务器分别运行脚本 |
| **安全性** | 令牌 + IP 白名单 | 取决于网络访问 |
| **最适合** | 生产部署 | 快速设置、测试、自动化 |

---

## 4. 管理面板 API 端点（服务端）

EasyMail 管理服务器在 `/api/v1/admin/` 下暴露以下端点用于 Postfix 配置管理。

### 4.1 Agent 管理

| 方法 | 路径 | 描述 |
|---|---|---|
| GET | `/postfix/agents` | 列出 agent（`?keyword`、`?page`、`?pageSize`） |
| GET | `/postfix/agents/:id` | 获取 agent 详情 |
| POST | `/postfix/agents` | 创建 agent |
| PUT | `/postfix/agents/:id` | 更新 agent |
| DELETE | `/postfix/agents/:id` | 删除 agent |

### 4.2 配置参数

| 方法 | 路径 | 描述 |
|---|---|---|
| GET | `/postfix/configs` | 列出配置参数（`?keyword`、`?page`、`?pageSize`） |
| GET | `/postfix/configs/:id` | 获取配置参数 |
| POST | `/postfix/configs` | 创建配置参数 |
| PUT | `/postfix/configs/:id` | 更新配置参数 |
| DELETE | `/postfix/configs/:id` | 删除配置参数 |

### 4.3 设置与变量

| 方法 | 路径 | 描述 |
|---|---|---|
| GET | `/postfix/settings` | 获取全局 Postfix 设置（EasyMailHost） |
| PUT | `/postfix/settings` | 更新全局设置 |
| GET | `/postfix/variables` | 列出可用的 `${section.field}` 变量 |
| GET | `/postfix/local-ips` | 获取本地 IP 地址 |

### 4.4 配置生成与交付

| 方法 | 路径 | 描述 |
|---|---|---|
| GET | `/postfix/preview` | 预览渲染后的配置 |
| GET | `/postfix/install-script` | **生成用于 curl 管道部署的 shell 脚本** |
| POST | `/postfix/agents/:id/push` | 推送配置到 agent（暂存） |
| POST | `/postfix/agents/:id/apply` | 在 agent 上应用配置 |
| POST | `/postfix/agents/:id/rollback` | 在 agent 上回滚配置 |
| POST | `/postfix/agents/:id/push-and-apply` | 一步推送并应用 |

### 4.5 状态与日志

| 方法 | 路径 | 描述 |
|---|---|---|
| GET | `/postfix/agents/:id/status` | 检查 agent 实时状态 |
| GET | `/postfix/agents/:id/logs` | 列出交付日志 |
| GET | `/postfix/status` | 所有 agent 的配置状态摘要 |

### 4.6 队列管理（通过 Agent）

| 方法 | 路径 | 描述 |
|---|---|---|
| GET | `/postfix/queue/agents/:id` | 列出队列消息 |
| GET | `/postfix/queue/agents/:id/stats` | 队列统计 |
| POST | `/postfix/queue/agents/:id/delete` | 删除消息 |
| POST | `/postfix/queue/agents/:id/resend` | 重新发送消息 |
| POST | `/postfix/queue/agents/:id/flush` | 刷新队列 |

---

## 5. 启动参数

当 `auto_migrate: true` 时启动，EasyMail 将这些默认 Postfix 参数种子到 `postfix_configs` 表中（按 `param_name` 更新插入）：

| 参数名称 | 默认值 | 管理 |
|---|---|---|
| `smtpd_relay_restrictions` | `permit_mynetworks permit_sasl_authenticated defer_unauth_destination` | ✅ |
| `mailbox_size_limit` | `102400000` (100MB) | ❌ (用户可编辑) |
| `smtpd_sasl_type` | `dovecot` | ✅ |
| `smtpd_sasl_path` | `inet:${dovecot.listen}` | ✅ |
| `smtpd_sasl_auth_enable` | `yes` | ✅ |
| `smtpd_recipient_restrictions` | `permit_mynetworks, permit_sasl_authenticated, reject_unauth_destination` | ✅ |
| `smtpd_sasl_security_options` | `noanonymous` | ✅ |
| `smtpd_sasl_tls_security_options` | `noanonymous` | ✅ |
| `local_recipient_maps` | (空 — 禁用检查) | ✅ |
| `virtual_mailbox_domains` | `tcp:${dovecot.listen}` | ✅ |
| `virtual_mailbox_maps` | `tcp:${dovecot.listen}` | ✅ |
| `smtpd_milters` | `inet:${milter.listen}` | ✅ |
| `milter_default_action` | `accept` | ✅ |
| `virtual_transport` | `lmtp:inet:${lmtp.listen}` | ✅ |

管理参数在管理 UI 中为只读。用户定义的参数可以自由添加、修改或删除。

---

## 6. 源码目录映射

| 区域 | 包/目录 |
|---|---|
| Postfix-agent 二进制文件 | `easymail/cmd/postfix-agent/main.go` |
| Postfix 领域实体 | `easymail/internal/domain/management/postfix_agent.go` |
| Postfix 配置领域实体 | `easymail/internal/domain/management/postfix_config.go` |
| 配置服务（服务端） | `easymail/internal/app/management/postfix_config_service.go` |
| 管理 HTTP 处理器 | `easymail/internal/portal/admin/handler/postfix_handler.go` |
| 管理路由 | `easymail/internal/portal/admin/router.go` |
| PO / 仓库 (MySQL) | `easymail/internal/infrastructure/persistence/mysql/postfix_po.go` |
| 仓库实现 | `easymail/internal/infrastructure/persistence/mysql/postfix_repo.go` |
| 启动迁移 | `easymail/internal/infrastructure/migrate/postfix_bootstrap.go` |
| Postfix 配置 (YAML) | `easymail/pkg/config/config.go` (PostfixConfig 结构体) |
| Milter 协议服务器 | `easymail/internal/protocol/milter/` |
| Milter 处理器（适配器） | `easymail/internal/adapter/milter/` |
| LMTP 协议服务器 | `easymail/internal/protocol/lmtp/` |
| LMTP 处理器（适配器） | `easymail/internal/adapter/lmtp/` |