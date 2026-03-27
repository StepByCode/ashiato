# 権限設計

## 0. 設計前提

| 項目 | 内容 |
| --- | --- |
| 権限モデル | RBAC + 最小限の ABAC |
| マルチテナント | DB は対応、MVP は単一 seed 組織 |
| 認証方式 | Firebase Authentication (Email/Password) |
| スコープ単位 | `organization_id` |
| MVP 方針 | `OWNER / EDITOR / VIEWER` のみ実装 |

## 1. 用語

| 用語 | 意味 |
| --- | --- |
| Subject | 操作主体。`user` または `service` |
| Resource | `task` `meeting` `announcement` などの対象 |
| Role | 組織単位の権限。`OWNER / EDITOR / VIEWER` |
| Policy | RBAC と属性条件の組み合わせ |

## 2. 認証

- 公開 API は Bearer JWT（Firebase ID トークン）を必須とする
- Firebase Admin SDK でトークンを検証し、`uid/email/displayName` を取得する
- 認証後、`uid/email/name` を正規化して `users` を upsert し、seed 組織へ自動所属させる
- ユーザーは Firebase コンソールで事前作成し、Email/Password でログインする

## 3. ロール設計

| ロール | 説明 |
| --- | --- |
| `OWNER` | 全操作可。seed owner email と一致するユーザー |
| `EDITOR` | 読み取りと業務更新可。MVP の既定ロール |
| `VIEWER` | 読み取りのみ |

補足:
- グローバルロールは MVP 対象外
- 認可の正本は IdP claim ではなく `organization_members.role`

## 4. 権限ルール

| 操作 | `OWNER` | `EDITOR` | `VIEWER` | 補足 |
| --- | --- | --- | --- | --- |
| `GET /api/v1/me` | 可 | 可 | 可 | - |
| `GET /api/v1/members` | 可 | 可 | 可 | - |
| `GET /api/v1/workflow-periods` | 可 | 可 | 可 | - |
| `GET /api/v1/tasks` | 可 | 可 | 可 | - |
| `POST/PATCH/DELETE /api/v1/tasks*` | 可 | 可 | 不可 | - |
| `POST /api/v1/tasks/{id}/approve` | 可 | 可 | 条件付き可 | approver 指定が必須 |
| `GET /api/v1/meetings/*` | 可 | 可 | 可 | - |
| `PUT /api/v1/meetings/*` | 可 | 可 | 不可 | - |
| `GET /api/v1/announcements/*` | 可 | 可 | 可 | - |
| `PUT/POST /api/v1/announcements*` | 可 | 可 | 不可 | 投稿指示は API が正データ化 |
| `GET/POST /internal/v1/announcement-publish-requests*` | service token | service token | service token | `X-Bot-Token` が必須 |

## 5. ABAC 条件

### 5.1 テナント境界

```text
if resource.organization_id != actor.organization_id:
    deny
```

### 5.2 タスク承認

```text
if actor.user_id is in task.approver_ids and task.status == "done":
    allow
else:
    deny
```

### 5.3 Bot internal API

```text
if request.header["X-Bot-Token"] == BOT_SHARED_TOKEN:
    allow
else:
    deny
```

## 6. 監査対象

- `task.create`
- `task.update`
- `task.delete`
- `task.approve`
- `meeting.upsert`
- `announcement.upsert`
- `announcement.publish_request`
- `announcement.publish_complete`
- `announcement.publish_fail`

監査ログでは `actor_type=user|service` を持ち、Bot 経由の反映も追跡可能にする。

## 7. API レイヤー統合

```text
1. Firebase ID トークン or bot token を検証
2. user の場合は users / organization_members を同期
3. organization_id 境界を強制
4. role を判定
5. 必要な ABAC 条件を判定
6. write 操作なら audit_logs を保存
```
