# ログ設計

## 0. 設計前提

| 項目 | 内容 |
| --- | --- |
| 対象システム | Frontend / API / Discord Bot |
| ログ方式 | JSON 構造化ログ |
| 集約方式 | Centralized Logging |
| 監査対象 | task / meeting / announcement の write 操作 |
| 個人情報 | 必要最小限。token は出力禁止 |

## 1. ログ分類

| 種別 | 用途 |
| --- | --- |
| Access Log | リクエスト追跡 |
| Application Log | 動作確認・障害解析 |
| Audit Log | セキュリティ監査・業務追跡 |
| Security Log | 認証失敗・異常検知 |

## 2. 標準フィールド

```json
{
  "timestamp": "2026-03-19T10:00:00Z",
  "level": "INFO",
  "service": "api",
  "trace_id": "uuid",
  "user_id": "uuid",
  "tenant_id": "uuid",
  "action": "task.update",
  "resource_type": "task",
  "resource_id": "uuid",
  "message": "request completed"
}
```

必須項目:

| フィールド | 説明 |
| --- | --- |
| `timestamp` | 時系列追跡 |
| `level` | 重要度 |
| `service` | サービス識別 |
| `trace_id` | 追跡 ID |
| `tenant_id` | 組織境界 |

## 3. Access Log

API では全リクエストで以下を出力する。

```json
{
  "timestamp": "...",
  "level": "INFO",
  "service": "api",
  "trace_id": "uuid",
  "method": "POST",
  "path": "/api/v1/tasks",
  "status": 201,
  "latency_ms": 42,
  "ip": "127.0.0.1",
  "user_agent": "..."
}
```

## 4. Audit Log

### 4.1 スキーマ

| フィールド | 内容 |
| --- | --- |
| `organization_id` | 組織 ID |
| `actor_user_id` | user actor の場合のみ設定 |
| `actor_type` | `user` または `service` |
| `actor_label` | service 名など |
| `action` | `task.create` など |
| `resource_type` | `task` `meeting` `announcement` |
| `resource_id` | リソース ID |
| `before_state` | 変更前 JSON |
| `after_state` | 変更後 JSON |
| `result` | `success` |
| `ip` | 実行元 IP |

### 4.2 user actor 例

```json
{
  "actor_type": "user",
  "actor_user_id": "uuid",
  "action": "task.approve",
  "resource_type": "task",
  "resource_id": "uuid",
  "result": "success"
}
```

### 4.3 service actor 例

```json
{
  "actor_type": "service",
  "actor_label": "discord-bot",
  "action": "announcement.publish_complete",
  "resource_type": "announcement",
  "resource_id": "uuid",
  "result": "success"
}
```

## 5. 監査対象操作

- `task.create`
- `task.update`
- `task.delete`
- `task.approve`
- `meeting.upsert`
- `announcement.upsert`
- `announcement.publish_request`
- `announcement.publish_complete`
- `announcement.publish_fail`

## 6. セキュリティログ

- Bearer token 未設定
- JWT 検証失敗
- `X-Bot-Token` 不一致
- tenant mismatch

## 7. マスキング

| 対象 | 方針 |
| --- | --- |
| Bearer token | 出力禁止 |
| Bot token | 出力禁止 |
| email | Access/Application log では不要時は出さない |
| IP | 監査要件に応じて保持 |

## 8. 保持ポリシー

| 種類 | 保持期間 |
| --- | --- |
| Access | 90日 |
| Application | 30日 |
| Audit | 1年以上 |
| Security | 1年以上 |
