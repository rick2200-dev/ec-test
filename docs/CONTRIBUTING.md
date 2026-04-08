# コントリビューションガイド

本プロジェクトへの貢献方法・開発規約をまとめたドキュメントです。

## 目次

- [開発環境セットアップ](#開発環境セットアップ)
- [ブランチ命名規則](#ブランチ命名規則)
- [コミットメッセージ規約](#コミットメッセージ規約)
- [Go コーディング規約](#go-コーディング規約)
- [フロントエンド コーディング規約](#フロントエンド-コーディング規約)
- [PR レビュープロセス](#pr-レビュープロセス)
- [新しいマイクロサービスの追加手順](#新しいマイクロサービスの追加手順)
- [新しい DB マイグレーションの追加手順](#新しい-db-マイグレーションの追加手順)
- [新しい Pub/Sub イベントの追加手順](#新しい-pubsub-イベントの追加手順)
- [テストガイドライン](#テストガイドライン)

---

## 開発環境セットアップ

### 必須ツール

| ツール                  | バージョン | 用途                                                |
| ----------------------- | ---------- | --------------------------------------------------- |
| Go                      | 1.25+      | バックエンドサービス                                |
| Node.js                 | 20+        | フロントエンド                                      |
| pnpm                    | 10+        | パッケージマネージャー (`corepack enable` で有効化) |
| Docker / Docker Compose | latest     | ローカル依存サービス                                |
| golang-migrate          | latest     | DB マイグレーション                                 |
| air                     | latest     | Go ホットリロード                                   |
| golangci-lint           | latest     | Go lint                                             |
| buf                     | latest     | Protocol Buffers コード生成                         |
| psql                    | latest     | DB シードデータ投入                                 |

### 初回セットアップ

```bash
# リポジトリクローン後
make deps-up          # PostgreSQL, Redis, Pub/Sub Emulator 起動
make migrate          # スキーマ作成
make seed             # 開発データ投入
pnpm install          # フロントエンド依存インストール
```

### 動作確認

```bash
make dev-gateway      # Gateway 起動後 → http://localhost:8080/healthz
make dev-auth         # Auth 起動後 → http://localhost:8081/healthz
```

---

## ブランチ命名規則

| プレフィックス | 用途                 | 例                                 |
| -------------- | -------------------- | ---------------------------------- |
| `feature/`     | 新機能開発           | `feature/add-cart-api`             |
| `fix/`         | バグ修正             | `fix/order-total-calculation`      |
| `docs/`        | ドキュメント変更     | `docs/update-api-spec`             |
| `refactor/`    | リファクタリング     | `refactor/extract-payment-service` |
| `chore/`       | 設定変更・依存更新等 | `chore/upgrade-go-1.26`            |
| `test/`        | テスト追加・修正     | `test/add-catalog-integration`     |
| `hotfix/`      | 本番緊急修正         | `hotfix/fix-rls-policy`            |

**命名ルール:**

- 英語小文字とハイフンのみ使用
- 簡潔に内容が分かる名前にする
- Issue がある場合は番号を含める: `feature/123-add-cart-api`

---

## コミットメッセージ規約

[Conventional Commits](https://www.conventionalcommits.org/) に準拠します。

### フォーマット

```
<type>(<scope>): <description>

[optional body]

[optional footer]
```

### Type 一覧

| Type       | 説明                                          |
| ---------- | --------------------------------------------- |
| `feat`     | 新機能                                        |
| `fix`      | バグ修正                                      |
| `docs`     | ドキュメントのみの変更                        |
| `style`    | コードの意味に影響しない変更 (フォーマット等) |
| `refactor` | バグ修正でも新機能でもないコード変更          |
| `perf`     | パフォーマンス改善                            |
| `test`     | テストの追加・修正                            |
| `chore`    | ビルドプロセスやツールの変更                  |
| `ci`       | CI 設定の変更                                 |

### Scope 一覧

サービス名やパッケージ名を scope として使用します。

```
feat(catalog): 商品のバリアント属性フィルタを追加
fix(order): コミッション計算の丸め誤差を修正
docs(readme): クイックスタート手順を更新
refactor(pkg/database): コネクションプール設定を環境変数化
test(inventory): 在庫引当の並行テストを追加
chore(deps): golangci-lint を v2 に更新
```

### Breaking Change

破壊的変更がある場合は `!` を付与するか footer に `BREAKING CHANGE:` を記載します。

```
feat(gateway)!: API パスプレフィックスを /api/v2 に変更

BREAKING CHANGE: 全 API エンドポイントのパスが /api/v1 から /api/v2 に変更されました。
```

---

## Go コーディング規約

### プロジェクトレイアウト

各サービスは以下の標準構成に従います:

```
services/<service>/
├── cmd/
│   └── server/
│       └── main.go          # エントリーポイント
├── internal/
│   ├── config/
│   │   └── config.go        # 環境変数の読み込み
│   ├── handler/
│   │   └── <resource>.go    # HTTP ハンドラー (chi router)
│   ├── service/
│   │   └── <resource>.go    # ビジネスロジック
│   └── repository/
│       └── <resource>.go    # データアクセス (pgx)
├── go.mod
├── go.sum
└── Dockerfile
```

### エラーハンドリング

```go
// pkg/errors の共通エラー型を使用
import pkgerr "github.com/Riku-KANO/ec-test/pkg/errors"

// サービス層でドメインエラーを返す
if product == nil {
    return nil, pkgerr.NotFound("product", id.String())
}

// ハンドラー層で HTTP レスポンスに変換
// pkg/httputil がエラー型に応じた適切なステータスコードを返す
```

### 命名規約

- **パッケージ名**: 小文字・単数形 (`handler`, `service`, `repository`)
- **インターフェース**: 動詞 + er (`Reader`, `ProductService`)
- **エクスポートする関数/型**: `PascalCase`
- **非公開関数/型**: `camelCase`
- **定数**: `PascalCase` (Go の慣例に従う)
- **テーブルカラム**: `snake_case` (SQL 側)

### テナントコンテキスト

全てのテナントスコープ操作で `pkg/tenant` パッケージを使用します:

```go
import "github.com/Riku-KANO/ec-test/pkg/tenant"

func (s *Service) GetProduct(ctx context.Context, id uuid.UUID) (*Product, error) {
    // テナント ID はコンテキストから取得 (ミドルウェアが設定済み)
    tc, err := tenant.FromContext(ctx)
    if err != nil {
        return nil, err
    }
    return s.repo.FindByID(ctx, tc.TenantID, id)
}
```

### DB アクセスと RLS

```go
import "github.com/Riku-KANO/ec-test/pkg/database"

// DB 操作前に必ず SET app.current_tenant_id を実行
// pkg/database のヘルパー関数が自動的に処理
func (r *Repository) FindByID(ctx context.Context, tenantID uuid.UUID, id uuid.UUID) (*Product, error) {
    conn, err := r.pool.Acquire(ctx)
    if err != nil {
        return nil, err
    }
    defer conn.Release()

    // RLS 用テナント設定
    database.SetTenant(ctx, conn, tenantID)

    // クエリ実行 (RLS が自動的に tenant_id で絞り込み)
    // ...
}
```

### ロガー

`log/slog` を使用します。構造化ログを JSON フォーマットで出力:

```go
slog.Info("order created",
    "order_id", order.ID,
    "tenant_id", tc.TenantID,
    "total", order.TotalAmount,
)
```

---

## フロントエンド コーディング規約

### Next.js App Router パターン

```
apps/<app>/
├── src/
│   ├── app/
│   │   ├── layout.tsx            # ルートレイアウト
│   │   ├── page.tsx              # トップページ
│   │   └── (routes)/
│   │       └── products/
│   │           ├── page.tsx      # Server Component (一覧)
│   │           └── [id]/
│   │               └── page.tsx  # Server Component (詳細)
│   ├── components/
│   │   ├── ui/                   # 汎用 UI コンポーネント
│   │   └── features/             # 機能固有コンポーネント
│   ├── lib/
│   │   ├── api/                  # API クライアント
│   │   └── utils/                # ユーティリティ
│   └── hooks/                    # カスタムフック
├── public/
└── next.config.ts
```

### 基本ルール

- **Server Components をデフォルトとする**: `"use client"` は必要な場合のみ
- **データ取得は Server Component で**: `fetch()` を直接呼び出す (React Server Components)
- **状態管理は最小限に**: URL パラメータ、Server Component での取得を優先
- **共有コンポーネントは `packages/` に**: 複数アプリで使う UI は共通パッケージ化
- **型安全**: OpenAPI 仕様から生成した型定義を使用

---

## PR レビュープロセス

### PR 作成時のチェックリスト

- [ ] Conventional Commits に従ったコミットメッセージ
- [ ] テストを追加・更新済み
- [ ] `make lint-go` でエラーなし
- [ ] `make test-go` で全テストパス
- [ ] マイグレーションがある場合、down マイグレーションも作成済み
- [ ] PR の説明に変更内容と動作確認手順を記載

### レビューフロー

1. **PR 作成**: ドラフト PR で早期フィードバックを得ることを推奨
2. **CI チェック**: GitHub Actions で lint・テスト・ビルドが自動実行
3. **コードレビュー**: 最低 1 名の Approve が必要
4. **マージ**: Squash Merge を使用 (コミット履歴をクリーンに保つ)

### レビュー観点

- **テナント分離**: RLS ポリシーが正しく適用されているか
- **エラーハンドリング**: 適切なエラー型を使用しているか
- **セキュリティ**: SQL インジェクション、認証バイパスの可能性がないか
- **パフォーマンス**: N+1 クエリ、不要なデータ取得がないか
- **テスト**: エッジケースがカバーされているか

---

## 新しいマイクロサービスの追加手順

### 1. ディレクトリ作成

```bash
mkdir -p services/<service-name>/cmd/server
mkdir -p services/<service-name>/internal/{config,handler,service,repository}
```

### 2. Go モジュール初期化

```bash
cd services/<service-name>
go mod init github.com/Riku-KANO/ec-test/services/<service-name>
```

### 3. go.work に追加

```go
// go.work
use (
    ./pkg
    ./services/<service-name>  // 追加
    // ... 既存サービス
)
```

### 4. エントリーポイント作成

`cmd/server/main.go` を既存サービス (例: `services/auth/cmd/server/main.go`) をテンプレートとして作成:

- 構造化ログ設定 (`slog` + JSON)
- DB 接続プール初期化
- Router 設定 (chi)
- `/healthz`, `/readyz` エンドポイント
- Graceful Shutdown

### 5. 設定ファイル作成

`internal/config/config.go`:

```go
package config

import "os"

type Config struct {
    HTTPPort    string
    DatabaseURL string
}

func Load() Config {
    return Config{
        HTTPPort:    getEnv("HTTP_PORT", "808X"),  // 適切なポート番号
        DatabaseURL: getEnv("DATABASE_URL", "postgres://ecmarket:localdev@localhost:5432/ecmarket_dev?sslmode=disable"),
    }
}

func getEnv(key, fallback string) string {
    if v := os.Getenv(key); v != "" {
        return v
    }
    return fallback
}
```

### 6. Dockerfile 作成

既存サービスの `Dockerfile` をコピーしてサービス名を変更。

### 7. Makefile に追加

```makefile
dev-<service-name>:
	cd services/<service-name> && air
```

`build-all`, `lint-go`, `test-go` のサービスリストにも追加。

### 8. Kubernetes マニフェスト作成

`deploy/base/<service-name>/` に Deployment, Service, ConfigMap を作成。

### 9. DB スキーマが必要な場合

マイグレーションを作成してスキーマとテーブルを追加 (次セクション参照)。

---

## 新しい DB マイグレーションの追加手順

### 1. マイグレーションファイル作成

```bash
make migrate-create
# プロンプトに名前を入力: create_<table_name>
```

これで `db/migrations/` に `NNNNNN_create_<table_name>.up.sql` と `.down.sql` が作成されます。

### 2. UP マイグレーション記述

```sql
-- db/migrations/NNNNNN_create_<table_name>.up.sql

CREATE TABLE <schema>_svc.<table_name> (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL,
    -- カラム定義...
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- tenant_id インデックス (必須)
CREATE INDEX idx_<table>_tenant ON <schema>_svc.<table_name>(tenant_id);

-- Row-Level Security (必須)
ALTER TABLE <schema>_svc.<table_name> ENABLE ROW LEVEL SECURITY;

CREATE POLICY tenant_isolation ON <schema>_svc.<table_name>
    USING (tenant_id = current_setting('app.current_tenant_id')::uuid);
```

**重要なルール:**

- **全テーブルに `tenant_id` カラムを追加** (テナント分離のため)
- **全テーブルに RLS ポリシーを設定** (`tenant_isolation` ポリシー)
- **`tenant_id` に対するインデックスを作成**
- **サービス固有の PostgreSQL スキーマを使用** (`auth_svc`, `catalog_svc` 等)
- **金額は BIGINT (最小通貨単位)** で保存 (例: 1000 = 1000円)

### 3. DOWN マイグレーション記述

```sql
-- db/migrations/NNNNNN_create_<table_name>.down.sql
DROP TABLE IF EXISTS <schema>_svc.<table_name>;
```

### 4. マイグレーション実行・検証

```bash
make migrate          # UP を実行
make migrate-down     # DOWN でロールバックできることを確認
make migrate          # 再度 UP して問題ないことを確認
```

---

## 新しい Pub/Sub イベントの追加手順

### 1. トピック・サブスクリプション定義

イベント名は `<domain>.<action>` の形式で命名:

```
order.created
order.paid
inventory.low_stock
catalog.product_updated
```

### 2. メッセージ構造体の定義

`pkg/pubsub/` にメッセージ型を追加:

```go
// pkg/pubsub/events.go
type OrderCreatedEvent struct {
    TenantID  string    `json:"tenant_id"`
    OrderID   string    `json:"order_id"`
    SellerID  string    `json:"seller_id"`
    BuyerID   string    `json:"buyer_id"`
    Total     int64     `json:"total"`
    Currency  string    `json:"currency"`
    CreatedAt time.Time `json:"created_at"`
}
```

**全イベントに `tenant_id` を含めること** (サブスクライバーがテナントコンテキストを復元するため)。

### 3. パブリッシャー実装

```go
import "github.com/Riku-KANO/ec-test/pkg/pubsub"

func (s *OrderService) CreateOrder(ctx context.Context, req CreateOrderRequest) (*Order, error) {
    // ... 注文作成ロジック

    // イベント発行
    err := s.publisher.Publish(ctx, "order.created", pubsub.OrderCreatedEvent{
        TenantID: tc.TenantID.String(),
        OrderID:  order.ID.String(),
        // ...
    })
    if err != nil {
        slog.Error("failed to publish order.created", "error", err)
        // Pub/Sub 失敗は注文作成自体を失敗させない (eventually consistent)
    }

    return order, nil
}
```

### 4. サブスクライバー実装

受信側サービスにサブスクリプションハンドラーを実装:

```go
func (h *NotificationHandler) HandleOrderCreated(ctx context.Context, msg *pubsub.Message) error {
    var event pubsub.OrderCreatedEvent
    if err := json.Unmarshal(msg.Data, &event); err != nil {
        return fmt.Errorf("unmarshal event: %w", err)
    }

    // テナントコンテキスト復元
    tenantID, _ := uuid.Parse(event.TenantID)
    ctx = tenant.WithContext(ctx, tenant.Context{TenantID: tenantID})

    // 処理実行
    return h.sendOrderConfirmation(ctx, event)
}
```

### 5. ローカル開発での動作確認

Pub/Sub Emulator が `make deps-up` で起動済み。環境変数でエミュレーターに接続:

```bash
export PUBSUB_EMULATOR_HOST=localhost:8085
```

---

## テストガイドライン

### テストの種類

| 種類       | 場所                       | 実行方法                          |
| ---------- | -------------------------- | --------------------------------- |
| 単体テスト | `*_test.go` (同パッケージ) | `go test ./...`                   |
| 統合テスト | `*_integration_test.go`    | `go test -tags=integration ./...` |
| E2E テスト | `tests/e2e/` (今後追加)    | TBD                               |

### 単体テスト

```go
// service/product_test.go
func TestCreateProduct_ValidInput(t *testing.T) {
    // Arrange: モックリポジトリを設定
    mockRepo := &MockProductRepository{...}
    svc := NewProductService(mockRepo)

    // Act
    product, err := svc.Create(ctx, CreateProductRequest{...})

    // Assert
    require.NoError(t, err)
    assert.Equal(t, "Test Product", product.Name)
}
```

### テスト命名規約

```
Test<Function>_<Scenario>
```

例: `TestCreateProduct_DuplicateSlug`, `TestGetOrder_NotFound`

### テナント分離テスト

マルチテナントのデータ分離が正しく機能していることを必ずテスト:

```go
func TestProductRepository_TenantIsolation(t *testing.T) {
    // テナント A のコンテキストでテナント B のデータにアクセスできないことを確認
}
```

### テストデータ

- テストではテスト用のテナント ID を使用
- DB テストでは `t.Cleanup()` で確実にクリーンアップ
- 共通のテストヘルパーを `internal/testutil/` に配置

### カバレッジ目標

- **サービス層 (ビジネスロジック)**: 80% 以上
- **リポジトリ層**: 統合テストでカバー
- **ハンドラー層**: 主要パスをカバー
