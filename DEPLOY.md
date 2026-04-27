# Software Web Manager 部署指南

统一软件版本管理平台（Go + React + Ant Design）。支持版本发布、灰度/预览、回滚、强更/可选更新、渠道分发与数据分析。

## 环境要求

| 组件 | 版本要求 |
|------|----------|
| Go | >= 1.22 |
| Node.js | >= 18 |
| MySQL | >= 5.7 |

## 项目结构

```
goprj/
├── backend/          # Go 后端 API 服务
│   ├── cmd/api/     # API 服务入口
│   ├── cmd/worker/  # Worker 服务入口
│   ├── internal/    # 内部包
│   └── migrations/  # 数据库迁移
├── web/             # React 前端
├── sdk/             # 多语言 SDK
├── deploy/          # Docker 部署配置
└── release/        # 打包输出目录
```

---

## 快速部署

### 方式一：一键打包（推荐）

```bash
# 运行打包脚本
./build.sh
```

打包脚本会自动：
1. 下载 Go 依赖
2. 编译后端主服务二进制（`swm-server`，内置聚合任务）
3. 可选编译兼容 Worker 二进制（`swm-worker`）
4. 安装前端依赖
5. 构建前端生产版本
6. 复制配置文件

### 方式二：手动打包

#### 1. 打包后端

```bash
cd backend

# 安装依赖
go mod tidy

# 编译主二进制（推荐，单进程部署）
CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags="-s -w" -o swm-server ./cmd/api/main.go

# 可选：兼容模式编译独立 worker
CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags="-s -w" -o swm-worker ./cmd/worker/main.go
```

#### 2. 打包前端

```bash
cd web

# 安装依赖
npm install

# 构建生产版本
npm run build
```

---

## 配置说明

### 环境变量

| 变量名 | 说明 | 默认值 |
|--------|------|--------|
| APP_ENV | 环境 | dev |
| HTTP_ADDR | 服务监听地址 | :8080 |
| DATABASE_URL | 数据库连接串 | - |
| JWT_SECRET | JWT 密钥 | - |
| JWT_ISSUER | JWT 发行者 | swm |
| STORAGE_DRIVER | 存储驱动 (local/s3) | local |
| LOCAL_STORAGE_PATH | 本地存储路径 | ./data/files |
| LOCAL_PUBLIC_BASE_URL | 文件访问地址 | http://localhost:8088/files |
| RUN_MIGRATIONS | 自动运行迁移 | true |
| ENABLE_EMBEDDED_WORKER | 是否启用内置聚合任务 | true |
| WORKER_INTERVAL_SECONDS | 聚合任务执行间隔（秒） | 3600 |
| CORS_ORIGINS | 跨域允许域名 | * |

### 示例配置

```bash
APP_ENV=production
HTTP_ADDR=:8080
DATABASE_URL=username:password@tcp(localhost:3306)/dbname?charset=utf8mb4&parseTime=true&loc=Local&multiStatements=true
JWT_SECRET=your-secure-random-secret-key
JWT_ISSUER=swm
STORAGE_DRIVER=local
LOCAL_STORAGE_PATH=./data/files
LOCAL_PUBLIC_BASE_URL=http://your-domain.com:8088/files
RUN_MIGRATIONS=false
ENABLE_EMBEDDED_WORKER=true
WORKER_INTERVAL_SECONDS=3600
CORS_ORIGINS=https://your-domain.com
```

---

## 运行服务

### 1. 准备数据库

```bash
# 登录 MySQL
mysql -u root -p

# 创建数据库和用户
CREATE DATABASE dbname CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci;
CREATE USER 'username'@'localhost' IDENTIFIED BY 'password';
GRANT ALL PRIVILEGES ON dbname.* TO 'username'@'localhost';
FLUSH PRIVILEGES;
```

### 2. 复制并修改配置

```bash
cd release

# 复制配置模板
cp .env.production .env

# 编辑配置文件
nano .env
```

### 3. 启动服务

```bash
# 启动后端主服务（内置聚合任务）
./swm-server

# 后台运行
./swm-server &
```

### 4. 访问服务

- 后端 API: http://localhost:8080
- 健康检查: http://localhost:8080/healthz

### 5. 前端部署

#### 开发模式
```bash
cd web
npm run dev
# 访问 http://localhost:5173
```

#### 生产模式（推荐使用 Nginx）

将 `web/dist` 目录配置为 Nginx root：

```nginx
server {
    listen 80;
    server_name your-domain.com;

    root /path/to/release/web/dist;
    index index.html;

    location / {
        try_files $uri $uri/ /index.html;
    }

    location /api/ {
        proxy_pass http://127.0.0.1:8080;
    }
}
```

---

## 验证部署

### API 测试

```bash
# 健康检查
curl http://localhost:8080/healthz

# 注册用户
curl -X POST http://localhost:8080/api/auth/register \
  -H "Content-Type: application/json" \
  -d '{"email":"admin@example.com","password":"password123","name":"Admin","org_name":"My Org"}'
```

---

## 常见问题

### 1. 编译错误：golang-migrate 版本

如遇到 `version "v4.17.1" invalid` 错误，修改 `backend/go.mod`：

```go
// 删除这两行
github.com/golang-migrate/migrate/v4/database/mysql v4.17.1
github.com/golang-migrate/migrate/v4/source/file v4.17.1

// 替换为
github.com/pressly/goose/v3 v3.23.0
```

### 2. Node.js 版本过低

```bash
# 使用 n 升级 Node.js
npm install -g n
n 20
```

### 3. 数据库连接失败

- 检查 MySQL 是否运行：`systemctl status mysql`
- 检查 DATABASE_URL 格式是否正确
- 检查数据库用户权限

---

## 目录说明

| 目录 | 说明 |
|------|------|
| release/swm-server | 后端主服务二进制（推荐） |
| release/swm-worker | 后端 Worker 二进制 |
| release/web/ | 前端静态文件 |
| release/migrations/ | 数据库迁移文件 |
| release/.env | 运行配置文件 |

---

## 技术栈

- **后端**: Go + Gin + GORM + MySQL
- **前端**: React + Ant Design + Vite + TypeScript
- **存储**: 本地文件系统 / S3 兼容对象存储
