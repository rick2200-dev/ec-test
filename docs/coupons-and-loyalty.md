# クーポン・ポイント設計書

買い手向けの **クーポン割引** と **ポイント付与・利用** の設計・運用仕様をまとめる。購入フロー全体は [カート・チェックアウト設計書](./cart-and-checkout.md) と [決済設計書](./payment.md)、キャンセル時の挙動は [注文キャンセル設計書](./order-cancellation.md) と合わせて参照のこと。

## 目次

- [機能概要と MVP スコープ](#機能概要と-mvp-スコープ)
- [サービス構成](#サービス構成)
- [データモデル](#データモデル)
- [API エンドポイント](#api-エンドポイント)
- [チェックアウト時の処理](#チェックアウト時の処理)
- [決済成功時の確定 (Commit)](#決済成功時の確定-commit)
- [キャンセル時の補償 (Refund / Reverse)](#キャンセル時の補償-refund--reverse)
- [イベント設計](#イベント設計)
- [並行性・冪等性](#並行性冪等性)
- [フィーチャーフラグ](#フィーチャーフラグ)
- [既知の制約と将来拡張](#既知の制約と将来拡張)

---

## 機能概要と MVP スコープ

### クーポン

- 買い手は チェックアウト時に **クーポンコード** を入力して割引を受けられる
- **プラットフォーム発行** のみ (セラー発行は Phase 5 以降の予定、DB 列は先行用意済み)
- 割引タイプ: **固定額** (`fixed_amount`) または **パーセント** (`percent`, basis points 指定)
- 制約: `min_order_amount` / `max_discount_amount` / `usage_limit_total` / `usage_limit_per_user` / `expires_at` を個別に設定可能
- 割引額は カート全体の小計に対して計算され、マルチセラー注文では小計に比例して各注文に配分 (端数は最後の注文が吸収)

### ポイント

- **購入確定時 (order.paid)** に `paid_subtotal × earn_rate_bps / 10_000` を floor したポイントを自動付与 (既定 1%)
- **次回以降の購入時** にポイントを指定して支払い充当 (1pt = 1 JPY)
- ポイント残高は **append-only 台帳** (`loyalty_svc.point_transactions`) で管理し、`point_accounts.balance` がその集計結果
- 決済後キャンセル時は **利用ポイントを返還 + 獲得ポイントを取消** (ユーザー確認済み方針)

### 確定仕様 (2026-04-18 ユーザー確認)

| 項目 | 方針 |
|---|---|
| MVP スコープ | クーポン + ポイント付与 + ポイント利用すべて一括実装 |
| クーポン発行主体 | プラットフォームのみ |
| コミッション計算基準 | 常に **割引前小計** (割引はプラットフォーム負担、セラー売上は不変) |
| キャンセル時 | 利用ポイントは返還、獲得ポイントは取消 |
| 通貨 | JPY のみ |

---

## サービス構成

```
┌──────────────┐       ┌─────────┐       ┌─────────────────┐
│  Buyer App   │──────▶│ Gateway │──────▶│    Cart Svc     │
│              │       │  :8080  │       │     :8088       │
└──────────────┘       └─────────┘       └────────┬────────┘
                                                  │ POST /internal/checkouts
                                                  ▼
                         ┌─────────────┐   ┌─────────────┐   ┌─────────────┐
                         │ Coupon Svc  │◀──│  Order Svc  │──▶│ Loyalty Svc │
                         │   :8093     │   │    :8083    │   │   :8094     │
                         │ postgres    │   │  postgres   │   │  postgres   │
                         │ :5444       │   │  :5443      │   │  :5445      │
                         └─────────────┘   └─────────────┘   └─────────────┘
                                │                 │                 │
                                │       Pub/Sub: order-events       │
                                ▼                 ▼                 ▼
                         ┌──────────────────────────────────────────────┐
                         │ coupon subscribes order.cancelled (refund)   │
                         │ loyalty subscribes order.paid + .cancelled   │
                         └──────────────────────────────────────────────┘
```

- **Coupon Svc** `:8093` — 管理者によるクーポン作成・失効、買い手のプレビュー、`Reserve`/`Commit`/`Release` の in-cluster API、`order.cancelled` 購読で redemption 取消
- **Loyalty Svc** `:8094` — 残高・履歴参照、`AwardEarn` (earn-only)、`Reserve`/`Commit`/`Release` (redeem + earn)、`order.paid` 購読 (earn fan-in)、`order.cancelled` 購読 (refund + reverse_earn)
- **Order Svc** — チェックアウト時に Coupon/Loyalty と同期 RPC で予約 (`Reserve`) を取得、webhook で確定 (`Commit`)、キャンセル時に `order.cancelled` イベントで discount / 予約 ID を下流に伝播

---

## データモデル

### Order 側 (既存テーブルへの追加列)

`order_svc.orders` に次の列が追加済み (migration 006):

| 列 | 型 | 用途 |
|---|---|---|
| `coupon_discount_amount` | BIGINT NOT NULL default 0 | この注文に割り当てられたクーポン割引分 (比例配分後) |
| `point_discount_amount` | BIGINT NOT NULL default 0 | この注文に割り当てられたポイント利用分 |
| `coupon_id` | UUID NULL | 使用されたクーポン (表示/参照用) |
| `coupon_reservation_id` | UUID NULL | coupon-svc の予約 ID (**anchor order のみ**に格納) |
| `point_reservation_id` | UUID NULL | loyalty-svc の予約 ID (**anchor order のみ**) |
| `points_earned` | BIGINT NOT NULL default 0 | 購入確定後に付与されたポイント (表示用ミラー、権威は loyalty ledger) |

`total_amount = subtotal + shipping_fee - coupon_discount_amount - point_discount_amount`

**anchor order** とは、マルチセラー注文におけるカート内の最初の注文のこと。予約 ID はカート全体で 1 件なので、anchor だけが保持する。

### Coupon Svc

- `coupon_svc.coupons` — クーポン定義本体 (`discount_type` / `usage_limit_*` / `expires_at` / `status` / `usage_count` 等)
- `coupon_svc.coupon_reservations` — チェックアウト時の pending 予約 (TTL 30 分、`reaper` が期限切れを `expired` に)
- `coupon_svc.coupon_redemptions` — 確定された利用ログ (immutable、UNIQUE `(coupon_id, order_id)` で webhook リプレイを無害化)
  - `refunded_at` / `refunded_reason` 列でキャンセル時の返還をマーク (migration 004)

### Loyalty Svc

- `loyalty_svc.point_accounts` — 買い手ごとの残高 (balance / pending_redemption / lifetime_*、version で楽観ロック)
- `loyalty_svc.point_reservations` — ポイント利用の pending 予約
- `loyalty_svc.point_transactions` — **append-only 台帳**
  - `type`: `earn` / `redeem` / `refund` / `reverse_earn` / `adjust` / `expire`
  - UNIQUE `(source_type, source_id, type)` で二重書き込み防止
  - `balance_after` を各行に記録し、履歴ページングで残高を再集計しない

---

## API エンドポイント

### 買い手向け (Gateway `/api/v1/buyer/*`、JWT 必須)

| Method | Path | 目的 |
|---|---|---|
| GET  | `/points/balance` | 残高 + pending + lifetime 取得 |
| GET  | `/points/transactions?limit=&offset=` | 履歴ページング |
| POST | `/coupons/preview` | コード + カート内容を dry-run して割引額を返す |
| GET  | `/coupons/redemptions` | 自分の過去 redemption 一覧 |

### 管理者向け (`/api/v1/admin/*`、platform_admin ロール必須)

| Method | Path | 目的 |
|---|---|---|
| POST | `/coupons/` | クーポン作成 (code, discount_type, 制限, 有効期間) |
| GET  | `/coupons/` | 一覧 (status フィルタ + ページング) |
| GET  | `/coupons/{id}` | 詳細 |
| POST | `/coupons/{id}/revoke` | 失効 (既存 redemption は保持) |
| GET  | `/coupons/{id}/stats` | 利用状況 (redeemed_count, total_discount, pending_reservation) |

### 内部 API (クラスタ内のみ、X-Internal-Token 必須)

Coupon Svc `:8093`:
- `POST /internal/reservations` → `Reserve`
- `POST /internal/reservations/{id}/commit` → `Commit`
- `POST /internal/reservations/{id}/release` → `Release`

Loyalty Svc `:8094`:
- `POST /internal/earn` — 純粋な earn (reservation なしの order.paid 経路)
- `POST /internal/reservations` → `Reserve` (利用予約)
- `POST /internal/reservations/{id}/commit` → `Commit` (redeem + earn)
- `POST /internal/reservations/{id}/release` → `Release`

### チェックアウトリクエスト拡張

`POST /api/v1/buyer/cart/checkout` の body に次の任意フィールドを追加:

```json
{
  "shipping_address": { ... },
  "currency": "JPY",
  "coupon_code": "WELCOME10",
  "points_to_redeem": 500
}
```

- `coupon_code`: 未指定または空文字は「クーポン不使用」
- `points_to_redeem`: 0 または未指定は「ポイント不使用」、正の値は充当希望額
- feature flag が OFF の環境で値を指定すると `400 FEATURE_DISABLED` を返す

---

## チェックアウト時の処理

### 全体シーケンス

```
Buyer ──▶ Gateway ──▶ Cart ──▶ Order: POST /internal/checkouts
                                   │
                                   ▼
  [Order Svc.CreateCheckout]
  ├─ 1. validate (feature flag / lines) 
  ├─ 2. group lines by seller_id → pre-discount subtotals
  ├─ 3. shipping fee (subscription check)
  ├─ 4. per-seller commission (pre-discount subtotal × rate_bps / 10_000)
  ├─ 5a. coupon_code あり → coupon-svc.Reserve(code, buyer, subtotals)
  │       ├─ fail (expired/limit/etc.) → 400 with stable error code
  │       └─ ok   → CouponReservation{reservation_id, discount_amount, ...}
  ├─ 5b. points_to_redeem > 0 → **clamp** to (cart_subtotal - coupon_discount)
  │       └─ loyalty-svc.Reserve(buyer, effective_amount)
  ├─ 6. domain.DistributeDiscount で各注文への配分 (後述)
  ├─ 7. total_amount = subtotal + shipping - coupon_share - point_share
  ├─ 8. atomic DB insert (N 注文 + payout)
  ├─ 9. Stripe PaymentIntent 作成 (cart_total で)
  └─10. order.created イベント N 件 publish

  [失敗時 (Stripe 失敗 / DB 失敗)]
  defer で coupon.Release + loyalty.Release を呼び、予約を解放
```

### ポイント clamp ルール (重要)

買い手が `points_to_redeem=5000` を指定しても、カート小計 (クーポン適用後) が 1000 JPY なら **1000 のみを loyalty-svc に予約する**。これは `domain.DistributeDiscount` が per-seller subtotal でクランプするため、予約額と実際の割引額がズレて「余分に burn される」事故を防ぐため。

```go
maxApplicable := cartSubtotal - couponDiscount
effective := min(input.PointsToRedeem, maxApplicable)
if effective <= 0 { /* 予約スキップ */ }
```

### 割引の比例配分 (`domain.DistributeDiscount`)

マルチセラー注文では、クーポン割引・ポイント割引とも **カート全体から各注文へ subtotal 比例で配分** する。残り端数は最後の非ゼロバケットが吸収する (決定的な挙動で、テストで pin されている)。

| 例: coupon_discount = 300, subtotals = [1000, 2000] |
| --- |
| share[0] = floor(300 × 1000 / 3000) = 100 |
| share[1] = floor(300 × 2000 / 3000) = 200 |
| remainder = 0 |
| result = [100, 200] |

端数ケース (discount=100, subtotals=[333, 333, 334]):
- shares = [33, 33, 34] (最後が +1 で端数吸収)

### 予約 ID の保持場所

マルチセラー注文ではカート全体で 1 つの予約しか持たないため、予約 ID は **anchor order (batch[0])** にのみ格納する。Commit / Release はこの anchor order 行を見て判断する。

---

## 決済成功時の確定 (Commit)

`OrderService.HandlePaymentSuccess` は Stripe webhook の `payment_intent.succeeded` を受け取り、次の順序で処理する。**webhook リトライに耐えるため** 3 段階構造を取る:

```
[HandlePaymentSuccess] — 同じ PI に紐づく N 件の order を順に処理

for each order:
  ├─ 0. order.status == cancelled → 完全スキップ
  │      (キャンセル済み注文へのリトライ webhook は無視)
  ├─ 1. SetPaid(order) — ErrOrderNotPending は fall-through 許容
  │      (既払注文へのリトライは (A) を skip し (B)(C) だけ再実行)
  ├─ 2. GetPayout(order)
  ├─ 3. payout.Status allow-list: pending / completed 以外は continue
  │      (failed / reversed は全 skip — 手動復旧対象)
  │
  ├─ [A] One-shot Stripe Transfer (payout.status == pending のときだけ)
  │   ├─ Stripe.CreateTransfer(payout.amount, seller_connected_account)
  │   ├─ 失敗 → payout を failed に、PayoutFailed イベント発行、continue
  │   └─ 成功 → payout を completed に (transfer_id と共に)
  │
  ├─ [B] Idempotent downstream commits (pending / completed 両方で実行)
  │   ├─ coupon_reservation_id があれば → coupon.Commit(reservation_id, order_id, pi_id)
  │   │    失敗 → return error → Stripe が webhook を再配信 → 次回 (B) が再実行
  │   ├─ point_reservation_id または enable_loyalty なら → loyalty.Commit(...)
  │   │    redeem + earn をアトミックに実行、失敗は同上 return error
  │   └─ earn 結果を order.points_earned に書き戻し (失敗は non-critical)
  │
  └─ [C] One-time publish (orders.paid_event_published_at IS NULL のときだけ)
      ├─ order.paid + payout.completed を発行
      └─ orders.paid_event_published_at = NOW() に stamp
           (WHERE paid_event_published_at IS NULL で重複防止)
```

### なぜこの 3 ブロック構造が必要か

**旧実装 1**: `payout.status != pending` で即 `continue` → retry で Commit に到達せず。

**旧実装 2**: (A)(B) に分けたが publish を `transferPending` ブロック内に置いていたため、Commit 初回失敗 → retry 成功のシナリオで publish が永遠に発行されず、shipping / notification / loyalty-earn-fanin が停滞。

**新実装 (3 ブロック)**:
- `(A)` は payout `pending` のときだけ → 二重 Stripe Transfer を防ぐ
- payout allow-list で `failed`/`reversed` は全 skip → 誤コミット防止
- `(B)` は pending/completed 両方で実行 → 初回失敗した Commit を retry で補償可能
- `(C)` は `paid_event_published_at` DB フラグで gate → Commit 成功した attempt で必ず publish が走り、かつ重複 publish は DB の NULL ガードで弾く

### Commit / publish の冪等性キー

| 操作 | idempotency key |
|---|---|
| coupon redeemption | `UNIQUE (coupon_id, order_id)` on `coupon_redemptions` |
| loyalty redeem    | `UNIQUE (source_type='reservation_commit', source_id=reservation_id, type='redeem')` on `point_transactions` |
| loyalty earn       | `UNIQUE (source_type='order_paid', source_id=order_id, type='earn')` on `point_transactions` |
| event publish     | `orders.paid_event_published_at IS NULL` ガード付き UPDATE |

---

## キャンセル時の補償 (Refund / Reverse)

`order.cancelled` イベントは `order-events` トピックに publish される。coupon と loyalty はそれぞれ自分の subscription でこれを購読する。

### OrderCancelled イベントのペイロード (抜粋)

```proto
message OrderCancelled {
  string order_id = 1;
  string buyer_auth0_id = 3;
  string coupon_id = 8;              // クーポン使用時のみ
  string coupon_reservation_id = 9;  // 〃
  int64 coupon_discount_amount = 10; // 〃
  string point_reservation_id = 11;  // ポイント使用時のみ
  int64 point_discount_amount = 12;  // 〃
  int64 points_earned = 13;          // 購入時に獲得した earn
}
```

### Coupon 側 (coupon-svc が `order-events-coupon` を購読)

`RefundRedemption(coupon_id, order_id, reason)`:

1. `coupon_redemptions` を `(coupon_id, order_id)` で検索
2. 見つからなければ `Skipped=true` で終了 (クーポンを使わなかったキャンセル)
3. `refunded_at` が既に埋まっていれば `AlreadyRefunded=true` で終了 (リプレイ)
4. 初回なら:
   - `refunded_at = NOW(), refunded_reason = reason` で UPDATE
   - `coupons.usage_count` を `DecrementUsage` で 1 戻す (seat を取り戻す)
5. `CouponRefunded` イベント発行 (将来の analytics/notification 用、現状オプショナル)

### Loyalty 側 (loyalty-svc が `order-events-loyalty` を購読)

`ApplyCancellation({ buyer, order_id, point_discount_amount, points_earned })`:

1. account を `LockForUpdate` で取得 (並行性保証)
2. **refund**: `point_discount_amount > 0` なら
   - idempotency check `(order_cancelled, order_id, refund)`
   - 初回なら balance + point_discount を credit、台帳に `type=refund, amount=+N` を INSERT
3. **reverse_earn**: `points_earned > 0` なら
   - idempotency check `(order_cancelled, order_id, reverse_earn)`
   - 初回なら balance - points_earned を debit (ただし `balance >= 0` の CHECK を守るため残高超過分は cap、cap 発生時は warn ログ)
   - 台帳に `type=reverse_earn, amount=-N` を INSERT

### なぜ lifetime_earned を戻さないか

キャンセル時に `reverse_earn` で balance は減らすが `lifetime_earned` は減らす (対称的に)。ただし `lifetime_redeemed` は **戻さない** ことを選択 — 「買い手が購入時にポイントを使う意思を示した」記録は保持する方が運用上有用だから。将来のランク制度や不正検知で「利用実績」として参照できる。

---

## イベント設計

### トピックとサブスクリプション

| トピック | パブリッシャ | サブスクリプション | コンシューマ |
|---|---|---|---|
| `order-events` | order-svc | `order-events-inventory` | inventory (stock release) |
| | | `order-events-notification` | notification (email) |
| | | `order-events-recommend` | recommend (purchase signal) |
| | | `order-events-shipping` | shipping (shipment create) |
| | | `order-events-coupon` | **coupon (refund redemption)** |
| | | `order-events-loyalty` | **loyalty (earn + refund + reverse_earn)** |
| `coupon-events` | coupon-svc | *(将来)* | notification / analytics |
| `loyalty-events` | loyalty-svc | `loyalty-events-notification` | notification |

### coupon-events (publish のみ、現状 subscribe 未実装)

- `coupon.issued` — admin によるクーポン作成時
- `coupon.redeemed` — Commit 成功時
- `coupon.revoked` — admin による失効
- `coupon.refunded` (予定) — キャンセル時

### loyalty-events

- `points.earned` — earn 台帳行 INSERT 時
- `points.redeemed` — redeem 台帳行 INSERT 時
- `points.refunded` — refund / reverse_earn 時
- `points.adjusted` — admin による手動調整

---

## 並行性・冪等性

### account 行ロック

`loyalty_svc.point_accounts` への書き込みはすべて `LockForUpdate` (UPSERT + RETURNING で行ロック取得) を経由する。これにより:

- 同一買い手の earn / redeem / refund が並行してきても直列化される
- idempotency check → delta 適用 → ledger insert が同一トランザクション内で矛盾なく完了する
- 初期実装の「optimistic lock + Insert 後の duplicate-key 握り潰し」に潜んでいた 二重付与リスクを排除

回帰テスト: `loyalty_service_test.go` に「並行リプレイでも balance が二重加算されない」を pin している。

### webhook リトライ

Stripe の `payment_intent.succeeded` は最大 3 日間リトライされる。本サービスは以下の方針で対応:

| 状況 | 動作 |
|---|---|
| 既払注文への重複 webhook | SetPaid は NotPending エラーを吸収、(A) 送金ブロックはスキップ、(B) Commit は毎回実行 |
| (B) の一部失敗 (coupon OK, loyalty NG) | error を返し Stripe にリトライさせる → 次回 (B) で両方 dedupe されて確定 |
| order.cancelled の重複 delivery | refund / reverse_earn の UNIQUE キーで 2 回目は no-op |

### チェックアウト失敗時の予約解放

`CreateCheckout` 内で Reserve 後に Stripe PaymentIntent 作成や DB insert が失敗すると、`defer` で両予約を `Release` する。Release 自体も冪等なので複数回呼ばれても問題ない。

---

## フィーチャーフラグ

order-svc の環境変数で機能単位に on/off できる:

- `ENABLE_COUPONS=true|false` — checkout の `coupon_code` を有効化
- `ENABLE_LOYALTY=true|false` — checkout の `points_to_redeem` を有効化

どちらも **既定 false** (docker-compose の order エントリ)。OFF のまま `coupon_code` や `points_to_redeem>0` が送られてきた場合は `400 FEATURE_DISABLED` を返す。

段階的ロールアウト手順:

1. order 側フラグ OFF のまま coupon / loyalty サービスをデプロイ (subscriber は動作)
2. admin から 1 枚テストクーポン発行 → buyer で preview 確認
3. `ENABLE_COUPONS=true` に切替、E2E 確認
4. `ENABLE_LOYALTY=true` に切替

---

## 既知の制約と将来拡張

| 項目 | 状態 | 対応予定 |
|---|---|---|
| セラー発行クーポン | 未実装 (DB 列のみ用意) | Phase 5 以降 |
| クーポンのスタッキング (複数同時使用) | 未実装 (MVP 仕様通り 1 枚のみ) | Phase 5 以降 |
| ポイント有効期限ジョブ | 未実装 (`point_transactions.expires_at` 列は用意済み) | 運用実績を見てから |
| 階段還元率 (会員ランク) | 未実装 (earn_rate_bps は単一値) | Phase 5 以降 |
| Frontend UI | 未実装 | API は完成、次フェーズで実装 |
| `lifetime_redeemed` の補償時減算 | しない設計 | 仕様として固定 |
| ¥0 決済への対応 | `total_amount = 0` の場合でも Stripe PI を作成 (最低額は Stripe 側ガード) | 実用上問題視する報告があれば調整 |

---

## 関連ドキュメント

- [カート・チェックアウト設計書](./cart-and-checkout.md) — 購入フロー本体
- [決済設計書](./payment.md) — Stripe 連携と webhook
- [注文キャンセル設計書](./order-cancellation.md) — キャンセル経路全体
- [アーキテクチャ設計書](./architecture.md) — サービス境界
