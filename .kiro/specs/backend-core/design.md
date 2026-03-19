# backend-core Design

## アーキテクチャ

- `api/cmd/server` が Echo サーバーを起動する
- OpenAPI を `api/openapi/openapi.yaml` に置き、`oapi-codegen` で Echo 向けサーバー型を生成する
- PostgreSQL へのアクセスは `sqlc` 生成コードと `pgxpool` を使う
- 認証は `stub` と `oidc` の 2 モードを持つ
- 認証後にユーザーと seed 組織メンバーシップを自動同期する

## 認証・認可

- `stub`
  - `DEV_JWT_SECRET` で署名された JWT を受け付ける
  - `sub`, `email`, `name` claim を正規化する
- `oidc`
  - `OIDC_ISSUER_URL` の discovery から Pocket ID の issuer/JWKS を取得する
  - `OIDC_CLIENT_ID` を audience として JWT を検証する
- ロールは `OWNER / EDITOR / VIEWER`
- 認可ルール
  - `OWNER`: 全 API 操作可
  - `EDITOR`: 読み取りと業務更新可
  - `VIEWER`: 読み取りのみ
  - `POST /api/v1/tasks/{id}/approve` はロールに加えて approver 指定が必要
- 全クエリで `organization_id` を絞り込む

## データモデル

- `organizations`
  - `slug`, `name` を持つ単一 seed 組織
- `users`
  - OIDC 主体は `oidc_subject`
- `organization_members`
  - ロール付与の正本
- `tasks`
  - `organization_id`, `year`, `month`, `title`, `status`, `due_date`, `reference_url`, `assignee_id`, `created_by`, `version`, `deleted_at`
- `task_approvals`
  - `task_id + approver_user_id` ごとに 1 行
  - `approved_at IS NULL` で未承認を表す
- `meetings`
  - 1 組織・1 年月で 1 件
  - `scheduled_at`, `meeting_url`, `notes`, `status`
- `announcements`
  - 1 組織・1 年月で 1 件
  - `body`, `status`, `publish_channel`, `discord_message_id`, `last_error`
- `audit_logs`
  - 成功した監査対象操作の before/after を保持する

## API 契約

- 公開 API
  - `GET /api/v1/me`
  - `GET /api/v1/members`
  - `GET /api/v1/workflow-periods`
  - `GET /api/v1/meetings/{year}/{month}`
  - `PUT /api/v1/meetings/{year}/{month}`
  - `GET /api/v1/tasks`
  - `POST /api/v1/tasks`
  - `PATCH /api/v1/tasks/{id}`
  - `DELETE /api/v1/tasks/{id}`
  - `POST /api/v1/tasks/{id}/approve`
  - `GET /api/v1/announcements/{year}/{month}`
  - `PUT /api/v1/announcements/{year}/{month}`
  - `POST /api/v1/announcements/{id}/publish`
- internal API
  - `GET /internal/v1/announcement-publish-requests`
  - `POST /internal/v1/announcement-publish-requests/{id}/complete`
  - `POST /internal/v1/announcement-publish-requests/{id}/fail`

## 状態遷移

- `tasks.status`
  - `todo` -> `in_progress` -> `done`
- `task approval_state`
  - `not_required`, `pending`, `partially_approved`, `approved`
- `meetings.status`
  - `planned`, `completed`, `cancelled`
- `announcements.status`
  - `draft` -> `publish_requested` -> `published`
  - `publish_requested` -> `publish_failed`
  - 下書き更新時は `draft` に戻す

## Bot 連携

- 公開 API からの `publish` 操作で `announcements.status=publish_requested` にする
- Bot は internal API で pending を取得する
- Bot 成功時に `published` と `discord_message_id` を反映する
- Bot 失敗時に `publish_failed` と `last_error` を反映する

## ログ・監査

- 全リクエストで `trace_id` を発行または継承する
- access log と application log は JSON で出す
- 監査対象
  - task create/update/delete/approve
  - meeting upsert
  - announcement upsert/publish request/publish complete/publish fail
