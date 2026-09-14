# HOLE CLUB

单门店夜店**员工订台与现场运营**系统。销售 / 前台 / 服务员 / 店长在手机上完成：登录 → 工作台 / 台位 → 两步预订 → 到店 → 开台 → 加单 → 登记线下收款 → 清台。店长可部分退款、管员工、管商品、看报表和审计。

服务端是业务事实来源，订单不放在 localStorage。

| 文档 | 职责 |
| --- | --- |
| 本文件 | 项目入口、怎么跑、当前能用什么 |
| [docs/PRD.md](docs/PRD.md) | 产品范围、角色、流程、验收；**已实现 / 未实现** |
| [docs/TRD.md](docs/TRD.md) | 技术栈、接口、数据、部署；**已实现 / 未实现** |

仓库只维护这三份文档。改功能时必须三份一起改，状态不得互相矛盾。约定见文末。

---

## 当前能用什么

**已上线（H5 + Go API + SQLite）**

员工 PIN 登录、四角色权限、工作台四 Tab（单页 `ClubConsole`）、18 个台位、两步预订、订台全闭环、改台 / 改预订 / 备注 / 取消、加单、线下结账、店长部分退款、1–90 天净营收报表与 CSV、审计筛选与 CSV、员工账号、商品上下架、服务端定价 / 幂等 / 乐观锁 / 占台冲突、5 秒轮询、多设备同步。

测试环境：`http://115.191.3.226:18080/club/`（my-cloud 用户级 systemd + SQLite）。预置账号口令均为 `holeclub`（仅测试，不要用于生产）。

**未实现（仍要做）**

| 项 | 说明 |
| --- | --- |
| 微信小程序员工端 | UniApp 脚本和胶囊避让已有，无 AppID、无备案域名、未发体验版 |
| 阿里云 HTTPS | 试营业唯一对外 API；H5 与小程序打同一 SQLite |
| 去掉 Railway / GitHub Pages | 进行中：停止把它们当发布入口 |
| 真实场地平面图、台位停用 | 现为 VIP/卡座 18 座简化网格 |
| 手机号加密与查看审计 | 现为明文，只允许虚构数据 |
| 金额改「分」 | 现为整数元 |
| 并台、留台超时、销售排行、反结账 | 产品 P1 |
| SQLite 备份演练 | 启动时已清过期会话；备份脚本待上云 cron |
| MySQL 迁移 | 后续，不是现状 |

**本阶段不做**

顾客会员端、线上支付、库存、排班、跨门店、JWT、原生 App、把工作台拆成多页。

完整对照表见 [PRD](docs/PRD.md) 与 [TRD](docs/TRD.md)。

---

## 技术栈（现状）

| 层 | 技术 |
| --- | --- |
| 前端 | UniApp + Vue 3 + TypeScript + Vite，目前只跑 **H5** |
| 后端 | Go 1.23 + **Gin 1.10**，`server/cmd/api` |
| 数据 | SQLite（`CLUB_DB_PATH`）；MySQL 驱动仅 `/readyz` 探测，**不是订单库** |
| 鉴权 | bcrypt 口令 + SQLite `sessions` + `Authorization: Bearer` |
| 测试 | `go test ./...`；Playwright H5 E2E（`front/tests/console.spec.ts`） |

---

## 本地运行

终端 1 — API：

```bash
mkdir -p .artifacts
cd server
go test ./...
PORT=18080 CLUB_ACCESS_CODE=local-test-only \
  CLUB_DB_PATH=../.artifacts/local-test.db \
  CLUB_WEB_DIR=../front/dist/build/h5 \
  go run ./cmd/api
```

终端 2 — H5（Vite 把 `/api` 代理到 `127.0.0.1:18080`）：

```bash
cd front
npm ci
npm run dev:h5
```

浏览器打开 Vite 提示的地址。构建并让 Go 托管静态资源：

```bash
cd front && npm run type-check && npm run build:h5
```

E2E（会写入「自动验收」前缀的测试订单，请用独立库）：

```bash
cd front
npx playwright install chromium
npx playwright test
```

---

## 测试环境（my-cloud）

- 地址：`http://115.191.3.226:18080/club/`
- SSH：`my-cloud`，用户 `deploy`，目录 `/home/deploy/club-test`
- 进程：`systemctl --user status club-test.service`
- 数据：`club.db`（部署不覆盖）
- 预置身份：店长、销售小林、前台小周、服务员阿杰，口令 `holeclub`

发布：`front` 构建 H5 到 `web/`，`GOOS=linux GOARCH=amd64 CGO_ENABLED=0` 编译 `club-api`，上传后重启用户服务。细节与验收步骤见 [TRD 部署](docs/TRD.md#部署)。

本机转发：`ssh -N -L 18081:127.0.0.1:18080 my-cloud` → `http://127.0.0.1:18081/club/`。

---

## 接口速查

- `GET /healthz` 进程存活
- `GET /readyz` SQLite（或可选 MySQL）可连
- `GET /api/v1/meta` 名称与版本
- `POST /api/v1/login` 起，业务 API 见 [TRD](docs/TRD.md)

---

## 文档同步约定

1. 仓库文档只有：`README.md`、`docs/PRD.md`、`docs/TRD.md`。
2. 新增、下线、改范围的功能，同一次提交改三份：README 改摘要与入口，PRD 改产品与验收，TRD 改实现与接口。
3. 每条能力必须标 **已实现 / 部分实现 / 未实现 / 本阶段不做**，三份用同一套状态。
4. 代码与文档冲突时，以代码为准，并立刻改文档。
