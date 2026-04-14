# Buyer 認証セットアップ (Auth0)

Buyer アプリはエンドユーザー認証に Auth0 (Universal Login + Authorization
Code with PKCE) を利用します。セッションは `@auth0/nextjs-auth0` SDK が
管理する HttpOnly Cookie に保存され、Next.js から Go gateway への
`/api/gateway/*` プロキシ呼び出しで access token を Bearer ヘッダとして
転送します (BFF パターン)。

このドキュメントでは、Auth0 の初回セットアップ手順と、
`frontend/apps/buyer/.env.local` に必要な環境変数を説明します。

---

## 1. Auth0 アカウントの作成

1. <https://auth0.com/> でサインアップ (無料枠で OK)。
2. ダッシュボードからアプリケーションを作成します。リージョンは開発機に近いものを
   選んでください。ドメインは `dev-<random>.us.auth0.com` のようになります。

## 2. Regular Web Application の作成

`Applications → Applications → Create Application`

- Name: `EC Marketplace Buyer (dev)`
- Application Type: **Regular Web Applications**
  _(SPA ではなく、client secret を持つ confidential client が必要です)_

作成したアプリの `Settings` タブで以下を設定します。

| 項目                  | 値                                    |
| --------------------- | ------------------------------------- |
| Allowed Callback URLs | `http://localhost:3000/auth/callback` |
| Allowed Logout URLs   | `http://localhost:3000`               |
| Allowed Web Origins   | `http://localhost:3000`               |

**Domain / Client ID / Client Secret** を控えてください。後で
`.env.local` に貼り付けます。

## 3. API (audience) の作成

`Applications → APIs → Create API`

- Name: `EC Marketplace API`
- Identifier: `https://api.ecmarket.local`
  _(これはブラウザがアクセスする URL ではなく、JWT の `aud` クレームです)_
- Signing Algorithm: **RS256**

## 4. Connections

`Authentication → Database → Username-Password-Authentication` は
デフォルトで有効です。手動テストを楽にしたい場合は
`Authentication → Social` から `Google` を任意で有効化してください。

## 5. Buyer アプリの設定

```bash
cd frontend/apps/buyer
cp .env.local.example .env.local
```

`.env.local` を編集します。

```bash
AUTH0_SECRET=$(openssl rand -hex 32)
APP_BASE_URL=http://localhost:3000
AUTH0_DOMAIN=dev-xxxxxx.us.auth0.com
AUTH0_CLIENT_ID=<Auth0 アプリ設定から>
AUTH0_CLIENT_SECRET=<Auth0 アプリ設定から>
AUTH0_AUDIENCE=https://api.ecmarket.local
AUTH0_SCOPE=openid profile email offline_access

GATEWAY_URL=http://localhost:8080
NEXT_PUBLIC_API_URL=/api/gateway

AUTH_SERVICE_URL=http://localhost:8081
AUTH_INTERNAL_TOKEN=dev-internal-token
```

## 6. Gateway の設定

`infra/docker/docker-compose.yaml` は以下の変数をシェル環境から読み取る
設定になっています。`infra/docker/.env` を作成する (またはシェルで
export する) ことで設定します。

```bash
JWKS_URL=https://dev-xxxxxx.us.auth0.com/.well-known/jwks.json
JWT_ISSUER=https://dev-xxxxxx.us.auth0.com/
JWT_AUDIENCE=https://api.ecmarket.local
AUTH_INTERNAL_TOKEN=dev-internal-token
```

`JWT_ISSUER` の**末尾のスラッシュ**に注意してください。Auth0 は `iss` が
`/` で終わるトークンを発行しますが、gateway は完全一致で比較するため
一致しないと 401 になります。

## 7. 起動

```bash
make deps-up        # postgres + redis
make migrate        # auth_svc.buyers などを作成
make seed
docker compose -f infra/docker/docker-compose.yaml up -d gateway auth review catalog order
make dev-buyer      # Next.js を :3000 で起動
```

<http://localhost:3000> を開いて **ログイン** をクリックし、Auth0 で
サインアップ/ログインを完了させると、商品ページに戻りヘッダに
メールアドレスが表示されます。

---

## トラブルシューティング

**ログイン後に gateway から `401 invalid_token` が返る** — access token の
`iss` または `aud` が `docker-compose.yaml` の設定と一致していません。
<https://jwt.io/> でトークンをデコードして `iss` と `aud` をそのまま
gateway の環境変数にコピーし、`docker compose restart gateway` で
再起動してください。

**ログインは成功するが `auth_svc.buyers` が空** — BFF からの upsert 呼び出し
(`POST /internal/buyers/upsert`) が失敗しています。Next.js dev サーバーの
ログで `"buyer upsert failed"` を確認してください。よくある原因は
`AUTH_INTERNAL_TOKEN` の不一致、または
ホストから auth service コンテナに到達できないことです。

**レビュー投稿が `403 PURCHASE_REQUIRED` を返す** — これは想定通りの
動作です。review service は対象商品に対して完了済みの注文が存在する
ことを要求します。seed で注文を追加するか、cart → checkout フローを
一度通してから再度試してください。
