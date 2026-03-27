# backend-core Requirements

## 概要

`backend-core` は Ashiato の API 正本として、P0 の進捗管理と認可を成立させつつ、P1 の定例情報と広報下書き・投稿指示までを扱うバックエンド機能である。

## 対象ユーザー

- StepByCode 運営メンバー
- Discord Bot サービス

## 対象ユースケース

1. 認証済みユーザーが初回アクセス時に自動でユーザー登録される
2. 組織メンバーが月次の定例情報を登録・参照できる
3. 組織メンバーが月次の作成タスクを一覧・作成・更新・削除できる
4. 指定された複数承認者がタスクを承認できる
5. 組織メンバーが月次の広報下書きを保存できる
6. 組織メンバーが広報投稿を Bot に依頼できる
7. Bot が投稿結果を API へ反映できる
8. API が認可・監査・テナント境界の正本になる

## 成功条件

- `AUTH_MODE=stub|oidc` の切替で同じ API を動かせる
- OIDC モードでは Pocket ID の JWT を検証できる
- 初回ログイン時に `users` と単一 seed 組織への `organization_members` が自動で整う
- `OWNER / EDITOR / VIEWER` の RBAC が API 側で適用される
- `tasks`, `meetings`, `announcements` が `year/month` 単位で管理される
- `task_approvals` が複数承認者の未承認・承認済みを表現できる
- Bot が `publish_requested` の広報を取得し、完了または失敗を書き戻せる
- 監査対象操作で `audit_logs` が残る
- 構造化ログが JSON で出力される

## 非対象

- 組織管理 UI
- グローバルロール運用
- Discord への実送信処理そのもの
- P2 の `attachments`
- フロントエンドの API 接続実装

## 制約

- API は Go / Echo / OpenAPI / PostgreSQL / `sqlc` / `pgx` を使う
- 認可の正本は DB の `organization_members` とする
- Bot は internal API を使い、共有トークンで認証する
- フロントエンドの `approved` 表現は API の task status ではなく承認進捗から導出する
- `doc/04_permission-design.md` `doc/05_erd.md` `doc/08_logging.md` を同期更新する
