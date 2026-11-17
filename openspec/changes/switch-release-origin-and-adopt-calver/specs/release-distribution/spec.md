## Purpose

规定 `c4lnn/sub2api` 独立发行的安装、容器镜像、在线升级、历史回滚及重新登录 runtime 的来源和版本一致性，确保首次切换及后续发布不再意外获取其他仓库的应用产物，并明确正式版本和旧 tag 清理的兼容边界。

## ADDED Requirements

### Requirement: 默认应用分发来源统一到本 fork

系统提供的默认安装、部署文件下载和应用镜像入口 MUST 使用 `c4lnn/sub2api` 及 `ghcr.io/c4lnn/sub2api`。未固定版本的源码下载入口 SHALL 使用该仓库的 `production` 分支，沿用 `ranxi2001/sub2api` 的生产分支约定；固定版本入口 SHALL 使用指定版本 tag。安装脚本既有的 `SUB2API_GITHUB_REPO` 显式覆盖能力 SHALL 保留，但不作为后台在线升级源的配置接口。

#### Scenario: 下载默认部署文件
- **WHEN** 操作者运行本 fork 的部署准备脚本且未提供自定义来源
- **THEN** 脚本从 `c4lnn/sub2api` 的 `production` 分支下载部署文件，所得应用配置使用 `ghcr.io/c4lnn/sub2api:latest`

#### Scenario: 使用默认二进制安装源
- **WHEN** 操作者运行安装脚本且未设置 `SUB2API_GITHUB_REPO`
- **THEN** 版本查询及应用压缩包下载均来自 `c4lnn/sub2api` 的 Releases

#### Scenario: 显式覆盖安装仓库
- **WHEN** 操作者为安装脚本明确设置有效的 `SUB2API_GITHUB_REPO`
- **THEN** 安装脚本继续使用该指定仓库，而后台在线升级仍使用本 fork 的既定来源

### Requirement: 在线升级和历史回滚仅跟踪本 fork

系统 MUST 从 `c4lnn/sub2api` 查询更新及历史回滚版本；默认情况下 MUST NOT 请求旧 fork 或上游的应用 Release。历史回滚 SHALL 沿用现有筛选规则：在近期 15 条 Release 的查询窗口内，排除 Draft、Pre-release 和不低于当前版本的条目，按数字版本降序最多展示 3 个候选。没有候选时 SHALL 返回空列表，不补入旧仓库版本或仅有 Git tag 的版本。

#### Scenario: 当前版本为首次正式版本
- **WHEN** 当前版本为 `2026.11.17` 且本 fork 没有符合条件的更低正式 Release
- **THEN** 历史版本回滚列表为空，不从 `ranxi2001/sub2api` 补充候选

#### Scenario: 存在多个较旧正式发布
- **WHEN** 近期查询窗口包含比当前版本低的 4 个正式 Release，以及 Draft 或 Pre-release
- **THEN** 列表只包含其中版本号最高的 3 个较旧正式 Release

#### Scenario: 候选缺少适用平台压缩包
- **WHEN** 操作者选择一个缺少当前操作系统或架构应用压缩包的候选执行回滚
- **THEN** 系统报告无兼容产物，且不从其他仓库下载替代文件

#### Scenario: 恢复已有本地备份
- **WHEN** 操作者使用本地备份回滚且有效的本地备份存在
- **THEN** 系统继续恢复该备份，不要求它出现在本 fork 的 Release 列表中；恢复后的升级来源取决于备份二进制自身的配置

### Requirement: 切换后不得复用旧源升级缓存

系统 MUST 隔离仓库切换前保存的更新信息。切换后的首次查询及网络错误回退 MUST NOT 返回旧仓库缓存中的 Release 链接或附件作为可安装更新；同一新来源的有效缓存及其过期规则 SHALL 保留。

#### Scenario: 旧缓存存在且新源不可访问
- **WHEN** 缓存中只有旧仓库的更新结果且查询新仓库失败
- **THEN** 系统报告查询失败或无法确定更新，不将旧缓存中的产物提供为可执行更新

#### Scenario: 使用新来源有效缓存
- **WHEN** 已成功缓存本 fork 的更新结果且缓存尚未过期
- **THEN** 后续非强制查询可以沿用现有缓存行为返回本 fork 的结果

### Requirement: 手动回滚指令指向本 fork

管理界面生成的手动回滚命令 MUST 使用与候选版本一致的本 fork 版本。脚本命令 SHALL 从 `c4lnn/sub2api` 对应 tag 获取安装脚本；容器命令 SHALL 使用 `ghcr.io/c4lnn/sub2api:<版本>`，镜像 tag 不包含前导 `v`。

#### Scenario: 显示脚本回滚命令
- **WHEN** 操作者为一个合格的较旧版本查看脚本回滚指令
- **THEN** 命令中的脚本地址属于 `c4lnn/sub2api` 的对应 `v<版本>` tag，回滚参数也是该 tag

#### Scenario: 显示容器回滚命令
- **WHEN** 操作者为一个合格的较旧版本查看容器回滚指令
- **THEN** Compose 镜像指令为 `ghcr.io/c4lnn/sub2api:<版本>`，而不是上游 Docker Hub 镜像

### Requirement: 重新登录 runtime 与应用版本匹配

自动准备及受管迁移工具 MUST 从 `c4lnn/sub2api` 的对应应用版本 Release 获取 `sub2api-reauth_<版本>_linux_<架构>.tar.gz`。下载 SHALL 继续要求有效的 Release 附件 SHA-256 并验证内容；缺少附件、摘要无效或内容不匹配时 MUST 失败，不回退到旧仓库或跳过校验。

#### Scenario: 准备首次正式版本 runtime
- **WHEN** `2026.11.17` 的 Linux amd64 或 arm64 实例需要准备 runtime
- **THEN** 下载来自本 fork 的 `v2026.11.17` Release，并使用匹配版本和架构的附件及其摘要进行校验

#### Scenario: runtime 校验失败
- **WHEN** 对应附件不存在、摘要缺失或压缩包校验失败
- **THEN** 准备过程失败，不执行未经校验的下载内容或其他仓库的替代 runtime

### Requirement: 各受支持容器部署入口一致

Compose、Apple Container、K3s 升级工具和 Kubernetes 模板 MUST 默认使用本 fork 的应用镜像；Kubernetes 渲染 SHALL 仅接受 `ghcr.io/c4lnn/sub2api@sha256:<64位小写十六进制摘要>`，不得因迁移放宽为任意仓库或浮动 tag。应用及其同源 init container SHALL 使用一致的已验证镜像摘要。

#### Scenario: 渲染本 fork 的固定镜像
- **WHEN** 操作者向 Kubernetes 渲染工具提供本 fork 的有效 digest 镜像
- **THEN** 输出中应用和相关 init container 均使用该镜像，不残留未固定版本的模板镜像

#### Scenario: 拒绝旧仓库或浮动镜像
- **WHEN** 操作者提供旧仓库镜像、其他 owner 的镜像或 `:latest`
- **THEN** Kubernetes 渲染工具拒绝该输入

#### Scenario: 新建 Apple Container 部署
- **WHEN** 操作者未覆盖 Apple Container 的应用镜像配置
- **THEN** 默认应用镜像为 `ghcr.io/c4lnn/sub2api:latest`

### Requirement: 首次正式版本的各类产物一致

首次新发行 MUST 使用 Git tag `v2026.11.17`，应用报告版本和带版本的镜像、附件 SHALL 使用 `2026.11.17`，Release MUST 为非 Draft、非 Pre-release。完整发布 MUST 提供既有支持平台的主程序压缩包、`checksums.txt`、Linux amd64/arm64 runtime 及双架构应用镜像，并使 `ghcr.io/c4lnn/sub2api:latest` 指向该次正式发行。版本比较 SHALL 沿用现有三段数字规则；后续正式版本 MUST 按此规则单调递增，不依赖后缀区分修订。

#### Scenario: 完整发布新版本
- **WHEN** 对包含本次实现的源码 commit 完整发布 `v2026.11.17`
- **THEN** Release、应用版本、主程序附件、runtime 和镜像的版本一致，产物可追溯到同一应用源码 commit，且 GHCR `latest` 更新

#### Scenario: dry-run 验证
- **WHEN** 操作者为所选源码执行完整发布 dry-run
- **THEN** 构建与产物验证执行，但不向 GHCR、Docker Hub 或 GitHub Releases 发布，也不删除旧 tag

### Requirement: 旧 tag 清理具有明确范围和前置条件

在新正式发布完成且附件、校验、镜像及版本验证通过后，发布切换 MUST 删除 `c4lnn/sub2api` 的远端 tag `v2.21.6-yuanxu.1` 和本地同名 tag，不将其移动至新 commit，也不重新创建该旧 tag。该操作 MUST NOT 删除其他仓库的 tag、旧 Release、附件、镜像或其他版本。规划 artifact 的创建 SHALL 不执行这些发布和删除操作。

#### Scenario: 新正式发行已验证
- **WHEN** `v2026.11.17` 的正式发布验证全部通过且执行旧 tag 清理
- **THEN** 本地与 `c4lnn/sub2api` 均不再存在 `refs/tags/v2.21.6-yuanxu.1`，新 tag 保持不变，其他仓库和旧发布产物不被清理

#### Scenario: 新发布未完成或验证失败
- **WHEN** 新版本发布未完成或任一必要产物验证失败
- **THEN** 不执行旧 tag 删除，保留现有恢复路径并报告失败阶段

### Requirement: 首次迁移说明与实际边界一致

当前部署与发布文档 MUST 描述本 fork 的仓库、版本与操作方式，并明确已有实例需要显式首次部署新二进制或镜像，已有本地配置不会因模板更新自动改变。文档 SHALL 保留第三方署名与历史记录，并说明删除旧 tag 不等于删除旧 Release、附件或镜像，亦不代表业务数据已回滚。

#### Scenario: 操作者迁移已有实例
- **WHEN** 操作者按文档从旧发行来源迁移
- **THEN** 可以找到现有 Compose 或 Apple Container 配置的修改要求、明确版本部署方法、后续更新来源验证和恢复边界，不会被指引通过仅设置安装脚本环境变量来切换后台升级源
