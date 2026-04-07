# Ubik Admin Web

基于 Vue 3 + TypeScript + Vite + Pinia + Vue Router + Element Plus 的管理后台前端。

## 快速开始

1. 安装依赖

```bash
npm install
```

2. 启动开发环境（默认 5173）

```bash
npm run dev
```

3. 构建生产包

```bash
npm run build
```

## 环境变量

可在项目根目录创建 `.env.local`：

```bash
VITE_USE_MOCK=false
VITE_ADMIN_BASE_URL=/api/admin
VITE_SYSTEM_BASE_URL=/api/system
VITE_REQUEST_TIMEOUT=12000
```

- `VITE_USE_MOCK=true` 时使用 MSW 模拟后端。
- `VITE_USE_MOCK=false` 时走真实后端。

## 跨域与多端口

已在 `vite.config.ts` 配置开发代理：

- `/api/admin` -> `http://localhost:8081/api/v1`
- `/api/system` -> `http://localhost:8082/api/v1`

浏览器只访问前端源，避免本地开发时跨域问题。

## 测试命令

```bash
npm run typecheck
npm run test:unit
npm run test:e2e
npm run test
```

- `test:unit` 包含单元测试与接口层集成测试（MSW）。
- `test:e2e` 使用 Playwright 覆盖登录与导航主流程。

## 当前功能覆盖

- 登录与会话管理（Access/Refresh）
- 赛事管理（创建/编辑/删除）
- 赛道管理（按赛事筛选，创建/编辑/删除）
- 看板、全局配置、角色权限、管理员管理（可扩展页面骨架）

## 扩展约定

新增模块建议遵循：

- `src/types`：接口与领域类型
- `src/services/repositories`：数据访问层
- `src/stores`：业务状态层
- `src/views`：页面层
- `src/tests`：测试覆盖
