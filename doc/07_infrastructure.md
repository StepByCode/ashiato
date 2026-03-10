Infrastructure Guidelines

System Architecture

%%{init: {
  "theme": "base",
  "themeVariables": {
    "background": "#FFFFFF",
    "primaryColor": "#EEF5FF",
    "primaryTextColor": "#0A2540",
    "primaryBorderColor": "#3572EF",
    "lineColor": "#6B7280",
    "tertiaryColor": "#F6FAFF"
  },
  "flowchart": { "curve": "basis", "nodeSpacing": 60, "rankSpacing": 80 }
}}%%
flowchart LR

    %% =======================
    %% User / Edge Layer
    %% =======================
    USER["Users<br/>(StepByCode運営)"]:::entry
    subgraph EDGE["Edge Layer"]
        CDN["Vercel CDN"]:::edge
        EDGE_FUNC["Vercel Edge<br/>(Rewrite / Header Injection)"]:::svc
    end

    %% =======================
    %% Application Layer
    %% =======================
    subgraph APP["Application Layer"]
        FE["Frontend Service<br/>(Next.js on Vercel)"]:::app
        API["API Service<br/>(Go + Echo on Coolify)"]:::app
        BOT["Discord Bot Service<br/>(Go on Coolify)"]:::app
    end

    %% =======================
    %% Identity Layer
    %% =======================
    subgraph IDENTITY["Identity Layer"]
        IDP["ZITADEL Selfhosted<br/>(OIDC)"]:::svc
    end

    %% =======================
    %% Data Layer
    %% =======================
    subgraph DATA["Data Layer"]
        RDB["Primary PostgreSQL"]:::db
        LOGS["Log Storage"]:::data
    end

    DISCORD["Discord API"]:::svc

    USER --> CDN
    CDN --> EDGE_FUNC
    EDGE_FUNC --> FE
    FE --> IDP
    FE --> API
    API --> RDB
    API --> LOGS
    BOT --> LOGS
    API --> BOT
    API --> DISCORD
    BOT --> DISCORD

    classDef edge fill:#fff2e6,stroke:#ff7a00,color:#4a2f00;
    classDef svc fill:#eef9f1,stroke:#2a9d8f,color:#073b4c;
    classDef app fill:#e8f1ff,stroke:#3572ef,color:#0a2540;
    classDef entry fill:#eaf7ff,stroke:#0091d5,color:#073b4c;
    classDef db fill:#f5faff,stroke:#2b6cb0,color:#0a2a4a;
    classDef data fill:#fff0fb,stroke:#b1008a,color:#4a0040;

System Components

1️⃣ Edge Layer

CDN / Edge Network

静的アセット配信

TLS終端

キャッシュ

地理的最適化

Edge Functions

役割：

共通ヘッダー注入

キャッシュ制御

静的ルーティング最適化

設計意図：

フロント配信負荷をEdgeへ集約

アプリ本体の応答を安定化

2️⃣ Application Layer

Application Service

Frontend Service (Vercel)

API Service (Coolify)

Discord Bot Service (Coolify)

責務：

Frontend: 画面表示 / OIDCログイン導線

API: タスク・承認・監査ログの中核処理

Bot: リマインド通知・Discord連携

3️⃣ Identity Layer（分離推奨）

Auth Middleware

API側JWT検証 + RBAC/ABAC最終判定

Identity Provider

ZITADEL Selfhosted

設計原則：

原則

理由

アプリと分離

ID基盤を交換可能にするため

独立スケール

ログイン集中時に追従するため

DB直接アクセス禁止

認証責務を分離するため

4️⃣ Data Layer

コンポーネント

用途

RDB

タスク / 承認 / 組織 / 監査のトランザクション管理

Log Storage

アプリログ / アクセスログ / 監査ログの保存

設計意図

小規模運営に必要な構成へ絞り、運用負荷を下げる

Edgeで先に配信と共通処理を行い、APIは業務処理に集中

IdentityとApplicationの分離

なぜ分離するか？

OIDC基盤の独立運用を可能にする

権限判定の責任境界を明確にする

Containerを使う理由（Lambdaではなく）

Lambdaの課題

コールドスタートで通知遅延が発生しやすい

長時間接続・常駐処理の実装が煩雑

Containerの利点

Bot常駐処理と相性がよい

API/Botを同一運用で管理しやすい

Discord通知責務を分離する理由

通知失敗時の再試行をAPI本体から分離できる

レート制限対応をBot側に閉じ込められる

Cacheを利用する理由

現フェーズでは専用Cacheは非採用

必要時にセッション/レート制限用途で追加予定

スケーリング戦略

水平スケール

レイヤー

方法

Edge

Vercelの自動スケール

App

CoolifyでAPI/Botを個別スケール

DB

接続上限監視 + 必要時Read Replica

Bot

キュー長に応じたレプリカ追加

局所アクセス対策

ログイン時間帯のピークを監視

Connection Pool管理

通知再試行のバックオフ制御
