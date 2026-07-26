# lite-scope-boundary Specification

## Purpose
Define the sales-side surface that `develop-lite` has removed and MUST keep unreachable, so an upstream rebase or a careless re-merge cannot silently reintroduce payment, affiliate, promo, announcement, user-redeem, or invitation-code functionality.

## ADDED Requirements

### Requirement: Payment endpoints SHALL NOT exist
The system SHALL NOT expose any internal payment, order, or payment-webhook HTTP surface.

#### Scenario: Payment routes are unrouted
- **GIVEN** a running server with any authentication state
- **WHEN** a request is made to `/api/v1/payment/config`, `/api/v1/payment/plans`, `/api/v1/payment/orders`, `/api/v1/payment/public/orders/verify`, `/api/v1/payment/webhook/alipay`, or `/api/v1/admin/payment/dashboard`
- **THEN** the server SHALL respond `404`
- **AND** the response SHALL NOT be produced by a payment handler

#### Scenario: WeChat payment OAuth is unrouted
- **GIVEN** a running server
- **WHEN** a request is made to `/api/v1/auth/oauth/wechat/payment/start` or `/api/v1/auth/oauth/wechat/payment/callback`
- **THEN** the server SHALL respond `404`

#### Scenario: No payment provider SDK is imported
- **GIVEN** the backend source tree
- **WHEN** imports are resolved from `cmd/server`
- **THEN** no Alipay, WeChat Pay, Stripe, Airwallex, or EasyPay provider package SHALL be reachable

#### Scenario: No payment provider SDK is declared as a dependency
- **GIVEN** `backend/go.mod` and `backend/go.sum`
- **WHEN** `go mod tidy` is run
- **THEN** it SHALL produce no change
- **AND** `github.com/stripe/stripe-go/*`, `github.com/smartwalle/alipay/*`, and `github.com/wechatpay-apiv3/wechatpay-go` SHALL NOT appear as direct requirements
- **AND** those modules SHALL therefore stay out of `govulncheck` and `gosec` scope

### Requirement: Affiliate, promo, and announcement endpoints SHALL NOT exist
The system SHALL NOT expose affiliate rebate, promo code, or announcement HTTP surface.

#### Scenario: Affiliate routes are unrouted
- **WHEN** a request is made to `/api/v1/user/aff`, `/api/v1/user/aff/transfer`, `/api/v1/admin/affiliates/invites`, or `/api/v1/admin/affiliates/users`
- **THEN** the server SHALL respond `404`

#### Scenario: Promo routes are unrouted
- **WHEN** a request is made to `/api/v1/auth/validate-promo-code` or `/api/v1/admin/promo-codes`
- **THEN** the server SHALL respond `404`

#### Scenario: Announcement routes are unrouted
- **WHEN** a request is made to `/api/v1/announcements` or `/api/v1/admin/announcements`
- **THEN** the server SHALL respond `404`

### Requirement: User-facing redeem flow SHALL NOT exist
The system SHALL NOT let end users redeem codes for themselves, and SHALL NOT expose admin redeem-code management.

#### Scenario: Redeem routes are unrouted
- **WHEN** a request is made to `/api/v1/redeem`, `/api/v1/redeem/history`, or `/api/v1/admin/redeem-codes`
- **THEN** the server SHALL respond `404`

#### Scenario: Redeem data layer survives without the flow
- **GIVEN** the redeem routes are gone
- **WHEN** the server starts
- **THEN** the `redeem_codes` table and its repository SHALL still be present
- **AND** admin balance and concurrency adjustments SHALL still write audit rows into it

### Requirement: Registration SHALL be gated only by registration_enabled
The system SHALL NOT accept, validate, or require an invitation code on any registration path.

#### Scenario: Invitation validation endpoint is unrouted
- **WHEN** a request is made to `/api/v1/auth/validate-invitation-code`
- **THEN** the server SHALL respond `404`

#### Scenario: OAuth first login auto-registers
- **GIVEN** `registration_enabled` is true
- **AND** an OAuth identity that has never logged in before completes its callback
- **WHEN** the pending-auth flow resolves
- **THEN** the system SHALL create the account without requesting an invitation code
- **AND** the pending session SHALL NOT carry an `invitation_required` state

#### Scenario: Invitation fields are not accepted
- **GIVEN** any registration or pending create-account request
- **WHEN** the payload contains `invitation_code`
- **THEN** the system SHALL ignore it
- **AND** the outcome SHALL depend only on `registration_enabled`

### Requirement: Sales-side tables SHALL be dropped
The system SHALL remove the sales-side tables on migration while keeping the historical migrations that created them replayable.

#### Scenario: Migration 154 drops the sales tables
- **GIVEN** a database at any migration version below 154
- **WHEN** migrations are applied
- **THEN** `payment_orders`, `payment_provider_instances`, `payment_audit_logs`, `promo_codes`, `promo_code_usages`, `announcements`, `announcement_reads`, `user_affiliates`, and `user_affiliate_ledger` SHALL NOT exist
- **AND** `redeem_codes` SHALL still exist

#### Scenario: Foreign key order is respected
- **GIVEN** `user_affiliate_ledger.source_order_id` references `payment_orders`
- **WHEN** migration 154 runs
- **THEN** `user_affiliate_ledger` SHALL be dropped before `payment_orders`

#### Scenario: Historical migrations are not rewritten
- **GIVEN** an existing deployment several versions behind
- **WHEN** it migrates forward
- **THEN** the migrations that originally created the sales tables SHALL still apply successfully before 154 drops them
- **AND** their recorded checksums SHALL be unchanged

### Requirement: Sales-side settings SHALL NOT be readable or writable
The system SHALL NOT expose payment, promo, affiliate, or invitation setting keys through the admin settings API.

#### Scenario: Admin settings payload excludes sales keys
- **GIVEN** an authenticated admin who has accepted the compliance gate
- **WHEN** `GET /api/v1/admin/settings` is called
- **THEN** the response SHALL NOT contain `payment_enabled`, `promo_code_enabled`, `invitation_code_enabled`, `affiliate_enabled`, `affiliate_rebate_rate`, `affiliate_rebate_freeze_hours`, `affiliate_rebate_duration_days`, or `affiliate_rebate_per_invitee_cap`

#### Scenario: Writing a removed key has no effect
- **WHEN** an admin submits a settings update containing a removed sales key
- **THEN** the system SHALL ignore the field
- **AND** the settings audit diff SHALL NOT record it

### Requirement: Sales-side frontend routes SHALL NOT exist
The frontend SHALL NOT register routes for the removed subsystems, and SHALL NOT link to them.

#### Scenario: Removed routes are absent from the router
- **WHEN** the router table is built
- **THEN** it SHALL NOT contain `/redeem`, `/affiliate`, `/purchase`, `/orders`, `/payment/qrcode`, `/payment/result`, `/payment/stripe`, `/payment/airwallex`, `/payment/stripe-popup`, `/auth/wechat/payment/callback`, `/admin/announcements`, `/admin/redeem`, `/admin/promo-codes`, `/admin/affiliates`, `/admin/orders`, or their sub-paths

#### Scenario: No surviving view navigates to a removed route
- **GIVEN** any rendered view or sidebar entry
- **WHEN** its navigation targets are enumerated
- **THEN** none SHALL resolve to a removed route
- **AND** none SHALL fall through to the 404 catch-all
