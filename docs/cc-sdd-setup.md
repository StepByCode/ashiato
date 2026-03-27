# cc-sdd セットアップガイド

このドキュメントでは、Ashiato リポジトリで採用している **cc-sdd（Claude Code Steering-Driven Development）** を各 AI CLI で使い始めるための手順を説明します。

---

## cc-sdd とは

cc-sdd は、AIコーディングエージェントが **仕様 → 設計 → タスク → 実装** の順に開発を進めるためのフレームワークです。

- **steering**（`.kiro/steering/`）: プロダクト概要、技術スタック、ディレクトリ構成をプロジェクトの長期メモリとして保持
- **specs**（`.kiro/specs/`）: 機能ごとの要件定義・設計・タスク分解を管理
- **settings**（`.kiro/settings/`）: テンプレートとルールの共通基盤

### リポジトリ内の配置構成

```text
.kiro/
├── settings/
│   ├── rules/              # 設計・タスク生成のルール定義
│   └── templates/
│       ├── steering/       # steering 生成テンプレート
│       ├── steering-custom/ # カスタム steering テンプレート
│       └── specs/          # spec 生成テンプレート
├── steering/
│   ├── product.md          # プロダクト概要
│   ├── tech.md             # 技術スタック
│   └── structure.md        # ディレクトリ構成
└── specs/
    └── {feature}/          # 機能ごとの仕様
        ├── requirements.md
        ├── design.md
        └── tasks.md
```

### 基本ワークフロー

```
1. steering 読み込み（初回 or 方針変更時）
2. spec-init → spec-requirements → spec-design → spec-tasks
3. spec-impl でタスクを実装
4. validate-* で検証
```

---

## Claude Code 用セットアップ

### 前提条件

- Claude Code CLI がインストール済み
- Node.js 20 以上

### インストール手順

```bash
# 1. リポジトリをクローン
git clone https://github.com/stepbycode/ashiato.git
cd ashiato

# 2. npm 依存を解決（package-lock.json がある場合）
npm install

# 3. Claude Code を起動
claude
```

### 設定ファイルの場所

| ファイル | 役割 |
|---|---|
| `CLAUDE.md` | Claude Code が自動読み込みするプロジェクトガイド |
| `.claude/commands/kiro/*.md` | `/kiro:*` スラッシュコマンド定義 |
| `.kiro/steering/*.md` | 長期メモリ（プロジェクト知識） |

### 使い方

Claude Code を起動したら、以下のスラッシュコマンドが使えます。

#### steering（プロジェクト知識の管理）

```
/kiro:steering              # steering の読み込み・同期
/kiro:steering-custom       # カスタム steering の追加
```

#### 仕様化フロー

```
/kiro:spec-init "機能の説明"           # 新しい spec を作成
/kiro:spec-requirements {feature}     # 要件定義を生成
/kiro:spec-design {feature}           # 技術設計を生成
/kiro:spec-tasks {feature}            # タスク分解を生成
```

#### 実装・検証

```
/kiro:spec-impl {feature} [tasks]     # タスクを実装
/kiro:spec-status {feature}           # 進捗確認
/kiro:validate-gap {feature}          # 要件とコードのギャップ分析
/kiro:validate-design {feature}       # 設計レビュー
/kiro:validate-impl {feature}         # 実装検証
```

#### 例：新機能を追加する場合

```
# 1. まず steering を読み込む
/kiro:steering

# 2. spec を作成
/kiro:spec-init "タスクにコメント機能を追加"

# 3. 要件 → 設計 → タスク の順に進める
/kiro:spec-requirements task-comments
/kiro:spec-design task-comments
/kiro:spec-tasks task-comments

# 4. 実装
/kiro:spec-impl task-comments

# 5. 検証
/kiro:validate-impl task-comments
```

---

## Codex 用セットアップ

### 前提条件

- OpenAI Codex CLI がインストール済み
- Node.js 20 以上

### インストール手順

```bash
# 1. リポジトリをクローン
git clone https://github.com/stepbycode/ashiato.git
cd ashiato

# 2. npm 依存を解決
npm install

# 3. Codex を起動
codex
```

### 設定ファイルの場所

| ファイル | 役割 |
|---|---|
| `AGENTS.md` | Codex が自動読み込みするプロジェクトガイド |
| `.codex/prompts/kiro-*.md` | `/prompts:kiro-*` プロンプトコマンド定義 |
| `.kiro/steering/*.md` | 長期メモリ（プロジェクト知識） |

### 使い方

Codex を起動したら、以下のプロンプトコマンドが使えます。

#### steering（プロジェクト知識の管理）

```
/prompts:kiro-steering              # steering の読み込み・同期
/prompts:kiro-steering-custom       # カスタム steering の追加
```

#### 仕様化フロー

```
/prompts:kiro-spec-init "機能の説明"           # 新しい spec を作成
/prompts:kiro-spec-requirements {feature}     # 要件定義を生成
/prompts:kiro-spec-design {feature}           # 技術設計を生成
/prompts:kiro-spec-tasks {feature}            # タスク分解を生成
```

#### 実装・検証

```
/prompts:kiro-spec-impl {feature} [tasks]     # タスクを実装
/prompts:kiro-spec-status {feature}           # 進捗確認
/prompts:kiro-validate-gap {feature}          # 要件とコードのギャップ分析
/prompts:kiro-validate-design {feature}       # 設計レビュー
/prompts:kiro-validate-impl {feature}         # 実装検証
```

#### 例：新機能を追加する場合

```
# 1. まず steering を読み込む
/prompts:kiro-steering

# 2. spec を作成
/prompts:kiro-spec-init "タスクにコメント機能を追加"

# 3. 要件 → 設計 → タスク の順に進める
/prompts:kiro-spec-requirements task-comments
/prompts:kiro-spec-design task-comments
/prompts:kiro-spec-tasks task-comments

# 4. 実装
/prompts:kiro-spec-impl task-comments

# 5. 検証
/prompts:kiro-validate-impl task-comments
```

---

## Gemini CLI 用セットアップ

### 前提条件

- Gemini CLI がインストール済み
- Node.js 20 以上

### インストール手順

```bash
# 1. リポジトリをクローン
git clone https://github.com/stepbycode/ashiato.git
cd ashiato

# 2. npm 依存を解決
npm install

# 3. Gemini CLI を起動
gemini
```

### 設定ファイルの場所

| ファイル | 役割 |
|---|---|
| `GEMINI.md` | Gemini CLI が自動読み込みするプロジェクトガイド |
| `.gemini/commands/kiro/*.toml` | `/kiro:*` コマンド定義（TOML 形式） |
| `.kiro/steering/*.md` | 長期メモリ（プロジェクト知識） |

### 使い方

Gemini CLI を起動したら、以下のコマンドが使えます。

#### steering（プロジェクト知識の管理）

```
/kiro:steering              # steering の読み込み・同期
/kiro:steering-custom       # カスタム steering の追加
```

#### 仕様化フロー

```
/kiro:spec-init "機能の説明"           # 新しい spec を作成
/kiro:spec-requirements {feature}     # 要件定義を生成
/kiro:spec-design {feature}           # 技術設計を生成
/kiro:spec-tasks {feature}            # タスク分解を生成
```

#### 実装・検証

```
/kiro:spec-impl {feature} [tasks]     # タスクを実装
/kiro:spec-status {feature}           # 進捗確認
/kiro:validate-gap {feature}          # 要件とコードのギャップ分析
/kiro:validate-design {feature}       # 設計レビュー
/kiro:validate-impl {feature}         # 実装検証
```

#### 例：新機能を追加する場合

```
# 1. まず steering を読み込む
/kiro:steering

# 2. spec を作成
/kiro:spec-init "タスクにコメント機能を追加"

# 3. 要件 → 設計 → タスク の順に進める
/kiro:spec-requirements task-comments
/kiro:spec-design task-comments
/kiro:spec-tasks task-comments

# 4. 実装
/kiro:spec-impl task-comments

# 5. 検証
/kiro:validate-impl task-comments
```

---

## CLI ごとの違いまとめ

| 項目 | Claude Code | Codex | Gemini CLI |
|---|---|---|---|
| ガイドファイル | `CLAUDE.md` | `AGENTS.md` | `GEMINI.md` |
| コマンド定義 | `.claude/commands/kiro/*.md` | `.codex/prompts/kiro-*.md` | `.gemini/commands/kiro/*.toml` |
| コマンド形式 | `.md`（Markdown） | `.md`（Markdown + meta タグ） | `.toml`（TOML） |
| コマンド接頭辞 | `/kiro:` | `/prompts:kiro-` | `/kiro:` |
| 共通基盤 | `.kiro/steering/`, `.kiro/specs/`, `.kiro/settings/` |||

---

## 共通の注意事項

1. **順序を守る**: Requirements → Design → Tasks → Implementation の順を崩さない
2. **横断確認**: 変更が複数ディレクトリへ波及するなら、関連箇所まで完結させる
3. **ドキュメント同期**: 仕様変更時はコードと `doc/` を同じ作業で更新する
4. **日本語を基本**: 対話とプロジェクト内 Markdown は日本語で記述する
5. **steering を最新に保つ**: 大きな方針変更時は steering の sync を実行する

---

## トラブルシューティング

### コマンドが見つからない

各 CLI のコマンドディレクトリが正しく配置されているか確認してください。

```bash
# Claude Code
ls .claude/commands/kiro/

# Codex
ls .codex/prompts/

# Gemini CLI
ls .gemini/commands/kiro/
```

### steering が空

初回セットアップ時は steering の bootstrap を実行してください。

```
# Claude Code / Gemini CLI
/kiro:steering

# Codex
/prompts:kiro-steering
```

### spec が見つからない

`.kiro/specs/` 配下に対象のフィーチャーディレクトリが存在するか確認してください。

```bash
ls .kiro/specs/
```
