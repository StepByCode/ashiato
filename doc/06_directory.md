📁 ディレクトリ構成図

0️⃣ 設計前提

項目

内容

リポジトリ構成

Polyrepo

アーキテクチャ

Clean Architecture

デプロイ単位

3サービス（Frontend / API / Discord Bot）

言語

TypeScript（Frontend） / Go（API・Bot）

MVP方針

P0に必要なディレクトリのみ

1️⃣ 全体構成（Monorepo想定）

root/
├── frontend/          # Webアプリ（Vercel）
├── api/               # APIサーバー（Coolify）
├── bot/               # Discord bot（Coolify）
├── doc/               # 設計書
└── README.md

2️⃣ フロントエンド構成テンプレ

frontend/
├── src/
│   ├── app/           # ルーティング層
│   ├── features/      # 機能単位モジュール
│   ├── components/    # 共通UI
│   ├── hooks/
│   ├── lib/           # APIクライアント
│   ├── stores/        # Zustand
│   └── types/
├── public/
└── tests/

Featureベース構成（推奨）

features/
├── auth/
│   ├── components/
│   ├── api.ts
│   ├── hooks.ts
│   └── types.ts
├── tasks/
│   ├── components/
│   ├── api.ts
│   ├── hooks.ts
│   └── types.ts
├── meeting/
│   ├── components/
│   ├── api.ts
│   ├── hooks.ts
│   └── types.ts
└── announcement/
    ├── components/
    ├── api.ts
    ├── hooks.ts
    └── types.ts

3️⃣ バックエンド構成テンプレ（Clean Architecture）

api/
├── cmd/                # エントリポイント
├── internal/
│   ├── domain/         # エンティティ・ビジネスルール
│   ├── usecase/        # アプリケーションロジック
│   ├── repository/     # sqlc + pgx
│   ├── handler/        # Echo HTTP層
│   ├── middleware/     # 認証・認可
│   └── config/
├── migrations/
└── tests/

4️⃣ DDDベース構成テンプレ

src/
├── modules/
│   ├── task/
│   │   ├── domain/
│   │   ├── application/
│   │   ├── infrastructure/
│   │   └── presentation/
│   ├── approval/
│   ├── meeting/
│   └── auth/

5️⃣ マイクロサービス構成

services/
├── frontend-service/
├── api-service/
└── discord-bot-service/

6️⃣ インフラ構成

infra/
├── vercel/            # FE設定
├── coolify/           # API/Bot設定
└── ci/                # GitHub Actions等

※ IaC（Terraform）は現フェーズ非採用。Vercel/Coolifyの設定中心で運用。

7️⃣ ドキュメント構成

doc/
├── 00designdoc_template.md
├── 01_feature-list.md
├── 02_tech-stack.md
├── 03_screen-flow.md
├── 04_permission-design.md
├── 05_erd.md
├── 06_directory.md
├── 07_infrastructure.md
├── 08_logging.md
└── 09_schedule_and_issues.md

8️⃣ テスト構成テンプレ

tests/
├── unit/
├── integration/
├── e2e/
└── fixtures/

9️⃣ ベクトルDB / AI機能がある場合

packages/
├── embeddings/
│   ├── generator.ts
│   ├── repository.ts
│   └── vector-client.ts
├── rag/
│   ├── retriever.ts
│   └── prompt-builder.ts

※ 現行スコープでは非採用（将来検討）。

🔟 状態管理分離パターン（FE）

stores/
├── auth.store.ts
├── tasks.store.ts
├── meeting.store.ts
└── ui.store.ts

11️⃣ API設計分離パターン

api/
├── client.ts
├── endpoints/
│   ├── auth.ts
│   ├── tasks.ts
│   ├── approvals.ts
│   ├── meeting.ts
│   └── announcements.ts
