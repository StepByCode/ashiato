# 10. フロントエンド繋ぎ込みガイド

> フロントエンド（`frontend/`）から API（`api/`）へ接続するための実装ガイド。
> 認証フロー、API クライアント設計、状態管理、型安全、エラーハンドリングを網羅する。

---

## 1. 前提

| 項目 | 値 |
|------|----|
| フロントエンド | Next.js / React / TypeScript |
| 状態管理 | Zustand |
| UI コンポーネント | shadcn/ui |
| API 仕様 | OpenAPI 3.0.3（`api/openapi/openapi.yaml`） |
| 認証方式 | OAuth2 (OIDC) + JWT Bearer Token |
| 認可の責務 | API 側が正。フロントエンドは UX 補助のみ |

**原則**: API が認証・認可・永続化の正（Single Source of Truth）。フロントエンドはトークンを付与してリクエストを送り、API のレスポンスに従って UI を制御する。

---

## 2. 認証フロー

### 2.1 概要

```
┌──────────┐     ①リダイレクト      ┌──────────────┐
│ Frontend │ ─────────────────────→ │ OIDC Provider│
│          │ ←───────────────────── │ (Pocket ID)  │
└──────────┘   ②認可コード返却      └──────────────┘
      │
      │ ③トークン取得
      ▼
┌──────────┐   ④Bearer Token付与    ┌──────────┐
│ Frontend │ ─────────────────────→ │   API    │
│          │ ←───────────────────── │          │
└──────────┘   ⑤レスポンス返却      └──────────┘
```

### 2.2 トークン管理

- 取得したアクセストークンは `sessionStorage` または認証ライブラリのメモリ内に保持する
- `localStorage` への保存は XSS リスクがあるため避ける
- トークンの有効期限を監視し、期限切れ前にリフレッシュする

### 2.3 ログイン・ログアウト

```typescript
// ログインチェック（ページ遷移時）
// トークンが無い or 期限切れ → OIDC Provider へリダイレクト

// ログアウト
// 1. トークンを破棄
// 2. Zustand ストアをリセット
// 3. ログインページへリダイレクト
```

### 2.4 ローカル開発（stub モード）

API が `AUTH_MODE=stub` の場合、`DEV_JWT_SECRET` で署名された JWT を手動生成して利用できる。
開発時は `.env.local` に API のベース URL を設定する。

```env
NEXT_PUBLIC_API_BASE_URL=http://localhost:8080
```

---

## 3. API クライアント設計

### 3.1 配置場所

```
frontend/src/lib/
├── api-client.ts      # fetch ラッパー（認証ヘッダー付与、エラーハンドリング）
├── api-types.ts       # OpenAPI から生成した型定義
└── api-endpoints.ts   # エンドポイント定義（リソースごと）
```

### 3.2 fetch ラッパー

```typescript
// frontend/src/lib/api-client.ts

const API_BASE = process.env.NEXT_PUBLIC_API_BASE_URL ?? "";

type RequestOptions = {
  method?: string;
  body?: unknown;
  params?: Record<string, string | number>;
};

export class ApiError extends Error {
  constructor(
    public status: number,
    public code: string,
    message: string,
  ) {
    super(message);
    this.name = "ApiError";
  }
}

export async function apiFetch<T>(
  path: string,
  token: string,
  options: RequestOptions = {},
): Promise<T> {
  const { method = "GET", body, params } = options;

  // クエリパラメータを組み立て
  const url = new URL(`${API_BASE}${path}`);
  if (params) {
    Object.entries(params).forEach(([k, v]) =>
      url.searchParams.set(k, String(v)),
    );
  }

  const headers: Record<string, string> = {
    Authorization: `Bearer ${token}`,
    "Content-Type": "application/json",
  };

  const res = await fetch(url.toString(), {
    method,
    headers,
    body: body ? JSON.stringify(body) : undefined,
  });

  // 204 No Content（DELETE 成功時）
  if (res.status === 204) return undefined as T;

  const json = await res.json();

  if (!res.ok) {
    throw new ApiError(res.status, json.code, json.message);
  }

  return json as T;
}
```

### 3.3 エンドポイント関数

```typescript
// frontend/src/lib/api-endpoints.ts

import { apiFetch } from "./api-client";
import type {
  MeResponse,
  MembersResponse,
  WorkflowPeriodsResponse,
  TasksResponse,
  Task,
  CreateTaskRequest,
  UpdateTaskRequest,
  Meeting,
  PutMeetingRequest,
  Announcement,
  PutAnnouncementRequest,
  PublishAnnouncementRequest,
} from "./api-types";

// ── 認証 ──
export const getMe = (token: string) =>
  apiFetch<MeResponse>("/api/v1/me", token);

// ── メンバー ──
export const getMembers = (token: string) =>
  apiFetch<MembersResponse>("/api/v1/members", token);

// ── ワークフロー ──
export const getWorkflowPeriods = (token: string) =>
  apiFetch<WorkflowPeriodsResponse>("/api/v1/workflow-periods", token);

// ── 定例（Meeting） ──
export const getMeeting = (token: string, year: number, month: number) =>
  apiFetch<Meeting>(`/api/v1/meetings/${year}/${month}`, token);

export const putMeeting = (
  token: string,
  year: number,
  month: number,
  body: PutMeetingRequest,
) => apiFetch<Meeting>(`/api/v1/meetings/${year}/${month}`, token, {
  method: "PUT",
  body,
});

// ── タスク ──
export const getTasks = (token: string, year: number, month: number) =>
  apiFetch<TasksResponse>("/api/v1/tasks", token, {
    params: { year, month },
  });

export const createTask = (token: string, body: CreateTaskRequest) =>
  apiFetch<Task>("/api/v1/tasks", token, { method: "POST", body });

export const updateTask = (token: string, id: string, body: UpdateTaskRequest) =>
  apiFetch<Task>(`/api/v1/tasks/${id}`, token, { method: "PATCH", body });

export const deleteTask = (token: string, id: string, version: number) =>
  apiFetch<void>(`/api/v1/tasks/${id}`, token, {
    method: "DELETE",
    params: { version },
  });

export const approveTask = (token: string, id: string) =>
  apiFetch<Task>(`/api/v1/tasks/${id}/approve`, token, { method: "POST" });

// ── 広報（Announcement） ──
export const getAnnouncement = (token: string, year: number, month: number) =>
  apiFetch<Announcement>(`/api/v1/announcements/${year}/${month}`, token);

export const putAnnouncement = (
  token: string,
  year: number,
  month: number,
  body: PutAnnouncementRequest,
) => apiFetch<Announcement>(`/api/v1/announcements/${year}/${month}`, token, {
  method: "PUT",
  body,
});

export const publishAnnouncement = (
  token: string,
  id: string,
  body: PublishAnnouncementRequest,
) => apiFetch<Announcement>(`/api/v1/announcements/${id}/publish`, token, {
  method: "POST",
  body,
});
```

---

## 4. 型定義の生成

OpenAPI スキーマからフロントエンド用の TypeScript 型を自動生成する。

### 4.1 推奨ツール

- **openapi-typescript**: OpenAPI YAML → TypeScript 型定義

```bash
npx openapi-typescript ../api/openapi/openapi.yaml -o src/lib/api-types.ts
```

### 4.2 主要な型（参考）

```typescript
// OrganizationRole
type OrganizationRole = "OWNER" | "EDITOR" | "VIEWER";

// TaskStatus
type TaskStatus = "todo" | "in_progress" | "done";

// ApprovalState
type ApprovalState = "not_required" | "pending" | "partially_approved" | "approved";

// MeetingStatus
type MeetingStatus = "planned" | "completed" | "cancelled";

// AnnouncementStatus
type AnnouncementStatus = "draft" | "publish_requested" | "published" | "publish_failed";

// ErrorResponse（全エラー共通）
interface ErrorResponse {
  code: string;   // "unauthorized" | "forbidden" | "not_found" | "validation_error" 等
  message: string;
}
```

---

## 5. 状態管理（Zustand ストア）

### 5.1 ストア分割方針

| ストア | 責務 |
|--------|------|
| `authStore` | ユーザー情報、トークン、ログイン状態 |
| `periodStore` | 選択中の年月、ワークフロー期間一覧 |
| `taskStore` | 選択月のタスク一覧 |
| `meetingStore` | 選択月の定例情報 |
| `announcementStore` | 選択月の広報情報 |

### 5.2 authStore の例

```typescript
// frontend/src/stores/auth-store.ts

import { create } from "zustand";
import type { ActorUser, Organization } from "@/lib/api-types";

interface AuthState {
  token: string | null;
  user: ActorUser | null;
  organization: Organization | null;
  isAuthenticated: boolean;

  setAuth: (token: string, user: ActorUser, org: Organization) => void;
  clearAuth: () => void;
}

export const useAuthStore = create<AuthState>((set) => ({
  token: null,
  user: null,
  organization: null,
  isAuthenticated: false,

  setAuth: (token, user, organization) =>
    set({ token, user, organization, isAuthenticated: true }),

  clearAuth: () =>
    set({ token: null, user: null, organization: null, isAuthenticated: false }),
}));
```

### 5.3 taskStore の例

```typescript
// frontend/src/stores/task-store.ts

import { create } from "zustand";
import type { Task } from "@/lib/api-types";

interface TaskState {
  tasks: Task[];
  loading: boolean;
  error: string | null;

  setTasks: (tasks: Task[]) => void;
  setLoading: (loading: boolean) => void;
  setError: (error: string | null) => void;
  updateTask: (updated: Task) => void;
  removeTask: (id: string) => void;
}

export const useTaskStore = create<TaskState>((set) => ({
  tasks: [],
  loading: false,
  error: null,

  setTasks: (tasks) => set({ tasks, error: null }),
  setLoading: (loading) => set({ loading }),
  setError: (error) => set({ error, loading: false }),
  updateTask: (updated) =>
    set((state) => ({
      tasks: state.tasks.map((t) => (t.id === updated.id ? updated : t)),
    })),
  removeTask: (id) =>
    set((state) => ({
      tasks: state.tasks.filter((t) => t.id !== id),
    })),
}));
```

---

## 6. エラーハンドリング

### 6.1 HTTP ステータスごとの対応

| ステータス | `code` | フロントエンドの対応 |
|-----------|--------|---------------------|
| 400 | `validation_error` | フォームにバリデーションエラーを表示 |
| 401 | `unauthorized` | トークンを破棄し、ログイン画面へリダイレクト |
| 403 | `forbidden` | 権限不足の旨を表示。操作ボタンを無効化 |
| 404 | `not_found` | リソースが存在しない旨を表示 |
| 409 | `conflict` | 楽観ロック競合。最新データを再取得してリトライを促す |
| 5xx | — | 一般的なエラーメッセージを表示 |

### 6.2 楽観ロック（Optimistic Locking）

タスクの更新・削除には `version` フィールドが必須。

```typescript
// 更新時
await updateTask(token, taskId, {
  title: "更新後タイトル",
  status: "in_progress",
  approver_ids: [...],
  version: task.version,  // 現在の version を送る
});

// 409 Conflict が返ったら
// → タスクを再取得して最新 version を反映し、ユーザーに再操作を促す
```

### 6.3 グローバルエラーハンドリング

```typescript
// API 呼び出しのラッパーフック例
import { useAuthStore } from "@/stores/auth-store";
import { ApiError } from "@/lib/api-client";

function handleApiError(error: unknown) {
  if (error instanceof ApiError) {
    if (error.status === 401) {
      useAuthStore.getState().clearAuth();
      window.location.href = "/login";
      return;
    }
    // その他のエラーは呼び出し元で処理
    throw error;
  }
  throw error;
}
```

---

## 7. API エンドポイント一覧と画面マッピング

### 7.1 エンドポイント一覧

| メソッド | パス | 画面 ID | 用途 |
|---------|------|---------|------|
| GET | `/api/v1/me` | 全画面 | ログインユーザー・組織情報の取得 |
| GET | `/api/v1/members` | S-03, S-04 | メンバー一覧（担当者・承認者の選択肢） |
| GET | `/api/v1/workflow-periods` | S-01 | 月ごとのワークフロー進捗一覧 |
| GET | `/api/v1/meetings/{year}/{month}` | S-02 | 定例情報の取得 |
| PUT | `/api/v1/meetings/{year}/{month}` | S-02 | 定例情報の作成・更新 |
| GET | `/api/v1/tasks?year=YYYY&month=MM` | S-03 | タスク一覧の取得 |
| POST | `/api/v1/tasks` | S-04 | タスク新規作成 |
| PATCH | `/api/v1/tasks/{id}` | S-03, S-04 | タスク更新 |
| DELETE | `/api/v1/tasks/{id}?version=N` | S-03 | タスク削除 |
| POST | `/api/v1/tasks/{id}/approve` | S-03 | タスク承認 |
| GET | `/api/v1/announcements/{year}/{month}` | S-05 | 広報文の取得 |
| PUT | `/api/v1/announcements/{year}/{month}` | S-05 | 広報文の作成・更新 |
| POST | `/api/v1/announcements/{id}/publish` | S-05 | 広報の公開リクエスト |

> 画面 ID は `doc/03_screen-flow.md` を参照。

### 7.2 レスポンス例

**GET `/api/v1/me`**
```json
{
  "user": {
    "id": "550e8400-e29b-41d4-a716-446655440000",
    "subject": "oidc-subject-id",
    "email": "kido@example.com",
    "name": "kido",
    "role": "OWNER"
  },
  "organization": {
    "id": "660e8400-e29b-41d4-a716-446655440000",
    "slug": "stepbycode",
    "name": "StepByCode"
  }
}
```

**GET `/api/v1/tasks?year=2026&month=4`**
```json
{
  "tasks": [
    {
      "id": "770e8400-e29b-41d4-a716-446655440000",
      "title": "connpass イベントページ作成",
      "status": "in_progress",
      "due_date": "2026-04-15",
      "reference_url": "https://connpass.com/event/...",
      "assignee_id": "550e8400-e29b-41d4-a716-446655440000",
      "created_by": "550e8400-e29b-41d4-a716-446655440000",
      "approver_ids": ["880e8400-e29b-41d4-a716-446655440000"],
      "approved_approver_ids": [],
      "approval_state": "pending",
      "version": 1,
      "year": 2026,
      "month": 4
    }
  ]
}
```

**エラーレスポンス（共通）**
```json
{
  "code": "forbidden",
  "message": "VIEWER role cannot create tasks"
}
```

---

## 8. 権限による UI 制御

フロントエンドでは UX 向上のためロールに応じて UI 要素を出し分ける。
ただし最終的な権限チェックは API 側が行う。

| 操作 | OWNER | EDITOR | VIEWER |
|------|-------|--------|--------|
| タスク一覧の閲覧 | ○ | ○ | ○ |
| タスクの作成・編集 | ○ | ○ | × |
| タスクの削除 | ○ | ○（自分が作成したもの） | × |
| タスクの承認 | ○（承認者に含まれる場合） | ○（承認者に含まれる場合） | × |
| 定例の編集 | ○ | ○ | × |
| 広報文の編集 | ○ | ○ | × |
| 広報の公開リクエスト | ○ | ○ | × |

```typescript
// 権限チェックのユーティリティ例
function canEditTask(role: OrganizationRole): boolean {
  return role === "OWNER" || role === "EDITOR";
}

function canApproveTask(userId: string, approverIds: string[]): boolean {
  return approverIds.includes(userId);
}
```

---

## 9. データ取得パターン

### 9.1 初回ロード

ページ表示時に必要なデータを並列で取得する。

```typescript
// 月別ページの初回ロード
useEffect(() => {
  const controller = new AbortController();

  async function load() {
    setLoading(true);
    try {
      const [tasksRes, meeting, announcement] = await Promise.all([
        getTasks(token, year, month),
        getMeeting(token, year, month).catch(() => null),  // 404 は null
        getAnnouncement(token, year, month).catch(() => null),
      ]);
      setTasks(tasksRes.tasks);
      setMeeting(meeting);
      setAnnouncement(announcement);
    } catch (err) {
      handleApiError(err);
    } finally {
      setLoading(false);
    }
  }

  load();
  return () => controller.abort();
}, [year, month, token]);
```

### 9.2 楽観的更新（Optimistic Update）

タスクのステータス変更など、即座にフィードバックを返したい場合は楽観的更新を使う。

```typescript
async function handleStatusChange(task: Task, newStatus: TaskStatus) {
  // 1. UI を先に更新
  const optimistic = { ...task, status: newStatus };
  taskStore.updateTask(optimistic);

  try {
    // 2. API に反映
    const updated = await updateTask(token, task.id, {
      title: task.title,
      status: newStatus,
      approver_ids: task.approver_ids,
      version: task.version,
    });
    // 3. サーバーの結果で上書き（version が更新される）
    taskStore.updateTask(updated);
  } catch (err) {
    // 4. 失敗時はロールバック
    taskStore.updateTask(task);
    handleApiError(err);
  }
}
```

---

## 10. 実装チェックリスト

- [ ] 環境変数 `NEXT_PUBLIC_API_BASE_URL` を `.env.local` に設定
- [ ] `lib/api-client.ts` — fetch ラッパー実装
- [ ] `lib/api-types.ts` — OpenAPI から型を生成
- [ ] `lib/api-endpoints.ts` — エンドポイント関数を実装
- [ ] `stores/auth-store.ts` — 認証ストア
- [ ] `stores/task-store.ts` — タスクストア
- [ ] `stores/meeting-store.ts` — 定例ストア
- [ ] `stores/announcement-store.ts` — 広報ストア
- [ ] 認証フロー（ログイン / ログアウト / トークン管理）
- [ ] 401 時の自動リダイレクト
- [ ] 409 Conflict 時の再取得とリトライ UI
- [ ] ローディング表示（スケルトン / スピナー）
- [ ] ロールに応じた UI 出し分け

---

## 関連ドキュメント

- [02_tech-stack.md](./02_tech-stack.md) — 技術スタック
- [03_screen-flow.md](./03_screen-flow.md) — 画面遷移・画面 ID
- [04_permission-design.md](./04_permission-design.md) — RBAC / ABAC 設計
- [05_erd.md](./05_erd.md) — データモデル
- [api/openapi/openapi.yaml](../api/openapi/openapi.yaml) — OpenAPI 仕様（正）
