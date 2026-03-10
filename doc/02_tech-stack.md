技術スタック

0️⃣ 前提定義（最初に決める）

項目

内容

プロダクト種別   

SaaS

想定ユーザー規模  

MAU

可用性目標     

99.9%

セキュリティ要件  

監査対応

パフォーマンス要件 

レスポンスタイム

チーム体制     

スキル分布 / 専任有                

リリース頻度    

週次

予算制約      

開発時3,000円 リリース後2,000円/月

既存資産      

なし 

1️⃣ 技術スタック構成

🖥 フロントエンド

項目

採用技術

採用理由

評価観点

備考

言語      

HTML CSS JavaScript     

 webフロントエンドの構成

型安全性 / 学習コスト / 保守性       

   

フレームワーク 

React/Next.js

バックエンドとの繋ぎ込みの点において優れるため     

エコシステム / SSR可否 / パフォーマンス 

   

UIライブラリ 

shadcn

 管理画面の構成に適しているため    

デザイン一貫性 / 開発速度           

   

状態管理    

Zustand

 軽量であるため    

スケール耐性 / Server State分離  

   

フォーム管理  

(Reactのhookにて対応)

     

パフォーマンス / バリデーション        

   

テスト     

Playwright



エンドツーエンドテストを行いたいため。     

カバレッジ / 実行速度             

  UIに関しては目視が一番！！ 

🧠 バックエンド

項目

採用技術

採用理由

評価観点

備考

言語 

Go / Terraform 

実行速度 / 軽量 

生産性 / 型安全 / 採用市場 



実行環境 

Coolify

ラーニングコストの削減

成熟度 / エコシステム 



フレームワーク 

Echo 

高速・軽量・シンプル 

構造化 / 拡張性 



API方式 

OpenAPI 

契約駆動開発 

型安全 / 柔軟性 



ORM / DBアクセス 

sqlc + pgx 

型安全なクエリ生成 

型統合 / マイグレーション管理 



認証 

OAuth2 + JWT 

スケーラブル認証 

セキュリティ / OAuth対応 



非同期処理 

Pub/Sub 

再試行 / 耐障害性 

可用性 / 分離設計 



☁ インフラ / DevOps

項目

採用技術

採用理由

評価観点

備考

ホスティング 

Coolify

コスト削減

可用性 / コスト 

コンテナ単位課金 

コンテナ 

Docker 

環境統一 / 軽量Goと相性良 

再現性 / 可搬性 

distroless推奨 

IaC 

なし

Vercel・Coolify共に自動デプロイの機能があるため





CI/CD 

CI
oxlint・vitest・Gotest
CD
Vercel・Coolify

Push→自動Build/Deploy 

自動化 / 安定性 

OIDC連携可 

監視 

Atlassian Statuspage

マネージド監視 

可観測性 / アラート精度/障害時のサブスクリプション機能



ログ管理 

Dozzle

構造化ログ 

トレーサビリティ 



CDN 

Cloudflare CDN

エッジ配信 

レイテンシ最適化 

Cloudflare Zerotrust対応



🔐 セキュリティ

項目

方針

評価観点

認証方式 

OAuth2 / OpenID Connect + JWT 

標準準拠 / 拡張性 

認可 

RBAC（将来的にABAC拡張可能設計） 

RBAC / ABAC可否 

通信 

HTTPS強制（TLS1.2+） 

TLS強制 / 証明書管理 

データ保護 

保存時暗号化（Cloud KMS） + マスキング 

暗号化 / マスキング 

脆弱性対策 

Dependabotで自動スキャン + 依存パッチ自動化 

自動検査 / パッチ管理 