# EC Marketplace Platform

マルチテナント対応のマーケットプレイス EC プラットフォームです。複数のテナント（マーケットプレイス運営者）が、それぞれ独立したセラー・バイヤーエコシステムを構築できます。

## アーキテクチャ概要

```
                         ┌──────────────────────────────────────────────┐
                         │              Frontend (Next.js)              │
                         │  ┌──────────┐ ┌──────────┐ ┌─────────────┐  │
                         │  │  Buyer   │ │  Seller  │ │   Admin     │  │
                         │  │  App     │ │  App     │ │   App       │  │
                         │  └────┬─────┘ └────┬─────┘ └──────┬──────┘  │
                         └───────┼────────────┼──────────────┼─────────┘
                                 │            │              │
                              ┌──▼────────────▼──────────────▼──┐
                              │     API Gateway (Go / :8080)     │
                              │   認証検証 ・ テナント解決 ・ルーティング │
                              └──┬────┬────┬────┬────┬────┬────┬┘
         ┌───────────────────────┼────┼────┼────┼────┼────┼────┘
         │          ┌────────────┘    │    │    │    │    │
         ▼          ▼                 ▼    ▼    ▼    ▼    ▼
    ┌─────────┐┌─────────┐   ┌───────┐┌───────┐┌───────┐┌────────────┐
    │  Auth   ││ Catalog │   │Inven- ││ Order ││Search ││ Recommend  │
    │ :8081   ││ :8082   │   │tory   ││ :8084 ││:8085  ││   :8086    │
    └────┬────┘└────┬────┘   │:8083  │└───┬───┘└───┬───┘└────────────┘
         │          │        └───┬───┘    │        │
         │          │            │        │        │    ┌──────────────┐
         └────┬─────┴────────┬───┘        │        │    │ Notification │
              │              │            │        │    │   :8087      │
              ▼              ▼            ▼        ▼    └──────┬───────┘
    ┌─────────────────┐  ┌────────┐  ┌────────┐              │
    │  PostgreSQL 16  │  │ Redis  │  │Pub/Sub │◄─────────────┘
    │  (RLS tenant    │  │  7     │  │        │
    │   isolation)    │  └────────┘  └────────┘
    └─────────────────┘

         ┌─────────┐  ┌──────────┐  ┌──────────┐
         │  Cart   │  │ Inquiry  │  │ Review   │
         │  :8088  │  │  :8090   │  │  :8091   │
         └─────────┘  └──────────┘  └──────────┘
```

## 技術スタック

| カテゴリ                     | 技術                       | バージョン / 備考                 |
| ---------------------------- | -------------------------- | --------------------------------- |
| バックエンド                 | Go                         | 1.25                              |
| フロントエンド               | Next.js (App Router)       | latest                            |
| モノレポ管理 (Frontend)      | Turborepo + pnpm           | pnpm 10.x                         |
| モノレポ管理 (Backend)       | Go Workspaces              | go.work                           |
| データベース                 | PostgreSQL                 | 16 (RLS によるマルチテナント分離) |
| キャッシュ                   | Redis                      | 7                                 |
| メッセージング               | Cloud Pub/Sub              | GCP                               |
| 認証                         | Auth0                      | JWT + Organizations               |
| 決済                         | Stripe Connect             | マーケットプレイス決済            |
| 検索                         | Vertex AI Search           | GCP                               |
| コンテナオーケストレーション | GKE                        | Google Kubernetes Engine          |
| CI/CD                        | GitHub Actions + ArgoCD    | GitOps                            |
| API 定義                     | Protocol Buffers / OpenAPI | buf + openapi-generator           |

## クイックスタート

### 前提条件

| ツール                  | バージョン | インストール                                                                          |
| ----------------------- | ---------- | ------------------------------------------------------------------------------------- |
| Go                      | 1.25+      | https://go.dev/dl/                                                                    |
| Node.js                 | 20+        | https://nodejs.org/                                                                   |
| pnpm                    | 10+        | `corepack enable`                                                                     |
| Docker / Docker Compose | latest     | https://docs.docker.com/get-docker/                                                   |
| golang-migrate          | latest     | `go install -tags 'postgres' github.com/golang-migrate/migrate/v4/cmd/migrate@latest` |
| air (ホットリロード)    | latest     | `go install github.com/air-verse/air@latest`                                          |
| golangci-lint           | latest     | https://golangci-lint.run/welcome/install/                                            |
| buf (Proto)             | latest     | https://buf.build/docs/installation                                                   |

### セットアップ手順

```bash
# 1. リポジトリをクローン
git clone https://github.com/Riku-KANO/ec-test.git
cd ec-test

# 2. ローカル依存サービスを起動 (PostgreSQL, Redis, Pub/Sub Emulator)
make deps-up

# 3. DB マイグレーションを実行
make migrate

# 4. 開発用シードデータを投入
make seed

# 5. 各サービスを起動 (別ターミナルで)
make dev-gateway      # API Gateway   → :8080
make dev-auth         # Auth Service  → :8081
make dev-catalog      # Catalog       → :8082
make dev-inventory    # Inventory     → :8083
make dev-order        # Order         → :8084
make dev-cart         # Cart          → :8088
make dev-inquiry      # Inquiry       → :8090
make dev-review       # Review        → :8091

# 6. フロントエンドを起動 (別ターミナルで)
pnpm install
make dev-buyer        # Buyer App     → :3000
make dev-seller       # Seller App    → :3001
make dev-admin        # Admin App     → :3002
```

## プロジェクト構成

```
ec-test/
├── frontend/                # フロントエンド (TypeScript / Next.js)
│   ├── apps/                #   Next.js アプリケーション群
│   │   ├── buyer/           #     購入者向け画面 (:3000)
│   │   ├── seller/          #     セラー管理画面 (:3001)
│   │   ├── admin/           #     プラットフォーム管理画面 (:3002)
│   │   └── storybook/       #     Storybook コンポーネントカタログ
│   └── packages/            #   共有 Node.js パッケージ
│       ├── eslint-config/   #     ESLint 共通設定
│       ├── tsconfig/        #     TypeScript 共通設定
│       ├── vitest-config/   #     Vitest 共通設定
│       └── i18n/            #     多言語対応 (next-intl)
├── backend/                 # バックエンド (Go)
│   ├── services/            #   Go マイクロサービス群
│   │   ├── gateway/         #     API Gateway (認証・ルーティング)
│   │   ├── auth/            #     テナント・セラー・ユーザー管理
│   │   ├── catalog/         #     商品・SKU・カテゴリ管理
│   │   ├── inventory/       #     在庫管理・在庫移動
│   │   ├── order/           #     注文・決済・コミッション
│   │   ├── search/          #     商品検索 (Vertex AI Search)
│   │   ├── recommend/       #     レコメンデーション
│   │   ├── notification/    #     通知 (メール・プッシュ)
│   │   ├── cart/            #     カート・チェックアウトオーケストレーション (Redis)
│   │   ├── inquiry/         #     買い手→売り手お問い合わせ (購入済み SKU 単位)
│   │   └── review/          #     商品レビュー・評価・セラー返信
│   ├── pkg/                 #   Go 共有パッケージ
│   │   ├── tenant/          #     テナントコンテキスト管理
│   │   ├── database/        #     DB 接続プール・RLS 設定
│   │   ├── middleware/      #     共通ミドルウェア (ログ等)
│   │   ├── errors/          #     共通エラー型
│   │   ├── httputil/        #     HTTP レスポンスヘルパー
│   │   ├── pagination/      #     ページネーション
│   │   └── pubsub/          #     Pub/Sub クライアント
│   ├── proto/               #   Protocol Buffers 定義
│   └── gen/                 #   生成コード (gRPC スタブ)
├── infra/                   # インフラ・運用
│   ├── db/                  #   DB マイグレーション / シード
│   ├── deploy/              #   Kubernetes / ArgoCD マニフェスト
│   │   ├── base/            #     Kustomize base
│   │   ├── overlays/        #     環境別オーバーレイ
│   │   └── argocd/          #     ArgoCD Application 定義
│   ├── docker/              #   Docker Compose 設定
│   └── scripts/             #   ユーティリティスクリプト
├── docs/                    # ドキュメント
├── .github/                 # GitHub Actions ワークフロー
├── go.work                  # Go Workspace 設定
├── package.json             # pnpm / Turborepo ルート設定
├── pnpm-workspace.yaml      # pnpm ワークスペース定義
├── turbo.json               # Turborepo パイプライン設定
└── Makefile                 # 開発用コマンド
```

## Make コマンド一覧

| コマンド              | 説明                                                  |
| --------------------- | ----------------------------------------------------- |
| `make deps-up`        | ローカル依存サービス起動 (PostgreSQL, Redis, Pub/Sub) |
| `make deps-down`      | ローカル依存サービス停止                              |
| `make migrate`        | DB マイグレーション実行                               |
| `make migrate-down`   | 直前のマイグレーションをロールバック                  |
| `make migrate-create` | 新規マイグレーションファイル作成                      |
| `make seed`           | 開発用シードデータ投入                                |
| `make dev-<service>`  | 指定サービスをホットリロードで起動 (air)              |
| `make dev-buyer`      | Buyer Next.js アプリ起動                              |
| `make dev-seller`     | Seller Next.js アプリ起動                             |
| `make dev-admin`      | Admin Next.js アプリ起動                              |
| `make build-all`      | 全 Go サービスをビルド                                |
| `make lint-go`        | 全 Go コードの lint 実行                              |
| `make test-go`        | 全 Go コードのテスト実行                              |
| `make proto-gen`      | Proto ファイルからコード生成                          |
| `make openapi-gen`    | OpenAPI 仕様から API クライアント生成                 |

## 開発ワークフロー

### サービスのローカル実行

各 Go サービスは [air](https://github.com/air-verse/air) によるホットリロードで実行されます。ソースコードを変更すると自動でリビルド・リスタートします。

```bash
# 必要なサービスだけ起動すれば OK
make dev-gateway    # 必須: 全リクエストの入口
make dev-auth       # テナント・セラー管理が必要な場合
make dev-catalog    # 商品関連の開発時
```

### DB マイグレーション

```bash
# 新しいマイグレーション作成
make migrate-create
# → "Migration name:" プロンプトに名前を入力

# マイグレーション実行
make migrate

# 1 つロールバック
make migrate-down
```

### テスト

```bash
# 全サービス + pkg のテスト
make test-go

# 特定サービスのみ
cd backend/services/catalog && go test ./...
```

### Lint

```bash
make lint-go
```

## 環境変数

ローカル開発時のデフォルト DB 接続先:

```
DATABASE_URL=postgres://ecmarket:localdev@localhost:5432/ecmarket_dev?sslmode=disable
```

各サービスの環境変数は `backend/services/<service>/internal/config/` を参照してください。

## ドキュメント

- [コントリビューションガイド](docs/CONTRIBUTING.md) -- 開発規約・PR ルール・新機能追加手順
- [アーキテクチャ設計書](docs/architecture.md) -- システム設計・データモデル・通信パターン
- [決済・Stripe Connect 設計](docs/payment.md) -- マルチセラー決済 (Separate Charges and Transfers) の設計・webhook・payout
- [カート・チェックアウト設計](docs/cart-and-checkout.md) -- カートのデータモデル・API・チェックアウトオーケストレーション
