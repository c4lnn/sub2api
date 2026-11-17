## Why

当前部署、在线升级、手动回滚和重新登录 runtime 分别引用 `ranxi2001/sub2api` 或 `Wei-Shaw/sub2api`，导致本 fork 发布后仍可能安装其他仓库的产物。将发行来源统一为 `c4lnn/sub2api`，并以正式版本 `v2026.11.17` 替代预发布 `v2.21.6-yuanxu.1`，使安装、镜像标签、更新检查与回滚使用一致的独立发布序列。

## What Changes

- 将应用分发链路统一到 `c4lnn/sub2api`：Compose、部署准备脚本、二进制安装、在线升级和历史版本回滚、前端手动回滚命令、重新登录 runtime、K3s/Kubernetes 与 Apple Container 默认镜像。
- 将首个独立正式版本设为 `v2026.11.17`；应用内版本、附件文件名和镜像版本使用 `2026.11.17`，沿用现有三段数字比较，不引入后缀比较或新版本解析库。
- 采用完整发布，提供主程序压缩包、`checksums.txt`、双架构重新登录 runtime 与 Linux amd64/arm64 镜像；正式发布更新 `latest`。
- 隔离旧升级源缓存，避免切换仓库后因缓存或网络失败重新返回旧仓库的更新产物。
- **BREAKING**：新版本的历史回滚列表只跟踪本仓库的合格正式 Release，不再显示旧仓库历史；不增加跨仓库回滚聚合。
- **BREAKING**：新正式发布验证通过后，删除 `c4lnn/sub2api` 的远端旧 tag `v2.21.6-yuanxu.1`，并清理本地同名 tag；不移动旧 tag、不操作其他仓库的 tag，不自动删除旧 Release、附件或镜像。
- 同步当前部署与发布文档、对应行为测试和首次迁移操作说明；保留上游署名、历史事件链接和 Go module 路径。

## Capabilities

### New Capabilities

- `release-distribution`：规定本 fork 的统一分发来源、正式版本与镜像发布契约、更新和回滚来源、首次发布切换及旧 tag 清理边界。

### Modified Capabilities

无。当前 `openspec/specs/` 为空，本变更新建相关行为规范。

## Impact

- 后端：`backend/internal/service/update_service.go`、升级缓存实现、`backend/internal/reauthruntime/runtime.go` 及对应行为测试；应用版本源 `backend/cmd/server/VERSION`。
- 前端：`frontend/src/components/common/VersionBadge.vue` 生成的手动回滚命令。
- 部署：`deploy/docker-compose*.yml` 中使用发布镜像的三个配置、`deploy/docker-deploy.sh`、`deploy/install.sh`、`deploy/migrate-reauth-managed.py`、`deploy/upgrade-k3s.*`、`deploy/kubernetes/`、`deploy/apple-container.sh` 和 `deploy/.env.example`。
- 发布：沿用 `.github/workflows/release.yml`、`.github/release-tools/` 和 GoReleaser 的动态仓库识别与正式版本机制；核对 Actions、Packages 权限、公开拉取和 `SIMPLE_RELEASE` 配置，不为 owner 增加第二套硬编码。
- 文档：README 多语言入口、部署/Kubernetes/Apple Container 文档，以及与新行为直接相关的安装、更新和回滚说明。
- 运维：已有实例需要首次显式部署新版本；源代码合并或 Release 成功不等于生产部署完成。实际发版、删除 tag 和生产部署不属于本次 artifact 编写操作。
- 不变：HTTP API 形状、Go module/import 路径、第三方数据源、数据库业务数据、现有版本比较规则和独立 CI 的触发策略。
