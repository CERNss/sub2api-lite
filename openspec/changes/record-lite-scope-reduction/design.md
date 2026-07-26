# Lite 收敛：删减清单与验证记录

> 覆盖 commit 区间 `ad3f36c65..c8d28dc09`（5 个 commit）。
> 汇总：**385 files changed, +15,852 / −103,928**，其中删除文件 241 个
> （`backend/ent/` 生成代码 56 个、`backend/internal/` 98 个、`frontend/src/` 85 个、`docs/` 2 个）。

## 1. 决策依据

lite 分支的定位是**内部管理员自用的「订阅转 API」网关**：不面向终端用户售卖额度，因此不需要收款、返利、促销、公告触达、邀请裂变。保留这些子系统的代价是持续维护 6 个上游子系统的 rebase 冲突、5 个第三方支付 SDK 的安全面，以及 9 张永远为空的表。

删减原则：

1. **能删干净的整体删**（支付、分销、优惠码、公告、用户兑换入口、邀请码门禁）。
2. **与核心网关共用数据的保留**（`redeem_codes` 作为余额/并发调整审计流水；订阅配额；用量与成本统计）。
3. **历史 migration 不改写**，只追加 `154_drop_sales_tables.sql` 做 drop —— 已部署实例必须能从任意旧版本顺序迁移上来。

## 2. 删除的子系统

### 2.1 支付（payment）

| 类别 | 内容 |
|------|------|
| Go 包 | `backend/internal/payment/`（整包：`amount`、`crypto`、`currency`、`fee`、`load_balancer`、`registry`、`types`、`wire`） |
| 支付渠道 | `provider/{airwallex,alipay,easypay,stripe,wxpay}.go` + `factory.go` |
| service | `payment_service.go`、`payment_order*.go`、`payment_config_*.go`、`payment_fulfillment.go`、`payment_refund.go`、`payment_resume_*.go`、`payment_webhook_provider.go`、`payment_stats.go`、`payment_amounts.go`、`payment_currency.go`、`payment_visible_method_instances.go`、`payment_order_expiry_service.go` |
| handler | `payment_handler.go`、`payment_webhook_handler.go`、`admin/payment_handler.go` |
| routes | `backend/internal/server/routes/payment.go`（整文件） |
| ent schema | `payment_order.go`、`payment_provider_instance.go`、`payment_audit_log.go` |
| 端点 | `/api/v1/payment/{config,checkout-info,plans,channels,limits}`、`/api/v1/payment/orders/**`、`/api/v1/payment/public/orders/{verify,resolve}`、`/api/v1/payment/webhook/{easypay,alipay,wxpay,stripe,airwallex}`、`/api/v1/admin/payment/{dashboard,config,orders/**,plans/**,providers/**}` |
| OAuth | `/api/v1/auth/oauth/wechat/payment/{start,callback}`（微信支付授权） |
| 前端 | `views/user/{PaymentView,PaymentQRCodeView,PaymentResultView,StripePaymentView,StripePopupView,AirwallexPaymentView,UserOrdersView}.vue`、`views/user/{paymentUx,paymentWechatResume}.ts`、`views/admin/orders/**`、`components/payment/**`（19 个）、`components/admin/payment/**`（7 个）、`views/auth/WechatPaymentCallbackView.vue`、`stores/payment.ts`、`api/payment.ts`、`api/admin/payment.ts`、`types/payment.ts` |
| 资源 | `assets/icons/{airwallex,alipay,easypay,stripe,wxpay}.svg` |
| 文档 | `docs/PAYMENT.md`、`docs/PAYMENT_CN.md` |

### 2.2 分销返利（affiliate）

| 类别 | 内容 |
|------|------|
| 后端 | `service/affiliate_service.go`、`repository/affiliate_repo.go`、`handler/admin/affiliate_handler.go` |
| 端点 | `/api/v1/user/aff`、`/api/v1/user/aff/transfer`、`/api/v1/admin/affiliates/{invites,rebates,transfers}`、`/api/v1/admin/affiliates/users/**` |
| 设置键 | `affiliate_enabled`、`affiliate_rebate_rate`、`affiliate_rebate_freeze_hours`、`affiliate_rebate_duration_days`、`affiliate_rebate_per_invitee_cap` |
| 前端 | `views/user/AffiliateView.vue`、`views/admin/affiliates/**`（4 个）、`api/admin/affiliates.ts`、`utils/oauthAffiliate.ts`（`aff_code` 引流管线） |
| 数据表 | `user_affiliates`、`user_affiliate_ledger` |

### 2.3 优惠码（promo code）

| 类别 | 内容 |
|------|------|
| 后端 | `service/{promo_service,promo_code,promo_code_repository}.go`、`repository/promo_code_repo.go`、`handler/admin/promo_handler.go` |
| 端点 | `/api/v1/auth/validate-promo-code`、`/api/v1/admin/promo-codes/**` |
| 设置键 | `promo_code_enabled` |
| ent schema | `promo_code.go`、`promo_code_usage.go` |
| 前端 | `views/admin/PromoCodesView.vue`、`api/admin/promo.ts` |
| 数据表 | `promo_codes`、`promo_code_usages` |

### 2.4 公告（announcement）

| 类别 | 内容 |
|------|------|
| 后端 | `service/{announcement_service,announcement}.go`、`repository/{announcement_repo,announcement_read_repo}.go`、`handler/announcement_handler.go`、`handler/admin/announcement_handler.go`、`handler/dto/announcement.go` |
| 端点 | `/api/v1/announcements`、`/api/v1/announcements/:id/read`、`/api/v1/admin/announcements/**` |
| ent schema | `announcement.go`、`announcement_read.go` |
| 前端 | `views/admin/AnnouncementsView.vue`、`components/admin/announcements/**`、`components/common/{AnnouncementBell,AnnouncementPopup}.vue`、`stores/announcements.ts`、`api/{announcements,admin/announcements}.ts` |
| 数据表 | `announcements`、`announcement_reads` |

### 2.5 用户自助兑换（user redeem flow）

删的是**用户侧入口**，不是 `redeem_codes` 数据本身（见 §3）。

| 类别 | 内容 |
|------|------|
| 后端 | `service/redeem_service.go`、`repository/redeem_cache.go`、`handler/redeem_handler.go`、`handler/admin/redeem_handler.go` |
| 端点 | `/api/v1/redeem`、`/api/v1/redeem/history`、`/api/v1/admin/redeem-codes/**`（list/stats/export/generate/batch-*/expire） |
| 前端 | `views/user/RedeemView.vue`、`views/admin/RedeemView.vue`、`api/{redeem,admin/redeem}.ts` |

### 2.6 邀请码注册门禁（invitation code）

| 类别 | 内容 |
|------|------|
| 端点 | `/api/v1/auth/validate-invitation-code` |
| 设置键 | `invitation_code_enabled` |
| 常量 | `RedeemTypeInvitation`、pending session 的 `invitation_required` 状态 |
| 行为变更 | 邮箱注册与全部 OAuth 首登路径不再校验邀请码；OAuth 首登在 `registration_enabled=true` 时直接自动注册。注册现在**仅**由 `registration_enabled` 控制 |
| 前端 | `RegisterView.vue`、`PendingOAuthCreateAccountForm.vue`、各 `*CallbackView.vue` 的邀请码输入与 `invitation_required` 分支 |

### 2.7 前端路由（20 条）

`/redeem`、`/affiliate`、`/purchase`、`/orders`、`/payment/qrcode`、`/payment/result`、`/payment/stripe`、`/payment/airwallex`、`/payment/stripe-popup`、`/auth/wechat/payment/callback`、`/admin/announcements`、`/admin/redeem`、`/admin/promo-codes`、`/admin/affiliates`、`/admin/affiliates/{invites,rebates,transfers}`、`/admin/orders`、`/admin/orders/dashboard`、`/admin/orders/plans`

### 2.8 i18n

`en.ts` −678 行、`zh.ts` −704 行：`affiliate`、`purchase`、`payment`、`announcements` 命名空间，以及 admin 下的 `redeem`、`promo` 文案。

### 2.9 数据表（migration `154_drop_sales_tables.sql`）

按外键依赖顺序 drop 9 张表：

```
user_affiliate_ledger   -- 先删：source_order_id FK 依赖 payment_orders
user_affiliates
promo_code_usages
promo_codes
payment_audit_logs
payment_orders
payment_provider_instances
announcement_reads
announcements
```

创建这些表的历史 migration（033/045/068/092/102/111/112/117/119/120/120a/130/131/132/133/134）**全部保留**，全新安装会先建再删。这是刻意的：migration 序列必须对任意历史版本可重放。

## 3. 刻意保留（删除时的红线）

| 保留项 | 理由 |
|--------|------|
| `redeem_codes` 表 + `redeem_code_repo.go` + `service/redeem_code.go` | admin 调整余额/并发时写入该表作为**审计流水**（`type` = `admin_balance` / `admin_concurrency` / `subscription`）。`/api/v1/admin/users/:id/balance-history` 直接读它。删表等于删掉全部账务审计 |
| 订阅（subscription）与订阅配额进度 | 是网关分组授权模型的一部分，与售卖无关 |
| 用量与成本统计（usage / ops / dashboard） | 网关核心可观测性 |
| `purchase_subscription_enabled` / `purchase_subscription_url` 设置键 | **注**：其消费方 `PurchaseSubscriptionView.vue` 已随支付一起删除，实际已成孤儿（见 §5 残留 R7） |

## 4. 冒烟验证记录（2026-07-26）

环境：`docker compose -f deploy/docker-compose.local.yml up -d`，postgres:18-alpine + redis:8-alpine + 本地构建镜像，全新空库。
R2 / R5 与并发审计另在一次**独立的 native 运行**（原生二进制 + 全新空库）上复验，结论一致。

| 检查项 | 结果 |
|--------|------|
| `go build ./...` | ✅ |
| `go vet ./...` | ✅ 0 |
| `go test -tags=unit ./...` | ✅ 42 包全过 |
| `go test -tags=integration ./...`（testcontainers） | ✅ 38 包全过 |
| `golangci-lint v2.7 run ./...` | ✅ 0 issues |
| 前端 `vue-tsc -b && vite build` | ✅ |
| 前端 `eslint` | ✅ 0 |
| 前端 `vitest run` | ✅ 100 文件 / 624 用例 |
| `docker build`（默认 build-arg） | ❌ 见残留 R1 |
| `docker build --build-arg GOLANG_IMAGE=golang:1.26.5-alpine` | ✅ |
| 容器启动 + `/health` | ✅ 200 |
| migration 全量应用（含 154） | ✅ 无错误，154 已记入 `schema_migrations` |
| 9 张销售侧表 | ✅ 全部不存在，剩余 65 张表 |
| `redeem_codes` 表 | ✅ 保留 |
| 11 个已删端点 | ✅ 全部 404（payment / redeem / promo / affiliate / announcement / validate-promo-code / validate-invitation-code） |
| admin 登录 + 合规确认 + 读设置 | ✅ 195 个设置键，无销售侧键 |
| admin 调整用户余额 → `redeem_codes` 审计行 | ✅ `POST /admin/users/:id/balance` 写入 `admin_balance` 行 |
| admin 调整用户并发 → `redeem_codes` 审计行 | ✅ `PUT /admin/users/:id` 改 `concurrency` 时写入 `admin_concurrency` 行（无专用 `/concurrency` 端点）。**注**：该路径当前无任何单测/集成测试覆盖，本次为运行时实测 |
| `GET /admin/users/:id/balance-history` | ✅ 200（此前因 UNION 已删的 `user_affiliate_ledger` 而 500，已修复） |
| admin 为用户创建 API key（fork 功能） | ✅ |
| 普通用户登录 / 建 key / 查订阅 | ✅ |
| 网关 `POST /v1/messages`（无上游账号） | ✅ 503 `No available accounts`（非 500） |
| 嵌入式前端首页 | ✅ 200 |

**结论：2.1–2.9 的删减本身是干净的** —— 编译、静态检查、全部测试、迁移、运行时端点行为都符合预期，没有因删减而破坏保留功能。

## 5. 残留（删减未清理干净的部分）

以下不是「删错了」，而是「没删完」。均已在 `tasks.md` 列为待办。
标 ✅ 的条目已在 2026-07-26 的 Codex review 轮次中修复（见 §7）。

| # | 严重度 | 残留 |
|---|--------|------|
| R1 | 高 ✅ | `Dockerfile` 的 `ARG GOLANG_IMAGE=golang:1.26.4-alpine` 与 `backend/go.mod` 的 `go 1.26.5` 不一致，alpine 镜像 `GOTOOLCHAIN=local`，`go mod download` 直接失败 → **镜像自 `dd212e9a1` 起构建不出来**。与删减无关，但冒烟必须记。已修复：`Dockerfile` / `deploy/Dockerfile` / `backend/Dockerfile` 对齐 1.26.5 并加同步注释；README/README_JA/DEV_GUIDE 的 1.25.7 一并纠正 |
| R2 | 高 | `service.InitializeDefaultSettings` **全代码库零调用方**（含测试）。commit `0a16904cb` 在其中播种 `backend_mode_enabled=true` 以实现「lite 全新安装即后台模式」，实际未生效 —— 全新安装实测 `backend_mode_enabled=false`，普通用户仍可登录自助 |
| R3 | 中 ✅ | `components/user/dashboard/UserDashboardQuickActions.vue` 仍有「兑换码」按钮 `router.push('/redeem')`，该路由已删 → 落到 404 catch-all。已修复：按钮删除，`dashboard.redeemCode` / `dashboard.addBalanceWithCode` 两个 i18n key 一并移除；`frontend/src/router/README.md` 里两条已删路由文档同步清理 |
| R4 | 中 | `backend/internal/domain/announcement.go`（232 行）完全孤儿，无任何引用 |
| R5 | 中 | migration `111_payment_routing_and_scheduler_flags.sql` 仍向 `settings` 插入 4 行 `payment_visible_method_*`，154 未清理，且 Go 侧已无对应 SettingKey 常量 → 全新库里 4 行死配置（实测存在） |
| R6 | 低 | `payment_enabled` 仍在注入前端的 `window.__APP_CONFIG__` 中（无生产者，恒为 false） |
| R7 | 低 | `purchase_subscription_enabled` / `purchase_subscription_url` 完全孤儿：后端设置键、admin 设置 API、`admin.settings.purchase.*` 文案都在，但视图与路由已删，前端无任何消费方 |
| R8 | 低 | OAuth 起始路径仍调用 `captureOAuthPromoCode()` 写 `oauth_promo_code` cookie，`auth_email_oauth.go` 仍写 `email_oauth_affiliate` cookie —— 只写不读 |
| R9 | 低 | 各 pending/create-account 请求 DTO 仍保留 `AffCode string \`json:"aff_code"\`` 字段（5 处），已无消费方 |
| R10 | 低 | 死类型：Go `dto.PromoCode` / `dto.PromoCodeUsage`；TS `Announcement*`（10 个）、`PromoCode*`（3 个）、`AffiliateInvitee`；`SystemSettings.{PromoCodeEnabled,Affiliate*,PaymentVisibleMethod*}`、`PublicSettings.{PromoCodeEnabled,AffiliateEnabled,PaymentEnabled}` |
| R11 | 低 ◐ | 前端死代码：`api/auth.ts` 的 `validatePromoCode()`（指向已 404 的端点，**已修复**：连同 `ValidatePromoCodeResponse` 与 export 一并删除）；仍待清理：`api/admin/settings.ts` 的 `PaymentVisibleMethod*` 类型与 helper、`vite-env.d.ts` 的 `AirwallexPayments` 声明、`router/meta.d.ts` 的 `requiresPayment` |
| R12 | 低 ◐ | 死白名单：`middleware/backend_mode_guard.go` 的 `/auth/oauth/wechat/payment/callback`（**已修复**：白名单项删除，`backend_mode_guard_test.go` 对应用例翻转为 expect Forbidden）；`guards.spec.ts` 复刻守卫里的 `/payment/result` 与 `/auth/wechat/payment/callback` 条目已删并新增「已删支付路由重定向到 /login」负向用例。仍待办：guards.spec 改为 import 真实守卫实现（当前复刻还缺 `/legal`、dingtalk 回调等真实白名单项，测的仍不是生产逻辑） |
| R13 | 低 | `admin_service.go` 的 `mergeBalanceHistoryCodes(redeemCodes, affiliateCodes, params)` 第二参数恒为 `nil`；孤立注释：`dto/settings.go:197`、`admin/setting_handler.go:563` 的 Alipay 说明、`api/admin/settings.ts` 的 `// Payment configuration` / `// Affiliate feature switch`（其下字段已删，现在错误地盖在 `risk_control_enabled` 上）、`AppHeader.vue` 的 `<!-- Announcement Bell -->` 空注释 |
| R14 | 低 | 遗留 i18n：`admin.settings.payment.*`（仅被 `apiError.ts` 的字段名翻译回退引用）、`admin.settings.purchase.*`、`admin.features.affiliate.*` |
| R15 | 低 ✅ | 指向 `main` 分支的链接全是死链（当时仓库只有 `develop-lite`）：合规文档 `blob/main`（Go 常量 + 前端 fallback），以及 README / README_JA / deploy/README 里的 `raw.githubusercontent .../main/deploy/*.sh` 安装命令 —— 用户照抄会得到 `bash: 404:: command not found`（实测踩中）。**最终解法：把 `main` 立为规范分支**（与 develop-lite 同历史），全部 17 处 URL 统一指 `main`；develop-lite 保留为冻结指针以兼容已发布版本中引用它的脚本 |
| R16 | 中高 ✅ | 删完代码后**没跑 `go mod tidy`**：`backend/go.mod` 仍把 `github.com/stripe/stripe-go/v85`、`github.com/smartwalle/alipay/v3`、`github.com/wechatpay-apiv3/wechatpay-go`、`github.com/shopspring/decimal` 列为**直接依赖**，全代码库零 import。已修复：tidy 移除 4 个直接依赖 + `smartwalle/{ncrypto,ngx,nsign}` 3 个间接依赖，`go.sum` −24 行 |
| R17 | 低 ✅ | 根 `Makefile` 引用不存在的路径：`build-datamanagementd` / `test-datamanagementd` 指向不存在的 `datamanagement/`，`secret-scan` 指向不存在的 `tools/secret_scan.py`（CI 无引用）。已修复：三个 target 与 .PHONY 项删除 |
| R18 | 中 ✅ | **CSP 仍为已删支付 SDK 放行域名**：`config.DefaultCSPPolicy` 与 `middleware/security_headers.go` 的注入规则仍包含 `*.stripe.com` 与 4 个 `airwallex.com` 域（script-src/style-src/frame-src），生产响应头持续为不存在的功能放宽 CSP。已修复：常量、注入规则、默认策略、对应测试全部移除，并新增「不得注入已删支付域名」负向用例 |
| R19 | 低 ✅ | `data/model_pricing.json` + `data/model_pricing.sha256`（6531 行运行时定价缓存）在 `1a342fcc3` 被**误提交**——从仓库根目录跑 server 时 `pricing.data_dir=./data` 落盘产物，commit message 未提及、无任何代码引用。已修复：文件删除，根 `.gitignore` 增加 `/data/` |
| R20 | 低 ✅ | `handler/auth_email_oauth.go` 邀请码删除后残留不可达分支 `if false { return }`。已修复：删除 |

## 6. Codex review 轮次修复（2026-07-26）

独立 Codex 多 agent review（`reviews/review-20260726-151839-b4b015.md`，14 条发现）对照本清单后，当轮修复了 R1、R3（含 router README）、R11 部分、R12 部分、R16–R20。**其中 R18（CSP 支付域名）与 R19（误提交的定价缓存）是 review 轮次新发现**，此前 R1–R16 清单漏掉了它们。

修复后验证：后端 `go build` / `go vet` / `golangci-lint` 零问题、`go test -tags=unit` 42 包全过（含翻转后的 backend-mode guard 用例与新的 CSP 负向用例）；前端 eslint / vitest 全量通过；**默认参数 `docker build` 端到端成功**（R1 的最终证明）。

review 中主动跳过的建议（理由）：
- R2 播种路径：三种方案对存量部署影响不同，仍留待人工定夺（§6 未决问题 → 现 §7）。
- `guards.spec.ts` 改 import 真实实现、`admin_concurrency` 补测（2.8）、CI 边界断言（3.4）：已在 tasks 里跟踪，属后续工作。
- `backend/internal/service` 拆包、前端大组件拆分、eslint 收紧：**刻意不做** —— 保持与上游相同的文件结构是本 fork 降低 rebase 冲突的核心策略。
- CI 跑全量前端测试：改动 CI 门槛需人工决策，且 review 工具对 CI 文件有环境修改红线。
- `user-token-api-key-automation` 补 OpenSpec：FORK.md #5 已声明这是有意的 interim 记录方式。

## 7. 未决问题

- R2 的修复方式需要定夺：是给 `InitializeDefaultSettings` 补调用方（影响所有默认设置的播种时机），还是改由 migration 播种 `backend_mode_enabled`，或改 `IsBackendModeEnabled` 的读取默认值。三者对既有部署的影响不同，不应顺手改。
- R5 的清理需要决定是在 154 里追加 `DELETE FROM settings WHERE key LIKE 'payment_visible_method_%'`，还是新开 155。
