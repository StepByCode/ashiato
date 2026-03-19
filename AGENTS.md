# Ashiato Codex Guide

このリポジトリでは `cc-sdd` を採用し、Codex では `AGENTS.md` と `.kiro/steering/` を長期メモリとして扱います。作業開始時は必ずリポジトリルートを基準にし、`frontend/` `api/` `bot/` `doc/` を横断して影響範囲を確認してください。

## プロジェクトメモリ

- 長期ルールは `.kiro/steering/` を参照する
- 機能ごとの仕様は `.kiro/specs/` を参照する
- 将来ディレクトリ単位の補足が必要になった場合は、その配下に追加の `AGENTS.md` を置く

## リポジトリ横断ルール

- UI変更時は `frontend/` だけで閉じず、`doc/03_screen-flow.md` と API 契約影響を確認する
- 認証・認可・業務ロジック変更時は `api/` を正とし、`doc/04_permission-design.md` と `doc/08_logging.md` を更新候補に含める
- データモデル変更時は `doc/05_erd.md` を同じ変更単位で見直す
- 通知・定期実行変更時は `bot/` と `doc/09_schedule_and_issues.md` を合わせて確認する
- 仕様や責務が変わる変更では、コードだけでなく `README.md` と `doc/` も同じ作業で同期する

## ディレクトリ責務

- `frontend/`: 運営メンバー向け Web UI。権限制御は UX 補助で、最終判定は API 側
- `api/`: 認証済み API、RBAC を軸にした認可、永続化、監査ログの責務を持つ
- `bot/`: Discord 通知と運用補助。正データ管理や最終認可判定は持たない
- `doc/`: 機能、画面、権限、ERD、ログ、運用の設計書
- `.kiro/`: `cc-sdd` の設定、steering、specs を置く

## 参照優先順

1. ユーザーの依頼
2. `.kiro/steering/product.md`
3. `.kiro/steering/tech.md`
4. `.kiro/steering/structure.md`
5. 関連する `doc/*.md`
6. 各ディレクトリの `README.md`

## Codex での `cc-sdd` ワークフロー

- 初回や方針更新時: `/prompts:kiro-steering`, `/prompts:kiro-steering-custom`
- 仕様化: `/prompts:kiro-spec-init "説明"` → `/prompts:kiro-spec-requirements {feature}` → `/prompts:kiro-spec-design {feature}` → `/prompts:kiro-spec-tasks {feature}`
- 実装: `/prompts:kiro-spec-impl {feature} [tasks]`
- 確認: `/prompts:kiro-spec-status {feature}`, `/prompts:kiro-validate-gap {feature}`, `/prompts:kiro-validate-design {feature}`, `/prompts:kiro-validate-impl {feature}`

## 開発ルール

- 思考は英語でもよいが、対話とプロジェクト内 Markdown は日本語を基本とする
- Requirements → Design → Tasks → Implementation の順を崩さない
- `-y` は意図的にレビューを省略する場合にだけ使う
- 変更が複数ディレクトリへ波及するなら、最初の1ファイル修正で止めずに関連箇所まで完結させる
