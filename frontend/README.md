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
