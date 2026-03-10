# 📁 ディレクトリ構成図

## 0. 設計前提

| 項目 | 内容 |
| --- | --- |
| リポジトリ構成 | Polyrepo |
| アーキテクチャ | Clean Architecture（API） + Feature-based（Frontend） |
| デプロイ単位 | 3サービス（Frontend / API / Discord Bot） |
| 言語 | TypeScript（Frontend） / Go（API・Bot） |
| MVP方針 | P0に必要なディレクトリのみ作成 |

## 1. 全体構成

```text
ashiato/
├── frontend/          # Webアプリ（Vercel）
├── api/               # APIサーバー（Coolify）
├── bot/               # Discord bot（Coolify）
├── doc/               # 設計書
└── README.md
```

## 2. フロントエンド構成（推奨）

```text
frontend/
├── src/
│   ├── app/           # ルーティング層
│   ├── features/      # 機能単位モジュール
│   ├── components/    # 共通UI
│   ├── hooks/         # 共通hooks
│   ├── lib/           # APIクライアント
│   ├── stores/        # Zustand
│   └── types/         # 型定義
├── public/
└── tests/
```

### Featureベース例

```text
features/
├── auth/
├── tasks/
├── meeting/
└── announcement/
```

## 3. API構成（Clean Architecture）

```text
api/
├── cmd/                # エントリポイント
├── internal/
│   ├── domain/         # エンティティ・ビジネスルール
│   ├── usecase/        # ユースケース
│   ├── repository/     # DB抽象
│   ├── handler/        # HTTP層
│   ├── middleware/     # 認証・認可
│   └── config/         # 設定
├── migrations/         # DBマイグレーション
└── tests/
```

## 4. Bot構成（推奨）

```text
bot/
├── src/
│   ├── commands/       # Discordコマンド
│   ├── jobs/           # 定期通知ジョブ
│   ├── services/       # API連携ロジック
│   ├── templates/      # 投稿テンプレート
│   └── config/         # 設定
├── scripts/
└── tests/
```

## 5. ドキュメント構成

```text
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
```

## 6. 命名ルール

- ドキュメント: `NN_topic.md`（2桁連番）
- Frontend: `kebab-case` ファイル名、Reactコンポーネントは `PascalCase`
- Go: パッケージは `lowercase`、ファイルは `snake_case.go`

## 7. 追加チェックリスト

- 追加したディレクトリの責務をREADMEに追記したか
- 設計変更時に `doc/` の関連ファイルを更新したか
- テスト配置先（`tests/`）を先に決めてから実装したか
