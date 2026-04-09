# Ubik Submission Web

作者端投稿系统，基于 Vue 3 + TypeScript + Vite + Pinia + Vue Router + Element Plus。

## 已实现功能

- 登录与注册，注册成功后自动登录
- 注册要求邮箱必填
- 比赛看板（进行中 > 未开始 > 已结束）
- 赛道投稿仅在比赛进行中开放，未开始和已结束阶段仅可查看
- 比赛倒计时进度条（显示倒计时文本，不显示百分比）
- 比赛详情与赛道详情浏览
- 赛道投稿（前端限制仅支持 doc/docx）
- 我的稿件列表、删除、修改稿件信息与可选替换文件
- 作者资料基础维护

## 快速开始

1. 安装依赖

```bash
npm install
```

2. 启动开发环境（默认 5174）

```bash
npm run dev
```

3. 运行类型检查

```bash
npm run typecheck
```

4. 构建生产包

```bash
npm run build
```

## 环境变量

可在项目根目录创建 `.env.local`：

```bash
VITE_SUBMISSION_BASE_URL=/api/submission
VITE_SYSTEM_BASE_URL=/api/system
VITE_REQUEST_TIMEOUT=12000
```

## 开发代理

`vite.config.ts` 已配置本地代理：

- `/api/submission` -> `http://localhost:80/api/v1`
- `/api/system` -> `http://localhost:8082/api/v1`
