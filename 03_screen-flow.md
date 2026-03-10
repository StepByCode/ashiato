🖥️ screen-flow.md テンプレート

0️⃣ 設計前提

項目

内容

対象ユーザー 

StepByCode運営

デバイス   

Responsive 

認証要否   

全面認証制               

権限制御   

RBAC

MVP範囲  

P0画面のみ                        

1️⃣ 画面一覧（Screen Inventory）

ID

画面名

役割

認証

優先度

S-01 

ログイン  

認証     

必須  

P0  

S-02 

Latest定例    

中核画面 定例    

必須 

P1  

S-03 

Latest作成

中核画面 作成  

必須  

P0  

S-04 

Latest広報    

中核画面 広報

必須  

P1  

S-05 

Past定例    

確認画面 定例 

必須  

P2 

S-06 

Past作成

確認画面 作成 

必須  

P2 

S-07 

Past広報

確認画面 広報

必須  

P2

2️⃣ 全体遷移図（高レベル）

flowchart TD
    S01[S-01: ログイン]

    subgraph Core [中核画面 - Latest]
        S02[S-02: Latest定例]
        S03[S-03: Latest作成]
        S04[S-04: Latest広報]
    end

    subgraph History [確認画面 - Past]
        S05[S-05: Past定例]
        S06[S-06: Past作成]
        S07[S-07: Past広報]
    end

    S01 --> Core
    
    S02 <--> S05
    S03 <--> S06
    S04 <--> S07

3️⃣ 認証フロー

flowchart LR
    S01[S-01: ログイン]
    subgraph Core [中核画面 - Latest]
    AuthCheck{Authenticated?}

    S01 --> AuthCheck
    AuthCheck -- No --> S01
    AuthCheck -- Yes --> Latest
    S01 --> Latest

4️⃣ CRUD標準遷移テンプレ

flowchart LR
    S02[n月の定例画面] --> 完了ボタンを押す --> S03[n月の作成画面]
    S03[n月の作成画面] --> すべての承認ボタンが押される --> S04[n月の広報画面]
    S03[n月の作成画面] --> 新規作成 --> S03[画面変化せず]
    S04[n月の広報画面] --> すべての承認ボタンが押される --> n+2月の中核画面作成
    S04[n月の作成画面] --> 新規作成 --> S04[画面変化せず]

5️⃣ 状態別分岐（State-based Flow）

flowchart TD
    Detail --> StatusCheck{Status?}
    StatusCheck -->|Draft| Edit
    StatusCheck -->|Active| ViewOnly
    StatusCheck -->|Archived| ReadOnly

6️⃣ 権限別分岐（RBAC/ABAC）

flowchart TD
    Detail --> RoleCheck{User Role}
    RoleCheck -->|Viewer| ReadOnly
    RoleCheck -->|Editor| Edit
    RoleCheck -->|Admin| AdminPanel

7️⃣  エラーフロー

//8行目以外はそのままでいいのでは？

flowchart TD
    Submit --> API
    API -->|Success| SuccessState
    API -->|ValidationError| ShowFormError
    API -->|ServerError| ErrorPage

8️⃣ 空状態 / 初回体験

flowchart TD
    S01[S-01: ログイン] --> HasData{Data exists?}
    HasData -->|No| S01[S-01: ログイン]
    HasData -->|Yes| n月の今進んでいるタスクの画面まで

９モバイル考慮（任意）

項目

Desktop

Mobile

ナビゲーション 

Sidebar 

ハンバーガーメニュー 

詳細表示    

2カラム    

1カラム       

新規作成      

ページ遷移なし   

なし（閲覧専用）    

🔟 URL設計テンプレ

/login
/dashboard
/dashboard/yyyy/mm
/create/yyyy/mm
/meeting/yyyy/mm
/influence/yyyy/mm



