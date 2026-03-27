# フロントエンド認証実装ガイド

## 概要

Ashiato フロントエンドは **Firebase Authentication（Email/Password）** を採用しています。
ユーザーは既存メンバーの招待によって事前に発行され、メールアドレスとパスワードでログインします。ログイン画面から自己登録はできません。

## アーキテクチャ

```text
ブラウザ
  ├── Firebase JS SDK → Firebase Auth サービス（Google 管理）
  │     └── signInWithEmailAndPassword() → ID トークン取得
  └── API リクエスト
        └── Authorization: Bearer <Firebase ID トークン>
              → API（Firebase Admin SDK でトークン検証）
```

## 環境変数

フロントエンドで必要な環境変数（`.env.local` に設定）:

| 変数 | 説明 | 例 |
| --- | --- | --- |
| `NEXT_PUBLIC_FIREBASE_API_KEY` | Firebase プロジェクトの API キー | `AIzaSy...` |
| `NEXT_PUBLIC_FIREBASE_AUTH_DOMAIN` | Firebase Auth ドメイン | `your-project.firebaseapp.com` |
| `NEXT_PUBLIC_FIREBASE_PROJECT_ID` | Firebase プロジェクト ID | `your-project` |

> **注意**: `NEXT_PUBLIC_` プレフィックスが付いた変数はブラウザに公開されます。
> API キーは Firebase のセキュリティルールで保護されており、公開しても安全です。

## ディレクトリ構成

```text
frontend/src/
├── lib/
│   ├── firebase.ts        # Firebase 初期化（シングルトン）
│   ├── auth-context.tsx   # 認証コンテキスト（AuthProvider / useAuth）
│   └── api.ts             # API クライアントヘルパー
├── app/
│   ├── layout.tsx         # AuthProvider をルートに配置
│   ├── page.tsx           # 認証ガード付きメインページ
│   ├── login/
│   │   ├── page.tsx       # ログインフォーム
│   │   └── login.css      # ログインページスタイル
│   └── members/
│       ├── page.tsx       # メンバー管理画面（一覧 + 作成フォーム）
│       └── members.css    # メンバー管理スタイル
```

## 主要モジュール

### `lib/firebase.ts` — Firebase 初期化

```typescript
import { initializeApp, getApps } from "firebase/app";
import { getAuth } from "firebase/auth";

const firebaseConfig = {
  apiKey: process.env.NEXT_PUBLIC_FIREBASE_API_KEY,
  authDomain: process.env.NEXT_PUBLIC_FIREBASE_AUTH_DOMAIN,
  projectId: process.env.NEXT_PUBLIC_FIREBASE_PROJECT_ID,
};

const app = getApps().length === 0 ? initializeApp(firebaseConfig) : getApps()[0];
export const auth = getAuth(app);
```

- `getApps().length === 0` で重複初期化を防止（Next.js の HMR 対策）

### `lib/auth-context.tsx` — 認証コンテキスト

提供する値:

| プロパティ | 型 | 説明 |
| --- | --- | --- |
| `user` | `User \| null` | ログイン中の Firebase User オブジェクト |
| `loading` | `boolean` | 認証状態の初回読み込み中かどうか |
| `getIdToken` | `() => Promise<string \| null>` | API リクエスト用の ID トークンを取得 |
| `logout` | `() => Promise<void>` | ログアウト |

### ログインフロー

1. ユーザーが `/login` にアクセス
2. メールアドレスとパスワードを入力
3. `signInWithEmailAndPassword(auth, email, password)` を呼び出し
4. 成功したら `/` にリダイレクト
5. 失敗したらエラーメッセージを表示

### 認証ガード

`page.tsx`（および認証が必要なページ）で `useAuth()` を使い、未認証なら `/login` にリダイレクト:

```typescript
const { user, loading } = useAuth();
const router = useRouter();

if (loading) return null;
if (!user) {
  router.replace("/login");
  return null;
}
```

## API リクエスト時のトークン付与

API を呼び出す際は、`getIdToken()` で取得したトークンを `Authorization` ヘッダーに付与します:

```typescript
const { getIdToken } = useAuth();

async function fetchFromAPI(path: string) {
  const token = await getIdToken();
  const res = await fetch(`${API_BASE_URL}${path}`, {
    headers: {
      Authorization: `Bearer ${token}`,
      "Content-Type": "application/json",
    },
  });
  return res.json();
}
```

## ユーザー管理

### 招待による発行

- `/invite` ページからログイン済みメンバーが新規アカウントを発行できます
- メールアドレスを入力すると API が Firebase Auth ユーザーを作成し、初回パスワードを自動生成します
- Resend が設定されていれば、招待メールで初回ログイン情報を送信します
- 招待されたユーザーはログイン後に `/profile-setup` で初期プロフィールを登録します

### ロール付与

- `OWNER_EMAILS` 環境変数に含まれるメールアドレスのユーザーは `OWNER` ロールが付与される
- それ以外のユーザーはデフォルトで `EDITOR` ロールが付与される

## トラブルシューティング

| 症状 | 原因 | 対処 |
| --- | --- | --- |
| ログインできない | 招待によるアカウント発行が未実施 | 既存メンバーに `/invite` からアカウントを発行してもらう |
| `auth/invalid-api-key` | `NEXT_PUBLIC_FIREBASE_API_KEY` が未設定または間違い | `.env.local` を確認 |
| API が 401 を返す | ID トークンが期限切れまたは未送信 | `getIdToken()` は自動で更新されるが、明示的に `getIdToken(true)` で強制更新も可能 |
| ログイン後すぐログアウトされる | `AuthProvider` がマウントされていない | `layout.tsx` で `<AuthProvider>` がルートを囲んでいるか確認 |
