# EasyMail 反垃圾技术

EasyMail 集成了多层反垃圾系统，运行在 SMTP Milter（Milter 协议）层面。它结合了实时特征提取、基于 JSON 的规则引擎、自定义/正则/组合特征、ML 分类模型（进程内 FastText、gRPC 上的 DistilBERT）以及反病毒扫描（ClamAV），在邮件到达用户收件箱之前对其进行分类并执行相应操作。

---

## 1. 内置特征

特征是在 SMTP 事务每个阶段提取的数值。它们是规则评估的输入信号。提取器在可配置的超时预算内并发运行。

### 1.1 提取管道

邮件流程分为六个 Milter 阶段。提取器注册到一个或多个阶段，并在相应的 SMTP 事件发生时被调用。

```
Connect -> Helo -> MailFrom -> RcptTo -> Headers -> Body
```

### 1.2 Connect 阶段

| 提取器键 | 特征 | 描述 |
|---|---|---|
| `connect_rdns` | `ip_ptr_ok`、`ip_ptr_count`、`ip_ptr_has_dot` | 对连接 IP 进行反向 DNS 查询。检查 PTR 记录是否存在、其数量以及是否包含点（完全限定域名）。 |
| `connect_fcrdns` | `ip_fcrdns_ok`、`ip_ptr_forward_match_count` | 前向确认的反向 DNS：执行 PTR 查询，然后验证返回主机名的至少一条前向（A/AAAA）记录与原始连接 IP 匹配。 |
| `stats_connect` | `ip_conn_5m`、`ip_conn_1d`、`ip_accept_5m`、`ip_reject_5m`、`ip_spam_5m`、`ip_accept_1d`、`ip_reject_1d`、`ip_spam_1d` | 基于 Redis 的每个连接 IP 的连接速率和历史结果计数器（5 分钟滑动窗口和每日日历窗口）。 |

### 1.3 Helo 阶段

| 提取器键 | 特征 | 描述 |
|---|---|---|
| `helo_rdns_match` | `helo_present`、`helo_rdns_match`、`helo_rdns_name_count` | 将 HELO/EHLO 主机名与连接 IP 的 PTR 记录进行比较。包括精确匹配、子域匹配和域后缀匹配。 |

### 1.4 MailFrom 阶段

| 提取器键 | 特征 | 描述 |
|---|---|---|
| `spf` | `spf_result`、`spf_pass`、`spf_fail`、`spf_softfail`、`spf_neutral`、`spf_none`、`spf_error`、`spf_skipped` | SPF（发件人策略框架）检查，遵循 RFC 7208。使用 `blitiri.com.ar/go/spf` 库。查询发件人域的 SPF TXT 记录，评估连接 IP 是否与公布的策略一致。七种结果码：None (0)、Neutral (1)、Pass (2)、SoftFail (3)、Fail (4)、TempError (5)、PermError (6)。 |
| `mailfrom_domain_dns` | `sender_domain_has_mx`、`sender_domain_has_a`、`sender_domain_mx_count`、`sender_domain_mx_pref_min`、`sender_domain_mx_is_null`、`sender_domain_a_count` | 发件人域的 DNS 健康检查。验证该域是否有 MX 和/或 A 记录，对其计数，并检测空 MX（`.`），这表示该域不接受邮件。 |
| `stats_mailfrom` | `sender_conn_5m`、`sender_conn_1d`、`sender_accept_5m`、`sender_reject_5m`、`sender_spam_5m`、`sender_accept_1d`、`sender_reject_1d`、`sender_spam_1d` | 通过 Redis 实现的每个发件人（信封 MAIL FROM）速率和结果计数器。 |

### 1.5 RcptTo 阶段

| 提取器键 | 特征 | 描述 |
|---|---|---|
| `rcpt_local` | `rcpt_domain_is_local`、`rcpt_mailbox_exists` | 检查收件人域是否为本地/已配置域，以及邮箱是否存在。依赖数据库的检查委托给应用层。 |
| `rcpt_contact_hit` | `rcpt_contact_sender_hit` | 检查发件人是否在收件人的地址簿中。依赖数据库。 |
| `stats_rcpt` | 每个收件人的计数器 | 基于 Redis 的每个收件人速率计数器。 |
| `stats_rcpt_domain` | 每个域的计数器 | 基于 Redis 的每个收件人域速率计数器。 |

### 1.6 Headers 阶段

| 提取器键 | 特征 | 描述 |
|---|---|---|
| （在 `Header()` 回调中内联） | `header_received`、`header_dkim_signature`、`header_authentication_results`、`header_content_type_multipart`、`header_has_message_id`、`header_has_date`、`header_list_id`、`header_list_unsubscribe`、`header_custom_count`、`header_fields` | 无需 MIME 解析的轻量级头部跟踪。统计 Received 头的数量、检测 DKIM-Signature 的存在、Content-Type multipart、List-* 头以及自定义 X-* 头的数量。 |

### 1.7 Body 阶段

| 提取器键 | 特征 | 描述 |
|---|---|---|
| `message_baseline` | `body_bytes`、`subject_len`、`has_list_unsubscribe`、`mime_part_count`、`attachment_count`、`rcpt_count` | 通过 `enmime` 进行结构化消息分析。解析完整邮件、统计 MIME 部分和附件数量。当 enmime 解析失败时回退到仅头部分析。 |
| `dkim` | `dkim_result`、`dkim_pass`、`dkim_fail`、`dkim_sig_count`、`dkim_pass_count`、`dkim_fail_count`、`dkim_verified_count`、`dkim_skipped` | DKIM（域名密钥识别邮件）签名验证，遵循 RFC 6376。解析 DKIM-Signature 头，通过 `_domainkey.<domain>` DNS TXT 记录查找选择器的公钥，使用 RSA/SHA-256 验证正文哈希和头部签名。支持每条消息多个 DKIM 签名。 |
| `dmarc` | `dmarc_result`、`dmarc_pass`、`dmarc_fail`、`dmarc_has_policy`、`dmarc_policy_none`、`dmarc_policy_quarantine`、`dmarc_policy_reject`、`dmarc_policy_code`、`dmarc_spf_aligned`、`dmarc_dkim_aligned`、`dmarc_spf_domain_match`、`dmarc_dkim_domain_match` | DMARC（基于域的消息认证、报告与一致性）策略评估，遵循 RFC 7489。查询 `_dmarc.<domain>` TXT 记录，解析策略（`p=`），使用宽松模式（通过公共后缀列表的组织域）检查 SPF 对齐（MAIL FROM 域与 From 头域）和 DKIM 对齐（`d=` 域与 From 域）。 |
| `classify_model` | 模型特定的特征（例如 `my_model`、`my_model_spam`、`my_model_ham`） | ML 分类器模型推理。支持 FastText（进程内）、DistilBERT 和其他 ONNX 模型（通过 gRPC worker）。特征被清理为 `modelName_label` 格式。详见第 4 节。 |
| `antivirus` | `antivirus_hit` | ClamAV 扫描。`1.0` 表示检测到病毒，`0.0` 表示干净，`-1.0` 表示出错。 |

#### 内容插件（Body 阶段）

| 插件键 | 特征 | 描述 |
|---|---|---|
| `url_basic` | `body_url_count`、`body_url_has_http`、`body_url_has_https`、`body_url_short_count`、`body_url_has_short` | URL 提取和分类。扫描邮件正文中的 HTTP/HTTPS URL，统计数量，基于 URL 形态分析（短主机 + 紧凑 slug）检测短 URL（如 t.co、bit.ly 等服务）。 |
| `attachment_risk` | `attachment_count`、`attachment_risky_count`、`attachment_double_ext_count`、`attachment_has_risky`、`attachment_has_double_ext` | 附件风险评估。识别可执行/脚本文件扩展名（`.exe`、`.dll`、`.scr`、`.bat`、`.ps1`、`.vbs`、`.js`、`.jar`、`.lnk`、`.iso`、`.hta`、`.msi`、`.reg`、`.wsf`）。检测双扩展名模式（例如 `invoice.pdf.exe`）。 |

### 1.8 DNS 基础设施

特征提取器使用共享的 DNS 解析器（`easymail/internal/infrastructure/easydns`），支持：

- 自定义 DNS 服务器配置
- 查询方法：LookupAddr (PTR)、LookupMX、LookupIPAddr (A/AAAA)、LookupTXT
- 基于上下文的所有查询超时
- 在未指定自定义服务器时回退到系统配置的 DNS

---

## 2. 自定义特征

除了内置提取器外，管理员可以通过管理 Web UI 定义自定义特征。这些特征存储在 `custom_features` 数据库表中（而不是 YAML 配置），并在每个管道阶段与内置特征一起评估。

### 2.1 自定义特征模式

| 字段 | 描述 |
|---|---|
| `id` | 自增主键 |
| `feature_key` | 特征的唯一键名 |
| `label` | 人类可读的显示名称 |
| `stage` | 该特征针对的管道阶段 |
| `type` | `meta_regex` 或 `composite` |
| `value_type` | 输出值的数据类型 |
| `enabled` | 此自定义特征是否激活 |
| `spec_json` | JSON 规范（因类型而异） |
| `description` | 人类可读的描述 |
| `unit` | 度量单位 |

### 2.2 元正则表达式特征（`meta_regex`）

使用正则表达式扫描邮件消息或会话元数据的特定部分，并输出布尔值或计数特征。

#### 规范字段

| 字段 | 描述 |
|---|---|
| `sources` | 要扫描的源字符串数组。支持的源：`connect_ip`、`mail_from`、`rcpt`、`subject`、`header_from_email`、`header_from_name`、`body`、`url_list`、`attachment_names` |
| `pattern` | 正则表达式模式 |
| `flags` | 正则标志：`i`（不区分大小写）、`m`（多行）、`s`（点号匹配所有） |
| `mode` | `any`（只要一个源匹配即为真）或 `all`（仅当所有源都匹配时才为真） |
| `emit` | `bool_hit`（输出 0/1）或 `count`（输出匹配次数） |

#### 按阶段划分的源可用性

| 源 | 最早阶段 | 内容 |
|---|---|---|
| `connect_ip` | Connect | 连接 IP 地址字符串 |
| `mail_from` | MailFrom | 信封 MAIL FROM |
| `rcpt` | RcptTo | 第一个信封 RCPT TO |
| `subject` | Headers | 解码后的主题行（RFC 2047） |
| `header_from_email` | Headers | From 头中的电子邮件地址（已解析） |
| `header_from_name` | Headers | From 头中的显示名称（已解析） |
| `body` | Body | 完整邮件正文文本（截断至 1 MiB） |
| `url_list` | Body | 从邮件正文中提取的 URL |
| `attachment_names` | Body | 来自 MIME 解析的附件文件名 |

#### 示例：检测钓鱼邮件主题模式

```json
{
  "sources": ["subject"],
  "pattern": "(?i)(urgent|account.*suspend|verify.*identity|security.*alert)",
  "flags": "",
  "mode": "any",
  "emit": "bool_hit"
}
```

如果主题匹配任何模式，则产生特征 `phishing_subject_hit` 值为 `1.0`，否则为 `0.0`。

#### 示例：统计可疑 URL 数量

```json
{
  "sources": ["url_list"],
  "pattern": "(?i)(bit\\.ly|tinyurl\\.com|short\\.link)",
  "flags": "",
  "mode": "all",
  "emit": "count"
}
```

统计邮件正文中发现的短 URL 数量。

### 2.3 组合特征（`composite`）

使用与规则引擎相同的条件 AST 组合多个现有特征（包括内置特征和其他自定义特征），产生布尔结果。

#### 规范字段

| 字段 | 描述 |
|---|---|
| `condition_json` | JSON 条件树（与规则相同的 AST 格式） |
| `emit` | 始终为 `"bool"` |

#### 示例：高风险发件人带可疑内容

```json
{
  "condition_json": {
    "op": "and",
    "children": [
      { "op": "feat", "feature": "ip_fcrdns_ok" },
      { "op": "cmp", "feature": "body_url_count", "kind": "gt", "value": 5 },
      { "op": "cmp", "feature": "spf_result", "kind": "ne", "value": 2 }
    ]
  },
  "emit": "bool"
}
```

组合特征可以引用其他组合特征，从而实现级联/链式评估。系统使用迭代不动点计算来解析组合特征之间的传递依赖关系。

### 2.4 自定义特征生命周期

1. **创建**：通过管理 UI 定义 → 存储在 `custom_features` 表中
2. **编译**：运行时从数据库加载特征并进行编译：
   - `meta_regex`：模式编译为 `regexp.Regexp`
   - `composite`：验证条件 JSON
3. **阶段分配**：系统根据源依赖关系（对于 meta_regex）和引用特征的阶段（对于 composite）自动确定所需的最早管道阶段
4. **评估**：在每个管道阶段，`applyCustomFeaturesForStage()` 运行：
   - 第一轮：内置提取器完成后
   - 第二轮（仅 Body 阶段）：内容插件完成后
5. **缓存失效**：数据库结果缓存有 10 秒的 TTL；管理的增删改操作调用 `InvalidateCustomFeatureDefsCache()` 立即清除缓存

---

## 3. 规则系统

规则定义反垃圾策略。每条规则都有一个 JSON 条件树、目标阶段、动作、优先级和启用/禁用标志。规则缓存在内存中并按阶段评估。

### 3.1 规则模式

```json
{
  "id": 1,
  "name": "拒绝缺少 SPF 的邮件",
  "enabled": true,
  "priority": 100,
  "stage": "mailfrom",
  "action": "reject",
  "conditionJson": "{...}"
}
```

| 字段 | 描述 |
|---|---|
| `id` | 自增主键 |
| `name` | 人类可读的规则名称 |
| `enabled` | 规则是否激活 |
| `priority` | 优先级高的规则先被评估 |
| `stage` | 此规则适用的管道阶段（可选；当条件中引用的特征不存在时自动推导） |
| `action` | 可选值：`accept`、`reject`、`spam`、`quarantine` |
| `conditionJson` | 条件的 JSON AST |

### 3.2 条件 AST

条件以 JSON 树的形式表达，包含以下节点类型。

#### 逻辑运算符

| 运算符 | 描述 | 子节点数 |
|---|---|---|
| `and` | 所有子节点必须为真 | 2+ |
| `or` | 至少一个子节点为真 | 2+ |
| `not` | 否定单个子节点 | 1 |

#### 特征运算符

| 运算符 | 描述 | 字段 |
|---|---|---|
| `feat` | 如果特征存在且值不为零则为真 | `feature`：特征键名 |
| `cmp` | 将特征值与阈值比较 | `feature`：特征键、`kind`：比较类型、`value`：数值阈值 |

#### cmp 类型

| Kind | 含义 |
|---|---|
| `eq` | 等于（`==`） |
| `ne` | 不等于（`!=`） |
| `gt` | 大于（`>`） |
| `ge` | 大于或等于（`>=`） |
| `lt` | 小于（`<`） |
| `le` | 小于或等于（`<=`） |

### 3.3 示例条件

**SPF 失败且 DMARC 策略为拒绝：**

```json
{
  "op": "and",
  "children": [
    { "op": "cmp", "feature": "spf_result", "kind": "eq", "value": 4 },
    { "op": "cmp", "feature": "dmarc_policy_code", "kind": "eq", "value": 2 }
  ]
}
```

**高连接速率且无 FCrDNS：**

```json
{
  "op": "and",
  "children": [
    { "op": "cmp", "feature": "ip_conn_5m", "kind": "gt", "value": 100 },
    { "op": "not", "children": [{ "op": "feat", "feature": "ip_fcrdns_ok" }] }
  ]
}
```

**分类器高垃圾邮件分数且附件可疑：**

```json
{
  "op": "and",
  "children": [
    { "op": "cmp", "feature": "my_model_spam", "kind": "gt", "value": 0.8 },
    { "op": "feat", "feature": "attachment_has_risky" }
  ]
}
```

**多个条件用 OR 连接：**

```json
{
  "op": "or",
  "children": [
    {
      "op": "and",
      "children": [
        { "op": "cmp", "feature": "spf_result", "kind": "eq", "value": 4 },
        { "op": "cmp", "feature": "dmarc_policy_code", "kind": "eq", "value": 2 }
      ]
    },
    {
      "op": "cmp", "feature": "antivirus_hit", "kind": "eq", "value": 1
    }
  ]
}
```

### 3.4 规则评估流程

```
对于每个管道阶段 S：
  1. 加载所有启用的规则（缓存，数据库变更时刷新）
  2. 过滤：仅匹配阶段 S 的规则（或自动推导的阶段匹配）
  3. 按优先级降序排序（数值大的先评估）
  4. 对排序后的每条规则：
     a. 将 conditionJson 反序列化为 AST（CondNode 树）
     b. 针对当前特征快照评估树
     c. 如果条件匹配 → 应用动作，存储匹配结果，跳出循环
  5. 如果没有规则匹配 → 应用 default_action（可配置，默认为 accept）
```

关键行为：
- **优先级排序**：`priority` 值越高的规则越先被评估。优先级相同时，顺序未定义（使用唯一的优先级以确保确定性）。
- **提前终止**：第一个匹配的规则获胜。后续规则不被评估。
- **阶段过滤**：规则仅在其所属的阶段进行评估。系统可以从规则条件中引用的特征自动推导规则的阶段，或者管理员可以显式设置。
- **特征快照**：在每个阶段，当前特征快照包含本阶段及之前所有阶段提取的所有特征，加上任何已评估的自定义特征。

### 3.5 动作

| 动作 | Milter 响应 | LMTP 路由 |
|---|---|---|
| `accept` | `SMFIR_CONTINUE`（放行） | 投递到收件箱 |
| `reject` | `SMFIR_REPLYCODE`（5xx SMTP 拒绝） | 不投递（除非绕过） |
| `spam` | `SMFIR_CONTINUE`（放行） | 路由到垃圾邮件文件夹 |
| `quarantine` | `SMFIR_CONTINUE`（放行） | 路由到隔离文件夹 |

### 3.6 从特征生成规则

在管理 UI 中定义规则时，所有可用特征都会列出其元数据：

| 元数据字段 | 描述 |
|---|---|
| `featureKey` | 在条件 JSON 中使用的特征标识符 |
| `origin` | `builtin`、`custom` 或 `model` |
| `builtinExtractor` | 提取器名称（用于内置特征） |
| `customFeatureId` | 自定义特征 ID（用于自定义特征） |
| `modelId` / `modelName` | 模型引用（用于模型特征） |

管理 UI 提供一个条件构建器，可以通过交互式表单控件构建 JSON AST，使管理员无需编写原始 JSON 即可组成复杂条件。

### 3.7 过滤日志

每个 milter 会话生成一条过滤日志记录，可通过管理仪表板访问。日志记录包含：

| 字段 | 描述 |
|---|---|
| `trace_id` | 会话的唯一跟踪标识符 |
| `queue_id` | Postfix 队列 ID |
| `connect_ip` | 连接 IP 地址 |
| `sender` | 信封 MAIL FROM |
| `recipient` | 信封 RCPT TO |
| `subject` | 解码后的主题 |
| `stage` | 发生匹配的管道阶段 |
| `rule_id` | 匹配规则的 ID（默认动作为 0） |
| `action` | 应用的动作（accept/reject/spam/quarantine） |
| `feature_snapshot` | 所有提取特征的完整 JSON 快照（启用 `log_feature_snapshot` 时） |
| `condition_trace` | 条件评估追踪——评估了哪些特征以及每个子条件是否匹配（启用 `log_condition_trace` 时） |
| `duration_ms` | 会话的总处理时长 |

条件追踪是一个结构化日志，显示条件树中的评估路径。例如，对于一个有两个子节点的 `and` 节点，追踪记录既记录父节点是否匹配，也记录每个子节点是否匹配，从而可以调试规则为何触发或为何不触发。

---

## 4. FastText 模型在线训练

EasyMail 为 FastText 分类模型提供了完整的在线训练工作流，包括样本管理、模型训练、模型发布和规则集成。

### 4.1 架构概览

```
Admin UI                    Milter Runtime
    |                            |
    v                            v
+-------------------+    +-------------------+
| 样本管理          |    | 模型缓存          |
| (model_samples)   |    | (按需打开)        |
+--------+----------+    +--------+----------+
         |                        |
         v                        v
+-------------------+    +-------------------+
| 训练引擎          |    | 特征提取器        |
| (fasttext CLI)    |    | (classify_model)  |
+--------+----------+    +--------+----------+
         |                        |
         v                        v
+-------------------+    +-------------------+
| 模型文件 (.bin)   |    | 规则引擎          |
| (磁盘存储)        |    | (条件评估)        |
+-------------------+    +-------------------+
```

### 4.2 数据库表

#### classify_models — 模型定义

| 列 | 类型 | 描述 |
|---|---|---|
| `id` | uint (PK) | 自增 |
| `name` | varchar(255) | 显示名称（也是特征键的根） |
| `algorithm` | varchar(50) | `FastText`、`DistilBERT` 或 `XGBoost` |
| `tokenizer` | varchar(50) | `GSE`、`WordPiece` 或 `distilbert-base-cased` |
| `languages` | longtext | JSON 数组，例如 `["en","zh"]` |
| `save_path` | varchar(500) | 模型文件路径（训练后设置） |
| `params` | longtext | JSON — `ModelParams`：lr、epoch、wordNgrams、dim、loss |
| `max_text_length` | int (默认: 256) | 输入截断长度 |
| `email_fields` | longtext | JSON 数组——从哪些字段提取文本 |
| `class_labels` | longtext | JSON 字符串数组——训练后从样本同步 |
| `enabled` | bool (默认: false) | 是否在运行时运行此模型 |
| `train_status` | varchar(50) | `pending`、`running`、`completed`、`failed` |
| `train_result` | text | 训练日志输出 |
| `train_time` | datetime | 训练完成时间 |
| `is_deleted` | bool | 软删除标志 |

#### model_samples — 每个模型的训练样本

| 列 | 类型 | 描述 |
|---|---|---|
| `id` | uint (PK) | 自增 |
| `classify_model_id` | uint (FK) | 引用 `classify_models.id` |
| `text` | text | 训练文本内容 |
| `label` | varchar(255) | 样本的金标标签 |
| `created_at` / `updated_at` | datetime | 时间戳 |

#### public_samples — 公共样本库

| 列 | 类型 | 描述 |
|---|---|---|
| `id` | uint (PK) | 自增 |
| `category_id` | uint (FK) | 引用 `public_sample_categories.id` |
| `tag` | varchar(255) | 分类标签（例如 `spam`、`phishing`） |
| `text` | text | 示例文本内容 |
| `created_at` / `updated_at` | datetime | 时间戳 |

#### public_sample_categories — 公共样本类别

| 列 | 类型 | 描述 |
|---|---|---|
| `id` | uint (PK) | 自增 |
| `name` | varchar(128) | 唯一的类别名称 |
| `description` | varchar(500) | 类别描述 |
| `sample_count` | bigint | 近似样本数量 |
| `created_at` / `updated_at` | datetime | 时间戳 |

#### training_tasks — 临时训练任务记录

| 列 | 类型 | 描述 |
|---|---|---|
| `id` | uint (PK) | 自增 |
| `model_name` | varchar(255) | 生成的模型名称 |
| `algorithm` | varchar(50) | 始终为 `FastText` |
| `params` | longtext | JSON 超参数 |
| `sample_mappings` | longtext | JSON 映射：TargetClass → 源组 |
| `status` | varchar(50) | `pending`、`running`、`completed`、`failed` |
| `train_result` | longtext | 日志输出 |
| `model_id` | uint | 生成的 `classify_models.id` |
| `created_at` / `updated_at` | datetime | 时间戳 |

### 4.3 API 端点

所有端点在 `/api/v1/admin/` 下，需要 JWT 认证。

#### 模型 CRUD（`/api/v1/admin/classify-models`）

| 方法 | 路由 | 描述 |
|---|---|---|
| GET | /classify-models | 列出模型（支持 `?keyword`、`?algorithm`、`?status`、`?page`、`?pageSize`） |
| GET | /classify-models/:id | 按 ID 获取模型 |
| POST | /classify-models | 创建模型（FastText 用 JSON，DistilBERT 用 multipart 上传 ONNX） |
| PUT | /classify-models/:id | 更新模型字段 |
| DELETE | /classify-models/:id | 软删除模型 + 移除样本 + 删除模型文件 |
| POST | /classify-models/:id/train | **启动 FastText 训练**（异步） |
| POST | /classify-models/:id/predict | 运行一次性推理用于测试 |
| POST | /classify-models/import | 从 zip 导入模型 |
| GET | /classify-models/:id/export | 将模型导出为 zip |

#### 每个模型的样本（`/classify-models/:id/samples`）

| 方法 | 路由 | 描述 |
|---|---|---|
| GET | /:id/samples | 列出样本（`?keyword`、`?label`、`?page`） |
| POST | /:id/samples | 创建单个或批量样本（`items` 数组） |
| GET | /:id/samples/labels | 列出此模型的唯一标签 |
| GET | /:id/samples/export | 下载 train.txt（`__label__<class>\t<text>`） |
| PUT | /:id/samples/:sampleId | 更新样本 |
| DELETE | /:id/samples/:sampleId | 删除样本 |

#### 临时训练（`/api/v1/admin/training`）

| 方法 | 路由 | 描述 |
|---|---|---|
| POST | /training | **从公共样本映射启动临时训练** |
| GET | /training/:id | 获取训练任务状态 + 日志 |

#### 公共样本（`/api/v1/admin/samples`）

| 方法 | 路由 | 描述 |
|---|---|---|
| GET | /samples | 列出样本（`?categoryId`、`?tag`、`?keyword`） |
| GET | /samples/tags | 列出唯一标签 |
| GET | /samples/stats | 按类别 + 标签统计数量 |
| POST | /samples | 创建单个或批量样本 |
| POST | /samples/batch-delete | 批量删除 |
| POST | /samples/batch-update | 批量更新标签/类别 |
| PUT | /samples/:id | 更新样本 |
| DELETE | /samples/:id | 删除样本 |
| GET | /sample-categories | 列出类别 |
| POST | /sample-categories | 创建类别 |

### 4.4 训练工作流 A：模型绑定训练

此工作流管理特定模型的样本并直接从中训练。

```
步骤 1：创建模型定义
  POST /classify-models
    { name, algorithm: "FastText", tokenizer, languages,
      params: { lr, epoch, wordNgrams, dim, loss },
      email_fields, max_text_length }
  → 模型创建成功，enabled=false, train_status=pending

步骤 2：导入样本
  POST /classify-models/:id/samples [{ text, label }, ...]
  → 存储在 model_samples 表中

步骤 3：开始训练
  POST /classify-models/:id/train
  → 验证：fasttext 可执行文件已配置、model_root 已配置、
            至少 1 个样本、没有并发训练正在运行
  → 设置 train_status=running，启动 goroutine：

    a. 从 DB 读取所有样本
    b. 写入 train.txt：每个样本一行
       格式："__label__<class> <tokenized_text>"
    c. 执行：
       fasttext supervised -input train.txt -output model \
         [-lr N] [-epoch N] [-wordNgrams N] [-dim N] [-loss softmax|hs|ns|ova]
    d. 流式输出 stdout/stderr，更新 DB 中的 train_result
    e. 成功时：
       - 将模型二进制复制到 model_root/<sanitized_name>/model.bin
       - 设置 save_path、train_status=completed、train_time=now
       - 从不同的样本标签同步 class_labels
       - 使模型缓存失效
    f. 失败时：
       - 设置 train_status=failed，在 train_result 中存储错误

步骤 4：发布模型
  PUT /classify-models/:id { enabled: true }
  → 验证 save_path 处的模型文件存在
  → 运行时缓存在 60 秒内获取已启用的模型
  → 下次邮件扫描时：ModelCache 按需打开 FastText 预测器
```

### 4.5 训练工作流 B：基于公共样本的临时训练

此工作流允许管理员使用公共样本库和标签映射快速创建模型。

```
步骤 1：准备公共样本
  通过 /samples API 将样本导入 public_samples 表
  组织到类别中（public_sample_categories）

步骤 2：配置训练
  POST /training
    { modelName, algorithm: "FastText",
      params: { learningRate, epoch, wordNgrams, dim, loss },
      sampleMappings: [{
        targetClass: "spam",
        sources: [{
          category: "General",
          tags: ["spam", "phishing"],
          limitType: "random",
          limitN: 5000
        }]
      }, {
        targetClass: "ham",
        sources: [{
          category: "General",
          tags: ["ham"],
          limitType: "random",
          limitN: 5000
        }]
      }]
    }

步骤 3：训练运行
  a. 创建 TrainingTaskPO 记录
  b. 对每个 TargetClass → 源映射：
     - 按 category_id + tags 查询 public_samples
     - 应用限制策略（random/first/last/middle）以实现平衡数据集
     - 构建训练行："__label__<class> <text>"
  c. 创建一个新的 ClassifyModelPO（enabled=false, train_status=running）
  d. 写入 train.txt，执行 fasttext supervised，流式输出日志
  e. 成功时：设置 save_path、train_status=completed
  f. 失败时：软删除生成的模型，设置 train_status=failed

步骤 4：发布（与工作流 A 相同）
  启用模型 → 运行时获取 → 规则条件可以引用它
```

### 4.6 运行时模型推理

当邮件到达 Body 阶段时，classify_model 提取器运行：

1. **组装**：根据 `email_fields` 配置从邮件中提取文本（主题、纯文本正文、发件人姓名等）
2. **截断**：截断至 `max_text_length`（默认：8192 个字符）
3. **模型缓存**：`ModelCache.PredictAll()` 遍历所有已启用的、未删除的模型：
   - **FastText 模型**：使用纯 Go FastText 引擎进行进程内推理（无外部进程，无 C++ 依赖）。加载 `.bin` 模型文件并运行预测。
   - **非 FastText 模型**（DistilBERT、XGBoost）：通过 gRPC 发送到分类器 worker 进程。
4. **特征输出**：每个模型的预测产生一组特征：
   - 如果模型有 `class_labels`（例如 `["spam", "ham"]`）：`modelname_spam` 和 `modelname_ham` 带概率分数
   - 如果没有类别标签：`modelname` 带有最高概率值
5. **合并**：特征合并到 MilterContext 特征快照中
6. **规则评估**：引用这些特征的规则（例如 `my_model_spam > 0.8`）针对快照进行评估

### 4.7 规则中的模型引用

模型通过其清理后的特征键在规则中被引用：

```
模型名称："My Spam Detector"
清理后：  "my_spam_detector"
特征：    "my_spam_detector_spam"（"spam" 标签的分数）
          "my_spam_detector_ham" （"ham" 标签的分数）
```

清理过程（`SanitizeFeatureKey`）将模型名称转换为小写，并将非 `[a-z0-9]` 字符替换为下划线。

规则可以在条件中使用这些特征：

```json
{
  "op": "cmp",
  "feature": "my_spam_detector_spam",
  "kind": "gt",
  "value": 0.85
}
```

管理规则编辑器列出所有可用的模型特征及其源元数据（ModelID、ModelName、FeatureOrigin：`model`），方便构建包含 ML 模型输出的条件。

### 4.8 模型导出和导入

模型可以导出为包含以下内容的 zip 文件：
- `model.bin` — FastText 模型二进制文件
- `model.conf` — JSON 配置（名称、算法、参数、类别标签、邮件字段等）

导入会恢复二进制文件和模型配置，创建一个新的 `classify_models` 记录。

---

## 5. 过滤管道

反垃圾过滤器作为 Postfix Milter（Milter 协议）运行。Postfix 配置为将每封入站邮件委托给 EasyMail 的 milter 端点。管道通过一系列阶段处理消息，包括并发特征提取、规则评估和可选辅助服务。

### 5.1 架构概览

```
Postfix SMTPD
    |
    v  (Milter protocol over TCP)
EasyMail Milter Server
    |
    +---> Connect 阶段：rDNS、fcDNS、IP 统计
    +---> Helo 阶段：  HELO/rDNS 匹配
    +---> MailFrom 阶段：SPF 检查、域 DNS 健康检查、发件人统计
    +---> RcptTo 阶段：本地检查、联系人查询、收件人统计
    +---> Headers 阶段：头部追踪
    +---> Body 阶段：  DKIM、DMARC、基线、URL、附件、
    |                   反病毒（ClamAV）、ML 模型、
    |                   自定义特征（meta_regex + composite）
    |
    +---> 每个阶段的规则评估（缓存规则 + 所有特征）
    |
    +---> 结果：
            accept  -> SMFIR_CONTINUE（邮件投递）
            reject  -> SMFIR_REPLYCODE（5xx SMTP 拒绝）
            spam    -> SMFIR_CONTINUE + X-EasyMail 头部 -> 垃圾邮件文件夹
            quarantine -> SMFIR_CONTINUE -> 隔离文件夹
```

### 5.2 阶段超时管理

每个 milter 阶段都有一个可配置的超时（每个阶段有默认值，全局可覆盖 `stage_timeout_ms`）。超时预算分为：
- 一半用于并发特征提取（内置提取器）
- 一半用于自定义特征和内容插件

当阶段超时时，已接收的特征仍会合并到上下文中。这确保过滤器在负载下优雅降级。

### 5.3 并发执行

在每个阶段内，注册的提取器作为 goroutine 并发运行。超时通道收集结果；超出超时的提取器被放弃（其部分或缺失的特征被跳过）。

```
sequenceDiagram
    participant SMTP as Postfix SMTPD
    participant Milter as Milter 处理器
    participant Engine as 特征引擎
    participant Ext as 提取器 (goroutines)
    participant Rule as 规则引擎
    
    SMTP->>Milter: Connect(IP)
    Milter->>Engine: Extract(StageConnect, ctx, fc, timeout)
    Engine->>Ext: 并发运行阶段
    Ext-->>Engine: fcrdns, rdns, stats
    Engine->>Milter: 特征已合并
    
    Milter->>Rule: EvaluateWithFeaturesAtStage
    Rule-->>Milter: MatchResult (accept/reject)
    Milter-->>SMTP: 响应 (continue/reject)
    
    Note over SMTP,Milter: 对每个阶段重复
    
    SMTP->>Milter: Body(email data)
    Milter->>Engine: Extract(StageBody, ctx, fc, timeout)
    Engine->>Ext: dkim, dmarc, baseline, urls, attachments
    Engine->>Ext: classify_model (gRPC)
    Engine->>Ext: antivirus (ClamAV)
    Engine->>Ext: 自定义特征 (meta_regex, composite)
    Ext-->>Engine: 所有特征已合并
    Milter->>Rule: EvaluateWithFeaturesAtStage
    Rule-->>Milter: MatchResult (最终动作)
    Milter-->>SMTP: 接受 + 头部 / 拒绝
```

### 5.4 反病毒扫描（ClamAV）

当 ClamAV 启用并配置后，Body 阶段会扫描：
- 完整的邮件正文（当 `scan_body` 为 true 时）
- 每个附件单独扫描（当 `scan_attachments` 为 true 时）

扫描结果产生 `antivirus_hit` 特征：
- `1.0` 如果检测到病毒
- `0.0` 如果所有扫描都干净
- `-1.0` 如果扫描遇到错误

### 5.5 Redis 统计追踪

过滤器使用 Redis 追踪：
- **连接速率**：每个 IP（5 分钟滑动窗口和每日窗口）
- **结果历史**：每个 IP、每个发件人、每个收件人域
- **日内过滤结果**：用于仪表板报告

这些计数器为特征提取提供数据（例如 `ip_conn_5m`、`sender_reject_1d`），也为管理仪表板统计提供数据。

### 5.6 SMTP 拒绝

当过滤器动作为 `reject` 时，milter 发送自定义 SMTP 回复（SMFIR_REPLYCODE）。回复码、增强码和消息均可配置：

```yaml
filter:
  rules:
    reject_reply:
      smtp_code: "550"
      enhanced_code: "5.7.1"
      message: "Spam detected by EasyMail"
```

### 5.7 Milter 注入的头部

Body 阶段处理后，milter 在消息到达 LMTP 投递之前注入以下头部：

| 头部 | 值 |
|---|---|
| `X-EasyMail-Filter-Action` | 最终动作：`accept`、`spam`、`quarantine` 或 `reject` |
| `X-EasyMail-Filter-Rule-Id` | 匹配规则的 ID（默认动作为 0） |
| `X-EasyMail-Filter-Trace-Id` | 用于调试的唯一跟踪标识符 |

这些头部在 Headers 阶段由 milter 从入站消息中剥离（通过 `SMFIR_CHGHDRS`），以防止伪造。

### 5.8 LMTP 路由

LMTP 投递服务读取 `X-EasyMail-Filter-Action` 头部以确定目标文件夹：

| 头部值 | 路由到的文件夹 |
|---|---|
| `accept` | 收件箱 |
| `spam` | 垃圾邮件 |
| `quarantine` | 隔离 |
| `reject` | 默认（收件箱；拒绝应该在 milter 处已执行） |
| (absent) | 收件箱 |

### 5.9 配置参考

过滤器的主要配置选项（在 `easymail.yaml` 的 `milter.filter` 下）：

```yaml
milter:
  filter:
    enable: true                          # 主开关
    rules:
      default_action: accept              # 没有规则匹配时的默认策略
      log_feature_snapshot: true          # 记录所有提取的特征
      log_condition_trace: true           # 记录条件评估追踪
      skip_for_compose_delivery: true     # 跳过本地撰写的邮件过滤
      stage_timeout_ms: 5000              # 每个阶段的超时时间（毫秒）
      reject_reply:
        smtp_code: "550"                  # 拒绝时的 SMTP 回复码
        enhanced_code: "5.7.1"            # 增强状态码
        message: "Spam detected by EasyMail"
    classify_model:
      enable: true                        # 启用 ML 模型推理
      endpoint: "127.0.0.1:50051"         # gRPC 分类器 worker（用于 DistilBERT）
      infer_deadline_ms: 0                # 模型推理的截止时间（0 = 使用 stage_timeout_ms）
    clamav:
      enable: true                        # 启用 ClamAV 扫描
      addr: "127.0.0.1:3310"              # ClamAV 守护进程地址
      timeout_ms: 300000                  # 扫描超时时间（5 分钟）
      scan_body: true                     # 扫描邮件正文
      scan_attachments: true              # 扫描附件
```

#### FastText 训练配置

```yaml
milter:
  filter:
    classify_model:
      fasttext_executable: /opt/easymail/bin/fasttext   # fasttext CLI 二进制路径
      model_root: models            # 模型存储的根目录
```

---

## 6. 源码目录映射

| 区域 | 包/目录 |
|---|---|
| Milter 协议服务器 | `easymail/internal/protocol/milter/` |
| Milter 处理器（阶段回调） | `easymail/internal/adapter/milter/` |
| 特征提取器（内置） | `easymail/internal/infrastructure/filter/extractors/` |
| 自定义特征（meta_regex、composite） | `easymail/internal/infrastructure/filter/extractors/custom_features.go` |
| 特征引擎（编排） | `easymail/internal/infrastructure/filter/extractors/feature_engine.go` |
| 规则引擎（评估） | `easymail/internal/app/filter/engine.go` |
| 规则条件 AST | `easymail/internal/domain/filter/rule/cond_ast.go` |
| 规则/特征实体 | `easymail/internal/domain/filter/rule/feature_entity.go` |
| 自定义特征实体 | `easymail/internal/domain/filter/rule/feature_entity.go` |
| 提取器/内容插件注册表 | `easymail/internal/domain/filter/rule/registry.go` |
| MilterContext（会话状态） | `easymail/internal/domain/filter/session.go` |
| 过滤结果类型 | `easymail/internal/domain/filter/outcome.go` |
| DNS 解析器 | `easymail/internal/infrastructure/easydns/` |
| Redis 过滤统计 | `easymail/internal/infrastructure/filter/stats/redis/` |
| ClamAV 集成 | `easymail/internal/infrastructure/filter/antivirus/` |
| 分类器模型服务 | `easymail/internal/app/filter/classify_*.go` |
| 模型 CRUD + 训练 + 样本 | `easymail/internal/app/filter/classify_model_*.go` |
| 临时训练服务 | `easymail/internal/app/filter/training_service.go` |
| 模型处理器（管理 API） | `easymail/internal/portal/admin/handler/model_handler.go` |
| 样本处理器（管理 API） | `easymail/internal/portal/admin/handler/model_sample_handler.go` |
| 训练处理器（管理 API） | `easymail/internal/portal/admin/handler/training_handler.go` |
| 公共样本处理器（管理 API） | `easymail/internal/portal/admin/handler/public_sample_handler.go` |
| FastText 纯 Go 引擎 | `easymail/internal/infrastructure/filter/classifier/fasttext/` |
| 模型缓存 | `easymail/internal/infrastructure/filter/classifier/modelcache/` |
| 模型配置数据库缓存 | `easymail/internal/infrastructure/cache/classify_models_cache.go` |
| 模型存储（文件管理） | `easymail/internal/infrastructure/filter/assets/classify_model_storage.go` |
| 过滤 PO（GORM 模型） | `easymail/internal/infrastructure/persistence/mysql/filter_po.go` |
| 训练任务 PO | `easymail/internal/infrastructure/persistence/mysql/training_task_po.go` |
| 模型/样本仓库 | `easymail/internal/infrastructure/persistence/mysql/model_repo.go` |
| 样本仓库 | `easymail/internal/infrastructure/persistence/mysql/sample_repo.go` |
| 模型配置仓库 | `easymail/internal/infrastructure/persistence/mysql/model_config_repo.go` |
| 公共样本仓库 | `easymail/internal/infrastructure/persistence/mysql/public_sample_repo.go` |
| 过滤配置 | `easymail/pkg/config/filter.go` |
| 模型运行器（独立 gRPC） | `easymail/internal/runtime/launcher/classifier.go` |
| 分类器领域实体 | `easymail/internal/domain/filter/classifier/` |