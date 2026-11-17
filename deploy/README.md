# Sub2API Deployment Files

This directory contains files for deploying Sub2API on Linux servers and Apple-silicon Macs.

网页注册额度、登录失败封禁及可信代理升级要求见 [网页认证 IP 防护](AUTH_IP_GUARD.md)。

## API Key 并发槽位 TTL 与升级切换

`gateway.api_key_slot_ttl_seconds` 只调整共享 Redis 实现的普通 API Key active slots。默认 `0` 继承 `gateway.concurrency_slot_ttl_minutes`；保持默认值升级不会缩短原租约。显式值至少为 30 秒，120 秒可作为运维评估起点；30 秒下限是配置策略，未经过负载测量，不构成安全 SLA。account/user 槽位、Live 自身租约、WS ingress 租约及 regional concurrency authority 继续沿用各自期限。

运行中的请求复用既有续租、watchdog 和精确释放流程。进程硬退出后，槽位在最后一次成功 Redis 更新时间加有效 TTL 后，由相关准入或统计访问清理，或随 Redis key 到期释放；重启不会立即清零。单独设置短 TTL 不能代替请求取消与退出确认。

**共享同一 Redis 槽位命名空间的进程必须使用相同的有效 API Key TTL。首次启用短 TTL、修改有效 TTL、回退配置或回滚到不识别该字段的旧二进制时，都必须统一 drain，禁止长短 TTL 实例混跑。** 准入和统计读取会按调用进程的 TTL 裁剪成员；短 TTL 进程可能把仍按长周期续租的活跃请求误判为到期，提前重新授予容量。

1. 保存当前二进制与配置，记录旧、新有效 TTL，以及共享该槽位命名空间的全部实例。配置为 `0` 时，也要计入继承的分钟值；regional authority 不应用这个独立 TTL。
2. 从负载均衡和其他入口撤下全部旧 TTL 实例，停止接收新请求，等待已有 HTTP/SSE、WebSocket turn 和相关 Live 会话正常结束。确认实际转发及槽位所有权已结束后再停止旧实例；达到 drain 超时或进程退出，本身不能证明上游请求已停止。
3. 确认旧 TTL 进程及其相关统计读取已全部停止，再以统一的新有效 TTL 启动实例并恢复流量。不得逐个滚动切换不同 TTL 的实例，也不得通过清 Redis、删除其他进程成员或启动清理扫描替代 drain。
4. 核对实际加载的配置、服务日志、并发统计及请求路径。回滚执行同一套 drain；旧二进制忽略专用字段后会恢复共享 TTL，不能直接与短 TTL 实例并行运行。

新增回归覆盖默认继承、独立 TTL、失效回收、双客户端活跃成员、队列和 Live handoff，以及 account/user 与 regional 隔离。实际负载、多 Pod drain 和生产切换仍需在各自环境验收；测试通过不代表这些操作已验证。

## Deployment Methods

| Method | Best For | Setup Wizard |
|--------|----------|--------------|
| **Docker Compose** | Quick setup, all-in-one | Not needed (auto-setup) |
| **Apple container** | Native local stack on macOS 26 | Not needed (auto-setup) |
| **Binary Install** | Production servers, systemd | Web-based wizard |
| **[Kubernetes / K3s](kubernetes/README.md)** | Request replicas with a shared primary | Pre-provisioned config / Secrets |

## Files

| File | Description |
|------|-------------|
| `docker-compose.yml` | Docker Compose configuration (named volumes) |
| `docker-compose.local.yml` | Docker Compose configuration (local directories, easy migration) |
| `docker-deploy.sh` | **One-click Docker deployment script (recommended)** |
| `apple-container.sh` | Native Apple `container` lifecycle script |
| `APPLE_CONTAINER.md` | Apple `container` deployment and operations guide |
| `.env.example` | Container environment variables template |
| `DOCKER.md` | GHCR image documentation |
| `install.sh` | One-click binary installation script |
| `migrate-reauth-managed.py` | 将旧登录 Worker 迁入随主程序版本自动升级的内置管理器，见 [迁移说明](MANAGED_WORKER.md) |
| `install-datamanagementd.sh` | datamanagementd 一键安装脚本 |
| `sub2api.service` | Systemd service unit file |
| `sub2api-datamanagementd.service` | datamanagementd systemd service unit file |
| `DATAMANAGEMENTD_CN.md` | datamanagementd 部署与联动说明（中文） |
| `config.example.yaml` | Example configuration file |
| `install-mihomo-codex.sh` | Optional Mihomo sidecar for rotating Codex ticket harvest exits |
| `mihomo-codex.service` | Systemd unit for the Mihomo ticket proxy |
| `mihomo-codex.config.example.yaml` | Sanitized Mihomo subscription configuration example |
| `EDGE_SECURITY.md` | Reverse proxy, CDN/WAF, trusted proxy, and ingress hardening guide |

---

## Apple container Deployment

Apple-silicon Macs running macOS 26 can run the complete Sub2API, PostgreSQL, and Redis stack with Apple `container` 1.1.0 or newer:

```bash
./apple-container.sh init
./apple-container.sh up
./apple-container.sh status
./apple-container.sh logs app -f
```

The script uses Apple named volumes, starts dependencies in order, and performs live readiness checks. The application container supervises the Sub2API process so the Web UI's update-and-restart flow can relaunch an updated binary. It does not provide host-level automatic startup; run `./apple-container.sh up` after a host reboot. Docker Compose remains the recommended production deployment path.

See [APPLE_CONTAINER.md](./APPLE_CONTAINER.md) for configuration, upgrades, persistence, networking behavior, and limitations.

---

## Docker Deployment (Recommended)

### Method 1: One-Click Deployment (Recommended)

Use the automated preparation script for the easiest setup:

```bash
# Download and run the preparation script
curl -sSL https://raw.githubusercontent.com/c4lnn/sub2api/production/deploy/docker-deploy.sh | bash

# Or download first, then run
curl -sSL https://raw.githubusercontent.com/c4lnn/sub2api/production/deploy/docker-deploy.sh -o docker-deploy.sh
chmod +x docker-deploy.sh
./docker-deploy.sh
```

**What the script does:**
- Downloads `docker-compose.local.yml` and `.env.example`
- Automatically generates secure secrets (JWT_SECRET, TOTP_ENCRYPTION_KEY, POSTGRES_PASSWORD)
- Creates `.env` file with generated secrets
- Creates necessary data directories (data/, postgres_data/, redis_data/)
- **Displays generated credentials** (POSTGRES_PASSWORD, JWT_SECRET, etc.)

**After running the script:**
```bash
# Start services
docker compose -f docker-compose.local.yml up -d

# View logs
docker compose -f docker-compose.local.yml logs -f sub2api

# If admin email/password were auto-generated, find them in logs:
docker compose -f docker-compose.local.yml logs sub2api | grep "Generated admin"

# Access Web UI
# http://localhost:8080
```

### Method 2: Manual Deployment

If you prefer manual control:

```bash
# Clone repository
git clone --branch production https://github.com/c4lnn/sub2api.git
cd sub2api/deploy

# Configure environment
cp .env.example .env
chmod 600 .env
nano .env  # Set POSTGRES_PASSWORD and other required variables

# Generate secure secrets (recommended)
JWT_SECRET=$(openssl rand -hex 32)
TOTP_ENCRYPTION_KEY=$(openssl rand -hex 32)
echo "JWT_SECRET=${JWT_SECRET}" >> .env
echo "TOTP_ENCRYPTION_KEY=${TOTP_ENCRYPTION_KEY}" >> .env

# Create data directories
mkdir -p data postgres_data redis_data

# Start all services using local directory version
docker compose -f docker-compose.local.yml up -d

# View logs (check for auto-generated admin email and password)
docker compose -f docker-compose.local.yml logs -f sub2api

# Access Web UI
# http://localhost:8080
```

### Deployment Version Comparison

| Version | Data Storage | Migration | Best For |
|---------|-------------|-----------|----------|
| **docker-compose.local.yml** | Local directories (./data, ./postgres_data, ./redis_data) | ✅ Easy (tar entire directory) | Production, need frequent backups/migration |
| **docker-compose.yml** | Named volumes (/var/lib/docker/volumes/) | ⚠️ Requires docker commands | Simple setup, don't need migration |

**Recommendation:** Use `docker-compose.local.yml` (deployed by `docker-deploy.sh`) for easier data management and migration.

### Docker behind host Nginx: trusted client IPs

Registration quotas and authentication abuse protection always use the trusted
proxy chain. With host Nginx and a Docker published port, the application's TCP
peer may be a bridge gateway, which is not trusted by default. Configure only
the actual proxy IPs through `SERVER_TRUSTED_PROXIES` in `.env` or the persistent
`/app/data/config.yaml`. Leave the environment variable **unset**, not empty, to
keep YAML configuration. Updating `.env` alone has no effect with an older
Compose template that does not pass this variable through.

See [Docker + host Nginx trusted proxy configuration](../docs/docker-trusted-proxies.md)
for peer inspection, exact proxy configuration, application-only recreation,
registration IP diagnostics and preserving site settings across template updates.

### How Auto-Setup Works

When using Docker Compose with `AUTO_SETUP=true`:

1. On first run, the system automatically:
   - Connects to PostgreSQL and Redis
   - Applies database migrations (SQL files in `backend/migrations/*.sql`) and records them in `schema_migrations`
   - Generates JWT secret (if not provided)
   - Creates admin account (email and password auto-generated if not provided; a provided password must be 8-72 bytes)
   - Writes config.yaml

2. No manual Setup Wizard needed - just configure `.env` and start

3. If `ADMIN_EMAIL` / `ADMIN_PASSWORD` are not set, check logs for the generated admin email (login username) and password:
   ```bash
   docker compose logs sub2api | grep "Generated admin"
   ```

### Startup and Database Recovery

Sub2API applies database migrations during application startup. PostgreSQL can
remain in its recovery/startup phase briefly after a host or Docker daemon
restart. The application retries transient PostgreSQL startup and connection
errors with bounded exponential backoff, then starts automatically when the
database becomes ready. Authentication errors, migration checksum mismatches,
SQL errors, and other permanent configuration or data errors fail immediately.

The Compose example also uses a PostgreSQL health check that verifies both
server readiness and a simple SQL query. `depends_on: condition: service_healthy`
controls dependency ordering for a fresh Compose start, but it is not a
replacement for application-level retries when Docker restores existing
containers after a host restart.

For systemd deployments, keep `Restart=always` and `RestartSec` configured in
`sub2api.service`; the application retry covers transient database startup,
while systemd remains the supervisor for permanent process exits. For
Kubernetes, use a PostgreSQL readiness probe and retain the Sub2API startup
retry behavior; configure the application liveness probe separately so a
database recovery period is not treated as a permanent process failure.

### Database Migration Notes (PostgreSQL)

- Migrations are applied in lexicographic order (e.g. `001_...sql`, `002_...sql`).
- `schema_migrations` tracks applied migrations (filename + checksum).
- Migrations are forward-only; rollback requires a DB backup restore or a manual compensating SQL script.

## 本 fork 的发行迁移与首次部署

本 fork 的正式版本使用三个递增数字段（例如 `2026.11.17`）；不带 `v` 的应用版本用于附件和镜像，Git tag 为 `v2026.11.17`。首个独立正式版只有在完整 Release 验收后才可作为部署来源：必须同时核对五个平台主程序包及 `checksums.txt`、Linux amd64/arm64 runtime、GHCR amd64/arm64 manifest 与 `latest`。Release 成功不代表生产实例已部署。

后端升级检查、前端回滚候选与 runtime 下载只查询本 fork。首次显式部署新版本之前，运行中的旧二进制仍会沿用自身编译进去的更新源；只修改仓库、`.env` 或 Compose 文件不会切换它。新版本历史回滚列表只显示本仓库较旧的合格正式版，因此首次正式发布后的列表可能为空。恢复旧二进制或镜像不会回滚数据库，也会恢复旧二进制自己的升级源；数据库数据恢复须单独决策，不能把程序回滚当作数据回滚。

### 现有 Docker Compose / Apple Container 实例

现有 `.env` 和持久化数据由操作者管理。不要用新 `.env.example` 覆盖现有 `.env`，也不要删除或重建数据库、Redis 卷/目录。备份数据库及应用配置，检查 Compose 文件中的应用 `image`，只将应用镜像改为已验收的 `ghcr.io/c4lnn/sub2api:2026.11.17`（或明确选择的后续版本）；保留原有端口、凭据、卷和环境变量，再显式拉取并重建应用服务：

```bash
docker compose -f docker-compose.local.yml pull sub2api
docker compose -f docker-compose.local.yml up -d sub2api
```

命名卷配置使用相应的 `docker-compose.yml` 命令。Apple Container 的已有 `.env` 不会被脚本默认覆盖；将 `APPLE_CONTAINER_SUB2API_IMAGE` 手动设为同一已验收镜像，再按 [Apple Container 升级步骤](APPLE_CONTAINER.md#backup-and-upgrade)拉取和重建。核对实际应用版本、更新信息来源、runtime 和回滚列表。安装脚本的 `SUB2API_GITHUB_REPO` 仅是安装脚本显式覆盖项，不是后台升级源配置。

`GATEWAY_DISABLE_ENCRYPTED_AGENT_OWNER_BINDING` 本次保持既有透传且默认 `false`，不启用或改变该策略。只有确实计划切换策略时，才按[加密 agent owner 策略](../docs/encrypted-agent-owner-policy.md)统一共享 Redis 节点配置、停止新请求、排空在途请求和 WebSocket 后重启并重连；不能清 Redis 代替恢复，也不能承诺无条件滚动切换。

首次升级还需按下方[“V1 展示分组升级与管理”](#v1-展示分组升级与管理)完成备份和隔离副本演练，并先停止旧版管理写入。`383_channel_monitor_display_groups.sql` 已存在于此代码基线，本次不修改迁移或监控实现。旧监控升级后进入待分组，须手动移入正式组才会显示在用户页；用户页可能暂时为空，但已启用的待分组监控仍可能探测并计费。旧二进制回退可能重新公开待分组监控；回退不默认删除新表列或修改迁移记录。

仅在新正式发行已完整验证、记录旧 tag 的 commit 且恢复用旧二进制/镜像已另行保存后，先核对目标 remote URL 确实属于 `c4lnn/sub2api`，再删除该远端及本地的 `refs/tags/v2.21.6-yuanxu.1`；不凭 remote 名称猜测仓库，不移动或重建旧 tag，不删除其他 tag。保护规则或权限拒绝时停止该步骤，不绕过保护。删除 tag 会令依赖该 tag 的旧 raw URL 失效；旧 Release、附件和镜像保留，但不能依赖失效的 tag URL 恢复。先保存可用恢复产物，不以旧 tag 作为恢复路径。

### V1 展示分组升级与管理

V1 使用独立的 `channel_monitor_display_groups` 表；`383_channel_monitor_display_groups.sql` 只为原 `channel_monitors` 表增加可空 `display_group_id` 和默认 `0` 的 `sort_order`，以及必要约束和索引。原 ID、旧 `group_name`、凭据密文、启停、账号/模板关联、检测历史和日汇总不回填、不重建。旧标签与新组名称互不联动。

**首次升级后全部旧监控进入管理端待分组，用户页可能为空。** 系统不会按旧标签自动创建正式组。管理员需在「渠道监控 → V1 主动探测」创建展示组，将监控移入正式组后才向用户发布。只有正式组中已启用的监控进入用户列表、详情和全站状态；待分组不公开，但其已启用监控仍按原功能开关、模式及周期探测，可能继续产生原有上游流量或费用。空正式组不显示在用户页。

升级步骤：

1. 备份完整数据库及所需加密配置，保存旧二进制/镜像与配置；在隔离副本演练增量迁移，核对旧列值、记录数、ID、历史与关联。结构锁可能影响业务，不承诺零停机。
2. 停止旧版管理写入，避免新旧实例同时管理展示；部署新程序，由既有迁移 runner 按文件名及校验和应用迁移，不修改旧 migration 或清理迁移记录。
3. 在待分组整理旧监控，创建正式组并明确移动发布。配置分组与卡片的数值排序后保存：合法值为 `0..2147483647`，默认 `0`，小值优先，同值按各自 ID 升序；可用 `10、20、30` 留插入空间。移动保留排序，复制继承来源归属/排序/旧标签且副本禁用，不复制历史。
4. 核对用户列表与直接详情的可见性、全站状态、原有探测与健康检查，重启确认新归属和排序保留；本地合成数据验证不能替代生产环境验收。

程序回退优先保留新增表与列，禁止默认 DROP 新结构或修改迁移记录。实际旧版本的启动兼容性必须在隔离副本验证；旧版不识别新分组、排序和待分组隐藏，**可能重新向用户显示待分组监控**，必要时在回退前暂停用户监控入口。回退期间不继续编辑新分组，重新升级不能自动合并旧标签变化。数据恢复是另一个需要明确授权的操作，会覆盖备份之后的写入，不能与程序回退混为一谈。

#### 管理 API

以下路径均以 `/api/v1` 为前缀，沿用管理员认证、监控功能开关及响应 envelope，不要求站点当前运行模式为 V1。待分组由 `NULL` 归属合成，不是可编辑的正式组，也不控制探测优先级或业务路由。

| 方法与路径 | 语义 |
|---|---|
| `GET /admin/channel-monitor-display-groups` | 返回 `{items: [...]}`，正式组按 `sort_order`、ID 升序；不包含待分组实体 |
| `POST /admin/channel-monitor-display-groups` | `{name, sort_order?}`；名称去首尾空白后非空且最多 100 字符，精确唯一、大小写敏感 |
| `PUT /admin/channel-monitor-display-groups/:id` | 只更新提供的 `name`、`sort_order` |
| `DELETE /admin/channel-monitor-display-groups/:id` | 仅删除实际空组；包含禁用或筛选未显示的监控也不可删，不级联删除监控及历史 |
| `GET /admin/channel-monitors?display_group_id=...` | **查询参数**省略为全量，`0` 为待分组，正整数为正式组；保留搜索、供应商、启停及分页 |
| `POST /admin/channel-monitors` | 可提供 `display_group_id: 正整数或null`、`sort_order`；未提供归属默认为待分组，新界面明确传入当前组 |
| `PUT /admin/channel-monitors/:id/display` | 部分更新：省略 `display_group_id` 不改，`null` 移入待分组，正整数移入正式组；请求体的 `0` 无效。`sort_order` 省略不改，空对象无效 |

例如 `PUT /admin/channel-monitors/42/display` 的 `{"display_group_id":null}` 只撤下公开展示，`{"display_group_id":3,"sort_order":20}` 明确移动并设置组内顺序。展示更新不修改凭据、旧标签、启停及业务配置，也不重新调度；普通配置更新不写展示归属或排序。新版表单隐藏旧标签，保存时不提交 `group_name`，原 API 的显式旧标签读写仍保留。

输入错误为 400，未知资源为 404，重名、非空删除或并发外键冲突为 409；失败不会静默将监控移入待分组。用户直接请求待分组、禁用或不存在的监控详情均返回 404。用户页下次成功刷新会移除不可见卡片并关闭其旧详情；不承诺抹除用户已经看过的信息。


**Verify `users.allowed_groups` → `user_allowed_groups` backfill**

During the incremental GORM→Ent migration, `users.allowed_groups` (legacy `BIGINT[]`) is being replaced by a normalized join table `user_allowed_groups(user_id, group_id)`.

Run this query to compare the legacy data vs the join table:

```sql
WITH old_pairs AS (
  SELECT DISTINCT u.id AS user_id, x.group_id
  FROM users u
  CROSS JOIN LATERAL unnest(u.allowed_groups) AS x(group_id)
  WHERE u.allowed_groups IS NOT NULL
)
SELECT
  (SELECT COUNT(*) FROM old_pairs)           AS old_pair_count,
  (SELECT COUNT(*) FROM user_allowed_groups) AS new_pair_count;
```

### datamanagementd（数据管理）联动

如需启用管理后台“数据管理”功能，请额外部署宿主机 `datamanagementd`：

- 主进程固定探测 `/tmp/sub2api-datamanagement.sock`
- Docker 场景下需把宿主机 Socket 挂载到容器内同路径
- 详细步骤见：`deploy/DATAMANAGEMENTD_CN.md`

### Commands

For **local directory version** (docker-compose.local.yml):

```bash
# Start services
docker compose -f docker-compose.local.yml up -d

# Stop services
docker compose -f docker-compose.local.yml down

# View logs
docker compose -f docker-compose.local.yml logs -f sub2api

# Restart Sub2API only
docker compose -f docker-compose.local.yml restart sub2api

# Update to latest version
docker compose -f docker-compose.local.yml pull
docker compose -f docker-compose.local.yml up -d

# Remove all data (caution!)
docker compose -f docker-compose.local.yml down
rm -rf data/ postgres_data/ redis_data/
```

For **named volumes version** (docker-compose.yml):

```bash
# Start services
docker compose up -d

# Stop services
docker compose down

# View logs
docker compose logs -f sub2api

# Restart Sub2API only
docker compose restart sub2api

# Update to latest version
docker compose pull
docker compose up -d

# Remove all data (caution!)
docker compose down -v
```

### Environment Variables

| Variable | Required | Default | Description |
|----------|----------|---------|-------------|
| `POSTGRES_PASSWORD` | **Yes** | - | PostgreSQL password |
| `JWT_SECRET` | **Recommended** | *(auto-generated)* | JWT secret (fixed for persistent sessions) |
| `TOTP_ENCRYPTION_KEY` | **Recommended** | *(auto-generated)* | TOTP encryption key (fixed for persistent 2FA) |
| `SERVER_PORT` | No | `8080` | Server port |
| `ADMIN_EMAIL` | No | *(auto-generated)* | Admin email (login username) |
| `ADMIN_PASSWORD` | No | *(auto-generated)* | Admin password (8-72 bytes) |
| `TZ` | No | `Asia/Shanghai` | Timezone |
| `UPDATE_GITHUB_TOKEN` | No | *(empty)* | Token for `api.github.com` release checks only; asset downloads remain anonymous. |
| `GEMINI_OAUTH_CLIENT_ID` | No | *(builtin)* | Google OAuth client ID (Gemini OAuth). Leave empty to use the built-in Gemini CLI client. |
| `GEMINI_OAUTH_CLIENT_SECRET` | No | *(builtin)* | Google OAuth client secret (Gemini OAuth). Leave empty to use the built-in Gemini CLI client. |
| `GEMINI_OAUTH_SCOPES` | No | *(default)* | OAuth scopes (Gemini OAuth) |
| `GEMINI_QUOTA_POLICY` | No | *(empty)* | JSON overrides for Gemini local quota simulation (Code Assist only). |
| `GATEWAY_API_KEY_QUEUE_MAX_WAITING` | No | `5` | Extra waiting requests per API key with `concurrency_limit>0`; `0` disables key queueing and restores immediate `429`. Read at process start; recreate the container after changing. |
| `GATEWAY_API_KEY_QUEUE_TIMEOUT_SECONDS` | No | `30` | Per-request wait budget in seconds for key capacity; must be a positive integer. Read at process start; recreate the container after changing. |

See `.env.example` for all available options.

> **Note:** The `docker-deploy.sh` script automatically generates `JWT_SECRET`, `TOTP_ENCRYPTION_KEY`, and `POSTGRES_PASSWORD` for you.

### Easy Migration (Local Directory Version)

When using `docker-compose.local.yml`, all data is stored in local directories, making migration simple:

```bash
# On source server: Stop services and create archive
cd /path/to/deployment
docker compose -f docker-compose.local.yml down
cd ..
tar czf sub2api-complete.tar.gz deployment/

# Transfer to new server
scp sub2api-complete.tar.gz user@new-server:/path/to/destination/

# On new server: Extract and start
tar xzf sub2api-complete.tar.gz
cd deployment/
docker compose -f docker-compose.local.yml up -d
```

Your entire deployment (configuration + data) is migrated!

---

## Gemini OAuth Configuration

Sub2API supports three methods to connect to Gemini:

### Method 1: Code Assist OAuth (Recommended for GCP Users)

**No configuration needed** - always uses the built-in Gemini CLI OAuth client (public).

1. Leave `GEMINI_OAUTH_CLIENT_ID` and `GEMINI_OAUTH_CLIENT_SECRET` empty
2. In the Admin UI, create a Gemini OAuth account and select **"Code Assist"** type
3. Complete the OAuth flow in your browser

> Note: Even if you configure `GEMINI_OAUTH_CLIENT_ID` / `GEMINI_OAUTH_CLIENT_SECRET` for AI Studio OAuth,
> Code Assist OAuth will still use the built-in Gemini CLI client.

**Requirements:**
- Google account with access to Google Cloud Platform
- A GCP project (auto-detected or manually specified)

**How to get Project ID (if auto-detection fails):**
1. Go to [Google Cloud Console](https://console.cloud.google.com/)
2. Click the project dropdown at the top of the page
3. Copy the Project ID (not the project name) from the list
4. Common formats: `my-project-123456` or `cloud-ai-companion-xxxxx`

### Method 2: AI Studio OAuth (For Regular Google Accounts)

Requires your own OAuth client credentials.

**Step 1: Create OAuth Client in Google Cloud Console**

1. Go to [Google Cloud Console - Credentials](https://console.cloud.google.com/apis/credentials)
2. Create a new project or select an existing one
3. **Enable the Generative Language API:**
   - Go to "APIs & Services" → "Library"
   - Search for "Generative Language API"
   - Click "Enable"
4. **Configure OAuth Consent Screen** (if not done):
   - Go to "APIs & Services" → "OAuth consent screen"
   - Choose "External" user type
   - Fill in app name, user support email, developer contact
   - Add scopes: `https://www.googleapis.com/auth/generative-language.retriever` (and optionally `https://www.googleapis.com/auth/cloud-platform`)
   - Add test users (your Google account email)
5. **Create OAuth 2.0 credentials:**
   - Go to "APIs & Services" → "Credentials"
   - Click "Create Credentials" → "OAuth client ID"
   - Application type: **Web application** (or **Desktop app**)
   - Name: e.g., "Sub2API Gemini"
   - Authorized redirect URIs: Add `http://localhost:1455/auth/callback`
6. Copy the **Client ID** and **Client Secret**
7. **⚠️ Publish to Production (IMPORTANT):**
   - Go to "APIs & Services" → "OAuth consent screen"
   - Click "PUBLISH APP" to move from Testing to Production
   - **Testing mode limitations:**
     - Only manually added test users can authenticate (max 100 users)
     - Refresh tokens expire after 7 days
     - Users must be re-added periodically
   - **Production mode:** Any Google user can authenticate, tokens don't expire
   - Note: For sensitive scopes, Google may require verification (demo video, privacy policy)

**Step 2: Configure Environment Variables**

```bash
GEMINI_OAUTH_CLIENT_ID=your-client-id.apps.googleusercontent.com
GEMINI_OAUTH_CLIENT_SECRET=GOCSPX-your-client-secret

# 可选：如需使用 Gemini CLI 内置 OAuth Client（Code Assist / Google One）
# 安全说明：本仓库不会内置该 client_secret，请在运行环境通过环境变量注入。
# GEMINI_CLI_OAUTH_CLIENT_SECRET=GOCSPX-your-built-in-secret
```

**Step 3: Create Account in Admin UI**

1. Create a Gemini OAuth account and select **"AI Studio"** type
2. Complete the OAuth flow
   - After consent, your browser will be redirected to `http://localhost:1455/auth/callback?code=...&state=...`
   - Copy the full callback URL (recommended) or just the `code` and paste it back into the Admin UI

### Method 3: API Key (Simplest)

1. Go to [Google AI Studio](https://aistudio.google.com/app/apikey)
2. Click "Create API key"
3. In Admin UI, create a Gemini **API Key** account
4. Paste your API key (starts with `AIza...`)

### Comparison Table

| Feature | Code Assist OAuth | AI Studio OAuth | API Key |
|---------|-------------------|-----------------|---------|
| Setup Complexity | Easy (no config) | Medium (OAuth client) | Easy |
| GCP Project Required | Yes | No | No |
| Custom OAuth Client | No (built-in) | Yes (required) | N/A |
| Rate Limits | GCP quota | Standard | Standard |
| Best For | GCP developers | Regular users needing OAuth | Quick testing |

---

## Binary Installation

For production servers using systemd.

### One-Line Installation

```bash
curl -sSL https://raw.githubusercontent.com/c4lnn/sub2api/production/deploy/install.sh | sudo bash
```

### Manual Installation

1. Download the latest release from [GitHub Releases](https://github.com/c4lnn/sub2api/releases)
2. Extract and copy the binary to `/opt/sub2api/`
3. Copy `sub2api.service` to `/etc/systemd/system/`
4. Run:
   ```bash
   sudo systemctl daemon-reload
   sudo systemctl enable sub2api
   sudo systemctl start sub2api
   ```
5. Open the Setup Wizard in your browser to complete configuration

### Commands

```bash
# Install
sudo ./install.sh

# Upgrade
sudo ./install.sh upgrade

# Uninstall
sudo ./install.sh uninstall
```

### Service Management

```bash
# Start the service
sudo systemctl start sub2api

# Stop the service
sudo systemctl stop sub2api

# Restart the service
sudo systemctl restart sub2api

# Check status
sudo systemctl status sub2api

# View logs
sudo journalctl -u sub2api -f

# Enable auto-start on boot
sudo systemctl enable sub2api
```

### Codex 292 ticket proxy

The Codex ticket harvester can use a local Mihomo sidecar backed by an airport
subscription. This keeps subscription credentials outside the application
database and limits the proxy listener to localhost. Run the optional installer
as root on the same server as Sub2API:

```bash
sudo MIHOMO_CODEX_SUBSCRIPTION_URL='https://provider.example/subscription' \
  bash deploy/install-mihomo-codex.sh
```

For a fresh or versioned binary install, the same environment variable makes
`install.sh` configure the sidecar automatically after downloading the
release:

```bash
sudo MIHOMO_CODEX_SUBSCRIPTION_URL='https://provider.example/subscription' \
  bash deploy/install.sh install -v v2.7.2
```

Without the variable, the installer detects an already active sidecar and
leaves it untouched. It cannot provision an airport subscription on your
behalf; the subscription URL remains the only required provider input.

The installer downloads the pinned compatible Mihomo release, configures the
`airport` provider with health checks and a `CODEX-ROTATE` round-robin group,
and listens on `http://127.0.0.1:3101`. Set the admin setting **292 harvest
proxy** to that local URL. Mihomo rotates only the ticket-harvest traffic;
normal account requests continue to use their configured residential proxy.

The subscription URL is written only to `/etc/mihomo-codex/config.yaml`
(`0640`, `root:mihomo-codex`) and must not be committed or printed in logs.

### Configuration

#### Server Address and Port

During installation, you will be prompted to configure the server listen address and port. These settings are stored in the systemd service file as environment variables.

To change after installation:

1. Edit the systemd service:
   ```bash
   sudo systemctl edit sub2api
   ```

2. Add or modify:
   ```ini
   [Service]
   Environment=SERVER_HOST=0.0.0.0
   Environment=SERVER_PORT=3000
   ```

3. Reload and restart:
   ```bash
   sudo systemctl daemon-reload
   sudo systemctl restart sub2api
   ```

#### Gemini OAuth Configuration

If you need to use AI Studio OAuth for Gemini accounts, add the OAuth client credentials to the systemd service file:

1. Edit the service file:
   ```bash
   sudo nano /etc/systemd/system/sub2api.service
   ```

2. Add your OAuth credentials in the `[Service]` section (after the existing `Environment=` lines):
   ```ini
   Environment=GEMINI_OAUTH_CLIENT_ID=your-client-id.apps.googleusercontent.com
   Environment=GEMINI_OAUTH_CLIENT_SECRET=GOCSPX-your-client-secret
   ```

   如需使用“内置 Gemini CLI OAuth Client”（Code Assist / Google One），还需要注入：
   ```ini
   Environment=GEMINI_CLI_OAUTH_CLIENT_SECRET=GOCSPX-your-built-in-secret
   ```

3. Reload and restart:
   ```bash
   sudo systemctl daemon-reload
   sudo systemctl restart sub2api
   ```

> **Note:** Code Assist OAuth does not require any configuration - it uses the built-in Gemini CLI client.
> See the [Gemini OAuth Configuration](#gemini-oauth-configuration) section above for detailed setup instructions.

#### Application Configuration

The main config file is at `/etc/sub2api/config.yaml` (created by Setup Wizard).

Login rate limiting can be toggled in **System Settings → Security → Login rate limiting** (enabled by default). Saving applies immediately to admins and users, including password, two-factor, Passkey, and OAuth account-binding sign-ins. Disabling skips both the request counter and the IP failure guard, including existing bans and Redis/Lua failures on these login paths. Other authentication endpoints retain their protection. Existing counters and bans keep their original expiry and apply again if re-enabled before expiry. The setting is stored as `login_rate_limit_enabled` in the database; no `.env` change or restart is required. An older settings client that omits this field preserves the current value.

The independent **System Settings → Security → Registration account limit per IP** switch defaults to enabled (at most two registered accounts per IP, including email and OAuth signups). Administrators can disable it for shared/NAT networks without disabling CAPTCHA, email verification, request throttling or login abuse protection. Saving `registration_ip_limit_enabled` takes effect on the next auth request without a restart. Missing/invalid values and lookup failures retain the quota; older settings clients that omit the field preserve its current value. This uses the existing settings store and requires no schema migration. Existing IP quota rows are never deleted or reset. Signups while disabled do not increment the quota, and re-enabling resumes the previously retained counts rather than backfilling those signups. Disabling the quota can increase signup abuse, so retain other verification controls. Reverting to a binary without this switch restores the original always-on quota.

### Prerequisites

- Linux server (Ubuntu 20.04+, Debian 11+, CentOS 8+, etc.)
- PostgreSQL 14+
- Redis 6.0+
- systemd

### Directory Structure

```
/opt/sub2api/
├── sub2api              # Main binary
├── sub2api.backup       # Backup (after upgrade)
└── data/                # Runtime data

/etc/sub2api/
└── config.yaml          # Configuration file
```

---

## Troubleshooting

### Docker

For **local directory version**:

```bash
# Check container status
docker compose -f docker-compose.local.yml ps

# View detailed logs
docker compose -f docker-compose.local.yml logs --tail=100 sub2api

# Check database connection
docker compose -f docker-compose.local.yml exec postgres pg_isready

# Check Redis connection
docker compose -f docker-compose.local.yml exec redis redis-cli ping

# Restart all services
docker compose -f docker-compose.local.yml restart

# Check data directories
ls -la data/ postgres_data/ redis_data/
```

For **named volumes version**:

```bash
# Check container status
docker compose ps

# View detailed logs
docker compose logs --tail=100 sub2api

# Check database connection
docker compose exec postgres pg_isready

# Check Redis connection
docker compose exec redis redis-cli ping

# Restart all services
docker compose restart
```

### Binary Install

```bash
# Check service status
sudo systemctl status sub2api

# View recent logs
sudo journalctl -u sub2api -n 50

# Check config file
sudo cat /etc/sub2api/config.yaml

# Check PostgreSQL
sudo systemctl status postgresql

# Check Redis
sudo systemctl status redis
```

### Common Issues

1. **Port already in use**: Change `SERVER_PORT` in `.env` or systemd config
2. **Database connection failed**: Check PostgreSQL is running and credentials are correct
3. **Redis connection failed**: Check Redis is running and password is correct
4. **Permission denied**: Ensure proper file ownership for binary install

---

## TLS Fingerprint Configuration

Sub2API supports TLS fingerprint simulation to make requests appear as if they come from the official Claude CLI (Node.js client).

> **💡 Tip:** Visit **[tls.sub2api.org](https://tls.sub2api.org/)** to get TLS fingerprint information for different devices and browsers.

### Default Behavior

- Built-in `claude_cli_v2` profile simulates Node.js 20.x + OpenSSL 3.x
- JA3 Hash: `1a28e69016765d92e3b381168d68922c`
- JA4: `t13d5911h1_a33745022dd6_1f22a2ca17c4`
- Profile selection: `accountID % profileCount`

### Configuration

```yaml
gateway:
  tls_fingerprint:
    enabled: true  # Global switch
    profiles:
      # Simple profile (uses default cipher suites)
      profile_1:
        name: "Profile 1"

      # Profile with custom cipher suites (use compact array format)
      profile_2:
        name: "Profile 2"
        cipher_suites: [4866, 4867, 4865, 49199, 49195, 49200, 49196]
        curves: [29, 23, 24]
        point_formats: 0

      # Another custom profile
      profile_3:
        name: "Profile 3"
        cipher_suites: [4865, 4866, 4867, 49199, 49200]
        curves: [29, 23, 24, 25]
```

### Profile Fields

| Field | Type | Description |
|-------|------|-------------|
| `name` | string | Display name (required) |
| `cipher_suites` | []uint16 | Cipher suites in decimal. Empty = default |
| `curves` | []uint16 | Elliptic curves in decimal. Empty = default |
| `point_formats` | []uint8 | EC point formats. Empty = default |

### Common Values Reference

**Cipher Suites (TLS 1.3):** `4865` (AES_128_GCM), `4866` (AES_256_GCM), `4867` (CHACHA20)

**Cipher Suites (TLS 1.2):** `49195`, `49196`, `49199`, `49200` (ECDHE variants)

**Curves:** `29` (X25519), `23` (P-256), `24` (P-384), `25` (P-521)
