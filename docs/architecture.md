# アーキテクチャ設計書

## 目次

- [システムアーキテクチャ概要](#システムアーキテクチャ概要)
- [サービス一覧](#サービス一覧)
- [通信パターン](#通信パターン)
- [データモデル](#データモデル)
- [マルチテナント設計](#マルチテナント設計)
- [認証・認可フロー](#認証認可フロー)
- [イベント駆動アーキテクチャ](#イベント駆動アーキテクチャ)
- [検索・レコメンデーション](#検索レコメンデーション)
- [デプロイアーキテクチャ](#デプロイアーキテクチャ)
- [ローカル開発アーキテクチャ](#ローカル開発アーキテクチャ)

---

## システムアーキテクチャ概要

本システムはマイクロサービスアーキテクチャを採用し、各サービスが独立したドメインを担当します。全てのリクエストは API Gateway を経由し、テナント解決・認証検証が行われます。

```
                    ┌─────────────┐
                    │   Auth0     │
                    │  (IdP)      │
                    └──────┬──────┘
                           │ JWT 発行
                           ▼
┌──────────────────────────────────────────────────────────┐
│                     Clients                               │
│  Buyer App (:3000)  Seller App (:3001)  Admin App (:3002)│
└────────────────────────┬─────────────────────────────────┘
                         │ HTTPS (REST / JSON)
                         ▼
              ┌─────────────────────┐
              │   API Gateway       │
              │   (:8080)           │
              │                     │
              │  ・JWT 検証          │
              │  ・テナント解決      │
              │  ・レート制限        │
              │  ・リクエストルーティング │
              └──┬──┬──┬──┬──┬──┬──┘
                 │  │  │  │  │  │
    ┌────────────┘  │  │  │  │  └──────────────┐
    │     ┌─────────┘  │  │  └──────────┐      │
    ▼     ▼            ▼  ▼             ▼      ▼
┌──────┐┌──────┐┌────────┐┌──────┐┌──────┐┌────────┐
│ Auth ││Cata- ││Inven-  ││Order ││Search││Recom-  │
│      ││log   ││tory    ││      ││      ││mend   │
│:8081 ││:8082 ││:8083   ││:8084 ││:8085 ││:8086  │
└──┬───┘└──┬───┘└───┬────┘└──┬───┘└──┬───┘└───────┘
   │       │        │        │       │
   │       │        │        │       │    ┌────────────┐
   │       │        │        │       │    │Notification│
   │       │        │        │       │    │  :8087     │
   │       │        │        │       │    └─────┬──────┘
   │       │        │        │       │          │
   ▼       ▼        ▼        ▼       ▼          ▼
┌─────────────────────────────────────────────────────┐
│                  Infrastructure                      │
│  ┌───────────┐  ┌─────────┐  ┌───────────────────┐  │
│  │PostgreSQL │  │  Redis   │  │  Cloud Pub/Sub    │  │
│  │  16       │  │  7       │  │  (非同期メッセージ) │  │
│  │(RLS分離)  │  │(キャッシュ)│  │                   │  │
│  └───────────┘  └─────────┘  └───────────────────┘  │
│                                                      │
│  ┌──────────────────┐  ┌────────────────────┐        │
│  │ Vertex AI Search │  │  Stripe Connect    │        │
│  │ (商品検索)        │  │  (決済・送金)       │        │
│  └──────────────────┘  └────────────────────┘        │
└─────────────────────────────────────────────────────┘
```

---

## サービス一覧

| サービス         | ポート | 役割                                                                    | DB スキーマ     |
| ---------------- | ------ | ----------------------------------------------------------------------- | --------------- |
| **gateway**      | 8080   | API Gateway。JWT 検証、テナント解決、リクエストルーティング、レート制限 | なし            |
| **auth**         | 8081   | テナント管理、セラー登録・管理、ユーザー認証連携 (Auth0)                | `auth_svc`      |
| **catalog**      | 8082   | 商品・SKU・カテゴリ管理、商品公開・非公開制御                           | `catalog_svc`   |
| **inventory**    | 8083   | 在庫数量管理、在庫引当・解放、在庫移動履歴                              | `inventory_svc` |
| **order**        | 8084   | 注文作成・管理、決済処理 (Stripe)、コミッション計算、売上送金           | `order_svc`     |
| **search**       | 8085   | 商品検索 (Vertex AI Search 連携)、ファセット検索                        | なし (外部)     |
| **recommend**    | 8086   | レコメンデーション、パーソナライズ                                      | なし (外部)     |
| **notification** | 8087   | メール・プッシュ通知、イベント購読による自動通知                        | なし            |

---

## 通信パターン

本システムでは 3 種類の通信パターンを使い分けます。

### 1. REST (外部 → Gateway)

クライアント (ブラウザ / モバイルアプリ) から Gateway への通信は REST/JSON で行います。

```
Client  ──HTTP/JSON──▶  Gateway  ──HTTP/JSON──▶  各サービス
```

- 全リクエストに `Authorization: Bearer <JWT>` ヘッダーを付与
- テナント識別: JWT クレーム内の `tenant_id` または `X-Tenant-ID` ヘッダー
- レスポンス形式: JSON (`Content-Type: application/json`)

### 2. gRPC (サービス間同期通信)

サービス間の同期的なデータ取得には gRPC を使用します (今後実装予定)。

```
Order Service  ──gRPC──▶  Catalog Service (商品情報取得)
Order Service  ──gRPC──▶  Inventory Service (在庫引当)
```

- Proto 定義: `backend/proto/` ディレクトリ
- コード生成: `make proto-gen` (`buf generate`)

### 3. Cloud Pub/Sub (非同期イベント)

サービス間の非同期通信には Cloud Pub/Sub を使用します。

```
Order Service  ──publish──▶  [order.created]  ──subscribe──▶  Notification Service
                                              ──subscribe──▶  Inventory Service
Catalog Service ──publish──▶ [catalog.product_updated] ──subscribe──▶ Search Service
```

---

## データモデル

### ER 図

```mermaid
erDiagram
    tenants ||--o{ sellers : "has many"
    tenants ||--o{ categories : "has many"
    sellers ||--o{ seller_users : "has many"
    sellers ||--o{ products : "has many"
    sellers ||--o{ orders : "receives"
    sellers ||--o{ inventory : "manages"
    sellers ||--o{ payouts : "receives"
    products ||--o{ skus : "has many"
    products }o--o{ categories : "belongs to many"
    skus ||--o| inventory : "has one"
    skus ||--o{ stock_movements : "has many"
    skus ||--o{ order_lines : "referenced by"
    orders ||--o{ order_lines : "has many"
    orders ||--o| payouts : "has one"

    tenants {
        uuid id PK
        varchar name
        varchar slug UK
        varchar status
        jsonb settings
        timestamptz created_at
        timestamptz updated_at
    }

    sellers {
        uuid id PK
        uuid tenant_id FK
        varchar auth0_org_id
        varchar name
        varchar slug
        varchar status
        varchar stripe_account_id
        int commission_rate_bps
        jsonb settings
    }

    seller_users {
        uuid id PK
        uuid tenant_id FK
        uuid seller_id FK
        varchar auth0_user_id
        varchar role
    }

    categories {
        uuid id PK
        uuid tenant_id FK
        uuid parent_id FK
        varchar name
        varchar slug
        int sort_order
    }

    products {
        uuid id PK
        uuid tenant_id FK
        uuid seller_id FK
        varchar name
        varchar slug
        text description
        varchar status
        jsonb attributes
    }

    skus {
        uuid id PK
        uuid tenant_id FK
        uuid product_id FK
        uuid seller_id FK
        varchar sku_code
        bigint price_amount
        varchar price_currency
        jsonb attributes
        varchar status
    }

    inventory {
        uuid id PK
        uuid tenant_id FK
        uuid sku_id FK
        uuid seller_id FK
        int quantity_available
        int quantity_reserved
        int low_stock_threshold
    }

    stock_movements {
        uuid id PK
        uuid tenant_id FK
        uuid sku_id FK
        varchar movement_type
        int quantity
        varchar reference_type
        uuid reference_id
    }

    orders {
        uuid id PK
        uuid tenant_id FK
        uuid seller_id FK
        varchar buyer_auth0_id
        varchar status
        bigint subtotal_amount
        bigint commission_amount
        bigint total_amount
        varchar currency
        jsonb shipping_address
        varchar stripe_payment_intent_id
    }

    order_lines {
        uuid id PK
        uuid tenant_id FK
        uuid order_id FK
        uuid sku_id FK
        varchar product_name
        varchar sku_code
        int quantity
        bigint unit_price
        bigint line_total
    }

    commission_rules {
        uuid id PK
        uuid tenant_id FK
        uuid seller_id FK
        uuid category_id FK
        int rate_bps
        int priority
        timestamptz valid_from
        timestamptz valid_until
    }

    payouts {
        uuid id PK
        uuid tenant_id FK
        uuid seller_id FK
        uuid order_id FK
        bigint amount
        varchar currency
        varchar stripe_transfer_id
        varchar status
    }
```

### DB スキーマ構成

PostgreSQL のスキーマ機能を利用し、サービスごとにスキーマを分離:

| スキーマ        | 担当サービス | テーブル                                               |
| --------------- | ------------ | ------------------------------------------------------ |
| `auth_svc`      | auth         | `tenants`, `sellers`, `seller_users`                   |
| `catalog_svc`   | catalog      | `categories`, `products`, `skus`, `product_categories` |
| `inventory_svc` | inventory    | `inventory`, `stock_movements`                         |
| `order_svc`     | order        | `orders`, `order_lines`, `commission_rules`, `payouts` |

### 金額の表現

全ての金額は **BIGINT 型 (最小通貨単位)** で保存します:

- 日本円 (JPY): `1000` = 1,000円
- コミッション率: `rate_bps` (ベーシスポイント, 1bps = 0.01%) -- `1000` = 10%

---

## マルチテナント設計

### Pool モデル

本システムは **Pool (共有データベース) モデル** を採用し、全テナントが同一の PostgreSQL インスタンス・同一のテーブルを共有します。テナント分離は PostgreSQL の **Row-Level Security (RLS)** で実現します。

```
┌─────────────────────────────────────────────────┐
│                  PostgreSQL 16                    │
│                                                  │
│  ┌───────────────────────────────────────────┐   │
│  │  catalog_svc.products                      │   │
│  │  ┌─────────┬──────────────────────────┐   │   │
│  │  │tenant_id│ data                      │   │   │
│  │  ├─────────┼──────────────────────────┤   │   │
│  │  │ AAA     │ テナントAの商品 ...       │   │   │
│  │  │ AAA     │ テナントAの商品 ...       │   │   │
│  │  │ BBB     │ テナントBの商品 ...       │   │   │  ← 同一テーブル
│  │  │ BBB     │ テナントBの商品 ...       │   │   │
│  │  │ CCC     │ テナントCの商品 ...       │   │   │
│  │  └─────────┴──────────────────────────┘   │   │
│  │                                            │   │
│  │  RLS Policy: tenant_id =                   │   │
│  │    current_setting('app.current_tenant_id')│   │
│  └───────────────────────────────────────────┘   │
└─────────────────────────────────────────────────┘
```

### RLS ポリシーの仕組み

1. **リクエスト受信**: Gateway が JWT からテナント ID を抽出
2. **コンテキスト伝播**: `backend/pkg/tenant` パッケージで Go context にテナント ID を格納
3. **DB セッション設定**: クエリ実行前に `SET app.current_tenant_id = '<uuid>'` を実行
4. **自動フィルタリング**: PostgreSQL の RLS が全クエリに `WHERE tenant_id = ...` を自動付与

```sql
-- 全テナントスコープテーブルに適用される RLS ポリシー
ALTER TABLE catalog_svc.products ENABLE ROW LEVEL SECURITY;

CREATE POLICY tenant_isolation ON catalog_svc.products
    USING (tenant_id = current_setting('app.current_tenant_id')::uuid);
```

### テナント分離の保証

- **DB レベル**: RLS による強制的なフィルタリング (アプリケーションバグがあっても他テナントのデータは見えない)
- **アプリケーションレベル**: 全クエリで `tenant_id` を WHERE 条件に含める (RLS が無効化された場合の保険)
- **インデックス**: 全テーブルの `tenant_id` にインデックスを作成し、パフォーマンスを確保

---

## 認証・認可フロー

### 全体フロー

```mermaid
sequenceDiagram
    participant Client as Client (Browser)
    participant Auth0 as Auth0
    participant GW as API Gateway
    participant Svc as Backend Service
    participant DB as PostgreSQL

    Client->>Auth0: 1. ログイン (email/password, Google, etc.)
    Auth0-->>Client: 2. JWT (access_token) 発行
    Note over Client,Auth0: JWT Claims: sub, tenant_id, seller_id, roles

    Client->>GW: 3. API リクエスト + Authorization: Bearer <JWT>
    GW->>GW: 4. JWT 署名検証 (Auth0 JWKS)
    GW->>GW: 5. テナント ID 抽出 + コンテキスト構築
    GW->>Svc: 6. リクエスト転送 (X-Tenant-ID, X-User-ID ヘッダー)
    Svc->>DB: 7. SET app.current_tenant_id = '<tenant_id>'
    Svc->>DB: 8. クエリ実行 (RLS が自動フィルタ)
    DB-->>Svc: 9. テナントスコープの結果
    Svc-->>GW: 10. レスポンス
    GW-->>Client: 11. レスポンス
```

### JWT クレーム構造

```json
{
  "sub": "auth0|abc123",
  "iss": "https://<tenant>.auth0.com/",
  "aud": "https://api.ec-marketplace.example.com",
  "tenant_id": "550e8400-e29b-41d4-a716-446655440000",
  "seller_id": "6ba7b810-9dad-11d1-80b4-00c04fd430c8",
  "roles": ["seller:admin"],
  "exp": 1700000000
}
```

### ロール一覧

| ロール           | 説明                   | アクセス範囲         |
| ---------------- | ---------------------- | -------------------- |
| `platform:admin` | プラットフォーム管理者 | 全テナント・全操作   |
| `tenant:admin`   | テナント管理者         | 自テナント内の全操作 |
| `seller:admin`   | セラー管理者           | 自セラーの全操作     |
| `seller:member`  | セラーメンバー         | 自セラーの限定操作   |
| `buyer`          | 購入者                 | 商品閲覧・購入       |

---

## イベント駆動アーキテクチャ

### トピック一覧

| トピック                  | パブリッシャー | サブスクライバー        | 説明             |
| ------------------------- | -------------- | ----------------------- | ---------------- |
| `order.created`           | order          | notification, inventory | 注文作成時       |
| `order.paid`              | order          | notification, inventory | 決済完了時       |
| `order.cancelled`         | order          | notification, inventory | 注文キャンセル時 |
| `inventory.low_stock`     | inventory      | notification            | 在庫が閾値以下   |
| `catalog.product_updated` | catalog        | search                  | 商品情報更新時   |
| `catalog.product_deleted` | catalog        | search                  | 商品削除時       |
| `seller.registered`       | auth           | notification            | セラー新規登録時 |
| `payout.completed`        | order          | notification            | 売上送金完了時   |

### メッセージフォーマット

全メッセージは以下の共通構造を持ちます:

```json
{
  "event_type": "order.created",
  "tenant_id": "550e8400-e29b-41d4-a716-446655440000",
  "timestamp": "2026-04-06T12:00:00Z",
  "data": {
    "order_id": "...",
    "seller_id": "...",
    "buyer_id": "...",
    "total": 5000,
    "currency": "JPY"
  }
}
```

**設計方針:**

- **全メッセージに `tenant_id` を含める**: サブスクライバーがテナントコンテキストを復元するため
- **べき等処理**: サブスクライバーは同一メッセージの再配信に対応
- **Pub/Sub 失敗は呼び出し元を失敗させない**: 結果整合性 (eventual consistency) を許容

---

## 検索・レコメンデーション

### 検索アーキテクチャ (Vertex AI Search)

```
┌──────────┐    Pub/Sub     ┌──────────────┐
│ Catalog  │───────────────▶│ Search       │
│ Service  │ product_updated│ Service      │
└──────────┘                │              │
                            │  ・インデックス更新  │
                            │  ・検索クエリ処理    │
                            └──────┬───────┘
                                   │
                            ┌──────▼───────┐
                            │ Vertex AI    │
                            │ Search       │
                            │              │
                            │ ・全文検索     │
                            │ ・ファセット   │
                            │ ・ランキング   │
                            └──────────────┘
```

**データフロー:**

1. `catalog` サービスで商品が作成・更新されると `catalog.product_updated` イベントを発行
2. `search` サービスがイベントを購読し、Vertex AI Search のインデックスを更新
3. 検索リクエストは `search` サービス経由で Vertex AI Search API に問い合わせ
4. 結果にはテナント ID によるフィルタが自動適用

### レコメンデーション

`recommend` サービスは以下のデータを基にパーソナライズドレコメンデーションを提供:

- 閲覧履歴
- 購入履歴
- 類似商品 (Vertex AI Search のセマンティック検索)

---

## デプロイアーキテクチャ

### GKE + ArgoCD (GitOps)

```
┌────────────┐     push      ┌──────────────┐
│ Developer  │──────────────▶│   GitHub      │
└────────────┘               │   Repository  │
                             └──────┬───────┘
                                    │
                    ┌───────────────┼───────────────┐
                    │               │               │
                    ▼               ▼               ▼
            ┌──────────────┐ ┌──────────┐  ┌──────────────┐
            │GitHub Actions│ │  ArgoCD  │  │   Container  │
            │              │ │          │  │   Registry   │
            │・lint         │ │・Sync     │  │   (GCR)      │
            │・test         │ │・Diff     │  └──────────────┘
            │・build image  │ │・Rollback │
            │・push image   │ └────┬─────┘
            └──────────────┘      │
                                  ▼
                          ┌──────────────┐
                          │     GKE      │
                          │              │
                          │ ┌──────────┐ │
                          │ │ gateway  │ │
                          │ │ auth     │ │
                          │ │ catalog  │ │
                          │ │ inventory│ │
                          │ │ order    │ │
                          │ │ search   │ │
                          │ │ recommend│ │
                          │ │ notify   │ │
                          │ └──────────┘ │
                          └──────────────┘
```

### デプロイ構成

```
infra/deploy/
├── base/                    # Kustomize base (全環境共通)
│   ├── gateway/
│   │   ├── deployment.yaml
│   │   ├── service.yaml
│   │   └── kustomization.yaml
│   ├── catalog/
│   └── ...
├── overlays/                # 環境別オーバーレイ
│   ├── dev/
│   │   └── kustomization.yaml
│   ├── staging/
│   │   └── kustomization.yaml
│   └── production/
│       └── kustomization.yaml
└── argocd/                  # ArgoCD Application 定義
    └── apps.yaml
```

### デプロイフロー

1. `main` ブランチへのマージで GitHub Actions が起動
2. CI: lint → test → Docker イメージビルド → GCR にプッシュ
3. CI が `infra/deploy/overlays/<env>/kustomization.yaml` のイメージタグを更新
4. ArgoCD がマニフェスト変更を検知し、GKE クラスタに自動同期

---

## ローカル開発アーキテクチャ

### 構成図

```
開発者マシン
├── Docker Compose (make deps-up)
│   ├── PostgreSQL 16  (:5432)
│   │   └── DB: ecmarket_dev / User: ecmarket / Pass: localdev
│   ├── Redis 7        (:6379)
│   └── Pub/Sub Emulator (:8085)
│
├── Go Services (各 make dev-<service>)
│   ├── gateway      :8080  ← air (ホットリロード)
│   ├── auth         :8081
│   ├── catalog      :8082
│   ├── inventory    :8083
│   ├── order        :8084
│   ├── search       :8085  (※ Pub/Sub と同ポートに注意、実際には別ポートを使用)
│   ├── recommend    :8086
│   └── notification :8087
│
└── Next.js Apps (pnpm)
    ├── buyer        :3000
    ├── seller       :3001
    └── admin        :3002
```

### ローカル開発 Tips

**全サービスを起動する必要はない**: 開発対象のサービスと Gateway のみ起動すれば OK。

```bash
# カタログ機能を開発する場合
make deps-up          # インフラ起動
make dev-gateway      # Gateway (必須)
make dev-auth         # テナント解決に必要
make dev-catalog      # 開発対象
```

**DB を直接確認する場合:**

```bash
psql postgres://ecmarket:localdev@localhost:5432/ecmarket_dev
```

**Pub/Sub Emulator を使用する場合:**

```bash
export PUBSUB_EMULATOR_HOST=localhost:8085
```

**ログの確認**: 全サービスが JSON 形式の構造化ログを出力するため、`jq` を使うと読みやすくなります:

```bash
make dev-catalog 2>&1 | jq .
```
