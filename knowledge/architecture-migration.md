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

| 移行前の場所 | 移行後 | サービス |
|------------|--------|----------|
| `repository.ProductFilter` | `domain.ProductFilter` | catalog |
| `repository.CheckoutBatchItem` | `domain.CheckoutBatchItem` | order |
| `repository.PurchaseSKURecord` | `domain.PurchaseSKURecord` | order |
| `repository.ErrOrderNotPending` | `domain.ErrOrderNotPending` | order |
| `repository.CancellationLine` | `domain.CancellationLine` | inventory |

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

| 移行前 | 移行後 | 意図 |
|--------|--------|------|
| `internal/service/` | `internal/app/` | 「サービス」は曖昧。アプリケーション層であることを明示 |
| `internal/handler/` | `internal/adapter/http/` | HTTPアダプターの1つ。技術が名前に現れる |
| `internal/repository/` | `internal/adapter/postgres/` | PostgreSQL アダプター。別DBに変えたとき名前が正直になる |
| `internal/grpcserver/` | `internal/adapter/grpc/` | gRPC アダプター |
| `internal/redis/` | `internal/adapter/redis/` | Redis アダプター |
| `internal/stripe/` | `internal/adapter/stripe/` | Stripe アダプター |
| `internal/subscriber/` | `internal/adapter/pubsub/` | Pub/Sub アダプター |

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
