# HOLE CLUB

夜店订台系统，基于 UniApp、Vue 3、TypeScript 和 Go。

当前测试版本：员工账号登录 → 工作台 / 台位 → 两步预订、资料编辑与精确时段排期 → 到店 → 开台 → 加单 → 记录线下收款 → 清台；店长可登记部分退款。支持员工、商品、套餐、1–90 天净营收报表、财务审计筛选及 CSV 导出；服务端执行权限校验、销售数据隔离、服务端定价、多设备同步、变更审计与订单持久化。

新工作台已接入 Go API，不再使用 localStorage 保存业务订单。my-cloud 测试部署使用服务器 SQLite 和用户级 systemd 服务；当前已包含独立员工账号、角色权限、服务端会话失效和店长操作审计，MySQL 迁移属于后续阶段。

详见 [测试部署和验收说明](docs/TEST_DEPLOYMENT.md)、[UI PRD](docs/UI_PRODUCT_PRD.md) 和 [目标架构](docs/ARCHITECTURE_TECH_STACK_QA.md)。

## 本地运行

```bash
cd front
npm ci
npm run dev:h5
```

## 在线预览

GitHub Pages 只托管静态前端，无法提供本版同源 API；完整流程请使用 my-cloud 测试服务。

每次推送 main 后，GitHub Actions 自动部署到：https://zhoujie0420.github.io/club/

## Railway 后端

仓库根目录包含 `Dockerfile` 和 `railway.toml`，Railway 连接本仓库后可直接部署。

1. 在 Railway 创建 Project，选择 **Deploy from GitHub repo** 并选中 `zhoujie0420/club`。
2. 给 Project 添加 MySQL 服务。
3. 在 API 服务中配置 `MYSQL_DSN` 和 `CORS_ALLOWED_ORIGINS`。不要把密码写入仓库。
   本版订单仍使用 SQLite；额外配置 `CLUB_ACCESS_CODE` 和持久卷路径 `CLUB_DB_PATH=/data/club.db`。MySQL 连接仅为后续迁移保留。Docker 镜像仅包含 API，完整 H5 请按测试部署说明单独构建和上传。
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
