# 文件上传系统

一个前后端分离的文件上传产品骨架，包含：

- 前端：Vite + Vue 3 + TypeScript + Element Plus
- 后端：Go + Gin + Gorm
- 数据库：PostgreSQL
- 存储：文件和数据库分别使用 Docker volume 持久化
- 部署：Docker Compose + GitHub Actions

## 功能

- 用户注册和登录
- 登录态有效期 48 小时
- 文件上传、下载、列表展示
- 展示文件名、文件大小、上传时间

## 运行方式

### 本地开发

1. 启动数据库：

```bash
docker compose up -d postgres
```

2. 启动后端：

```bash
cd backend
go run ./cmd/server
```

3. 启动前端：

```bash
cd frontend
npm install
npm run dev
```

### 一键部署

```bash
docker compose up -d --build
```

## 环境变量

后端：

- `APP_PORT`：服务端口，默认 `8080`
- `DATABASE_DSN`：PostgreSQL 连接串
- `JWT_SECRET`：JWT 密钥
- `TOKEN_TTL_HOURS`：登录有效时长，默认 `48`
- `UPLOAD_DIR`：文件存储目录，默认 `/data/uploads`
- `MAX_UPLOAD_MB`：单文件最大值，默认 `100`
- `CORS_ALLOWED_ORIGINS`：逗号分隔的允许来源

## 数据库访问

PostgreSQL 默认只在 Docker Compose 内部网络可访问，不会映射到宿主机端口。后端通过 `postgres:5432` 连接数据库。

## GitHub Actions

仓库包含一个部署工作流示例。你需要在 GitHub Secrets 中配置：

- `SSH_HOST`
- `SSH_PORT`
- `SSH_USER`
- `SSH_KEY`
- `FILE_UPLOAD_DEPLOY_PATH`
- `BACKEND_ENV_FILE`（可选）
