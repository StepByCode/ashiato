# frontend-zustand Design

## ストア構成

- `workflow-store` に定例（meeting）と広報（publicity）のクライアント状態をまとめる
- 既存の `task-store` はそのまま並置し、workflow 用 store と同居させる
- 型
  - `ChannelState = "in_progress" | "done"`
  - `ChannelId = "x" | "instagram" | "facebook"`
  - `PublicityChannel` で各媒体の状態と補足メモを持つ
  - `WorkflowStore` に meeting と publicity の state + actions を集約
- actions
  - meeting: `setMeetingAt`, `setMeetUrl`, `resetMeeting`
  - publicity: `setTemplate`, `updateChannelState`, `resetPublicity`

## 永続化

- `zustand/middleware` の `persist` で `localStorage` に保存する
- ストアキーは `workflow-store-v1`
- シリアライズは JSON のまま、バージョン `1`

## UI からの利用

- `MeetingBoard` で `meetingAt` / `meetUrl` を store から取得し、日時・リンク変更時に actions を呼ぶ
- `PublicityBoard` で `template` / `channels` を store から読み書きし、媒体ごとの状態更新を actions に統一する
- UI 固有の一時的な状態（ダイアログの開閉やカウントダウン計算）はローカル `useState` / `useMemo` のまま保持する

## リセット方針

- meeting/publicity それぞれに初期値へ戻す `reset` を用意し、開発中の状態破棄を容易にする

## 非機能

- クライアント専用ストアとし、サーバー側では import しない
- TypeScript strict モードで型付けする
