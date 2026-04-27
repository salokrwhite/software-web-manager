# Software Web Manager

## Quickstart

### 1. 启动基础设施

```bash
cd deploy
cp .env.example .env
# 根据需要修改.env

docker compose up -d
```

如果使用 MinIO/S3，需要确保已创建 `S3_BUCKET` 对应的 bucket（默认 `swm`）。

### 2. 启动后端

```bash
cd backend
# Windows PowerShell
$env:RUN_MIGRATIONS="true"
$env:HTTP_ADDR=":8080"
$env:DATABASE_URL="swm:swm@tcp(localhost:3306)/swmanager?charset=utf8mb4&parseTime=true&loc=Local&multiStatements=true"

go run ./cmd/api
```

### 3. 启动前端

```bash
cd web
npm install
npm run dev
```

### 4. SDK 使用

查看 `sdk/` 目录下的 Go / Node / C# / Java / C++ / Rust / Python 示例代码。

## 发布流程（发布端）

1. 创建应用（管理后台或 `POST /api/apps`）。
   - 个人用户创建/修改应用后需要系统管理员审核，审核通过前无法进行任何应用操作。
2. 新建版本（`POST /api/apps/{id}/releases`）。
3. 上传制品（`POST /api/releases/{id}/artifacts`）。
4. 提交审核（`POST /api/releases/{id}/submit`）。
5. 审批通过（`POST /api/releases/{id}/approve`）。
6. 发布到渠道（`POST /api/releases/{id}/publish`，可设置灰度与目标规则）。
7. 需要回滚时使用（`POST /api/releases/{id}/rollback`）。

