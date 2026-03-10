# 📘 ログ設計

## 0. 設計前提

| 項目 | 内容 |
| --- | --- |
| 対象システム | Frontend / API / Discord Bot |
| ログ方式 | 構造化ログ（JSON） |
| 集約方式 | Centralized Logging |
| 保持期間 | 30日 / 90日 / 1年 |
| 個人情報 | マスキング必須 |

## 1. ログ分類

| 種別 | 目的 | 出力対象 |
| --- | --- | --- |
| Application Log | 動作確認・デバッグ | 開発・運用 |
| Access Log | リクエスト追跡 | 運用 |
| Audit Log | セキュリティ監査 | セキュリティ |
| Security Log | 異常検知 | 運用 |
| Business Log | KPI分析 | 運営 |
| Infrastructure Log | リソース監視 | SRE |

## 2. ログレベル定義

| レベル | 用途 |
| --- | --- |
| `DEBUG` | 詳細情報（本番では通常無効） |
| `INFO` | 正常動作 |
| `WARN` | 想定内の異常 |
| `ERROR` | 処理失敗 |
| `FATAL` | サービス停止級 |

## 3. 標準ログフォーマット

```json
{
  "timestamp": "2026-03-10T10:00:00Z",
  "level": "INFO",
  "service": "api-service",
  "environment": "prod",
  "trace_id": "uuid",
  "user_id": "uuid",
  "tenant_id": "uuid",
  "action": "task.update",
  "resource_type": "task",
  "resource_id": "uuid",
  "message": "Task updated successfully",
  "metadata": {}
}
```

## 4. 必須フィールド

| フィールド | 理由 |
| --- | --- |
| `timestamp` | 時系列追跡 |
| `level` | 重要度判定 |
| `service` | サービス識別 |
| `trace_id` | 分散トレーシング |
| `action` | 操作識別 |
| `tenant_id` | テナント境界追跡 |

## 5. ログ例

### 5.1 Access Log

```json
{
  "timestamp": "...",
  "method": "POST",
  "path": "/api/tasks",
  "status": 200,
  "latency_ms": 120,
  "ip": "xxx.xxx.xxx.xxx",
  "user_agent": "...",
  "trace_id": "..."
}
```

### 5.2 Audit Log

```json
{
  "timestamp": "...",
  "user_id": "uuid",
  "action": "role.grant",
  "resource_type": "user",
  "resource_id": "uuid",
  "before": {"role": "member"},
  "after": {"role": "admin"},
  "result": "allow",
  "ip": "..."
}
```

## 6. マスキングポリシー

| 対象 | 方針 |
| --- | --- |
| パスワード | 出力禁止 |
| アクセストークン | 先頭/末尾以外マスク |
| メールアドレス | ハッシュ化または部分マスク |
| IP | 監査要件に応じて匿名化 |

## 7. 保持ポリシー

| 種類 | 保持期間 |
| --- | --- |
| Application | 30日 |
| Access | 90日 |
| Audit | 1年以上 |
| Security | 1年以上 |

## 8. 監視・アラート

- `ERROR` 率が閾値超過でアラート
- 5xx連続発生時に即時通知
- 認証失敗の急増をSecurityイベントとして通知

## 9. 実装チェックリスト

- 全サービスでJSONログを採用したか
- `trace_id` をヘッダーで連携しているか
- PIIマスキングが有効か
- 監査対象操作を漏れなく記録しているか
