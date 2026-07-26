## 1. 子系统删除（已随 `ad3f36c65..c8d28dc09` 完成）

- [x] 1.1 删除支付子系统：`internal/payment/` 整包、5 个渠道 provider、订单/退款/履约/webhook service、公开与 admin 路由。
- [x] 1.2 删除分销返利：service / repo / admin handler / `aff_code` 引流管线 / 5 个设置键。
- [x] 1.3 删除优惠码：service / repo / admin handler / `validate-promo-code` 端点 / `promo_code_enabled` 设置键。
- [x] 1.4 删除公告：service / repo / 用户与 admin handler / DTO / 前端铃铛与弹窗。
- [x] 1.5 删除用户自助兑换入口与 admin 兑换码管理页（保留 `redeem_codes` 数据与仓储）。
- [x] 1.6 删除邀请码注册门禁：端点、设置键、`RedeemTypeInvitation`、`invitation_required` pending 状态；注册改为仅由 `registration_enabled` 控制。
- [x] 1.7 删除 7 个 ent schema 与对应生成代码，重跑 `go generate ./ent`。
- [x] 1.8 新增 `154_drop_sales_tables.sql`，按外键顺序 drop 9 张表；保留全部历史 migration。
- [x] 1.9 清理前端：20 条路由、85 个文件、1382 行 i18n、侧边栏菜单项、公共设置字段与类型。
- [x] 1.10 修复删除引入的回归：admin 余额历史不再 UNION 已删的 `user_affiliate_ledger`。
- [x] 1.11 删除 `docs/PAYMENT*.md` 及 README 引用。

## 2. 冒烟验证（已完成 2026-07-26）

- [x] 2.1 `go build` / `go vet` / `golangci-lint v2.7` 全部零问题。
- [x] 2.2 `go test -tags=unit`（42 包）与 `-tags=integration`（38 包）全过。
- [x] 2.3 前端 `vue-tsc` + `eslint` + `vitest`（100 文件 / 624 用例）全过。
- [x] 2.4 全新空库 docker compose 起栈，migration 全量应用无错，154 已记录。
- [x] 2.5 确认 9 张销售侧表不存在、`redeem_codes` 保留。
- [x] 2.6 确认 11 个已删端点全部 404。
- [x] 2.7 确认保留链路可用：admin 登录/设置、余额调整→审计流水、并发调整→审计流水、余额历史 200、admin 为用户建 key、用户自助建 key、订阅查询、网关 503（非 500）、嵌入式前端可访问。
- [ ] 2.8 补测：`admin_concurrency` 审计写入路径（`AdminService.UpdateUser` 的 `concurrencyDiff` 分支）当前**零测试覆盖**，仅靠本次运行时实测。

## 3. 规范固化（本 change）

- [x] 3.1 编写 `lite-scope-boundary` spec，锁定删减边界。
- [x] 3.2 编写 `lite-retained-surface` spec，锁定刻意保留项。
- [x] 3.3 在 `openspec/FORK.md` 追加 overlay 条目并更新快速概览表。
- [ ] 3.4 补一条 CI 检查，用 spec 里的端点/表清单做回归断言（当前只有人工冒烟）。

## 4. 残留清理（待办，详见 `design.md` §5）

### 4.1 阻塞级

- [x] 4.1.1 **R1** 修 `Dockerfile` 的 `ARG GOLANG_IMAGE`，与 `backend/go.mod` 的 Go 版本对齐。已同步 `Dockerfile` / `deploy/Dockerfile` / `backend/Dockerfile` 至 1.26.5 并加注释；README/README_JA/DEV_GUIDE 版本文案一并纠正；默认参数 `docker build` 实测通过。
- [ ] 4.1.2 **R2** 决定并实现 `backend_mode_enabled` 的播种路径 —— `InitializeDefaultSettings` 零调用方，`0a16904cb` 的「lite 默认后台模式」未生效。修复前需先定夺方案（见 `design.md` §7）。
- [x] 4.1.3 **R16** 跑 `go mod tidy` 并提交 `go.mod` / `go.sum`，移除 Stripe / Alipay / WeChat Pay / decimal 4 个已无 import 的直接依赖及其 3 个间接依赖（go.sum −24 行）。

### 4.2 用户可见

- [x] 4.2.1 **R3** 删除 `UserDashboardQuickActions.vue` 的「兑换码」按钮（指向已删的 `/redeem`）。连带删除 `dashboard.redeemCode` / `dashboard.addBalanceWithCode` i18n key，并清理 `frontend/src/router/README.md` 的两条已删路由文档。

### 4.3 死代码 / 死配置

- [ ] 4.3.1 **R4** 删除孤儿 `backend/internal/domain/announcement.go`。
- [ ] 4.3.2 **R5** 清理 migration 111 播种的 4 行 `payment_visible_method_*` 设置。
- [ ] 4.3.3 **R6** 从注入的公共设置里去掉 `payment_enabled`（注意这是前端可见契约变更）。
- [ ] 4.3.4 **R7** 决定 `purchase_subscription_*` 去留：要么恢复入口，要么连同后端设置键、admin API 字段、i18n 一起删。
- [ ] 4.3.5 **R8** 删除只写不读的 `oauth_promo_code` / `email_oauth_affiliate` cookie 写入。
- [ ] 4.3.6 **R9** 删除 5 处请求 DTO 的 `aff_code` 字段。
- [ ] 4.3.7 **R10/R11** 删除死类型与死 API helper。已完成：`validatePromoCode()` + `ValidatePromoCodeResponse` + export 删除。仍待办：`dto.PromoCode*`、TS `Announcement*`/`PromoCode*`/`AffiliateInvitee`、`SystemSettings`/`PublicSettings` 残留字段、`PaymentVisibleMethod*` helper、`AirwallexPayments` 声明、`requiresPayment` meta。
- [ ] 4.3.8 **R12** 已完成：`backend_mode_guard.go` 微信支付回调白名单项删除（对应测试翻转为 Forbidden）；`guards.spec.ts` 复刻数组中的支付路径删除并新增负向用例。仍待办：把 `guards.spec.ts` 改为 import 真实守卫实现，而不是复刻一份（当前复刻还缺 `/legal`、dingtalk 回调等真实项）。
- [ ] 4.3.9 **R13** 清理 `mergeBalanceHistoryCodes` 的空参数与各处孤立注释。
- [ ] 4.3.10 **R14** 清理遗留 i18n 命名空间。
- [x] 4.3.11 **R15** 死链修复（用户实测 404 触发）。最终解法：把 `main` 立为规范分支，合规文档常量（`admin_compliance.go`）、前端 fallback（`adminCompliance.ts` / `AdminComplianceDialog.vue`）、README / README_JA / `deploy/README.md` 的 curl 安装命令、`install.sh` / `docker-deploy.sh` 的 raw URL 共 17 处统一指向 `main`；develop-lite 保留为冻结指针兼容已发布脚本。

## 5. Codex review 轮次修复（2026-07-26，已完成）

> 独立 review（`reviews/review-20260726-151839-b4b015.md`）触发的当轮修复，验证记录见 `design.md` §6。

- [x] 5.1 **R17** 根 `Makefile` 删除指向不存在路径的 `build-datamanagementd` / `test-datamanagementd` / `secret-scan` target。
- [x] 5.2 **R18** CSP 去支付化：`config.DefaultCSPPolicy` 与 `middleware/security_headers.go` 移除 Stripe / Airwallex 域名注入，测试改为负向断言。
- [x] 5.3 **R19** 删除误提交的 `data/model_pricing.json` / `data/model_pricing.sha256`，根 `.gitignore` 增加 `/data/`。
- [x] 5.4 **R20** 删除 `auth_email_oauth.go` 的 `if false { return }` 死分支。
