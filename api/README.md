# api

`api/` は Ashiato のバックエンド API 実装です。
認証、認可、永続化、監査ログ、Bot 向け internal API を担います。

## 役割

- Firebase Authentication による ID トークン検証
- `organization_members` を正本にした RBAC
- `tasks`, `task_approvals`, `meetings`, `announcements` の永続化
- 監査ログ・構造化 access log の出力
- Bot 向け publish request API

## 実装構成

```text
api/
├── cmd/server/        # Echo サーバー起動
├── internal/
│   ├── auth/          # Firebase ID トークン検証と認証ミドルウェア
│   ├── config/        # 環境変数読み込み
│   ├── db/            # sqlc 生成コード
│   ├── domain/        # モデル・エラー
│   ├── handler/       # OpenAPI strict handler 実装
│   ├── logging/       # JSON access log
│   ├── middleware/    # request context
│   ├── repository/    # Firebase Realtime Database
│   └── usecase/       # 業務ロジック
├── migrations/        # SQL migration
├── openapi/           # OpenAPI 契約
├── sqlc/              # sqlc 設定と query
└── tests/             # 結合テスト補助
```

## 主要環境変数

| 変数 | 説明 |
| --- | --- |
| `FIREBASE_CREDENTIALS_JSON` | Firebase サービスアカウントの JSON（GCP 環境では ADC にフォールバック） |
| `FIREBASE_DATABASE_URL` | Firebase Realtime Database の URL |
| `DISCORD_WEBHOOK_URL` | Discord Webhook URL |
| `DEFAULT_ORG_NAME` | seed 組織名 |
| `DEFAULT_ORG_SLUG` | seed 組織 slug |
| `OWNER_EMAILS` | `OWNER` 扱いにする email の CSV |
| `PORT` | サーバーポート（デフォルト 8080） |

## 開発フロー

```bash
cd api
go test ./...
```

OpenAPI と `sqlc` の再生成:

```bash
cd api
$(go env GOPATH)/bin/oapi-codegen -generate types,strict-server,echo-server -package oapi -o internal/oapi/oapi.gen.go openapi/openapi.yaml
cd sqlc && $(go env GOPATH)/bin/sqlc generate
```
