# バックエンドAPI実装依頼書

## 1. 目的
フロントエンド実装（定例・作成・広報）に合わせて、バックエンドで必要なAPIを実装する。
本書は、バックエンド開発の着手に必要なエンドポイント仕様と処理要件を定義する。

## 2. 対象画面と要件サマリ
- 定例（Meeting）
  - 定例日時の取得・更新
  - Meet URLの取得・更新
- 作成（Task）
  - タスク一覧取得
  - タスク新規作成
  - タスク担当者更新
  - タスクURL更新
  - タスク状態更新（in_progress, done, approved）
- 広報（Publicity）
  - テンプレート本文の取得・更新（最大140文字）
  - チャネル進捗一覧取得
  - チャネル進捗状態更新（in_progress, done）

## 3. API共通ルール
- ベースパス: /api/v1
- 文字コード: UTF-8
- Content-Type: application/json
- 日時: ISO 8601（例: 2026-04-10T20:00:00+09:00）
- 認証: 今回は暫定で不要（将来JWT導入を想定してミドルウェア挿入可能な構造にする）
- エラーレスポンス形式:

{
  "error": {
    "code": "VALIDATION_ERROR",
    "message": "owner is required",
    "details": [
      { "field": "owner", "reason": "required" }
    ]
  }
}

- ステータスコード方針:
  - 200: 取得/更新成功
  - 201: 作成成功
  - 400: バリデーションエラー
  - 404: 対象なし
  - 409: 状態遷移競合
  - 500: サーバー内部エラー

## 4. データモデル（API視点）

### 4.1 Task
- id: string
- title: string
- owner: enum[kido, kitahara, sogo, nakai]
- state: enum[in_progress, done, approved]
- url: string
- createdAt: string (ISO)
- updatedAt: string (ISO)

### 4.2 MeetingSettings
- meetingAt: string (ISO)
- meetUrl: string
- updatedAt: string (ISO)

### 4.3 PublicityTemplate
- text: string（最大140文字）
- updatedAt: string (ISO)

### 4.4 PublicityChannel
- id: enum[x, instagram, facebook]
- name: string
- note: string
- state: enum[in_progress, done]
- updatedAt: string (ISO)

## 5. エンドポイント仕様

## 5.1 Task API

### 5.1.1 タスク一覧取得
- Method: GET
- Path: /api/v1/tasks
- Query:
  - month (任意): 2026-02 形式
- Response 200:

{
  "tasks": [
    {
      "id": "connpass",
      "title": "connpass",
      "owner": "kido",
      "state": "in_progress",
      "url": "",
      "createdAt": "2026-03-01T00:00:00+09:00",
      "updatedAt": "2026-03-01T00:00:00+09:00"
    }
  ]
}

### 5.1.2 タスク作成
- Method: POST
- Path: /api/v1/tasks
- Request:

{
  "title": "Figma",
  "owner": "kitahara"
}

- 処理要件:
  - title 必須（trim後1文字以上）
  - owner 必須
  - 初期 state は in_progress
  - url は空文字で作成
- Response 201:

{
  "task": {
    "id": "task-xxxxxxxx",
    "title": "Figma",
    "owner": "kitahara",
    "state": "in_progress",
    "url": "",
    "createdAt": "2026-03-26T10:00:00+09:00",
    "updatedAt": "2026-03-26T10:00:00+09:00"
  }
}

### 5.1.3 タスク担当者更新
- Method: PATCH
- Path: /api/v1/tasks/{taskId}/owner
- Request:

{
  "owner": "nakai"
}

- Response 200: 更新後 task

### 5.1.4 タスクURL更新
- Method: PATCH
- Path: /api/v1/tasks/{taskId}/url
- Request:

{
  "url": "https://example.com"
}

- 処理要件:
  - 空文字許可
  - URL文字列は最大2048文字
- Response 200: 更新後 task

### 5.1.5 タスク状態更新
- Method: PATCH
- Path: /api/v1/tasks/{taskId}/state
- Request:

{
  "state": "done"
}

- 処理要件:
  - state は in_progress / done / approved のみ
  - 想定遷移:
    - in_progress -> done
    - done -> in_progress
    - done -> approved
  - approved からの戻しは原則禁止（必要なら 409 を返却）
- Response 200: 更新後 task

## 5.2 Meeting API

### 5.2.1 定例設定取得
- Method: GET
- Path: /api/v1/meeting
- Response 200:

{
  "meetingAt": "2026-04-10T20:00:00+09:00",
  "meetUrl": "https://meet.google.com/abc-defg-hij",
  "updatedAt": "2026-03-26T10:00:00+09:00"
}

### 5.2.2 定例設定更新
- Method: PATCH
- Path: /api/v1/meeting
- Request:

{
  "meetingAt": "2026-04-10T20:00:00+09:00",
  "meetUrl": "https://meet.google.com/abc-defg-hij"
}

- 処理要件:
  - meetingAt はISO形式で受理
  - meetUrl は空文字許可
- Response 200: 更新後 meeting

## 5.3 Publicity API

### 5.3.1 テンプレート取得
- Method: GET
- Path: /api/v1/publicity/template
- Response 200:

{
  "text": "【イベント告知】...",
  "updatedAt": "2026-03-26T10:00:00+09:00"
}

### 5.3.2 テンプレート更新
- Method: PATCH
- Path: /api/v1/publicity/template
- Request:

{
  "text": "【イベント告知】4/10(金) 20:00から定例を開催します..."
}

- 処理要件:
  - text 最大140文字
- Response 200: 更新後 template

### 5.3.3 チャネル一覧取得
- Method: GET
- Path: /api/v1/publicity/channels
- Response 200:

{
  "channels": [
    {
      "id": "x",
      "name": "X",
      "note": "投稿文の最終チェック",
      "state": "in_progress",
      "updatedAt": "2026-03-26T10:00:00+09:00"
    }
  ]
}

### 5.3.4 チャネル状態更新
- Method: PATCH
- Path: /api/v1/publicity/channels/{channelId}/state
- Request:

{
  "state": "done"
}

- 処理要件:
  - state は in_progress / done のみ
- Response 200: 更新後 channel

## 6. 処理上の補足
- 画面表示速度のため、Task一覧・Publicity一覧は200ms以内の応答を目標とする。
- 更新系APIは更新後リソースを返却し、フロント側が即時再描画できる形にする。
- 将来的な認証導入に備え、ユーザーIDでデータ分離可能なスキーマ設計とする。

## 7. 最低限必要な実装順（推奨）
1. GET /tasks, POST /tasks, PATCH /tasks/{id}/state
2. PATCH /tasks/{id}/owner, PATCH /tasks/{id}/url
3. GET/PATCH /meeting
4. GET/PATCH /publicity/template
5. GET /publicity/channels, PATCH /publicity/channels/{id}/state

## 8. 受け入れ条件
- フロント3画面（定例・作成・広報）で、初期表示と主要更新操作がすべてAPI経由で動作すること。
- バリデーションエラー時に、4xx とエラーJSONが返ること。
- 状態遷移の不正操作で 409 が返ること。
