# frontend

`frontend/` は **Ashiato のWebフロントエンド** を実装するディレクトリです。  
運営メンバーがログインし、Latest/Pastフローを追えるUIを提供します。

## 役割

- ログイン後の業務画面（定例・作成・広報）の提供
- APIの表示状態管理（未完了/完了など）
- 権限に応じたUI制御（表示・非表示・無効化）
- レスポンシブ対応（Desktop / Mobile）

## 想定技術スタック

- React / Next.js
- TypeScript
- Zustand
- shadcn/ui
- Playwright（E2E）

## 推奨ディレクトリ構成（着手時の目安）

```text
frontend/
├── src/
│   ├── app/          # ルーティング層
│   ├── features/     # 機能単位モジュール
│   ├── components/   # 共通UI
│   ├── hooks/        # 共通フック
│   ├── lib/          # APIクライアント等
│   ├── stores/       # 状態管理
│   └── types/        # 型定義
├── public/           # 静的アセット
└── tests/            # E2E/統合テスト
```

## 実装方針

- 画面遷移は `doc/03_screen-flow.md` を正とする
- フロントの権限制御はUX補助であり、最終判定はAPI側で行う
- まずP0画面を固め、その後P1/P2を段階追加する

## 関連ドキュメント

- `../doc/01_feature-list.md`
- `../doc/03_screen-flow.md`
- `../doc/04_permission-design.md`
- `../doc/06_directory.md`


## Pencil

Pencil は VS Code 拡張 `highagency.pencildev` を使ってこのリポジトリで利用します。

- ルートの `ashiato.pen` を VS Code で開くと、Pencil のキャンバスとして使えます
- AI 連携はローカルの `claude` CLI を使う前提です。この環境では導入済みです
- `.vscode/extensions.json` に推奨拡張を入れてあるので、VS Code でワークスペースを開くと Pencil を案内できます
- 既存の画面トークンに合わせるときは `frontend/src/app/globals.css` の CSS 変数を参照してください
- 拡張が見えない場合は VS Code を再起動するか、拡張一覧で `Pencil` を有効化してください
