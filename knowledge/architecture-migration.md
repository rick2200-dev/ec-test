# アーキテクチャ移行記録

ヘキサゴナルアーキテクチャへの段階的リファクタリングの記録。  
「なぜこの順序で変更したか」「どんな問題を解決したか」「何を学んだか」を残す。

設計の原則と完成形については [`hexagonal-architecture.md`](./hexagonal-architecture.md) を参照してください。

## 目次

- [移行前の問題点](#移行前の問題点)
- [移行の全体方針](#移行の全体方針)
- [Phase 0: ドメインエラー分離 + 型の所有権整理](#phase-0-ドメインエラー分離--型の所有権整理)
- [Phase 1: トランザクション抽象化](#phase-1-トランザクション抽象化)
- [Phase 2: ポート抽出 + HTTPクライアント移動](#phase-2-ポート抽出--httpクライアント移動)
- [Phase 3: ディレクトリリネーム](#phase-3-ディレクトリリネーム)
- [Phase 4: ドメイン層の充実](#phase-4-ドメイン層の充実)
- [Phase 5: auth モノリスの分解と subscription サービス抽出](#phase-5-auth-モノリスの分解と-subscription-サービス抽出)
- [設計上の判断メモ](#設計上の判断メモ)

---

## 移行前の問題点

リファクタリング前のコードは `service/`, `handler/`, `repository/`, `domain/` というフラットな構造を持っていました。動くコードでしたが、以下の問題が蓄積していました。

### 1. pgx.Tx の漏洩

auth サービスのインターフェースには `pgx.Tx` 型のパラメータを持つメソッドが約15個ありました。

```go
// 問題のある例（移行前）
type SellerUserStore interface {
    Create(ctx context.Context, su *domain.SellerUser) error
    CreateTx(ctx context.Context, tx pgx.Tx, su *domain.SellerUser) error  // pgx.Tx が漏洩
    UpdateRoleTx(ctx context.Context, tx pgx.Tx, ...) error                // 同上
}
```

これは「`service/` がインフラ（pgx）を知っている」ことを意味します。  
サービス層のインターフェースにDBドライバーの型が現れるのは依存方向の逆転です。

### 2. リポジトリ型のサービス層への漏洩

`service/` が `repository/` パッケージの具体型に依存していました。

```go
// 問題のある例（移行前）— order_service.go
func (s *OrderService) CreateCheckout(ctx context.Context, ..., items []repository.CheckoutBatchItem) error
//                                                                   ^^^^^^^^^^^^^^^^^^^^^^^^^^^
//                                                                   リポジトリ型がサービス境界を越えている
```

`repository.ProductFilter`, `repository.CheckoutBatchItem`, `repository.ErrOrderNotPending` など、
本来ドメインに属するべき型がリポジトリパッケージに置かれていました。

### 3. HTTPクライアントがサービス層に混在

`cart/internal/service/catalog_client.go` というファイルが存在し、
HTTPクライアントの実装がサービスオーケストレーションコードと同じパッケージに置かれていました。  
「サービスロジック」と「インフラ実装」が区別できない状態でした。

### 4. ハンドラーが具体クラスに依存

```go
// 問題のある例（移行前）
type CartHandler struct {
    svc *service.CartService  // 具体型への依存
}
```

インターフェースへの依存ではないため、モックに差し替えることができません。

### 5. サービス層が HTTP 意味論を知っていた

```go
// 問題のある例（移行前）— cart_service.go
if quantity <= 0 {
    return nil, apperrors.BadRequest("quantity must be positive")
    //           ^^^^^^^^^^^^^^^^^^^^ HTTP 400 がビジネスロジックに現れている
}
```

`apperrors.BadRequest()` はHTTPステータスコード400を意味します。  
ビジネスロジックがHTTPに結合すると、同じロジックをgRPCやCLIから呼ぶときに支障が出ます。

### 6. 型なしイベントペイロード

```go
// 問題のある例（移行前）
pubsub.PublishEvent(ctx, s.publisher, tenantID, "order.created", "order-events", map[string]any{
    "order_id":  order.ID.String(),
    "selller_id": order.SellerID.String(),  // タイポがあってもコンパイルエラーにならない
})
```

`map[string]any` はフィールド名のタイポをコンパイル時に検出できず、
イベントのスキーマがコードから読み取れません。

---

## 移行の全体方針

問題を4つのフェーズに分けて解決しました。**順序に意味があります**。

```
Phase 0: 型の所有権を整理（後続フェーズの土台）
Phase 1: トランザクション抽象化（最も深いインフラ結合を除去）
Phase 2: ポート抽出 + HTTP クライアント移動（依存方向を明示）
Phase 3: ディレクトリリネーム（名前が設計を反映するように）
Phase 4: ドメイン充実（任意・優先度低）
```

各フェーズの後に `go build ./... && go test ./...` を実行して壊れていないことを確認しました。  
各サービスは独立したGoモジュールなので、1サービスずつ安全に進められました。

---

## Phase 0: ドメインエラー分離 + 型の所有権整理

**目的**: 後続フェーズで触るファイル全てに影響する基盤を先に整える。

### 0a. ドメインエラーの作成

`apperrors.BadRequest()` をサービス層から追い出し、`domain/errors.go` に移しました。

```go
// domain/errors.go（新規作成）
var (
    ErrEmptyCart       = errors.New("cart is empty")
    ErrSKUNotInCart    = errors.New("sku not in cart")
    ErrInvalidQuantity = errors.New("quantity must be positive")
    ErrOrderNotFound   = errors.New("order not found")
    ErrOrderNotPending = errors.New("order is not in pending status")
    // ...
)
```

```go
// app/cart_service.go（移行後）
if quantity <= 0 {
    return nil, domain.ErrInvalidQuantity  // HTTP と無関係
}
```

### 0b. 漏洩していた型をドメインへ移動

| 移行前の場所                    | 移行後                      | サービス  |
| ------------------------------- | --------------------------- | --------- |
| `repository.ProductFilter`      | `domain.ProductFilter`      | catalog   |
| `repository.CheckoutBatchItem`  | `domain.CheckoutBatchItem`  | order     |
| `repository.PurchaseSKURecord`  | `domain.PurchaseSKURecord`  | order     |
| `repository.ErrOrderNotPending` | `domain.ErrOrderNotPending` | order     |
| `repository.CancellationLine`   | `domain.CancellationLine`   | inventory |

### 0c. ハンドラーにエラーマッピング追加

```go
// adapter/http/error_mapper.go（各ハンドラーパッケージに追加）
func mapError(err error) *apperrors.AppError {
    switch {
    case errors.Is(err, domain.ErrOrderNotFound):
        return apperrors.NotFound(err.Error())
    case errors.Is(err, domain.ErrInvalidQuantity):
        return apperrors.BadRequest(err.Error())
    default:
        return apperrors.Internal("internal error", err)
    }
}
```

**なぜ Phase 0 が最初か**: ドメインエラーと型の整理を先にやっておかないと、  
Phase 1〜3 でファイルを移動するたびに import パスと型参照を同時に直す二重作業になります。

---

## Phase 1: トランザクション抽象化

**目的**: `pgx.Tx` をインターフェースから完全に除去する。

auth サービスに集中していた `pgx.Tx` 漏洩を解決しました。これが最もリスクの高い変更でした。

### 変更内容

**`pkg/database/` への追加**:

```go
// pkg/database/tx_context.go（新規追加）
type txKey struct{}

func WithTx(ctx context.Context, tx pgx.Tx) context.Context {
    return context.WithValue(ctx, txKey{}, tx)
}

func TxFromContext(ctx context.Context) pgx.Tx {
    tx, _ := ctx.Value(txKey{}).(pgx.Tx)
    return tx
}

func QueryerFromContext(ctx context.Context, pool *pgxpool.Pool) pgxutil.Queryer {
    if tx := TxFromContext(ctx); tx != nil {
        return tx
    }
    return pool
}
```

**`TxRunner` インターフェースの変更**:

```go
// 移行前
type TxRunner interface {
    RunTenantTx(ctx context.Context, tenantID uuid.UUID, fn func(tx pgx.Tx) error) error
}

// 移行後
type TxRunner interface {
    RunTenantTx(ctx context.Context, tenantID uuid.UUID, fn func(ctx context.Context) error) error
}
```

**リポジトリメソッドの統合**:

```go
// 移行前 — Tx有りと無しの2系統
func (r *SellerUserRepo) Create(ctx context.Context, su *domain.SellerUser) error { ... }
func (r *SellerUserRepo) CreateTx(ctx context.Context, tx pgx.Tx, su *domain.SellerUser) error { ... }

// 移行後 — context から取り出すので1系統で済む
func (r *SellerUserRepo) Create(ctx context.Context, su *domain.SellerUser) error {
    q := database.QueryerFromContext(ctx, r.pool)  // tx があれば tx、なければ pool
    return q.QueryRow(ctx, sql, ...).Scan(...)
}
```

**影響範囲**: auth サービス（重度、約15箇所）、order サービスのキャンセル処理（中度）。  
他のサービスには `Tx` 系メソッドがなかったため影響なし。

---

## Phase 2: ポート抽出 + HTTPクライアント移動

**目的**: 依存方向を明示的にする。インターフェースを使う側のパッケージに置く（Go イディオム）。

### 2a. `port/` パッケージの新設

各サービスに `internal/port/store.go` と `internal/port/service.go` を作成しました。

```
port/store.go   — driven ports（app 層が使うインフラのインターフェース）
port/service.go — driving port（ハンドラーが使うユースケースインターフェース）
```

インターフェースを使う側（`app/`）のパッケージに近い場所に置くのが Go のイディオムです。  
実装側（`adapter/`）ではなく、依存する側が「何が欲しいか」を宣言します。

### 2b. HTTPクライアントの移動

```
移行前: cart/internal/service/catalog_client.go   （サービスロジックと混在）
移行後: cart/internal/adapter/httpclient/catalog_client.go  （インフラアダプターとして分離）
```

移動と同時に、クライアントが `port.SKULookupClient` インターフェースを実装するよう変更しました。  
`app/cart_service.go` はインターフェース経由でしかクライアントを触りません。

### 2c. ハンドラーのインターフェース化

```go
// 移行前
type CartHandler struct { svc *service.CartService }

// 移行後
type CartHandler struct { svc port.CartUseCase }
```

**循環 import の回避**:  
`SKULookup` という型は `app/` でも `adapter/httpclient/` でも使われます。  
`domain/` に置くと「ドメインがHTTPレスポンス形状を知る」になり不適切。  
`adapter/httpclient/` に置くと `app/` が adapter を import することになり依存違反。  
→ **`port/store.go` に置く**（両者とも port は参照してよい）。

---

## Phase 3: ディレクトリリネーム

**目的**: 名前が設計を反映するようにする。純粋にコスメティックな変更。

| 移行前                 | 移行後                       | 意図                                                    |
| ---------------------- | ---------------------------- | ------------------------------------------------------- |
| `internal/service/`    | `internal/app/`              | 「サービス」は曖昧。アプリケーション層であることを明示  |
| `internal/handler/`    | `internal/adapter/http/`     | HTTPアダプターの1つ。技術が名前に現れる                 |
| `internal/repository/` | `internal/adapter/postgres/` | PostgreSQL アダプター。別DBに変えたとき名前が正直になる |
| `internal/grpcserver/` | `internal/adapter/grpc/`     | gRPC アダプター                                         |
| `internal/redis/`      | `internal/adapter/redis/`    | Redis アダプター                                        |
| `internal/stripe/`     | `internal/adapter/stripe/`   | Stripe アダプター                                       |
| `internal/subscriber/` | `internal/adapter/pubsub/`   | Pub/Sub アダプター                                      |

**実施順序**: シンプルなサービスから複雑なサービスへ。

1. search → recommend → notification（Pub/Subのみ）
2. cart → inventory → inquiry（小規模）
3. catalog → order → auth（複雑）

**ハマりポイント**: `handler/` を `adapter/http/` にリネームするとき、  
パッケージ宣言を `package http` にすると標準ライブラリの `net/http` と名前衝突します。  
Go はディレクトリ名ではなく宣言された `package` 名を識別子として使うので、  
ディレクトリが `adapter/http/` でもパッケージ宣言は `package handler` のままで問題ありません。

---

## Phase 4: ドメイン層の充実

**目的**: ドメイン層を薄い構造体の集合から「振る舞いを持つモデル」に育てる。  
ただし「やれるからやる」ではなく、「やることでサービス層が薄くなる」場合のみ実施。

### 4a. Cart エンティティへの振る舞いメソッド追加

```go
// domain/cart.go（追加）
func (c *Cart) AddItem(item CartItem)
func (c *Cart) RemoveItem(skuID uuid.UUID)
func (c *Cart) SetItemQuantity(skuID uuid.UUID, quantity int) error
```

これにより `app/cart_service.go` の `AddItem`, `UpdateItemQuantity`, `RemoveItem` メソッドが、
「スライス操作コード」を直接持たなくなり、「ロード→ドメインメソッド呼び出し→保存」という
純粋なオーケストレーションになりました。

### 4b. Order のステータス遷移メソッド

```go
// domain/order.go（追加）
func (o *Order) CanBeCancelled() bool {
    switch o.Status {
    case StatusPending, StatusPaid, StatusProcessing:
        return true
    }
    return false
}
```

`cancellation/service.go` の `canOrderBeCancelled(status string)` をこのドメインメソッドへの  
ラッパーに変更。キャンセル可否のロジックのソース・オブ・トゥルースがドメインに一本化されました。

```go
// cancellation/domain.go（移行後）
func canOrderBeCancelled(status string) bool {
    return (&domain.Order{Status: status}).CanBeCancelled()
}
```

テストコードがこの関数を直接テストしているため、削除はせずラッパー化で対応しました。  
これにより `cancellation/` のテストを壊さずにドメインを唯一の真実の源にできました。

### 4c. 型付きイベント構造体

全サービスで `map[string]any` を typed struct に置き換えました。

```go
// 移行前
pubsub.PublishEvent(ctx, s.publisher, tenantID, "order.paid", "order-events", map[string]any{
    "order_id":  order.ID.String(),
    "seller_id": order.SellerID.String(),
    // ...
})

// 移行後
pubsub.PublishEvent(ctx, s.publisher, tenantID, domain.EventTypeOrderPaid, "order-events",
    domain.OrderPaidEvent{
        OrderID:  order.ID.String(),
        SellerID: order.SellerID.String(),
        // ...
    })
```

イベントタイプ文字列も定数化（`domain.EventTypeOrderPaid`）したことで、  
文字列ベースの switch 文で定数以外の値を使ってしまうミスを防げます。

---

## Phase 5: auth モノリスの分解と subscription サービス抽出

**目的**: 8,000 LOC 超に肥大化していた auth サービスを責務ごとに分解し、境界の明確になった subscription 機能を独立サービスに切り出す。

### 背景

auth は Phase 3 完了時点で最大のサービスになっていました。`AuthService` という god struct が 9 ストアに依存し、identity（tenant / seller / buyer）・RBAC・API Token・subscription（seller tier / buyer free-shipping）を一手に抱えていました。`docs/architecture.md` が定義する auth のスコープは「テナント管理、セラー登録・管理、ユーザー認証連携」であり、subscription は本来オフスコープです。さらに order サービスが `httpclient.BuyerSubscriptionClient` 経由で auth の `/buyer-subscriptions/...` を呼んでおり、論理境界はすでに露出していました。

### 5a. Phase 1 — auth 内部の責務分離（破壊的変更なし）

デプロイ単位は auth のままで、`internal/app` を責務別パッケージに切り分けました。

```
app/identity.go     — Tenant, Seller, SellerUser, Buyer プロフィール
app/rbac.go         — 役割ルックアップ、監査ログ、platform admin 管理
app/subscription.go — seller/buyer プラン + 購読（後で切り出す単位）
app/credential.go   — API Token lifecycle
```

`AuthService` は 4 つの Service をポインタ埋め込みする facade に縮退させ、既存の呼び出し側を壊さないようにしました。`port/service.go` も責務別に分割し、ハンドラは対応する narrow interface（`IdentityUseCase`, `RBACUseCase`, `SubscriptionUseCase`, `CredentialUseCase`）に依存する形に変更しました。

**原則**:

- パブリック HTTP API は無変更（gateway・order・frontend への影響ゼロ）
- DB スキーマ無変更（`auth_svc.*` テーブルはそのまま）

### 5b. Phase 2 — subscription サービスの gRPC 抽出

分離済みの `app/subscription.go` を独立サービス `backend/services/subscription` として切り出し、gRPC + HTTP のデュアル公開にしました。

**Proto 定義** (`backend/proto/subscription/v1/subscription.proto`):

- `service SubscriptionService` に 12 RPC（`ListSellerPlans`, `GetSellerSubscription`, `SubscribeSeller`, `ListBuyerPlans`, `GetBuyerSubscription`, `SubscribeBuyer` ほか）
- 既存の `buyerSubscriptionResponse`（order 側）の互換フィールド（`status`, `plan_slug`, `features.free_shipping`）を過不足なくカバー

**ポート割り当て**: HTTP 8089 / gRPC 50058（既存の採番規則の次を取る）。

**DB 戦略**: 新スキーマ `subscription_svc` を作成し、4 テーブル（`subscription_plans`, `seller_subscriptions`, `buyer_plans`, `buyer_subscriptions`）を物理移管。跨サービスの RDB FK は張らず、`auth_svc.sellers.subscription_id` は論理 FK に格下げ。マイグレーション 000019 がデータコピーと旧テーブル DROP を一括で実行します。

**呼び出し側の変更**:

- **order**: `httpclient.BuyerSubscriptionClient` を削除し、`grpcclient.BuyerSubscriptionClient` で置換。`port.BuyerSubscriptionChecker` インターフェースは維持して実装差し替えのみ（DI 境界は変えない）。
- **gateway**: `/buyer-subscriptions/*`, `/seller-subscription-plans/*` の upstream を auth → subscription に切替。REST パスは維持して外部 API 互換を保つ。
- **auth**: subscription 関連コード（handler / port / repo / app / domain）を完全撤去。

### 5c. レビュー指摘で後から入れた 2 つの修正

最初の切り出しでは、gateway の認可に依存しすぎた boundary と、RLS × materialized view の設計ミスが残りました。長期運用に直結するため、000019 migration への統合と `pkg/middleware` の追加で補正しました（当初 000020 で追加した BYPASSRLS ロール + SECURITY DEFINER 関数は、000019 自体が FORCE RLS 下の `CREATE MATERIALIZED VIEW` で失敗して 000020 に到達しない順序バグがあったため、最終的に 000019 本体へ吸収しました）。

#### 修正 1 — FORCE RLS と materialized view の衝突

`catalog_svc.seller_plan_boost` は `subscription_svc.seller_subscriptions` / `subscription_plans` を join する projection ですが、000019 でこれらに `FORCE ROW LEVEL SECURITY` を付けた結果、アプリロールからの `REFRESH MATERIALIZED VIEW CONCURRENTLY` が `current_setting('app.current_tenant_id')` で弾かれるようになりました。migration 時の `CREATE MATERIALIZED VIEW` も `ecmarket` ロールで走ると同じ問題を踏みます。

**解決策**: 000019 内で `BYPASSRLS` 権限を持つ `ecmarket_rls_bypass` ロールを作り、マテビューの所有者をそのロールに付け替え、`catalog_svc.refresh_seller_plan_boost()` という `SECURITY DEFINER` 関数で refresh をラップしました。アプリは `REFRESH` ではなく `SELECT catalog_svc.refresh_seller_plan_boost()` を呼びます。FORCE RLS 有効化 → bypass ロール作成 → MV 作成（`SET LOCAL ROLE` 下）の順序を同一 migration に畳み込むことで、中間状態で migration が失敗する窓を潰しました。

```sql
-- 000019_move_subscriptions_to_subscription_svc.up.sql の要旨
CREATE ROLE ecmarket_rls_bypass BYPASSRLS;
GRANT ecmarket_rls_bypass TO ecmarket;  -- SET LOCAL ROLE を可能にする

DO $$ BEGIN
    SET LOCAL ROLE ecmarket_rls_bypass;
    DROP MATERIALIZED VIEW IF EXISTS catalog_svc.seller_plan_boost;
    CREATE MATERIALIZED VIEW catalog_svc.seller_plan_boost AS ... ;
END $$;

CREATE FUNCTION catalog_svc.refresh_seller_plan_boost()
RETURNS void LANGUAGE sql SECURITY DEFINER SET search_path = pg_catalog
AS $$ REFRESH MATERIALIZED VIEW CONCURRENTLY catalog_svc.seller_plan_boost; $$;

GRANT EXECUTE ON FUNCTION catalog_svc.refresh_seller_plan_boost() TO ecmarket;
```

**教訓**: マルチテナント RLS と全テナント横断のマテビューは同居できません。単一ロールで migration もアプリも回す構成では、マテビュー所有者を RLS をバイパスする別ロールに分離し、refresh は `SECURITY DEFINER` 経由で呼び出し元に権限だけ渡すのが正解です。`SET row_security = off` はスーパーユーザーしか使えないので、単一ロール運用では `SET LOCAL ROLE` + `BYPASSRLS` の組み合わせの方が汎用的です。

#### 修正 2 — 各サービス自身の認証境界

subscription サービスは最初、`InternalContext` ミドルウェア（ヘッダ `X-Tenant-ID` を読むだけ）に依存しており、gRPC サーバーには何の認可もありませんでした。しかも docker-compose でホストポート 8089/50058 を publish していたため、直接叩いて `X-Tenant-ID` を偽装すれば plan 作成や subscribe の mutation を通せる状態でした。

**解決策**:

- **HTTP**: 全ての非ヘルスエンドポイントに `X-Internal-Token` を要求するミドルウェア。環境変数は per-service 命名（`SUBSCRIPTION_INTERNAL_TOKEN`, `AUTH_INTERNAL_TOKEN`）とし、サービスごとに別シークレットを持たせる。
- **gRPC**: `pkg/middleware/grpc_internal_token.go` に再利用可能な `UnaryInternalTokenInterceptor` を新設。`x-internal-token` metadata を検証し、`/grpc.health.*` と `/grpc.reflection.*` はスキップ、空シークレット時は fail-closed。
- **gateway / order**: gateway は `ServiceClient.WithHeader` で、order の gRPC クライアントは `grpc.WithPerRPCCredentials` + 小さな creds struct でトークンを付与。
- **docker-compose**: `ports:` を `expose:` に変更し、ホストから直接到達できないようにした。

**教訓**: 「gateway で認可する」は必要条件であって十分条件ではありません。新サービスを立てたら、そのサービスの HTTP/gRPC 入口にも **サービスごとの** shared secret を要求する境界を置くのがデフォルトです。ローカル compose でも host port publish を避けることで「開発時は楽だが本番で想定外に到達できる」という事故を防げます。

#### 修正 3 — ホットパスのタイムアウトとログ

order サービスの `HasFreeShipping` は checkout hot path にいます。subscription サービスが遅延・停止したときに order 全体が引きずられるのを避けるため、以下を入れました。

- `context.WithTimeout(ctx, 500 * time.Millisecond)` を call-site で必ずラップ
- 失敗時は `slog.Warn("buyer subscription lookup failed", "error", err, "duration_ms", ...)` で deadline exceeded と hard RPC error を区別できるように記録
- 呼び出し側は error を受けても「送料不明 → 標準送料を請求」という safe default に倒す

**教訓**: gRPC 化自体が信頼性を上げるわけではなく、「タイムアウト + safe default + 観測ログ」の 3 点セットが揃って初めてホットパスの依存として健全になります。

### 5d. Phase 5 全体で得た教訓

- **god struct の解体は facade 経由で段階的に**: Phase 1 で内部分離だけ先にやることで、Phase 2 のサービス境界切り出しは「既に分かれているものを物理分離するだけ」になり、レビューと動作確認が 1 軸ずつ進められました。
- **論理 FK で跨サービス参照を許す**: subscription 側は seller/buyer の存在確認をせず、ID 整合はアプリ層（gateway の認可）で保証する方針。これにより subscription サービスが auth に逆依存することを避けられました。
- **REST パスを変えずに upstream だけ差し替える**: 外部 API 互換を保ったまま内部構造を変えられるのは、gateway が proxy 役を担っているからこそのメリット。サービス境界を動かすリファクタリングは、gateway ルーティングを先に書き換え→新サービス起動→旧サービスを削除、の順で段階実行できます。
- **DB スキーマ移管は「コピー → 参照先切替 → 旧 DROP」を一度のマイグレーションに詰め込める**: 000019 はこの 3 ステップを atomic に実行しています。中間状態（新旧両スキーマが並存する）を露出させないほうが、むしろ運用が楽になるケースがありました。
- **proto は RPC ごとに Request/Response を別メッセージにする**: 同じレスポンス型を複数 RPC で再利用すると buf lint の STANDARD ルールで弾かれます。最初から 1:1 で切っておくと後から楽。
- **migration は単独で valid な状態まで畳み込む**: 「不足分は次の migration で直す」という分割は、前の migration が先に失敗すると後続に到達しないため成立しません。FORCE RLS 有効化と BYPASSRLS ロール導入のような、相互依存する DDL 変更は同一 migration に入れるのが安全です。down 側も対称に作り、中間状態（RLS 有効だが MV の所有者がまだ app role）を露出させないこと。

---

## 設計上の判断メモ

### なぜ「フェーズ分け」したか

一度に全部やると差分が巨大になりレビューが不可能です。  
特に Phase 0 の型整理を先にやることで、後続フェーズの diff を「構造変更のみ」に絞れました。  
「型を直す変更」と「ファイルを移動する変更」が混ざると何が起きているか追えなくなります。

### なぜ `port/` に DTO を置くか（循環 import の回避）

Go のルール: パッケージ A が B を import し、B が A を import することはできません。  
`httpclient.CatalogClient` は `port.SKULookupClient` を実装し、`app.CartService` は `port.SKULookupClient` を使います。  
もし `SKULookup` 型が `httpclient/` にあると、`app/` が `adapter/httpclient/` を import することになり依存方向違反です。  
もし `domain/` にあると、`domain/` がHTTPレスポンス形状（インフラ詳細）を持つことになり不適切です。  
→ `port/` は `app/` も `adapter/` も import できるので、ここが唯一の正解です。

### なぜ `package handler` をディレクトリが `adapter/http/` でも使うか

Go では `import "path/to/adapter/http"` と書いたとき、参照名は `package` 宣言の名前になります。  
ディレクトリ名を `http` にしてパッケージ宣言も `http` にすると、  
`main.go` 内で `http.NewCartHandler(...)` と `http.ListenAndServe(...)` が衝突します。  
ディレクトリ名（URL的な意味）とパッケージ名（Go的な識別子）は別物として扱えます。

### なぜ `canOrderBeCancelled` をそのまま残したか（削除しなかったか）

`cancellation/service_test.go` がこの関数をパッケージ内部のホワイトボックステストとして直接呼んでいます。  
テストを変更せずに済む最小変更として「ラッパーに変える」を選びました。  
テストをドメインメソッドに向け直すリファクタリングは次の機会で十分です。

### recommend・search サービスが初期移行から外れた理由

recommend と search は他のサービスと構造が異なるため、Phase 0 を別コミットで実施しました。

**engine パターン**:  
これらのサービスは `adapter/postgres/` を持ちません。その代わり `internal/engine/` というパッケージに  
バックエンドを抽象化するインターフェースを置き、PostgreSQL 全文検索・Vertex AI などを実装として差し替えられる構造になっています。

```
recommend/
  internal/
    engine/
      engine.go      # RecommendEngine インターフェース
      postgres.go    # PostgreSQL マテリアライズドビュー実装
      vertexai.go    # Vertex AI 実装
    repository/
      view_refresher.go  # マテリアライズドビュー更新（app層のインターフェース経由で注入）
```

`engine.go` のインターフェースは他サービスの `port/store.go` に相当する役割を果たしますが、  
「検索・推薦エンジン」という独自の概念なのでパッケージ名に技術名を使っています。

**Phase 0 補足内容**:

- `domain/errors.go` 追加（recommend: 5種、search: 1種）
- `app/` 層の `apperrors.BadRequest()` をドメインエラーに置き換え
- `adapter/http/errors.go` 追加（mapError 関数）
- `adapter/http/*_handler.go` でサービスエラーを `mapError(err)` 経由にする

---

### バッチ処理について

現時点でバッチジョブはありません。`recommend.RefreshPopularProducts()` が  
定期実行される唯一のバックグラウンドタスクですが、これは `time.Ticker` で動く軽量なものです。  
将来バッチが必要になった場合の選択肢は `knowledge/hexagonal-architecture.md` に記載しています。

### notification サービスが特殊な理由

notification サービスは HTTP ハンドラーを持たず、Pub/Sub のサブスクライバーのみで構成されます。  
`internal/service/` が存在しなかったため Phase 3 の `service/ → app/` リネームは対象外でした。  
「サービス = HTTP エンドポイントを持つもの」ではなく、「サービス = 1つのドメイン責務を持つプロセス」  
という観点では notification サービスは完全に正当な設計です。
