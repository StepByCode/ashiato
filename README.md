# Ashiato

StepByCode運営向けのイベント進行管理プロジェクトです。  
「定例 → 作成 → 広報」の流れを、WebアプリとDiscord Botで扱える状態を目指します。

## リポジトリ構成

```text
ashiato/
├── frontend/   # Webフロントエンド
├── api/        # バックエンドAPI
├── bot/        # Discord連携Bot
└── doc/        # 設計・運用ドキュメント
```

各ディレクトリの詳細は以下を参照してください。

- `frontend/README.md`
- `api/README.md`
- `bot/README.md`
- `doc/README.md`

## まず読むドキュメント

- `doc/01_feature-list.md`（機能と優先度）
- `doc/02_tech-stack.md`（採用技術）
- `doc/03_screen-flow.md`（画面遷移）
- `doc/04_permission-design.md`（権限設計）
- `doc/05_erd.md`（DB設計）

## 開発方針

- P0（MVP）を先に固める
- API仕様を基準に、frontend/botを接続する
- 仕様変更時は実装と同じPRで `doc/` も更新する
- 権限制御の最終判定は必ずAPI側で行う

## ステータス

現在は設計ドキュメントを起点に、各コンポーネント実装を進めるフェーズです。

## AIエージェント環境

このリポジトリでは `cc-sdd` を採用し、`Codex` `Gemini CLI` `Claude Code` 向けの作業環境を配置しています。

- Codex: `AGENTS.md`、`.codex/prompts/`
- Gemini CLI: `GEMINI.md`、`.gemini/commands/kiro/`
- Claude Code: `CLAUDE.md`、`.claude/commands/kiro/`
- 共通: `.kiro/settings/`、`.kiro/steering/`

各CLIはリポジトリルートで起動し、`frontend/` `api/` `bot/` `doc/` を横断して影響範囲を確認する前提です。仕様化から始める場合は、最初に steering を読み込んでから spec を切ってください。

- Codex: `/prompts:kiro-steering` → `/prompts:kiro-spec-init "やりたいこと"`
- Gemini CLI: `/kiro:steering` → `/kiro:spec-init "やりたいこと"`
- Claude Code: `/kiro:steering` → `/kiro:spec-init "やりたいこと"`
