# 🖥️ 画面遷移設計テンプレート

## 0. 設計前提

| 項目 | 内容 |
| --- | --- |
| 対象ユーザー | StepByCode運営 |
| デバイス | Responsive |
| 認証要否 | 全面認証制 |
| 権限制御 | RBAC |
| MVP範囲 | P0画面のみ |

## 1. 画面一覧（Screen Inventory）

| ID | 画面名 | 役割 | 認証 | 優先度 |
| --- | --- | --- | --- | --- |
| S-01 | ログイン | 認証 | 必須 | P0 |
| S-02 | Latest定例 | 中核画面（定例） | 必須 | P1 |
| S-03 | Latest作成 | 中核画面（作成） | 必須 | P0 |
| S-04 | Latest広報 | 中核画面（広報） | 必須 | P1 |
| S-05 | Past定例 | 確認画面（定例） | 必須 | P2 |
| S-06 | Past作成 | 確認画面（作成） | 必須 | P2 |
| S-07 | Past広報 | 確認画面（広報） | 必須 | P2 |

## 2. 全体遷移図（高レベル）

```mermaid
flowchart TD
    S01["S-01: ログイン"]

    subgraph Core["中核画面 - Latest"]
        S02["S-02: Latest定例"]
        S03["S-03: Latest作成"]
        S04["S-04: Latest広報"]
    end

    subgraph History["確認画面 - Past"]
        S05["S-05: Past定例"]
        S06["S-06: Past作成"]
        S07["S-07: Past広報"]
    end

    S01 --> S02
    S02 <--> S05
    S03 <--> S06
    S04 <--> S07
```

## 3. 認証フロー

```mermaid
flowchart LR
    S01["S-01: ログイン"] --> AuthCheck{"Authenticated?"}
    AuthCheck -- No --> S01
    AuthCheck -- Yes --> Latest["Latest画面"]
```

## 4. CRUD標準遷移テンプレ

```mermaid
flowchart LR
    S02["n月の定例画面"] -->|"完了ボタンを押す"| S03["n月の作成画面"]
    S03 -->|"すべての承認ボタンが押される"| S04["n月の広報画面"]
    S03 -->|"新規作成"| S03
    S04 -->|"すべての承認ボタンが押される"| N2["n+2月の中核画面作成"]
    S04 -->|"新規作成"| S04
```

## 5. 状態別分岐（State-based Flow）

```mermaid
flowchart TD
    Detail["Detail"] --> StatusCheck{"Status?"}
    StatusCheck -->|"Draft"| Edit["Edit"]
    StatusCheck -->|"Active"| ViewOnly["ViewOnly"]
    StatusCheck -->|"Archived"| ReadOnly["ReadOnly"]
```

## 6. 権限別分岐（RBAC/ABAC）

```mermaid
flowchart TD
    Detail["Detail"] --> RoleCheck{"User Role"}
    RoleCheck -->|"Viewer"| ReadOnly["ReadOnly"]
    RoleCheck -->|"Editor"| Edit["Edit"]
    RoleCheck -->|"Admin"| AdminPanel["AdminPanel"]
```

## 7. エラーフロー

```mermaid
flowchart TD
    Submit["Submit"] --> API["API"]
    API -->|"Success"| SuccessState["SuccessState"]
    API -->|"ValidationError"| ShowFormError["ShowFormError"]
    API -->|"ServerError"| ErrorPage["ErrorPage"]
```

## 8. 空状態 / 初回体験

```mermaid
flowchart TD
    S01["S-01: ログイン"] --> HasData{"Data exists?"}
    HasData -->|"No"| S01
    HasData -->|"Yes"| CurrentTask["n月の今進んでいるタスク画面"]
```

## 9. モバイル考慮（任意）

| 項目 | Desktop | Mobile |
| --- | --- | --- |
| ナビゲーション | Sidebar | ハンバーガーメニュー |
| 詳細表示 | 2カラム | 1カラム |
| 新規作成 | ページ遷移なし | なし（閲覧専用） |

## 10. URL設計テンプレ

- `/login`
- `/dashboard`
- `/dashboard/yyyy/mm`
- `/create/yyyy/mm`
- `/meeting/yyyy/mm`
- `/influence/yyyy/mm`
