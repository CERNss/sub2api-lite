## Why

`develop-lite` 把上游 Sub2API 从「面向终端用户售卖额度的平台」收敛成「内部管理员自用的订阅转 API 网关」。这一收敛跨 5 个 commit（`92e839574` → `c8d28dc09`），删掉 241 个文件、净减约 10 万行，涉及支付、分销返利、优惠码、用户兑换、公告、邀请码注册六个子系统。

这些删减此前只散落在 commit message 里，没有单一事实来源。后果有三个：

1. 上游 rebase 时无从判断某个冲突文件「是我们删的」还是「我们漏合的」。
2. 无法机器校验删掉的端点/表/设置是否被悄悄改回来。
3. 已经出现了残留：`InitializeDefaultSettings` 播种的 `backend_mode_enabled=true` 是死代码（该函数无任何调用方），导致「lite 全新安装默认后台模式」这一意图**实际未生效**；用户仪表盘还留着指向已删 `/redeem` 路由的按钮。

本 change 把删减范围固化为可验证的 spec，并把冒烟中发现的残留列为待办。

## What Changes

- 新增 `lite-scope-boundary` capability：以「必须仍然不存在」的形式，锁定被删除的 HTTP 端点、数据表、设置键、前端路由。
- 新增 `lite-retained-surface` capability：锁定**故意保留**的部分（`redeem_codes` 兼作余额/并发审计流水、订阅配额、用量与成本统计），防止后续清理时误删。
- 在 `openspec/FORK.md` 增加对应 overlay 条目，使删减与既有 5 条 fork change 同级可追踪。
- 记录冒烟验证结论与残留清单（见 `design.md` / `tasks.md`）。
- Codex review 轮次（2026-07-26）修复其中机械性残留：Go 版本对齐（R1）、`go mod tidy`（R16）、CSP 去支付域名（R18）、误提交定价缓存（R19）、`/redeem` 死按钮（R3）、死分支/死白名单/死 helper（R11/R12/R20 部分）、Makefile 死 target（R17）。需要人工定夺的（R2 播种路径等）仍保持待办。

## Capabilities

### New Capabilities
- `lite-scope-boundary`: 销售侧子系统在 lite 分支上必须保持不可达（端点 404、表不存在、设置键不再被读写、前端路由不存在）。
- `lite-retained-surface`: lite 分支必须保留的、与销售侧同源但仍在使用的能力（redeem_codes 审计流水、订阅、用量统计）。

### Modified Capabilities
- None.

## Impact

- 后端：`backend/internal/{handler,service,repository,payment}`、`backend/ent/schema`、`backend/internal/server/routes`、`backend/migrations/154_drop_sales_tables.sql`
- 前端：`frontend/src/{views,components,api,stores,types,i18n,router}`
- 数据库：9 张表被 drop（migration 154）
- 部署：已发布实例升级到该分支后，`/api/v1/payment/*`、redeem、promo、affiliate、announcement 端点全部消失；集成方必须先迁走
- 文档：`docs/PAYMENT.md`、`docs/PAYMENT_CN.md` 已删除，README 相关段落已清理

## Fork Touchpoints

### New Files
- `openspec/changes/record-lite-scope-reduction/proposal.md`: 本提案。
- `openspec/changes/record-lite-scope-reduction/design.md`: 完整删减清单 + 保留项理由 + 冒烟证据 + 残留发现。
- `openspec/changes/record-lite-scope-reduction/tasks.md`: 已完成的删减动作与待处理残留。
- `openspec/changes/record-lite-scope-reduction/specs/lite-scope-boundary/spec.md`: 删减边界需求。
- `openspec/changes/record-lite-scope-reduction/specs/lite-retained-surface/spec.md`: 保留面需求。
- `backend/migrations/154_drop_sales_tables.sql`: drop 9 张销售侧表，保留 `redeem_codes`。

### Upstream Patch Files
- `backend/internal/server/routes/admin.go`: 移除 announcements / redeem-codes / promo-codes / affiliates 路由组。
- `backend/internal/server/routes/auth.go`: 移除 `validate-promo-code`、`validate-invitation-code`、微信支付 OAuth start/callback。
- `backend/internal/server/routes/user.go`: 移除 `/aff`、`/aff/transfer`、`/announcements`、`/redeem` 路由组。
- `backend/internal/service/setting_service.go`: 删除支付/优惠码/分销/邀请码设置键的持久化与校验。
- `backend/internal/service/domain_constants.go`: 删除对应 SettingKey 常量。
- `backend/internal/handler/dto/settings.go`: 删除对应 DTO 字段。
- `backend/internal/handler/admin/setting_handler.go`: 删除 admin 设置透传与审计 diff 字段。
- `backend/internal/service/auth_service.go`: 注册流程去掉邀请码/优惠码/分销绑定。
- `backend/internal/service/auth_oauth_email_flow.go`: OAuth 邮箱注册路径去掉邀请码校验与分销透传。
- `backend/internal/service/admin_service.go`: 余额历史不再 UNION 已删的 `user_affiliate_ledger`。
- `backend/internal/handler/auth_oidc_oauth.go`: OIDC 首登去掉邀请码门禁。
- `backend/internal/handler/auth_linuxdo_oauth.go`: LinuxDo 首登去掉邀请码门禁。
- `backend/internal/handler/auth_wechat_oauth.go`: 微信首登去掉邀请码门禁。
- `backend/internal/handler/auth_dingtalk_oauth.go`: 钉钉首登去掉邀请码门禁。
- `backend/internal/handler/auth_email_oauth.go`: 邮箱 OAuth 首登去掉邀请码门禁。
- `backend/internal/service/auth_email_oauth_auto.go`: 自动注册路径去掉邀请码线索。
- `backend/internal/handler/auth_oauth_pending_flow.go`: 去掉 `invitation_required` pending 状态。
- `backend/cmd/server/wire.go`: 去掉支付/分销/公告依赖注入。
- `backend/cmd/server/wire_gen.go`: wire 重新生成，同步移除上述注入。
- `backend/internal/server/api_contract_test.go`: 契约测试同步删除条目。
- `backend/go.mod`: tidy 移除 Stripe / Alipay / WeChat Pay / decimal 等已无 import 的支付依赖。
- `backend/go.sum`: 随 tidy 同步（−24 行）。
- `backend/internal/config/config.go`: `DefaultCSPPolicy` 去掉 Stripe / Airwallex 域名。
- `backend/internal/server/middleware/security_headers.go`: 删除支付 SDK 域名常量与 CSP 注入规则。
- `backend/internal/server/middleware/security_headers_test.go`: 支付域名用例替换为「不得注入」负向断言。
- `backend/internal/server/middleware/backend_mode_guard.go`: 删除已不存在的微信支付回调白名单项。
- `backend/internal/server/middleware/backend_mode_guard_test.go`: 对应用例翻转为 expect Forbidden。
- `frontend/src/router/index.ts`: 删除 20 条销售侧路由。
- `frontend/src/router/README.md`: 路由文档同步移除 `/redeem`、`/admin/redeem`。
- `frontend/src/router/__tests__/guards.spec.ts`: 复刻守卫中的支付路径删除，新增已删路由重定向负向用例。
- `frontend/src/components/layout/AppSidebar.vue`: 删除对应菜单项。
- `frontend/src/components/user/dashboard/UserDashboardQuickActions.vue`: 删除指向已删 `/redeem` 的「兑换码」快捷入口。
- `frontend/src/api/auth.ts`: 删除 promo/invitation API（含 `validatePromoCode` helper）。
- `frontend/src/views/auth/RegisterView.vue`: 去掉邀请码输入与校验。
- `frontend/src/views/auth/OAuthCallbackView.vue`: 去掉 `invitation_required` 处理。
- `frontend/src/views/auth/OidcCallbackView.vue`: 同上。
- `frontend/src/views/auth/LinuxDoCallbackView.vue`: 同上。
- `frontend/src/views/auth/WechatCallbackView.vue`: 同上。
- `frontend/src/views/auth/DingTalkCallbackView.vue`: 同上。
- `frontend/src/views/auth/DingTalkEmailCompletionView.vue`: 同上。
- `frontend/src/views/auth/EmailVerifyView.vue`: 去掉邀请码透传。
- `frontend/src/components/auth/PendingOAuthCreateAccountForm.vue`: 去掉邀请码输入与 `invitation_required` 分支。
- `frontend/src/api/user.ts`: 删除 aff/redeem 相关 API 与类型。
- `frontend/src/api/admin/settings.ts`: 删除销售侧设置字段。
- `frontend/src/i18n/locales/en.ts`: 删除 affiliate / purchase / payment / announcements / admin redeem / promo 文案（678 行）。
- `frontend/src/i18n/locales/zh.ts`: 同上（704 行）。
- `frontend/src/stores/app.ts`: 删除对应公共设置字段。
- `frontend/src/types/index.ts`: 删除对应公共设置类型。
- `deploy/docker-compose.local.yml`: 持久化目录收敛到 `./data`。
- `deploy/docker-deploy.sh`: 同上，并改为 compose v2 语法。
- `README.md`: 删除支付文档引用；Go 版本徽章对齐 go.mod；安装命令 URL 改指 `develop-lite`。
- `README_JA.md`: 同上。
- `deploy/README.md`: 安装命令 URL 改指 `develop-lite`（本 fork 无 `main` 分支）。
- `backend/internal/service/admin_compliance.go`: 合规文档 URL 从 `blob/main` 改为 `blob/develop-lite`。
- `frontend/src/stores/adminCompliance.ts`: 合规文档 fallback URL 同步。
- `frontend/src/components/admin/AdminComplianceDialog.vue`: 同上。

### Shared Touchpoints
- `backend/internal/handler/admin/setting_handler.go`: also owned by `add-external-custom-menu-token-open` and `control-oidc-local-email-verification` — 三者都改 admin 设置透传；本 change 删销售侧字段，对方两个加自己的字段，rebase 时三份改动都要在。
- `backend/internal/handler/dto/settings.go`: also owned by `add-external-custom-menu-token-open` and `control-oidc-local-email-verification` — 同一 DTO，删与加并存。
- `backend/internal/service/setting_service.go`: also owned by `add-external-custom-menu-token-open` and `control-oidc-local-email-verification` — 同一设置持久化路径。
- `backend/internal/service/domain_constants.go`: also owned by `control-oidc-local-email-verification` — 本 change 删销售侧 SettingKey，对方加 OIDC SettingKey。
- `backend/internal/service/admin_service.go`: also owned by `add-admin-user-api-key-creation` — 本 change 改余额历史合并路径，对方加 `CreateUserAPIKey` / `TransferAPIKey`。
- `backend/internal/server/api_contract_test.go`: also owned by `add-admin-user-api-key-creation` and `user-token-api-key-automation` — 契约清单同时被删条目与加条目修改。
- `backend/internal/handler/auth_oauth_pending_flow.go`: also owned by `control-oidc-local-email-verification` — 本 change 删 `invitation_required` 状态，对方加本地邮箱验证状态。
- `backend/internal/handler/auth_oidc_oauth.go`: also owned by `control-oidc-local-email-verification` — 同一 OIDC 回调，删邀请码门禁与加验证标志并存。
- `backend/internal/service/auth_oauth_email_flow.go`: also owned by `control-oidc-local-email-verification` — 同一 OAuth 邮箱注册路径。
- `frontend/src/components/layout/AppSidebar.vue`: also owned by `add-external-custom-menu-token-open` — 本 change 删销售侧菜单项，对方加 external menu 分发。
- `frontend/src/types/index.ts`: also owned by `add-external-custom-menu-token-open` and `2026-04-28-support-mounted-frontend-client-templates` — 三者都改类型定义。
- `frontend/src/i18n/locales/en.ts`: also owned by `control-oidc-local-email-verification` — 本 change 删销售侧文案，对方加 OIDC 文案。
- `frontend/src/i18n/locales/zh.ts`: also owned by `control-oidc-local-email-verification` — 同上。
- `deploy/docker-compose.local.yml`: also owned by `2026-04-28-support-mounted-frontend-client-templates` — 本 change 改持久化目录布局，对方加 client-templates 挂载注释。

### Non-OpenSpec Overlap
- `backend/internal/server/routes/auth.go`: 与 FORK.md #5 `user-token-api-key-automation` 重叠（该 overlay 尚无 OpenSpec proposal，故不列为 Shared Touchpoint）—— 本 change 删 promo/invitation/微信支付路由，#5 加 token alias 端点，rebase 时两侧都要在。
- `backend/internal/server/routes/user.go`: 同上 —— 本 change 删 aff/announcement/redeem 路由组，#5 加 `PUT /keys/:id/group`。
- `frontend/src/api/auth.ts`: 同上 —— 本 change 删 promo/invitation API，#5 加 `exchangeToken` 系列 helper。
- `.gitignore`: 新增 `deploy/data/`、根 `/data/` 等运行时目录忽略项，属基础设施 fork 区。
- `Makefile`: 删除已废弃/指向不存在路径的构建目标（datamanagementd、secret-scan），属基础设施 fork 区。
- `Dockerfile`: Go 基础镜像版本与 `backend/go.mod` 对齐（R1），属基础设施 fork 区。
- `deploy/Dockerfile`: 同上。
- `backend/Dockerfile`: 同上。
- `DEV_GUIDE.md`: Go 版本说明改为以 `backend/go.mod` 为准，属文档 fork 区。
- 已删除文件（verify 不追踪存在性，记录于此）：`data/model_pricing.json`、`data/model_pricing.sha256` —— `1a342fcc3` 误提交的运行时定价缓存，本 change 移除（R19）。
