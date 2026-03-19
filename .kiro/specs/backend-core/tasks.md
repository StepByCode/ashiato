# backend-core Tasks

## 1. 基盤初期化

- `api/` の Go module を整備する
- Echo / OpenAPI / `pgx` / `sqlc` / OIDC / logging 依存を追加する
- 完了条件: `go test ./...` が実行可能な構成になる

## 2. API 契約

- `api/openapi/openapi.yaml` に全 endpoint と request/response schema を定義する
- `oapi-codegen` で server/types を生成する
- 完了条件: 生成コードがコンパイルされる

## 3. DB スキーマ

- 初期 migration の `up/down` を作る
- `year/month` と複数承認者モデルを schema に反映する
- 完了条件: 空 DB に migration を適用できる

## 4. sqlc

- query 定義を作る
- `sqlc generate` で DB access code を生成する
- 完了条件: usecase が利用する全 query が生成される

## 5. 認証・ミドルウェア

- `stub` / `oidc` verifier を実装する
- trace / access log / auth / bot token middleware を実装する
- 初回ログイン同期を実装する
- 完了条件: 認証済みリクエストで actor context を取得できる

## 6. ユースケース・ハンドラ

- `me`, `members`, `workflow-periods`
- `meetings`
- `tasks` と `task approvals`
- `announcements`
- `internal publish request`
- 完了条件: OpenAPI endpoint が全て実装される

## 7. 監査ログ

- 監査対象操作の before/after 記録を追加する
- 完了条件: 主要 write 操作で `audit_logs` にレコードが残る

## 8. テスト

- auth の `stub` / `oidc` テスト
- repository / usecase の PostgreSQL 統合テスト
- API シナリオテスト
- 完了条件: 主要 happy path と権限/競合エラーを検証できる

## 9. ドキュメント同期

- `api/README.md`
- `doc/04_permission-design.md`
- `doc/05_erd.md`
- `doc/08_logging.md`
- 完了条件: 実装に合わせた記述へ更新される
