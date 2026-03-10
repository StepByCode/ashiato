# doc

`doc/` は **Ashiato の設計・運用ドキュメント** を管理するディレクトリです。  
実装前の判断を先に文章化し、`frontend` / `api` / `bot` で共通前提を持てる状態を目指します。

## 目的

- 仕様の認識ズレを減らす
- 実装優先度（P0/P1/P2）を明確にする
- 画面・権限・DB・インフラの整合性を保つ

## ドキュメント一覧

| ファイル | 概要 |
| --- | --- |
| `00designdoc_template.md` | 設計ドキュメント作成の土台テンプレート |
| `01_feature-list.md` | 機能一覧と優先度（MVP含む） |
| `02_tech-stack.md` | 技術選定と採用理由 |
| `03_screen-flow.md` | 画面一覧・遷移・URL設計 |
| `04_permission-design.md` | RBAC/ABAC を含む権限モデル |
| `05_erd.md` | DB設計・ERD・主要テーブル定義 |
| `06_directory.md` | 推奨ディレクトリ構成 |
| `07_infrastructure.md` | インフラ構成とスケーリング方針 |
| `08_logging.md` | ログ設計・監査ログ方針 |
| `09_schedule_and_issues.md` | スケジュールとIssue管理テンプレート |

## 運用ルール

- 仕様変更時は、影響するドキュメントを同じPRで更新する
- 新規ドキュメントは `NN_*.md` 形式（`NN` は2桁連番）で追加する
- テンプレートの値を埋めたら、未使用セクションは削除または `TBD` を明記する

## 更新の目安

- 機能追加・削除時: `01_feature-list.md` を更新
- 技術方針変更時: `02_tech-stack.md` と `07_infrastructure.md` を更新
- 画面や権限変更時: `03_screen-flow.md` と `04_permission-design.md` を更新
- DB変更時: `05_erd.md` を更新
