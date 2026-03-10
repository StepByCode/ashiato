# api

`api/` は **Ashiato のバックエンドAPI** を実装するディレクトリです。  
フロントエンドとBotから利用される共通の業務ロジックと認可判定を担当します。

## 役割

- 認証済みユーザー向けAPIの提供
- RBACを軸とした権限制御
- タスク/承認などコアデータの永続化
- 監査ログ・操作ログの出力

## 想定技術スタック

- Go
- Echo
- OpenAPI（契約駆動）
- PostgreSQL + `sqlc` + `pgx`
- OAuth2 / OIDC + JWT

## 推奨ディレクトリ構成（着手時の目安）

```text
api/
├── cmd/             # エントリポイント
├── internal/
│   ├── domain/      # エンティティ・ビジネスルール
│   ├── usecase/     # ユースケース
│   ├── repository/  # DBアクセス抽象
│   ├── handler/     # HTTPハンドラ
│   ├── middleware/  # 認証・認可・共通処理
│   └── config/      # 設定読み込み
├── migrations/      # DBマイグレーション
└── tests/           # テスト
```

## 実装方針

- API仕様を先に確定し、フロントとインターフェースを固定する
- 認可の最終判定は必ずサーバー側で実施する
- 監査対象操作（権限変更・削除・設定変更）は必ず監査ログを残す

## 関連ドキュメント

- `../doc/02_tech-stack.md`
- `../doc/04_permission-design.md`
- `../doc/05_erd.md`
- `../doc/08_logging.md`
