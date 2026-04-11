# カート・チェックアウト設計書

本書は買い手のショッピングカートとチェックアウト (マルチセラー注文作成) の設計を説明する。決済の詳細は [決済設計書](./payment.md) を参照のこと。

## 目次

- [全体像](#全体像)
- [カートのデータモデル](#カートのデータモデル)
- [カート操作 API](#カート操作-api)
- [価格スナップショット戦略](#価格スナップショット戦略)
- [チェックアウトオーケストレーション](#チェックアウトオーケストレーション)
- [マルチセラー注文の表示](#マルチセラー注文の表示)
- [在庫引当のタイミング](#在庫引当のタイミング)
- [フロントエンド連携 (未実装)](#フロントエンド連携-未実装)
- [既知の制約](#既知の制約)

---

## 全体像

本マーケットプレイスは Amazon 型 UX を目指しており、買い手は 1 つのカートに複数セラーの商品を混在させて一度に決済できる。この要件を満たすため:

- **カートは Redis で管理** する (高頻度更新 + TTL 前提 + 複雑なクエリ不要)
- **チェックアウト時に `seller_id` でグループ化** して N 個の注文を作成する
- **N 個の注文は同じ `stripe_payment_intent_id` を共有** し、1 回の決済で全セラーへの送金を処理する

カート機能はテナント分離に依存しない (本マーケットプレイスは現状 1 テナント構成)。ただし将来の SaaS 展開に備え、Redis キーには `tenant_id` を含めている。

### サービス構成

```
┌──────────────┐      ┌─────────┐      ┌──────────┐
│ Buyer App    │─────▶│ Gateway │─────▶│  Cart    │
│ (Next.js)    │      │  :8080  │      │  :8088   │
└──────────────┘      └─────────┘      └──┬─┬──┬──┘
                                          │ │  │
                        ┌─────────────────┘ │  │
                        │ ┌─────────────────┘  │
                        │ │ ┌──────────────────┘
                        ▼ ▼ ▼
                 ┌────────┐ ┌───────┐ ┌─────────┐
                 │ Redis  │ │Catalog│ │ Order   │
                 │ (cart) │ │ :8082 │ │  :8084  │
                 └────────┘ └───────┘ └─────────┘
```

- **Cart service** — Redis へのカート CRUD とチェックアウトオーケストレーション
- **Redis** — `cart:{tenant_id}:{buyer_auth0_id}` キーでカートを永続化
- **Catalog service** — SKU の価格・商品名・所属セラー ID の権威ソース
- **Order service** — チェックアウト時に N 個の注文と PaymentIntent を作成

---

## カートのデータモデル

### Redis キー設計

```
cart:{tenant_id}:{buyer_auth0_id}
```

- **`tenant_id`** — UUID 文字列。将来の SaaS 展開を見越した空間分離のため必須
- **`buyer_auth0_id`** — JWT の `sub` クレーム (例: `auth0|abc123`)
- **TTL** — 30 日 (2,592,000 秒)、**更新のたびにリセット** する

将来的にカート機能を匿名ユーザーに拡張する場合は、ログイン前のキーとして `cart:{tenant_id}:anon:{session_id}` を追加し、ログイン時にマージするフローを設計する (本 MVP では対象外)。

### JSON ペイロード

値は JSON 文字列として保存する。Redis の `HSET` ではなく `SET` で 1 キー 1 値の構造にすることで、アトミックな全体置換とシリアライゼーションの単純化を優先する。

```json
{
  "tenant_id": "550e8400-e29b-41d4-a716-446655440000",
  "buyer_auth0_id": "auth0|abc123",
  "items": [
    {
      "sku_id": "8d3f...",
      "seller_id": "11bb...",
      "quantity": 2,
      "unit_price_snapshot": { "amount": 1200, "currency": "JPY" },
      "product_name_snapshot": "ワイヤレスイヤホン Pro",
      "sku_code_snapshot": "WH-PRO-BLK-M",
      "added_at": "2026-04-11T09:34:12Z"
    },
    {
      "sku_id": "9e4a...",
      "seller_id": "22cc...",
      "quantity": 1,
      "unit_price_snapshot": { "amount": 3000, "currency": "JPY" },
      "product_name_snapshot": "折り畳み傘",
      "sku_code_snapshot": "UMB-FOLD-NAVY",
      "added_at": "2026-04-11T09:35:42Z"
    }
  ],
  "updated_at": "2026-04-11T09:35:42Z"
}
```

### フィールドの意味

| フィールド | 型 | 説明 |
| --- | --- | --- |
| `tenant_id` | UUID string | テナント ID |
| `buyer_auth0_id` | string | Auth0 の sub クレーム |
| `items[].sku_id` | UUID string | catalog service の SKU ID |
| `items[].seller_id` | UUID string | catalog から取得したセラー ID (チェックアウト時のグルーピングキー) |
| `items[].quantity` | int32 | 数量 (1 以上) |
| `items[].unit_price_snapshot` | Money | cart-add 時点の単価 (詳細: [価格スナップショット戦略](#価格スナップショット戦略)) |
| `items[].product_name_snapshot` | string | cart-add 時点の商品名 (UI 表示用) |
| `items[].sku_code_snapshot` | string | cart-add 時点の SKU コード |
| `items[].added_at` | RFC3339 | cart-add タイムスタンプ |
| `updated_at` | RFC3339 | カート最終更新時刻 |

### なぜ RDB ではなく Redis なのか

- **高頻度更新**: 商品ページで「カートに入れる」が連打される可能性がある
- **TTL 必須**: 放置カートの自動削除に Redis の EXPIRE が最適
- **複雑なクエリ不要**: 常に「自分のカート 1 件を取得する」のみ
- **トランザクション単位が小さい**: カート全体をアトミックに SET するだけで整合性が取れる

一方、注文確定後の履歴は `order_svc.orders` で永続化されるため、カート自体を長期保存する必要はない。

---

## カート操作 API

全ての API は Gateway 経由で buyer 用認証を必須とする。

| HTTP | Path | 説明 |
| --- | --- | --- |
| `GET` | `/api/v1/buyer/cart` | 現在のカートを取得 (空の場合は空配列) |
| `POST` | `/api/v1/buyer/cart/items` | カートに商品を追加 |
| `PUT` | `/api/v1/buyer/cart/items/{skuId}` | カート内商品の数量を更新 |
| `DELETE` | `/api/v1/buyer/cart/items/{skuId}` | カートから商品を削除 |
| `DELETE` | `/api/v1/buyer/cart` | カートを空にする |
| `POST` | `/api/v1/buyer/cart/checkout` | チェックアウト (注文作成 + PaymentIntent 発行) |

### リクエスト例

#### カートに追加

```http
POST /api/v1/buyer/cart/items HTTP/1.1
Authorization: Bearer <jwt>
Content-Type: application/json

{"sku_id": "8d3f...", "quantity": 2}
```

Cart service は `sku_id` から catalog service に `GetSKU` を呼び、価格・商品名・セラー ID をスナップショットとして保存する。既にカートに存在する SKU の場合は数量を加算する。

#### チェックアウト

```http
POST /api/v1/buyer/cart/checkout HTTP/1.1
Authorization: Bearer <jwt>
Content-Type: application/json

{
  "shipping_address": {
    "name": "山田太郎",
    "postal_code": "1000001",
    "line1": "千代田区千代田 1-1"
  },
  "currency": "JPY"
}
```

レスポンス:

```json
{
  "order_ids": ["a1b2...", "c3d4..."],
  "stripe_payment_intent_id": "pi_1ABC...",
  "stripe_client_secret": "pi_1ABC..._secret_xyz",
  "total": { "amount": 5400, "currency": "JPY" }
}
```

フロントは `stripe_client_secret` を Stripe.js の `confirmCardPayment` に渡して決済を確定させる。

---

## 価格スナップショット戦略

### なぜスナップショットが必要か

カート投入時と決済確定時の間に価格が変動する可能性があるため、カート UI では **cart-add 時の価格** を表示する。これにより「カートに入れた時と値段が違う」という UX 不整合を防ぐ。

### スナップショットのライフサイクル

```
1. cart-add 時:
   Cart service → Catalog service GetSKU → snapshot を Redis に保存

2. 表示時 (GET /cart):
   Redis から snapshot を返す (Catalog を再照会しない)

3. checkout 時:
   Redis の snapshot を Order service に渡す
   Order service は受け取った単価をそのまま使用 (権威ソースとして信頼する)
```

### 価格変動への対応方針

現状 MVP では **cart-add 時点の価格** で決済する。カート投入から 1 時間後に値下げされていても、買い手は古い値段で購入することになる。これは以下の理由による:

- 価格変動を検知する実装コストが高い (catalog に毎回問い合わせる、イベント購読で無効化する、など)
- 値上げより値下げの方が買い手に有利な方向なので、UX 影響が少ない
- 値上げ方向の変動はセラー側の運用リスクとして許容する

**将来拡張**: catalog service が `sku.price_changed` イベントを発行し、cart service が購読して該当 SKU を含むカートを削除 or フラグ付けするフローが考えられる。本 MVP の範囲外。

### 在庫切れスナップショット

商品名や SKU コードも一緒にスナップショットすることで、catalog から該当商品が削除された後もカートに「商品名 (販売終了)」と表示できる。チェックアウト時に catalog 側で該当 SKU が見つからない場合、order service が 400 エラーを返し cart service がフロントにエラーレスポンスを返す。

---

## チェックアウトオーケストレーション

### フロー

```
1. Cart service が Redis からカートを取得
   ↓ 空なら 400 BadRequest
2. Cart service が Order service の内部 API に items を POST
   (POST /internal/checkouts — JWT 検証なしのクラスタ内 API)
3. Order service:
   a. items を seller_id でグループ化
   b. 1 トランザクションで N 個の order + order_lines + pending payout を作成
      (トランザクション先頭で cross-schema lookup: `auth_svc.sellers.name` から
       `orders.seller_name` を、`catalog_svc.skus.product_id` から
       `order_lines.product_id` をスナップショット。
       詳細は [docs/architecture.md § 購入履歴と商品スナップショット](./architecture.md#購入履歴と商品スナップショット))
   c. Stripe に 1 回 PaymentIntent を発行 (合計金額)
   d. 発行された PI ID を全 order に書き戻す
   e. order.created イベントを N 回 publish
   f. レスポンスに order_ids + client_secret + payment_intent_id を含めて返す
4. Cart service が Redis から該当カートキーを削除
5. Cart service が cart.checked_out イベントを publish
6. Cart service が checkout レスポンスをフロントに返す
```

### なぜ Cart service が Order service を呼ぶのか

チェックアウトの「カートを読み出して → 注文を作って → カートを消す」一連の流れは cart service の責務にまとめるのが自然。gateway 層に振り分け込みロジックを持たせると薄いプロキシを崩してしまう。

Order service 側には `POST /internal/checkouts` という **クラスタ内部専用のエンドポイント** を用意する。Gateway は呼び出さないため JWT 検証は不要で、入力の `buyer_auth0_id` をそのまま信用する (cart service が既に検証済みの値を渡す前提)。本番では Istio / NetworkPolicy で `/internal/*` パスを外部から到達不能にする運用を想定している。

### トランザクション境界

Order service の `CreateCheckout` は以下のトランザクション境界で動作する:

| 境界 | 内容 |
| --- | --- |
| **DB Tx #1** | 全 order + order_lines + pending payout の INSERT |
| **(外部呼び出し)** | Stripe PaymentIntent 作成 |
| **DB Tx #2** | 全 order の `stripe_payment_intent_id` を UPDATE |

Stripe 呼び出しを挟んで DB トランザクションを分ける理由は [payment.md の「なぜトランザクションを 2 段階に分けるか」](./payment.md#決済フロー) を参照。

### 補償ロジック

Stripe 呼び出しが失敗した場合、Tx #1 で作成した注文を `status='cancelled'` にマークする補償処理を実行する。Tx #2 の UPDATE 失敗は運用アラートで対応する (稀なケース)。

---

## マルチセラー注文の表示

チェックアウト後、買い手にとっては「1 回の決済」だが、DB 上では複数の注文が存在する。UI ではこの抽象化を適切に行う必要がある:

### 推奨表示方針

- **注文履歴画面**: `stripe_payment_intent_id` でグループ化し、1 つの「注文 (バスケット)」として表示
- **詳細画面**: セラーごとにセクションを分けて表示 (「○○商店から発送」など)
- **ステータス追跡**: 各セラーの配送状況を個別に表示 (セラー A は「発送済み」、セラー B は「準備中」など)

### API サポート

`GET /api/v1/buyer/orders` は現状 1 レコード = 1 注文だが、フロント側でクライアントサイドに `stripe_payment_intent_id` でグルーピングする。将来的には order service に `ListCheckouts` (PI ID でグルーピング済み) を追加することを検討する。

---

## 購入履歴の取得

`/orders`, `/orders/{id}` 画面は、削除済み / アーカイブ済みの商品も含めて購入者が過去の注文を参照できる必要があります。表示に必要な情報は **スナップショット** と **クエリ時エンリッチ** の 2 系統に分けて保持します (詳細は [docs/architecture.md § 購入履歴と商品スナップショット](./architecture.md#購入履歴と商品スナップショット))。

### 一覧: `GET /api/v1/buyer/orders`

Gateway は order service の既存 REST を薄くプロキシします。レスポンスの各要素には checkout 時に固定された `seller_name` が含まれるため、販売者が削除されていてもカード表示に支障はありません。フロントは `seller_name` が空文字の場合 `orders.unknownSeller` にフォールバックします。

### 詳細: `GET /api/v1/buyer/orders/{id}`

Gateway は以下を行います:

1. `GET /orders/{id}` を order service に発行 (スナップショット値を含む)
2. 各行の `product_id` に対して `catalogGRPC.GetProduct` を `errgroup.WithContext` で **並列** に呼び出し
3. エンリッチ結果を以下のように合成:
   - `NotFound` または `product.status == "archived"` → `is_deleted=true`, `image_url=""`, `product_slug=""`
   - `active` → `image_url` と `slug` を乗せる
   - その他のエラー → warn ログ + `is_deleted=true` (グレースフル縮退: 1 行の失敗で注文全体の取得を失敗させない)
4. 商品名 (`product_name`) は **常にスナップショット値** を返す。現在の catalog 値にはフォールバックせず、購入時点の表示を保存する

注: catalog RPC に `BatchGetProducts` は実装していません。典型的な注文は ≤10 行で並列 `GetProduct` で十分です。行数が大きくなる領域 (B2B 一括注文など) に拡張する場合は batch RPC を追加してください。

---

## 在庫引当のタイミング

現状の設計では、在庫引当は **注文作成時 (`CreateCheckout`)** に `inventory service` と連携して実施する予定だが、**MVP では未実装** である。

現時点の挙動:
- カート追加時は在庫チェックなし (「0 在庫だがカートに入れられる」状態)
- 注文作成時も在庫引当なし (超過販売の可能性あり)

**あるべき姿**:
1. カート追加時: 在庫 > 0 を確認 (ソフトチェック、他の買い手の購入は考慮しない)
2. 注文作成時: `inventory.ReserveStock(sku_id, quantity)` を呼び、失敗なら注文キャンセル
3. 決済成功時: `inventory.CommitReservation(reservation_id)` で確定
4. 決済失敗時: `inventory.ReleaseReservation(reservation_id)` で解放

実装は別タスクとする。本書では **現状未実装であること** を明示するに留める。

---

## フロントエンド連携 (未実装)

現状フロントは `frontend/apps/buyer/src/app/(storefront)/products/[slug]/page.tsx:158` に "Add to cart" ボタンのプレースホルダがあるだけで、バックエンドに繋がっていない。後続タスクで以下を実装する:

### 1. API クライアントに JWT を付与

`frontend/apps/buyer/src/lib/api.ts` の `fetchAPI` に `Authorization: Bearer <token>` ヘッダを追加する。現状は `Content-Type` のみで、認証ヘッダなし。

### 2. Add-to-Cart ボタンの配線

```tsx
const handleAddToCart = async () => {
  await fetchAPI("/api/v1/buyer/cart/items", {
    method: "POST",
    body: JSON.stringify({ sku_id: sku.id, quantity: 1 }),
  });
};
```

### 3. カートページの追加

`frontend/apps/buyer/src/app/(storefront)/cart/page.tsx` を新設。セラーごとにグルーピングして商品を表示し、合計金額とチェックアウトボタンを配置する。

### 4. チェックアウトフロー

- カートページ → 住所入力 → Stripe.js の `CardElement` で決済情報入力
- `confirmCardPayment(clientSecret)` で Stripe に決済リクエスト
- 決済成功後、注文完了画面にリダイレクト

### 5. i18n キーの追加

`frontend/packages/i18n/messages/{en,ja}/buyer.json` に以下を追加:

```json
{
  "cart": {
    "title": "Cart",
    "empty": "Your cart is empty",
    "itemCount": "{count} items",
    "subtotal": "Subtotal",
    "total": "Total",
    "checkout": "Proceed to checkout",
    "removeItem": "Remove",
    "groupedBySeller": "Items from {seller}"
  }
}
```

---

## 既知の制約

| 制約 | 対応方針 |
| --- | --- |
| 在庫引当が未実装 | 別タスク。設計は上記参照 |
| 価格変動の反映なし (スナップショットのまま) | MVP では許容。将来 `sku.price_changed` イベントで対応 |
| 匿名ユーザーカート非対応 | 認証済みユーザーのみ。将来 session-id ベースのキーを追加 |
| カートマージ (匿名 → ログイン) 未対応 | 上記と同じく将来対応 |
| 複数通貨混在のバリデーション | checkout で 400 を返す想定だが未検証 |
| 在庫切れ SKU のカート削除通知 | catalog 側イベントとの連携が必要。未実装 |

---

## 関連ドキュメント

- [決済設計書](./payment.md) — Stripe 連携、コミッション、Payout、webhook の詳細
- [アーキテクチャ設計書](./architecture.md) — 全体像、データモデル、RLS
- [コントリビューションガイド](./CONTRIBUTING.md) — 開発プロセス、新サービス追加手順
