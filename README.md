# HOLE CLUB

夜店订台系统的可交互 MVP，基于 UniApp、Vue 3 和 TypeScript。

当前流程：主页 → 查看房台 → 选择台位 → 确认预订 → 查看/取消订单 → 会员统计。

演示订单保存在浏览器 localStorage；后续将接入 Go API、数据库、登录、支付与后台管理。

## 本地运行

```bash
cd front
npm ci
npm run dev:h5
```

## 在线预览

每次推送 main 后，GitHub Actions 自动部署到：https://zhoujie0420.github.io/club/
