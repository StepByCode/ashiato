# Project Structure

## Organization Philosophy

トップレベルをサービス責務ごとに分離し、サービス間連携は API 契約と設計ドキュメントで管理します。1つの変更が複数サービスへ波及しやすいため、実装時は「1ディレクトリだけ修正して終える」のではなく、関連するサービスとドキュメントを同じ変更単位で揃えます。

## Directory Patterns

### Frontend
**Location**: `frontend/`  
**Purpose**: 運営メンバー向け Web UI、画面遷移、表示状態管理、権限に応じた UI 制御  
**Example**: 画面追加時は `frontend/` に加えて `doc/03_screen-flow.md` を確認する

### API
**Location**: `api/`  
**Purpose**: 認証済み API、業務ロジック、認可、永続化、監査ログ  
**Example**: 権限変更時は `api/` 実装と `doc/04_permission-design.md` をセットで見直す

### Bot
**Location**: `bot/`  
**Purpose**: Discord 通知、投稿補助、定期実行、API 連携  
**Example**: 通知文面やジョブ変更時は `bot/` と `doc/09_schedule_and_issues.md` を確認する

### Documentation
**Location**: `doc/`  
**Purpose**: 機能、画面、権限、ERD、インフラ、ログ、運用ルールの設計書  
**Example**: コード仕様が変わる変更では、影響する `doc/*.md` を同じ作業で更新する

### Spec-Driven Files
**Location**: `.kiro/`  
**Purpose**: `cc-sdd` の settings、steering、specs を保持する  
**Example**: 新機能は `.kiro/specs/<feature>/` で requirements/design/tasks を管理する

## Naming Conventions

- **Documentation**: `doc/NN_topic.md`
- **Frontend files**: `kebab-case`
- **React components**: `PascalCase`
- **Go packages**: `lowercase`
- **Go files**: `snake_case.go`

## Import and Dependency Rules

- `frontend/` `api/` `bot/` の間でソースコードを直接 import しない
- サービス間連携は API 契約、HTTP、設計ドキュメントを通して行う
- フロントエンド内の詳細な import ルールは、Next.js プロジェクト作成時にその配下で定義する

## Code Organization Principles

- 認証・認可・監査の最終責務は `api/` に置く
- `bot/` は通知と補助に専念し、正データや権限判定を持たない
- `frontend/` はユーザー体験のための表示制御を行うが、権限の最終判定はしない
- 仕様変更時は `README.md` と `doc/` を必ず見直す
