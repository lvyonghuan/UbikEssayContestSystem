# Ubik

征文系统。

## 测试体系

项目内包含三层测试：

1. 单元/模块测试：`go test ./...`
2. 提交端私有样本测试：`go test ./submission -run TestPrivateFilesE2ERegisterToUploadWithHooks -v`
3. 强依赖全链路测试（管理员建赛到作者投稿上传）：`go test ./integration -run TestStrongDepsFullChainAllFiles -v -count=1`

## 强依赖全链路测试说明

`TestStrongDepsFullChainAllFiles` 会拉通以下流程：

1. 管理员登录
2. 创建比赛与赛道
3. 创建脚本定义、上传脚本版本、创建流程与挂载
4. 作者注册/登录
5. 使用 `tests_files` 全量原始文件执行投稿与上传
6. 校验数据库与 `files/submissions` 文件落盘结果

### 内置脚本策略

- 限制投稿数脚本: scripts/submission_hooks/limit_three_submissions.py
	- 事件: submission/submission_pre
	- 规则: 同一作者在同一比赛维度达到上限后拒绝投稿
- 字数统计脚本: scripts/submission_hooks/count_docx_words.py
	- 事件: submission/file_post
	- 规则: 读取已保存 docx 内容，统计字数后写入 work_infos.word_count
	- 依赖: 运行脚本的 Python 环境需安装 PyICU

### 字数统计脚本挂载说明

`scripts/submission_hooks/count_docx_words.py` 仅提供脚本实现，不会在系统启动时自动创建默认挂载。

管理员可通过脚本流配置将该脚本挂载到 `submission/file_post`：

1. 创建脚本定义（interpreter=`python`）
2. 上传脚本版本（文件路径使用 `scripts/submission_hooks/count_docx_words.py`）
3. 创建脚本流（例如 `file_post_word_count`）
4. 为脚本流添加步骤，建议 `failureStrategy=fail_close`
5. 创建挂载：`scope=submission`，`eventKey=file_post`，`targetType=track` 或 `global`

脚本输入取自上传接口 file_post payload，核心字段为：

- `payload.savedPath`: 已落盘 docx 路径
- `payload.workID`: 作品 ID
- `payload.trackID`: 赛道 ID

脚本输出 JSON 中的 patch 仅写入：

- `word_count` -> `works.work_infos.word_count`

### 场景覆盖

强依赖全链路测试默认覆盖以下场景：

1. 比赛进行中：全量文件执行
2. 比赛未开始：投稿拒绝
3. 比赛已结束：投稿拒绝
4. 达到上限：第 4 篇投稿拒绝
5. 删除后重投：删除 1 篇后重新投稿成功
6. 跨赛道累计上限：同一比赛跨赛道累计仍受限
7. 文件后置脚本异常输出：上传失败并返回可追踪错误
8. 文件名解析回退：异常命名文件仍可测试

## 运行前准备

执行强依赖全链路测试前，需要：

1. PostgreSQL 可连接，且已初始化 `ubik` 数据库与表结构
2. Redis 可连接
3. 设置 JWT 环境变量（测试中未设置时会使用默认测试值）
4. `tests_files` 下存在测试文件样本

常用初始化：

- `database/sqls/init_database.sql`
- `database/sqls/init_tables.sql`
- `database/sqls/drop_tables.sql`

## 结果留存

测试结果默认保留，不自动清理：

1. 报告文件：`tests_files/private_e2e_results/`
2. 投稿文件：`files/submissions/<track_id>/<author_id>/<work_id>.docx`
3. 数据库记录：`contests`、`tracks`、`authors`、`works`、`script_*`、`action_logs`

## 私有数据策略

私有样本和产物默认通过 `.gitignore` 过滤，避免误提交：

1. `tests_files/`
2. `tests_files/private_e2e_results/`
3. `files/`
