# lite-retained-surface Specification

## Purpose
Pin down the capabilities that share code or data with the removed sales-side subsystems but MUST survive on `develop-lite`. Without this, a future cleanup pass that follows the "delete anything named redeem/subscription" heuristic would destroy the balance audit trail and the gateway's quota model.

## ADDED Requirements

### Requirement: redeem_codes SHALL remain the balance and concurrency audit trail
The system SHALL keep the `redeem_codes` table, repository, and service as the storage for administrative balance and concurrency adjustments, even though the user-facing redeem flow is gone.

#### Scenario: Admin balance adjustment writes an audit row
- **GIVEN** an authenticated admin and an existing user
- **WHEN** `POST /api/v1/admin/users/:id/balance` is called with `operation=add`
- **THEN** the user balance SHALL change
- **AND** a `redeem_codes` row SHALL be created with type `admin_balance`, the applied value, the operator notes, and `used_by` set to the target user

#### Scenario: Admin concurrency adjustment writes an audit row
- **GIVEN** an authenticated admin and a user whose concurrency is being changed
- **WHEN** `PUT /api/v1/admin/users/:id` is called with a different `concurrency`
- **THEN** a `redeem_codes` row SHALL be created with type `admin_concurrency` and the signed delta as its value
- **AND** the row SHALL appear in that user's balance history

#### Scenario: Balance history reads only surviving tables
- **GIVEN** a user with at least one adjustment
- **WHEN** `GET /api/v1/admin/users/:id/balance-history` is called
- **THEN** the system SHALL respond `200`
- **AND** the query SHALL NOT reference `user_affiliate_ledger` or any other dropped table
- **AND** the response SHALL include the adjustment rows and the aggregate `total_recharged`

#### Scenario: Adjustment types remain distinguishable
- **WHEN** balance history entries are returned
- **THEN** each SHALL carry a `type` that distinguishes at least `admin_balance`, `admin_concurrency`, and `subscription` records

### Requirement: Subscription and quota capabilities SHALL survive
The system SHALL keep subscriptions, group binding, and subscription quota progress, which belong to the gateway authorization model rather than to sales.

#### Scenario: User subscription listing works
- **GIVEN** an authenticated non-admin user
- **WHEN** `GET /api/v1/subscriptions` is called
- **THEN** the system SHALL respond `200` with the user's subscription list

#### Scenario: Admin subscription management works
- **GIVEN** an authenticated admin
- **WHEN** admin subscription endpoints are called
- **THEN** they SHALL remain routed and functional

### Requirement: Usage and cost statistics SHALL survive
The system SHALL keep token-level usage tracking, cost calculation, dashboards, and ops metrics.

#### Scenario: Usage endpoints remain routed
- **GIVEN** an authenticated user or admin
- **WHEN** usage, dashboard, or ops metrics endpoints are called
- **THEN** they SHALL respond successfully
- **AND** they SHALL NOT depend on any dropped sales table

### Requirement: The gateway forward path SHALL be unaffected
Removing the sales subsystems SHALL NOT change how the gateway authenticates API keys or forwards upstream requests.

#### Scenario: API key auth still works end to end
- **GIVEN** an API key created through the admin or user key flow
- **WHEN** it is used against `POST /v1/messages`
- **THEN** authentication SHALL succeed
- **AND** the request SHALL reach account scheduling

#### Scenario: No upstream account yields a service error, not a crash
- **GIVEN** a valid API key and a group with no schedulable upstream account
- **WHEN** a gateway request is made
- **THEN** the system SHALL respond `503` with an explicit "no available accounts" error
- **AND** the system SHALL NOT respond `500`
