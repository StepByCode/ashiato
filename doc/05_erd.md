05_erd.md

🗄️ DB設計書テンプレート

0️⃣ 設計観点

項目

内容

権限モデル

RBAC

ID戦略

Auto Increment

論理削除

無

監査ログ

必須

1️⃣ テーブル一覧テンプレート

ドメイン

テーブル名

役割

Phase

アカウント

users

ユーザー主体

P0

コア機能

tasks

中核リソース

P0

コア機能

approvals

関係テーブル

P0

拡張

custom_attributes

拡張属性

P2

1️⃣ テーブル一覧テンプレート

ドメイン

テーブル名

役割

Phase

アカウント

users

ユーザー主体

P0

認可

roles

ロール定義

P0

認可

user_roles

ロール付与

P0

コア機能

entities

中核リソース

P0

コア機能

entity_relations

関係テーブル

P1

補助

comments

コメント

P1

補助

logs

操作ログ

P0

通知

notifications

通知管理

P1

拡張

custom_attributes

拡張属性

P2

監査

audit_logs

監査ログ

P0

2️⃣ ERDテンプレート（抽象版）

erDiagram
    USERS ||--o{ TASKS : 
    USERS ||--o{ APPROVALS : 
    TASKS ||--o{ APPROVALS : 

    USERS {
        uuid id PK
        string zitadel_sub UK
        string email UK
        string name
    }

    TASKS {
        uuid id PK
        string title
        string url
        string status
        uuid assignee_id FK
        uuid created_by FK
        integer version
        timestamp created_at
        timestamp updated_at
        timestamp deleted_at
    }

    APPROVALS {
        uuid id PK
        uuid task_id FK
        uuid user_id FK
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



zitadel_sub

String

UNIQUE NOT NUL



email

String

UNIQUE NOT NULL



name

String

NOT NULL



Tasks

カラム

型

制約

説明

id

UUID

PK



title

String

UNIQUE NOT NUL



url

String

UNIQUE NOT NULL



status

Enum

NOT NULL

in progress/Done

assignee_id (FK)

UUID



users.id

created_by (FK)

UUID

NOT NULL

users.id

version

Integer

DEFAULT 1



created_at

Timestamp

DEFAULT now()



updated_at

Timestamp

DEFAULT now()



deleted_at

Timestamp





Approvals

カラム

型

制約

説明

id (PK)

UUID

NOT NULL



task_id (FK)

UUID

NOT NULL



user_id (FK)

UUID

NOT NULL



created_at

Timestamp

DEFAULT now()



(unique_key)

-

(task_id, user_id)





4️⃣ 権限設計テンプレート

RBAC

role.level 比較で許可判定

ABAC（任意）

{
  "subject.role": "EDITOR",
  "resource.status": "active",
  "environment.time": "<= deadline"
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

App
 └── PostgreSQL (RDB + Vector)

メリット

トランザクション整合性

シンプル

デメリット

大規模時のスケール制限

B. 外部ベクトルDB分離

App
 ├── RDB（メタデータ）
 └── Vector DB（検索専用）

メリット

高速検索・水平スケール

フィルタリング最適化

デメリット

整合性管理が必要

ベクトル格納設計パターン

🔹 パターン1：既存テーブルに直接持つ（小規模向け）

ALTER TABLE entities
ADD COLUMN embedding VECTOR(1536);

適用条件

1エンティティ = 1ベクトル

更新頻度低い

🔹 パターン2：専用ベクトルテーブル（推奨）

erDiagram
    entities {
        uuid id PK
        varchar title
    }

    embeddings {
        uuid id PK
        uuid entity_id FK
        varchar content_type
        vector embedding
        jsonb metadata
        timestamp created_at
    }

    entities ||--o{ embeddings : "持つ"

embeddings テーブル定義テンプレ

カラム

型

説明

id

UUID

PK

entity_id

UUID

紐づくリソース

content_type

VARCHAR

title/body/comment 等

embedding

VECTOR(N)

ベクトル

metadata

JSONB

フィルタ用属性

model_name

VARCHAR

使用モデル

created_at

TIMESTAMP



3️⃣ メタデータ設計（検索フィルタ用）

{
  "group_id": "uuid",
  "status": "active",
  "visibility": "public",
  "language": "ja",
  "created_by": "uuid"
}

※ RAGやマルチテナントでは必須

4️⃣ インデックス設計

pgvector（Cosine距離）

CREATE INDEX idx_embeddings_vector
ON embeddings
USING ivfflat (embedding vector_cosine_ops)
WITH (lists = 100);

HNSW（高速）

CREATE INDEX idx_embeddings_hnsw
ON embeddings
USING hnsw (embedding vector_cosine_ops);

5️⃣ クエリテンプレ

類似検索（TopK）

SELECT entity_id, 1 - (embedding <=> :query_vector) AS similarity
FROM embeddings
WHERE metadata->>'group_id' = :group_id
ORDER BY embedding <=> :query_vector
LIMIT 10;

6️⃣ 更新戦略テンプレ

戦略

説明

同期更新

レコード保存時に即生成

非同期キュー

保存→Job→生成

再生成バッチ

モデル変更時に全更新

7️⃣ RAG設計テンプレ

flowchart LR
    UserQuery --> EmbedQuery
    EmbedQuery --> VectorSearch
    VectorSearch --> ContextChunks
    ContextChunks --> LLM
    LLM --> Answer

チャンク設計指針

項目

推奨

文字数

300〜800 tokens

オーバーラップ

10〜20%

単位

意味単位（段落）

8️⃣ 多ベクトル対応

用途別に分ける：

種類

例

semantic_vector

本文検索

keyword_vector

タイトル重視

user_profile_vector

レコメンド

skill_vector

マッチング

vector_semantic VECTOR(1536),
vector_title VECTOR(1536)