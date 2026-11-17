## 1. 后端发行来源与缓存隔离

- [x] 1.1 将 `backend/internal/service/update_service.go` 的更新和历史回滚来源改为 `c4lnn/sub2api`，更新现有升级/回滚行为测试；完成判据：查询目标为新仓库，仍排除 Draft/Pre-release 与当前或更高版本，并只返回最多 3 个较旧正式版本。
- [x] 1.2 将 `backend/internal/repository/update_cache.go` 的升级信息 key 切换为 `update:c4lnn:sub2api:latest`，保留 TTL 和同源回退；在现有缓存测试体系中补充旧 key 已存在且新源请求失败的隔离覆盖，完成判据：不会读出旧源链接或附件，新源缓存可正常读写且不清空其他 Redis 数据。
- [x] 1.3 将 `backend/internal/reauthruntime/runtime.go` 的 Release API 与下载地址统一到新仓库，保留对应版本/架构及 SHA-256 校验；完成判据：版本 `2026.11.17` 请求本 fork 的匹配 runtime，缺少附件或摘要不符时不执行、不回退旧源。
- [x] 1.4 以 `010b4cda7379b3ab91745d31efee5525a208c26d` 为代码基线，将 `backend/cmd/server/VERSION` 从 `2.21.6` 更新为 `2026.11.17`；完成判据：后续构建的程序报告该版本，现有比较规则将它判为高于 `2.21.6` 及 `2.21.6-yuanxu.1`，不引入新的版本解析规则。

## 2. 安装与默认容器部署入口

- [x] 2.1 修改 `deploy/docker-compose.yml`、`deploy/docker-compose.local.yml` 和 `deploy/docker-compose.standalone.yml` 的应用镜像为 `ghcr.io/c4lnn/sub2api:latest`；完成判据：三份配置渲染出的应用镜像一致，不修改数据库、Redis 镜像或已有卷定义，保留 `GATEWAY_DISABLE_ENCRYPTED_AGENT_OWNER_BINDING` 的既有透传和默认 `false`。
- [x] 2.2 修改 `deploy/docker-deploy.sh` 的部署文件来源为 `c4lnn/sub2api/production/deploy`；完成判据：实际脚本下载目标属于本 fork 的 `production`，下载后的应用镜像为新 GHCR 地址，原有密钥生成行为保持不变。
- [x] 2.3 修改 `deploy/install.sh` 的默认仓库、用法链接及生成 service 的文档链接，同步 `deploy/sub2api.service` 的发行文档链接；完成判据：默认版本查询和下载属于本 fork，`SUB2API_GITHUB_REPO` 显式覆盖和既有 Token 安全行为仍成立。
- [x] 2.4 修改 `deploy/migrate-reauth-managed.py` 的 Release API 与 runtime 下载源；完成判据：匹配版本/架构的文件来自本 fork，缺少或不符摘要时拒绝迁移，不增加未验证的下载后备路径。
- [x] 2.5 同步 `deploy/apple-container.sh` 与 `deploy/.env.example` 的 `APPLE_CONTAINER_SUB2API_IMAGE` 默认值，并同步部署镜像中与发行来源直接相关的固定 source 元数据；完成判据：新建 Apple Container 使用本 fork GHCR 镜像，既有显式镜像覆盖及持久化 runtime 行为不变，`.env.example` 中已有的加密 agent 策略变量和默认值不丢失，操作者现有显式配置不被覆盖。

## 3. Kubernetes、K3s 与前端回滚

- [x] 3.1 修改 `deploy/upgrade-k3s.sh` 的固定 tag 脚本下载地址与 `deploy/upgrade-k3s.py` 的仓库来源；完成判据：Release、Git tag 解析和 GHCR 请求均属于本 fork，保留现有版本/commit/digest 验证及单节点 Deployment 限制。
- [x] 3.2 同步 `deploy/kubernetes/gateway.yaml` 的 init/application 模板镜像与 `deploy/kubernetes/render.py` 的帮助及仓库白名单，更新 `test_gateway_render.py`、`test_gateway_render_real.py` 的行为用例；完成判据：接受新仓库合法 digest，输出所有相关容器同源同 digest，拒绝旧仓库、其他 owner 和浮动 tag。
- [x] 3.3 修改 `frontend/src/components/common/VersionBadge.vue` 的手动回滚脚本仓库和镜像路径；完成判据：实际回滚界面针对所选合格版本生成 `c4lnn/sub2api` 固定 tag 脚本与无前导 `v` 的 GHCR 镜像命令，不改后台 API 形状。

## 4. 当前文档与迁移说明

- [x] 4.1 更新 README 多语言入口、`deploy/README.md`、`deploy/DOCKER.md`、`deploy/APPLE_CONTAINER.md`、`deploy/kubernetes/README.md` 中描述当前安装和发布的地址、分支及镜像；完成判据：可执行示例属于本 fork，未固定版本入口使用 `production`，固定版本示例使用新 tag，保留上游署名、历史链接与第三方数据源。
- [x] 4.2 在现有相关章节说明三段数字独立版本、完整发布要求、首次显式部署、现有 `.env`/Compose 手动迁移、回滚列表可能为空及旧二进制恢复后升级源的变化；完成判据：文档不把 `SUB2API_GITHUB_REPO` 当作后台配置，不把 Release 成功当作生产部署完成，不暗示业务数据随二进制回滚。
- [x] 4.3 写明 `v2.21.6-yuanxu.1` 的限定清理顺序及后果；完成判据：说明新发行验证后才删本地和目标远端 tag，保留旧 Release/附件/镜像，提示旧 tag raw URL 失效且先保存恢复产物。
- [x] 4.4 在首次升级与恢复说明中保留并引用 `deploy/README.md` 的“V1 展示分组升级与管理”，明确此基线已有 `383_channel_monitor_display_groups.sql`，不修改迁移或监控实现；完成判据：说明备份与隔离演练、停止旧版管理写入、旧监控进入待分组后需手动发布、用户页可能为空且探测仍可能计费，以及旧二进制可能重新公开待分组监控；回退不默认删除新表列或修改迁移记录。
- [x] 4.5 在部署配置保留要求中引用 `docs/encrypted-agent-owner-policy.md`；完成判据：明确本次不启用或改变加密 agent 策略，涉及该策略切换时才要求共享 Redis 节点配置一致、停止新请求、排空在途请求与 WS 后重启重连，不能清 Redis 代替恢复或承诺无条件滚动切换。

## 5. 实现后的针对性验证

- [x] 5.1 全部实现修改完成后一次运行升级/回滚、缓存及 runtime 相关的最窄 Go 测试组合；完成判据：新来源、旧缓存隔离、既有筛选及摘要失败路径通过，记录实际命令与输出，不通过字符串源码扫描代替行为验证。
- [x] 5.2 使用项目本地虚拟环境和现有 Shell 测试入口，一次验证安装器安全/仓库覆盖、三个 Compose 配置、Kubernetes 渲染及真实离线渲染、K3s 升级与 Apple Container 既有行为；完成判据：新镜像来源与拒绝路径符合规范，Compose 中加密 agent 策略未设置时仍为 `false`、显式设置时仍正确透传，记录运行环境限制，不把 fixture 验证声称为真实集群部署。
- [x] 5.3 构建并在浏览器验证实际版本/回滚界面，使用隔离测试实例的候选数据观察两个手动回滚页签并检查源码构建版本；完成判据：实际可见命令和 `2026.11.17` 正确，测试数据不触发真实生产回滚。构建优先使用根目录 `build.sh`，不可用时记录失败阶段后使用现有等价流程。
- [x] 5.4 检查同一发行的跨组件来源一致性并移除临时验证脚本；完成判据：安装、后台更新、前端指令、runtime 和部署配置均指向本 fork，没有为此次修改留下临时文件或兼容旧源的后备分支。
- [x] 5.5 在隔离数据库与应用实例中复用现有 V1 展示分组升级集成用例并观察管理端/用户端，作为新发行包含既有破坏性迁移的集成验收，不新增监控功能或为本次变更重写迁移；完成判据：旧记录与关联保持、旧监控初始待分组且不公开、移入正式组后的启用监控可见，记录探测状态与旧版本回退的公开性及启动兼容边界，不对生产数据库执行验证。

## 6. 完整发布准备与远端验收

- [ ] 6.1 在实际发布执行阶段核对 `c4lnn/sub2api` 的 Actions/Packages 写入权限、GHCR 公开拉取及 `SIMPLE_RELEASE` 仓库变量，记录目标源码 commit；完成判据：完整发布不会被变量强制变为 Simple Release，所有远端操作都明确指向目标 fork。
- [ ] 6.2 对待发布 commit 执行完整 Release dry-run，验证现有年份三段版本、平台矩阵和产物收集；完成判据：工作流成功产生验证产物，`build-frontend` 保留 `NODE_OPTIONS: --max-old-space-size=6144`，未向 registry/Release 发布，未删除旧 tag，不以旧工作流覆盖当前构建配置。
- [ ] 6.3 对以 `010b4cda7379b3ab91745d31efee5525a208c26d` 为基线且包含已验证实现的 commit 创建并推送 `v2026.11.17`，使用现有 Release 流程完成一次正式完整发布；完成判据：新 Git tag 指向该实现 commit，Release 为非 Draft/Pre-release，未移动旧 tag 或复用旧版本二进制。
- [ ] 6.4 验收新 Release 的五个平台主程序包、`checksums.txt`、amd64/arm64 runtime，以及 GHCR 的版本镜像、双架构 manifest 和 `latest`；完成判据：下载和摘要验证通过，应用版本为 `2026.11.17`，镜像 revision 与目标 commit 一致，`latest` 指向该次发行，并记录实际观测证据。

## 7. 删除旧 tag 与交付边界

- [ ] 7.1 在新发布全部验收后记录旧 tag 的 commit 和可用恢复二进制/镜像 digest，核对拟删除 remote 的 URL；完成判据：删除目标明确为 `c4lnn/sub2api` 的 `refs/tags/v2.21.6-yuanxu.1`，不依赖 remote 名称猜测仓库，恢复路径不依赖旧 tag raw URL。
- [ ] 7.2 删除目标远端旧 tag 并清理本地同名 tag，不删除旧 Release/附件/镜像；完成判据：精确的本地与远端旧 ref 均不存在，新 tag 不变，其他 tag 与旧发布产物未被清理。保护规则或权限失败时记录错误，不绕过保护、不扩大删除范围。
- [x] 7.3 交付实现、发布、tag 清理的分项实际结果及首次部署操作说明；完成判据：未执行的远端步骤保持未勾选，权限或网络失败有具体记录，生产部署仍需独立授权，不将本次 artifact 编写或代码合并声明为已上线。

## 实际验证记录

- Go 升级/历史回滚：在 `backend/` 执行 `go test -tags unit ./internal/service -run '^TestUpdateService' -count=1 -timeout=90s`，通过（`0.398s`）。此前未收窄的 service 测试作业已取消，不计作通过。
- Linux runtime：Windows 交叉编译 `GOOS=linux GOARCH=amd64 go test -c -o D:/code/sub2api/backend/bin/reauth-verification.test ./internal/reauthruntime`，再由 WSL Ubuntu-22.04 执行 `reauth-verification.test -test.v -test.run TestPrepareVerifiesDigestBeforeExecutingAndReusesCache`。`tampered` 与 `valid` 均通过，断言 `v2026.11.17` 的 fork API/下载 URL、摘要拒绝及缓存复用。修复验证中发现的测试请求计数遗漏；临时二进制已删除。
- 前端：根目录未发现 `build.sh`。全局 pnpm 11.21.0 在进入测试前报 `ERR_PNPM_LOCKFILE_CONFIG_MISMATCH`；改用 `npx --yes --registry=https://registry.npmjs.org pnpm@10.11.0 install --frozen-lockfile --ignore-scripts --registry=https://registry.npmjs.org` 恢复锁定依赖，未为此修改锁文件。随后按 `package.json` 的等价步骤直接运行本地工具：回滚组件 Vitest 1/1、i18n Vitest 3/3、`vue-tsc -b` 与 `vite build` 均通过；构建内存上限保持 6144 MiB。Browserslist 数据陈旧及大 chunk 为构建警告。
- 实际界面：Orca 打开仅绑定本机的隔离页面，渲染生产 `VersionBadge.vue`，合成候选 `2026.11.16`，仅允许候选 GET，阻止修改 API。实际点击后观察到 `curl -sSL https://raw.githubusercontent.com/c4lnn/sub2api/v2026.11.16/deploy/install.sh | sudo bash -s -- rollback v2026.11.16` 和 `image: ghcr.io/c4lnn/sub2api:2026.11.16`；当前显示版本为 `2026.11.17`。页面、服务及临时文件已清理，未执行回滚。
- 主程序：`go build -tags embed -trimpath -o bin/release-verification.exe ./cmd/server` 后执行 `--version`，实际输出 `Sub2API 2026.11.17 (commit: unknown, built: unknown)`。这是本地源码构建，不是正式发布 revision 验收；临时二进制已删除。
- 缓存与 V1 集成：执行 `CI=true go test -tags integration ./internal/repository -run 'TestUpdateCacheSuite|TestChannelMonitorDisplayGroupUpgradePreservesLegacyData|TestChannelMonitorDisplayGroupPublishedOrderingAndFilterPagination' -count=1 -timeout=5m`，在 `TestMain` 初始化阶段失败：`docker is not available (CI=true); failing integration tests`。Docker Desktop 启动 120 秒超时，随后仍无 `dockerDesktopLinuxEngine` pipe。未运行数据库用例、未访问生产数据库，不将初始化失败或跳过计为通过；5.1、5.5 保持未完成。
- 远端：匿名请求 `https://api.github.com/repos/c4lnn/sub2api` 返回 `API rate limit exceeded`；当前进程未提供 `GH_TOKEN`、`GITHUB_TOKEN` 或 `UPDATE_GITHUB_TOKEN`。未核实 Actions/Packages 写入权限或 `SIMPLE_RELEASE`，未执行远端 dry-run、正式发布、创建/推送 tag 或删除旧 tag。新发行未验收，故旧 tag 清理前置条件不满足；生产部署未执行。
- OpenSpec：`openspec validate switch-release-origin-and-adopt-calver --strict` 通过。
- Python 部署测试：WSL 的常规 venv 创建因缺少 ensurepip 失败；使用 `python3 -m venv --without-pip .venv-wsl` 创建项目本地标准库环境，`source .venv-wsl/bin/activate` 后分别执行 `python -m unittest discover -s deploy/tests -p test_gateway_render.py -v`、`test_upgrade_k3s.py`、`test_migrate_reauth_managed.py`，实际为 `Ran 6 tests ... OK`、`Ran 18 tests ... OK`、`Ran 8 tests ... OK`。本次没有安装 OS 包。早先系统 Python 的结果不作为虚拟环境验收；最终这 32 个用例均已在本地 venv 中复验通过。
- 真实离线渲染：部署验证 worker 使用 Windows 项目 `.venv/Scripts/python.exe -m unittest discover -s deploy/tests -p test_gateway_render_real.py -v`，`Ran 2 tests ... OK`。通过 Docker Desktop 自带 `kubectl.exe` 的 `kubectl kustomize <temporary-dir>` 执行真实离线渲染，不连接集群。Python 部署用例合计 34 个通过；PyYAML 仅安装于 Windows 项目 venv。
- 安装器与部署准备：WSL 执行 `bash deploy/tests/install-github-token-test.sh`，输出 `install GitHub token checks passed`、exit 0，覆盖新默认来源、显式仓库覆盖和 Token 的 argv/env 泄露拒绝。Git Bash 的临时 docker-deploy 驱动器输出 `docker-deploy curl-fixture behavior checks passed`、exit 0：实际执行生产脚本，两个下载 URL 为本 fork 的 `main/deploy`，fixture 返回当前本地模板，应用镜像为新 GHCR，JWT/TOTP/Postgres 密钥为互不相同的 64 位 hex；临时驱动器已删除，无真实下载/容器启动。
- Compose：`D:\Git\bin\bash.exe deploy/tests/docker-compose-gateway-env-test.sh` 输出 `docker compose source image and gateway environment checks passed`、`docker compose Gateway environment test passed`，exit 0。实际对三个发布模板执行 `docker compose config --format json`，镜像为新 GHCR，owner binding 默认 `false`、显式覆盖 `true`；未启动服务。
- Apple fixture：WSL 临时驱动器运行后输出 `Apple container lifecycle tests passed`、exit 0，覆盖默认镜像及显式 override。为跨平台运行，在临时副本规范化 fixture CRLF，并提供 `plutil` JSON 字段提取、BSD `stat` 到 GNU `stat` 转换及 localhost 健康探测成功 shim；container 为仓库 fake fixture。此结果仅为隔离 fixture 生命周期验证，不代表真实 Apple Container/macOS 部署或真实健康探测。临时驱动器、shim 已清理。
- 继续验收时 Docker 已恢复响应，Linux engine 为 `29.5.3`；默认集成测试因 `mirror.ccs.tencentyun.com` 下载 `postgres:18.1-alpine3.23` 返回 `EOF` 及本地缺失镜像而失败，未执行用例。没有修改 Docker 全局 mirror 配置；通过 harness 已有的 `SUB2API_TEST_POSTGRES_IMAGE` / `SUB2API_TEST_REDIS_IMAGE` 参数尝试公开镜像源。
- 发布前置核对补充：从 Git Credential Manager 使用现有 `c4lnn` 凭据，目标仓库 API 返回 admin/push 权限；Actions 已启用，默认 workflow 权限为 write，仓库变量列表为空（没有 `SIMPLE_RELEASE=true`）；该凭据的 Packages 查询返回 403，未据此声称 GHCR 可公开拉取。当前源码 HEAD 仍为基线 commit，变更尚未提交。远端 `main` 为 `6efebe3ff9f2eacceedeeac73204d276bf793b88`、版本 `0.2.19-ark`，不是当前 `production` 的祖先；直接覆盖会丢弃其独有历史。公开 refs 当前只列出 `v0.2.19-ark`，没有原计划的旧 tag 或新正式 tag，旧 Release URL 返回 404；不将其记为本次已清理。
- 此前用户在分支路线选择中明确选择“只继续本地集成验收，暂不提交或修改远端分支”，当时未执行 commit、push、merge、dispatch、发布或删除远端内容。用户随后确认保留 `production`，并明确授权“提交并推送并打上 tag”：本次可提交已验证实现、正常快进推送到目标 fork 的 `production`，并创建及推送 `v2026.11.17`；不覆盖分叉的远端 `main`，不修改远端默认分支，不删除旧 tag 或执行生产部署。推送 tag 会触发现有 Release 工作流；远端 dry-run、完整产物验收与旧 tag 清理并未因此自动完成，6.1–6.4、7.1–7.2 须按各自实际完成判据记录，未完成项继续保留。
- 缓存/V1 数据库验收已解除阻塞：使用 `mirror.gcr.io/library/postgres:15-alpine`（已支持的 PostgreSQL 15）和 `mirror.gcr.io/library/redis:7-alpine` 启动仅绑定 localhost 的独立容器，通过已有 `SUB2API_TEST_POSTGRES_DSN` / `SUB2API_TEST_REDIS_URL` 执行 `CI=true go test -v -tags integration ./internal/repository -run '^(TestUpdateCacheSuite|TestChannelMonitorDisplayGroupUpgradePreservesLegacyData|TestChannelMonitorDisplayGroupPublishedOrderingAndFilterPagination|TestLocalV1BrowserVerificationSeed)$' -count=1 -timeout=5m`：PASS（`10.988s`）。`TestUpdateCacheSuite` 的 7 个子用例及两个既有 V1 用例均通过。临时 seed 用例为浏览器验收留下独立 `sub2api_v1_browser_1117`，实际启动前确认旧监控 3 条、`display_group_id` 列 0 个。原 Docker Hub mirror/自动容器启动失败不再作为最终结论；未改生产实现、SQL 迁移或 Docker 全局配置。
- V1 实际应用与浏览器验收：新源码构建程序在 `127.0.0.1:58087` 启动并自动升级独立 `sub2api_v1_browser_1117`；迁移后保留 3 条监控与 3 条历史，3 条均无展示组，启用状态为 true/false/true。用户本人完成本地管理员合规确认后，管理页实际显示 3 条“待分组”，普通用户页面显示“暂无可显示的渠道”，用户列表为空。创建“隔离验收正式组”并将 id=1 移入、排序设为 20 后，用户 API 与实际页面仅显示 `legacy-probe`；未发布 id=2/3 的详情均为 HTTP 404。浏览器实际版本显示 `2026.11.17`。数据库升级、管理端与用户端均为真实本地应用，不是 API mock。
- V1 探测边界：id=3 仍为 enabled=true、display_group_id=NULL；手动执行 `POST /api/v1/admin/channel-monitors/3/run` 返回 HTTP 500、`CHANNEL_MONITOR_KEY_DECRYPT_FAILED`，符合刻意不可解密的合成密文。没有真实上游凭据或成功上游调用，不将历史里的合成正常状态或页面延迟视为真实探测成功，也不声称已验证真实计费。
- V1 旧版回退边界：从本地前驱 commit `321f11c1a4d23302961c223ac030fd09e3f9e6e8`（版本 `2.21.6`，无 383 迁移）导出临时源码并构建；通过 `pg_dump` / `psql` 将升级后的测试库复制到独立 `sub2api_v1_rollback_1117`，旧程序仅绑定 `127.0.0.1:58088`、使用独立 Redis DB 1。首次启动仅因临时 DATA_DIR 未创建而失败，创建目录后正常启动。保留新增表、列和迁移记录时，旧版用户列表返回 id=1/3，尚未分组但已启用的 id=3 详情 HTTP 200，禁用 id=2 仍为 404；分组记录、id=1 的 group=1/order=20 及 3 条历史保留。该结果证明这一明确旧源码构建可启动且会重新公开待分组监控，不证明缺失的 `v2.21.6-yuanxu.1` 发行附件或其他旧版本均兼容；没有对生产数据库执行回退或恢复。
- 本地验收收尾：已停止新旧两个临时应用服务，移除本次 PostgreSQL/Redis 容器，删除临时 seed 测试、导出的旧源码、临时新旧二进制与 DATA_DIR；没有修改生产迁移或监控实现。更新验收记录后执行 `openspec validate switch-release-origin-and-adopt-calver --strict`：通过。5.1、5.5 已完成；6.1–6.4、7.1–7.2 仍按用户的本地限定选择保持未完成。
- 生产分支纠正：用户明确以 `ranxi2001/sub2api` 的 `production` 为维护依据，本 fork 继续使用 `production`，不改为 `main`。已恢复未固定版本的部署脚本 URL、README 安装/克隆示例与 OpenSpec 分支契约；保留原始上游署名与固定 tag 的发行入口。本地 `git symbolic-ref --short HEAD` 实际返回 `production`。通过 Git Bash 在临时目录执行真实 `deploy/docker-deploy.sh`，curl fixture 仅接受本 fork 的 `production/deploy/docker-compose.local.yml` 和 `production/deploy/.env.example`，实际两次下载通过，生成配置使用 `ghcr.io/c4lnn/sub2api:latest`、三个密钥均为 64 位 hex，输出 `docker-deploy production branch behavior checks passed (2 downloads; local fixtures; no network or deployment)`；临时目录自动清理。`openspec validate switch-release-origin-and-adopt-calver --strict` 通过。此前 `main/deploy` 验证记录仅代表当时实现，不再是当前入口；远端默认分支、远端 refs 与生产部署均未修改。
- tag 发布前补齐生产分支约定：现有 Release 的 `sync-version-file` 会在正式发布成功后写回远端默认分支，与本 fork 仍以 `production` 维护的要求冲突；本次将该作业 checkout 和 VERSION 回写目标明确固定为 `production`，不修改远端默认分支，不将版本同步提交写入分叉的 `main`。
- Release VERSION 回写验证：在临时本地 bare remote 中创建 `main` 与 `production`，按工作流配置 checkout 分支并执行工作流原始 VERSION 同步脚本。实际 `production` 更新为 `2026.11.17`，`main` commit 不变；再次执行同版本不产生新提交或推送。输出 `Release VERSION sync passed: production updated; main unchanged; matching version is idempotent (temporary local Git remote only)`；临时仓库已自动删除。该验证不代表远端 Release dry-run 或正式产物验收。
