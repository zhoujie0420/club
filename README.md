# HOLE CLUB

夜店订台系统的可交互 MVP，基于 UniApp、Vue 3 和 TypeScript。

当前流程：主页 → 查看房台 → 选择台位 → 确认预订 → 查看/取消订单 → 会员统计。

演示订单当前保存在浏览器 localStorage。`server/` 已提供可部署到 Railway 的 Go API 骨架，后续订单、权限和收银模块会逐步迁移到真实 API。

## 本地运行

```bash
cd front
npm ci
npm run dev:h5
```

## 在线预览

每次推送 main 后，GitHub Actions 自动部署到：https://zhoujie0420.github.io/club/

## Railway 后端

仓库根目录包含 `Dockerfile` 和 `railway.toml`，Railway 连接本仓库后可直接部署。

1. 在 Railway 创建 Project，选择 **Deploy from GitHub repo** 并选中 `zhoujie0420/club`。
2. 给 Project 添加 MySQL 服务。
3. 在 API 服务中配置 `MYSQL_DSN` 和 `CORS_ALLOWED_ORIGINS`。不要把密码写入仓库。
4. 生成 Public Domain，访问 `https://你的域名/healthz` 验证服务。

本地验证：

```bash
cd server
go mod tidy
go test ./...
go run ./cmd/api
```

接口：

- `GET /healthz`：进程健康状态，Railway 部署检查使用。
- `GET /readyz`：数据库连接状态。
- `GET /api/v1/meta`：服务名称和版本。
