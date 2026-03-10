05_erd.md

🗄️ DB設計書テンプレート

0️⃣ 設計観点

項目

内容

権限モデル

RBAC + ABAC

ID戦略

UUID

論理削除

有（tasks中心）

監査ログ

必須

1️⃣ テーブル一覧テンプレート

ドメイン

テーブル名

役割

Phase

アカウント

users

運営メンバー主体

P0

組織

organizations

組織スコープ境界

P0

組織

organization_members

組織内ロール付与

P0

コア機能

tasks

進捗管理の中核リソース

P0

コア機能

task_approvals

タスク承認の関係テーブル

P0

監査

audit_logs

監査ログ

P0

1️⃣ テーブル一覧テンプレート

ドメイン

テーブル名

役割

Phase

アカウント

users

運営メンバー主体

P0

認可

roles

ロール定義

P0

認可

user_roles

ロール付与

P0

組織

organizations

組織情報

P0

組織

organization_members

組織スコープ権限

P0

コア機能

tasks

中核リソース

P0

コア機能

task_approvals

承認フロー管理

P0

補助

meeting_infos

定例情報管理（日時 / Meet URL）

P2

補助

announcement_templates

広報テンプレート管理

P1

通知

discord_accounts

Discordアカウント紐づけ

P2

通知

reminders

リマインド送信キュー

P2

監査

audit_logs

監査ログ

P0

2️⃣ ERDテンプレート（抽象版）

erDiagram
    ORGANIZATIONS ||--o{ ORGANIZATION_MEMBERS : contains
    USERS ||--o{ ORGANIZATION_MEMBERS : belongs_to

    USERS ||--o{ TASKS : assignee
    USERS ||--o{ TASKS : created_by
    TASKS ||--o{ TASK_APPROVALS : requires
    USERS ||--o{ TASK_APPROVALS : approves

    USERS ||--o{ MEETING_INFOS : updates
    USERS ||--o{ ANNOUNCEMENT_TEMPLATES : edits
    USERS ||--o{ DISCORD_ACCOUNTS : links
    TASKS ||--o{ REMINDERS : triggers

    USERS ||--o{ AUDIT_LOGS : acts
    ORGANIZATIONS ||--o{ TASKS : owns

    USERS {
        uuid id PK
        string zitadel_sub UK
        string discord_user_id UK
        string email UK
        string name
        timestamp created_at
        timestamp updated_at
    }

    ORGANIZATIONS {
        uuid id PK
        string name
        timestamp created_at
        timestamp updated_at
    }

    ORGANIZATION_MEMBERS {
        uuid id PK
        uuid organization_id FK
        uuid user_id FK
        string scope_role
        timestamp created_at
    }

    TASKS {
        uuid id PK
        uuid organization_id FK
        string title
        string phase
        string status
        date due_date
        string reference_url
        uuid assignee_id FK
        uuid created_by FK
        integer version
        timestamp created_at
        timestamp updated_at
        timestamp deleted_at
    }

    TASK_APPROVALS {
        uuid id PK
        uuid task_id FK
        uuid approver_id FK
        timestamp approved_at
        timestamp created_at
    }

3️⃣ カラム定義テンプレート

users

カラム

型

制約

説明

id

UUID

PK

内部ID

zitadel_sub

String

UNIQUE NOT NULL

OIDC subject

discord_user_id

String

UNIQUE

Discord連携ID

email

String

UNIQUE NOT NULL

連絡先

name

String

NOT NULL

表示名

created_at

Timestamp

DEFAULT now()

作成日時

updated_at

Timestamp

DEFAULT now()

更新日時

Tasks

カラム

型

制約

説明

id

UUID

PK

タスクID

organization_id (FK)

UUID

NOT NULL

organizations.id

title

String

NOT NULL

タスク名

phase

Enum

NOT NULL

meeting/create/pr

status

Enum

NOT NULL

todo/in_progress/done

due_date

Date

NOT NULL

期限日

reference_url

String

NULL

関連URL（任意）

assignee_id (FK)

UUID

NOT NULL

users.id

created_by (FK)

UUID

NOT NULL

users.id

version

Integer

DEFAULT 1

楽観ロック

created_at

Timestamp

DEFAULT now()

作成日時

updated_at

Timestamp

DEFAULT now()

更新日時

deleted_at

Timestamp

NULL

論理削除

Approvals

カラム

型

制約

説明

id (PK)

UUID

NOT NULL

承認ID

task_id (FK)

UUID

NOT NULL

tasks.id

approver_id (FK)

UUID

NOT NULL

users.id

approved_at

Timestamp

NULL

承認時刻

created_at

Timestamp

DEFAULT now()

作成日時

(unique_key)

-

(task_id, approver_id)

同一ユーザーの重複承認防止

4️⃣ 権限設計テンプレート

RBAC

role.level 比較で許可判定

ABAC（任意）

{
  "tenant boundary": "resource.organization_id == user.organization_id",
  "role level": "user.scope_role.level >= required_level",
  "abac condition": "resource.status != confirmed && now <= due_date"
}

テーブル

役割

policies

条件定義

policy_logs

評価ログ

🧠 ベクトルDB設計テンプレート

アーキテクチャ選択パターン

A. 同一DB内（pgvector）

現行スコープ外（将来検討）。

メリット

将来の検索拡張に流用しやすい

デメリット

MVPの目的（進捗可視化）に不要

B. 外部ベクトルDB分離

現行スコープ外（将来検討）。

メリット

将来の高度検索でスケールしやすい

デメリット

運用コストが増える

ベクトル格納設計パターン

🔹 パターン1：既存テーブルに直接持つ（小規模向け）

-- 現フェーズでは未実装方針
-- ALTER TABLE tasks ADD COLUMN embedding VECTOR(1536);

適用条件

将来、自然言語検索を導入する場合のみ検討

🔹 パターン2：専用ベクトルテーブル（推奨）

erDiagram
    TASKS {
        uuid id PK
        varchar title
    }

    EMBEDDINGS {
        uuid id PK
        uuid task_id FK
        vector embedding
        jsonb metadata
        timestamp created_at
    }

    TASKS ||--o{ EMBEDDINGS : "将来検討"

embeddings テーブル定義テンプレ

カラム

型

説明

id

UUID

PK

task_id

UUID

紐づくタスク

content_type

VARCHAR

title/body 等

embedding

VECTOR(N)

現行未使用

metadata

JSONB

将来の検索フィルタ用

model_name

VARCHAR

将来の埋め込みモデル名

created_at

TIMESTAMP

作成日時

3️⃣ メタデータ設計（検索フィルタ用）

{
  "organization_id": "uuid",
  "phase": "meeting",
  "status": "done",
  "created_by": "uuid"
}

※ 現行スコープ外、将来検討

4️⃣ インデックス設計

pgvector（Cosine距離）

-- 現フェーズでは作成しない方針
-- CREATE INDEX idx_embeddings_vector ON embeddings USING ivfflat (embedding vector_cosine_ops);

HNSW（高速）

-- 現フェーズでは作成しない方針
-- CREATE INDEX idx_embeddings_hnsw ON embeddings USING hnsw (embedding vector_cosine_ops);

5️⃣ クエリテンプレ

類似検索（TopK）

-- 現フェーズでは実装しない方針
-- SELECT task_id FROM embeddings ORDER BY embedding <=> :query_vector LIMIT 10;

6️⃣ 更新戦略テンプレ

戦略

説明

同期更新

現フェーズでは未採用

非同期キュー

将来導入時に候補

再生成バッチ

将来導入時に候補

7️⃣ RAG設計テンプレ

flowchart LR
    UserQuery --> ScopeCheck
    ScopeCheck -->|Current| TaskSearch
    ScopeCheck -->|Future| VectorSearch

チャンク設計指針

項目

推奨

文字数

現フェーズでは未定義

オーバーラップ

現フェーズでは未定義

単位

現フェーズでは未定義

8️⃣ 多ベクトル対応

用途別に分ける：

種類

例

semantic_vector

現行では未使用

keyword_vector

現行では未使用

user_profile_vector

現行では未使用

skill_vector

現行では未使用

-- 現フェーズ未採用
-- vector_semantic VECTOR(1536),
-- vector_title VECTOR(1536)
