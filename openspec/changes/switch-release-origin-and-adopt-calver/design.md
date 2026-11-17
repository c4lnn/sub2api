## Context

动机见 `proposal.md` 的 Why。此变更跨后端升级、前端指令、安装与容器部署、runtime 下载和远端发行迁移，且涉及旧 tag 删除，因此需要技术设计。

已核实的现状：

- 后端升级源在 `backend/internal/service/update_service.go` 写死为 `ranxi2001/sub2api`；版本比较只比较三段数字，忽略 `-` 后缀。历史回滚查询近期 15 条 Release，排除 Draft/Pre-release，最多显示 3 个更低版本。
- Redis 升级缓存的实际 key 是 `backend/internal/repository/update_cache.go` 中的 `update:latest`，有效期 20 分钟；新源查询失败时会回退到有效缓存，缓存没有来源字段。
- `VersionBadge.vue` 的手动回滚脚本和镜像仍引用 `Wei-Shaw/sub2api`、`weishaw/sub2api`；runtime 自动准备和迁移工具各自写死旧 fork 地址。
- Compose、K3s 和 Kubernetes 默认引用旧 fork；Kubernetes 渲染器还限制了允许的镜像仓库。Apple Container 默认仍为上游 Docker Hub 镜像。
- Release 工作流由 `v*` tag push 或手动触发，使用运行仓库的 owner/name 动态发布。带 `-` 后缀的版本为预发布且不更新 `latest`；完整发布与 Simple Release 的主程序附件行为不同。
- 本变更的代码基线为 `010b4cda7379b3ab91745d31efee5525a208c26d`，基线版本文件为 `2.21.6`。本 fork 依据 `ranxi2001/sub2api` 的 `production` 分支维护，生产源码与未固定版本入口使用 `production`；初次调查观测到目标远端默认分支为 `main`，不据此更改生产分支约定，也不自动修改远端设置。初次公开页面显示 `v2.21.6-yuanxu.1` 和 `v0.2.19-ark` 均为 Pre-release，GitHub API 排查时返回 403，公开 HTML 可访问；这不是对发布权限或附件完整性的验证，后续远端状态另见 tasks 中的实际记录。
- 此基线的 Release 前端构建已设置 `NODE_OPTIONS: --max-old-space-size=6144`；Compose 与 `.env.example` 已透传 `GATEWAY_DISABLE_ENCRYPTED_AGENT_OWNER_BINDING`，默认 `false`。来源切换必须保留这些现有配置。
- 此基线已包含 `383_channel_monitor_display_groups.sql`：旧监控的展示组归属保持 `NULL`，不自动创建或分配正式展示组。首次升级后的公开展示和旧二进制回退边界见 `deploy/README.md` 的“V1 展示分组升级与管理”。

## Goals / Non-Goals

**Goals:**

- 保持现有架构和配置约定，按入口进行小范围替换，不新增通用“仓库来源框架”。
- 切换后即使新源暂不可达，也不从旧源缓存安装其他仓库产物。
- 将源码版本、Release、主程序包、runtime、镜像及其来源 commit 作为同一发行单元验证。
- 将可本地验证的实现、远端发布验收和生产部署明确分开，不把远端待执行操作标成已完成。

**Non-Goals:**

- 不修改 Go module/import 路径，不全局替换上游署名、历史 Issue/PR 或第三方定价来源。
- 不增加后缀版号比较、跨仓库回滚、任意后台升级源配置或自动部署生产实例。
- 不重写 Release 工作流触发策略，不因 owner 迁移将动态发布地址硬编码。
- 不删除旧 Release、其附件、容器 tag 或其他仓库的 tag；此次明确删除对象只有本地及 `c4lnn/sub2api` 的 `v2.21.6-yuanxu.1` Git tag。
- 不重新实现或调整此基线已有的 V1 展示分组、迁移 SQL 或加密 agent 账号策略；仅将其升级风险纳入发行说明与验收。

## Decisions

### 1. 按现有入口统一默认来源，不新增配置层

后端升级仓库、runtime API/下载路径、迁移脚本及部署默认值改为 `c4lnn/sub2api`；前端手动回滚镜像改为完整 GHCR 路径。保留安装脚本已有 `SUB2API_GITHUB_REPO` 覆盖语义，不把它扩展成后端环境变量。

未固定版本的 raw URL 和当前仓库克隆/说明使用 `production`，沿用 `ranxi2001/sub2api` 的生产分支约定；Release 中安装和 K3s 入口继续固定到对应 tag。正式发布成功后的 VERSION 同步作业也明确 checkout 并写回 `production`，不依赖远端默认分支，以免改动分叉的 `main`。安装器生成的 service 文档链接、部署镜像的来源标签等与发行来源直接相关的元数据一并同步，不改与分发无关的外部链接。远端默认分支设置与本地源码入口是独立操作，不自动改名或合并远端分支；分支/tag 推送须有用户明确授权。

替代方案是引入跨 Go、Vue、Shell、Python 的共享动态仓库配置。它需要额外生成/注入机制，而本次只有一个固定发行目标，维护成本高于收益，故不采用。

### 2. 用独立固定缓存 key 完成一次性来源隔离

将仓库层升级缓存 key 改为 `update:c4lnn:sub2api:latest`，不读取旧 `update:latest`。保持缓存序列化、TTL、同源错误回退和 API 响应结构不变。旧 key 自然过期，不扫描或清空 Redis。

只要求运维手动清缓存不能覆盖所有现有实例；给缓存增加 schema 和来源校验可更通用，但本次没有可配置多源需求。固定新 key 是更小且可测试的切换，不承诺多个不同发行源未来共享该 key。

### 3. 保留三段数字比较，首次正式发布为 v2026.11.17

将 `backend/cmd/server/VERSION` 更新为 `2026.11.17`；为包含完整实现的 commit 新建 `v2026.11.17`，不把旧 tag 改名后直接复用旧二进制。新附件和镜像由该 commit 重新构建，以免旧源码中的升级地址继续存在。

后续正式版本仍使用三段、无前导零的数字并单调递增；本次不增加日历有效性检查、四段版本或同日修订解析。`v2026.11.17` 是本次确定的版本，后续同日再发版也必须满足已有三段数字比较，不使用 `-yuanxu.N` 作为正式升级序列。

### 4. 沿用动态发布，首次必须完整发布

保留现有 `github.repository_owner`、仓库名和 `GITHUB_REPOSITORY` 的派生行为。首次发布前确认 `vars.SIMPLE_RELEASE` 不为 `true`，手动参数 `simple_release=false`、`dry_run=false`；否则仓库变量的 OR 逻辑仍可能强制简化发布。

保留此基线 `build-frontend` 作业中的 `NODE_OPTIONS: --max-old-space-size=6144`，不以旧工作流替换新配置；本次不调整前端内存限制或 CI 架构。

完整发行产物包括：Linux amd64/arm64、macOS amd64/arm64、Windows amd64 主程序包及 `checksums.txt`，Linux 双架构 runtime，以及 `ghcr.io/c4lnn/sub2api:2026.11.17` 双架构 manifest。现有规则同时生成 `latest`、`2026.11`、`2026` 镜像 tag，不另建分发规则。

正常 push tag 与手动运行均沿用现有事件机制。先做本地针对性验证及工作流 dry-run，再进行一次实际完整发布；不将 dry-run 成功当作远端发行成功。

### 5. 保留下载安全边界和部署限制

不修改 GitHub 下载域名白名单、不绕过主程序现有校验、不降低 runtime 附件摘要要求。Kubernetes 渲染器的仓库正则只从旧 owner 改为新 owner，保留强制 digest、拒绝其他仓库和拒绝浮动 tag 的行为；模板内 init/application 镜像必须同步。

K3s 脚本仍只支持它现有的单节点 Deployment 场景，不借迁移扩展到 StatefulSet。Apple Container 默认镜像与 `.env.example` 同步，文档明确现有 `.env` 需操作者修改；Compose 本地文件同理。

修改 Compose 镜像及 `.env.example` 时，保留 `GATEWAY_DISABLE_ENCRYPTED_AGENT_OWNER_BINDING` 的透传、默认 `false` 和操作者已有的显式值，不因发行来源变更开启策略。只有部署涉及该策略切换时，才按 `docs/encrypted-agent-owner-policy.md` 要求统一所有共享 Redis 节点的值、停止新请求、排空在途请求和旧 WS 后重启并重连；不得用清 Redis 代替恢复，也不承诺无条件滚动切换。

### 6. 历史回滚与本地恢复分开描述

保持既有候选筛选，不导入旧仓库 Release。首次新正式发布后历史列表可能为空，这是正确结果。候选有平台包缺失时仍失败，不自动去其他仓库补包。

保留 `.backup` 本地恢复能力；恢复旧二进制会恢复它自身的升级来源，且不回滚数据库。生产恢复还可以使用操作者事先保存的旧镜像 digest/二进制，不依赖即将删除的旧 tag 重新下载脚本。

此基线的 V1 升级需先保存完整数据库、加密配置和旧产物，在隔离副本演练既有迁移，停止旧版管理写入后再部署。旧监控首次进入管理员“待分组”，用户页面可能为空；需管理员创建正式展示组并移入监控才公开，已启用的待分组监控仍可能继续探测及计费。验收区分管理端待分组、正式组公开展示与原有探测，不自动按旧标签发布。

回退旧二进制可能重新向用户展示待分组监控，必要时先暂停用户监控入口，并在隔离副本验证实际旧版本启动兼容性；默认保留新增表、列和迁移记录，不执行 DROP 或修改 migration 校验。回退期间不编辑新分组；数据库恢复另需明确授权，且会覆盖备份之后的写入。

### 7. 新发布验收后再删除旧 tag

先记录旧 tag 的 commit 与旧产物可用的恢复信息，验证 remote URL 确实指向 `c4lnn/sub2api`，再执行仅针对 `refs/tags/v2.21.6-yuanxu.1` 的远端删除并清理本地同名 tag。不得因为本地同时配置 upstream 就按 remote 名称猜测删除对象，不批量删 tag，也不 force-move 旧 tag。

删除 tag 并不等于删除 GitHub Release 或镜像。旧 Release/附件/镜像本次明确保留，其部分依赖旧 Git tag 的源码与脚本链接可能失效；文档需说明这一结果，不能宣称旧版本仍完整可重建。若 GitHub 的受保护 tag/发布规则阻止删除，应记录错误并停止该删除步骤，不绕过保护或扩大清理范围。

替代方案是先删旧 tag 再发布，或者直接覆盖旧 tag。前者在新发布失败时削弱恢复路径，后者破坏发行身份且仍保留不一致的附件，因此都不采用。

## Risks / Trade-offs

- [旧实例仍运行写死旧仓库的二进制] → 首次显式部署 `2026.11.17`；只改仓库源码、环境变量或模板不算切换完成。
- [年份版本无法被未来低编号的 2.x/3.x 自动升级替换] → 后续本 fork 继续使用单调递增的三段数字，记录同步的上游基线但不复用其版本作为本 fork 发布号。
- [镜像已推送但 Release/附件发布失败] → 工作流当前先推镜像后发布 Release；验收全部产物后才确认发行完成和删除旧 tag，不把某个作业的部分成功当成整体成功。
- [已有 runtime 按版本目录缓存且不含仓库身份] → 新应用版本创建独立版本目录，不主动清理或复用旧版 runtime；保留既有摘要校验。
- [共享 Redis 的旧、新实例存在不同更新源] → 新 key 隔离更新信息；生产是否混跑、何时切换所有实例由部署验收明确记录，不擅自修改业务缓存。
- [旧 tag 删除后老 raw URL 失效] → 先保存恢复所需旧二进制或镜像 digest，再删除；不承诺 tag 删除可逆，也不重新创建旧 tag。
- [GitHub API 限流、发布权限或 GHCR 可见性不足] → 远端验收检查凭据和公开拉取，失败明确记为阻塞；不回退到其他仓库掩盖问题。

## Migration Plan

1. 实现本变更的来源、缓存、版本、指令和文档修改；仅更新覆盖这些行为的现有测试，在旧缓存回退等缺少覆盖且影响安装来源的边界补充隔离测试。
2. 完成一次最窄验证组合：升级/回滚与缓存行为、runtime 来源/摘要路径、安装器覆盖、Compose 配置、Kubernetes 渲染与 K3s 升级、Apple Container 既有测试及前端实际回滚指令。Python 检查在项目本地虚拟环境内运行；构建优先使用根目录 `build.sh`，如果实际不存在或不能执行则记录阶段并按现有构建流程执行，不添加替代构建框架。
3. 为待发布 commit 运行完整发布 dry-run，验证三段年份版本和目标架构产物；核对 Actions 权限、Packages 可见性以及 `SIMPLE_RELEASE`。此时不删除旧 tag。
4. 对包含已验证修改的 commit 创建并推送新 tag `v2026.11.17`，执行实际完整发布。验证 Release 非 Draft/Pre-release、附件和校验、双架构 runtime、镜像 manifest、源码 revision label、应用 `--version` 及 `latest`。
5. 记录旧 tag commit 和恢复产物，确认新 tag 正确且旧 tag 所在 remote 是目标 fork；删除目标远端旧 tag 和本地同名 tag。验证旧 Git ref 不存在，检查旧 Release/附件/镜像没有被清理；该项单独记录完成状态。
6. 代码与发行交付不自动部署生产。在明确授权的部署窗口，操作者修改现有配置并首次部署固定版本或 digest，检查应用版本、更新响应来源、实际 runtime 来源和回滚列表；无较旧正式版本时接受空列表。首次升级同时按现有 V1 部署文档完成数据库备份、隔离演练、停止旧版管理写入及分组公开性验收；保留既有配置，涉及加密 agent 策略切换时再执行节点配置一致性及排空 WS 流程。
7. 失败恢复：新发布未验收前不删除旧 tag；生产验证失败时恢复预留的二进制/镜像 digest 或有效本地备份，不假定应用降级可恢复数据库。按上述 V1 回退边界防止待分组监控意外公开，不删除新增数据库结构或修改迁移记录；报告恢复后升级源可能重新指向旧仓库。
