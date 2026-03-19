# Technology Stack

## Architecture

単一リポジトリの中に `frontend/` `api/` `bot/` `doc/` を持つ構成です。実行系は Web フロントエンド、API、Discord Bot の3サービスを想定し、サービス間の接続は API 契約と設計ドキュメントを基準にします。

## Core Technologies

- **Frontend**: React / Next.js / TypeScript / Zustand / shadcn/ui
- **API**: Go / Echo / OpenAPI / PostgreSQL / `sqlc` / `pgx`
- **Auth**: OAuth2 / OpenID Connect / JWT
- **Bot**: Discord 連携 Bot。SDK とジョブ実行方式は実装着手時に確定する
- **Ops**: Docker / Coolify / Cloudflare CDN / Dozzle

## Key Libraries and Standards

- API 契約は OpenAPI を基準にし、フロントエンドと Bot はそれに合わせる
- 認可の最終判定は API 側で行い、フロントエンドは UX 補助の表示制御に留める
- ログは JSON の構造化ログを前提にし、監査対象操作は監査ログを残す
- RBAC を基軸にし、必要に応じて ABAC 拡張できる設計を維持する

## Development Environment

### Required Tools

- Node.js 20 以上
- npm 11 以上
- Go toolchain
- Docker
- Codex CLI / Gemini CLI / Claude Code

### Common Working Rules

- 実装前に関連する `doc/*.md` を確認する
- 仕様変更時はコード変更と同じ単位で `doc/` を更新する
- 画面変更時は `doc/03_screen-flow.md`、権限変更時は `doc/04_permission-design.md`、DB変更時は `doc/05_erd.md`、ログ変更時は `doc/08_logging.md` を更新候補にする

## Key Technical Decisions

- 業務ロジックと認可は API に集約する
- Bot は通知と運用補助に専念し、正データ管理は持たない
- 各サービスの実装より先に仕様と設計を固めるため、`cc-sdd` の Requirements → Design → Tasks → Implementation を基本フローにする
- Bot の言語や SDK はまだ固定していないため、着手時には既存ドキュメントと整合する形で明文化してから進める
