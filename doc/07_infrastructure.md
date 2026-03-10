# Infrastructure Guidelines

## System Architecture

```mermaid
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
    USER["Users<br/>(StepByCode運営)"]:::entry

    subgraph EDGE["Edge Layer"]
        CDN["Vercel CDN"]:::edge
        EDGE_FUNC["Vercel Edge<br/>(Rewrite / Header Injection)"]:::svc
    end

    subgraph APP["Application Layer"]
        FE["Frontend Service<br/>(Next.js on Vercel)"]:::app
        API["API Service<br/>(Go + Echo on Coolify)"]:::app
        BOT["Discord Bot Service<br/>(Go on Coolify)"]:::app
    end

    subgraph IDENTITY["Identity Layer"]
        IDP["ZITADEL Selfhosted<br/>(OIDC)"]:::svc
    end

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
```

## System Components

### 1. Edge Layer

#### CDN / Edge Network

- 静的アセット配信
- TLS終端
- キャッシュ制御

#### Edge Functions

- ヘッダー注入
- リライト
- 早期リジェクト（未認証アクセス）

### 2. Application Layer

#### Frontend Service

- Next.js on Vercel
- 認証フローの起点
- API呼び出しと画面描画

#### API Service

- Go + Echo
- ビジネスロジック
- 認可の最終判定
- DB永続化

#### Discord Bot Service

- 定例/締切通知
- Discord API連携
- API連携によるステータス通知

### 3. Identity Layer

- ZITADEL Selfhosted（OIDC）
- JWT発行とユーザー同定
- 将来のIdP差し替えを想定した分離

### 4. Data Layer

| コンポーネント | 用途 |
| --- | --- |
| PostgreSQL | トランザクションデータ |
| Log Storage | アプリ・監査ログ保存 |

## Environment Strategy

| 環境 | 用途 | デプロイ |
| --- | --- | --- |
| `dev` | 開発・検証 | 手動/ブランチ単位 |
| `staging` | 結合確認 | main反映前 |
| `prod` | 本番運用 | 承認後自動反映 |

## Security Baseline

- 通信はHTTPS（TLS1.2+）を強制
- APIはJWT検証を必須化
- テナント境界（organization_id）を全クエリで担保
- 監査対象操作を `audit_logs` に記録

## Scaling / Reliability

- Frontend: CDNキャッシュで配信最適化
- API: Coolify側で水平スケール可能な構成を維持
- DB: read-heavy化したらRead Replicaを検討
- 障害時: APIとBotはログを最優先で保全

## Monitoring

- アプリログ: 失敗率・5xx率・処理時間
- インフラ: CPU/Memory/再起動回数
- アラート: 高優先度は即時通知（Discord/メール）

## Release Policy

1. `doc/` で設計変更を先に確定
2. 実装PRでテストとドキュメントを同時更新
3. `staging` で確認後に `prod` へ反映
