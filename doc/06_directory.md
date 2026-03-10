# 📁 ディレクトリ構成図

## 0. 設計前提

| 項目 | 内容 |
| --- | --- |
| リポジトリ構成 | Polyrepo |
| アーキテクチャ | Clean Architecture |
| デプロイ単位 | マイクロサービス |
| 言語 | 任意（TypeScript / Go / Python など） |
| MVP方針 | P0に必要なディレクトリのみ |

## 1. 全体構成（Monorepo想定）

```text
root/
├── apps/              # 実行可能アプリ
│   ├── web/
│   ├── api/
│   └── admin/
├── packages/          # 共有パッケージ
│   ├── ui/
│   ├── domain/
│   ├── config/
│   └── utils/
├── infra/             # IaC / Terraform / Docker
├── scripts/           # 補助スクリプト
├── docs/              # 設計書
└── README.md
```

## 2. フロントエンド構成テンプレ

```text
apps/web/
├── src/
│   ├── app/           # ルーティング層
│   ├── features/      # 機能単位モジュール
│   ├── components/    # 共通UI
│   ├── hooks/
│   ├── lib/           # APIクライアント等
│   ├── stores/        # 状態管理
│   └── types/
├── public/
└── tests/
```

### Featureベース構成（推奨）

```text
features/
├── auth/
│   ├── components/
│   ├── api.ts
│   ├── hooks.ts
│   └── types.ts
├── entity/
│   ├── components/
│   ├── api.ts
│   ├── hooks.ts
│   └── types.ts
```

## 3. バックエンド構成テンプレ（Clean Architecture）

```text
apps/api/
├── cmd/                # エントリポイント
├── internal/
│   ├── domain/         # エンティティ・ビジネスルール
│   ├── usecase/        # アプリケーションロジック
│   ├── repository/     # DB抽象
│   ├── handler/        # HTTP層
│   ├── middleware/
│   └── config/
├── migrations/
└── tests/
```

## 4. DDDベース構成テンプレ

```text
src/
├── modules/
│   ├── user/
│   │   ├── domain/
│   │   ├── application/
│   │   ├── infrastructure/
│   │   └── presentation/
│   ├── organization/
│   └── core/
```

## 5. マイクロサービス構成

```text
services/
├── auth-service/
├── core-service/
├── notification-service/
└── gateway/
```

## 6. インフラ構成

```text
infra/
├── terraform/
│   ├── modules/
│   └── environments/
│       ├── dev/
│       ├── staging/
│       └── prod/
├── docker/
└── ci/
```

## 7. ドキュメント構成

```text
docs/
├── 01_feature-list.md
├── 02_db-design.md
├── 03_screen-flow.md
├── 04_permission-design.md
├── 05_api-spec.md
└── 06_directory.md
```

## 8. テスト構成テンプレ

```text
tests/
├── unit/
├── integration/
├── e2e/
└── fixtures/
```

## 9. ベクトルDB / AI機能がある場合

```text
packages/
├── embeddings/
│   ├── generator.ts
│   ├── repository.ts
│   └── vector-client.ts
├── rag/
│   ├── retriever.ts
│   └── prompt-builder.ts
```

## 10. 状態管理分離パターン（FE）

```text
stores/
├── auth.store.ts
├── entity.store.ts
└── ui.store.ts
```

## 11. API設計分離パターン

```text
api/
├── client.ts
├── endpoints/
│   ├── auth.ts
│   ├── entities.ts
│   └── users.ts
```
