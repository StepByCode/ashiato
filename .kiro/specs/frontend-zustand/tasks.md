# frontend-zustand Tasks

## 1. ストア基盤を追加

- `frontend/src/store/workflow-store.ts` を新規作成し、meeting/publicity 状態と actions、`persist` 設定を定義する
- 初期値は現行 UI と同じ文面・媒体構成を採用する
- 完了条件: 単体で import でき、型エラーがない

## 2. MeetingBoard をストア接続

- `meeting-board.tsx` で `meetingAt` と `meetUrl` の状態を store から取得・更新するように置き換える
- カレンダー表示・カウントダウン表示の挙動が従来と一致すること

## 3. PublicityBoard をストア接続

- `publicity-board.tsx` でテンプレートと各チャネル状態を store から読み書きするように置き換える
- ボタン押下による状態遷移と文字数カウントの挙動が従来と一致すること

## 4. ドキュメント反映

- `frontend/README.md` に Zustand の利用場所とストア参照方法を追記する
- 完了条件: 読むだけで開発者がストア構造を把握できる
