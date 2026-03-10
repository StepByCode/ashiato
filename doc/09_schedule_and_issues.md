🚀 プロジェクトIssue管理テンプレート（工数見積もり付き）

📌 前提条件

1人日 = 8時間

想定チーム規模：3名（例：3〜8名、スキル混在）

見積単位：

XS = 2h

S = 3h

M = 5h

L = 7h

XL = 11h以上

優先度ラベル：

🔴 P0：MVP必須

🟡 P1：早期追加

🟢 P2：中期対応

⚪ P3：将来構想

3月 スケジュール

A:きど(BE)
B:なかい(FE)

C:たかの(FE)

凡例

内容

○

作業可能(5時間)

△

作業可能(2～3時間)

×

作業不可

月

火

水

木

金

土

日

1<br>A: ○<br>B: △<br>C: ○

2<br>A: ○<br>B: △<br>C: ×

3<br>A: ○<br>B: ○<br>C: ○

4<br>A: ○<br>B: △<br>C: ×

5<br>A: ○<br>B: ○<br>C: ○

6<br>A: ○<br>B: △<br>C: △

7<br>A: △<br>B: ×<br>C: ×

8<br>A: △<br>B: ○<br>C: ○

9<br>A: ○<br>B: △<br>C: ×

10<br>A: ○<br>B: ○<br>C: ○

11<br>A: ○<br>B: △<br>C: △

12<br>A: ○<br>B: ○<br>C: ○

13<br>A: ○<br>B: △<br>C: ×

14<br>A: △<br>B: ×<br>C: ×

15<br>A: △<br>B: ○<br>C: △

16<br>A: ○<br>B: △<br>C: ○

17<br>A: ○<br>B: ○<br>C: ○

18<br>A: ○<br>B: △<br>C: ×

19<br>A: ○<br>B: ○<br>C: ○

20<br>A: ○<br>B: △<br>C: △

21<br>A: △<br>B: ×<br>C: ×

22<br>A: △<br>B: ○<br>C: ×

23<br>A: ○<br>B: △<br>C: ○

24<br>A: ○<br>B: ○<br>C: ○

25<br>A: ○<br>B: △<br>C: ×

26<br>A: ○<br>B: ○<br>C: ○

27<br>A: ○<br>B: △<br>C: △

28<br>A: △<br>B: ×<br>C: ×

29<br>A: △<br>B: ○<br>C: ○

30<br>A: ○<br>B: △<br>C: ○

31<br>A: ○<br>B: ○<br>C: △

🏁 Sprint別 Issue一覧テンプレート

🏗️ Sprint 1：インフラ

🔢 推定合計：17h

#

タイトル

工数

時間

優先度

担当

備考

#A1,C3

OIDC基盤（ZITADEL連携）

L

7h

🔴 P0

BE

JWT検証基盤とログイン導線

#C1

FEサービス初期構成（Vercel）

M

5h

🟡 P1

FE

フロント配信設定

#C2

BEサービス初期構成（Coolify）

M

5h

🟡 P1

BE

APIデプロイ設定

Sprint 合計

17h

（P0のみ）

7h

🏗️ Sprint 2：コア機能（mock）

🔢 推定合計：23h

#

タイトル

工数

時間

優先度

担当

備考

#B1

タスク新規作成（テンプレート化）

XL

20h

🔴 P0

FE

担当者/承認者/締切/URL入力

#B2

タスク確認

S

3h

🔴 P0

FE

未完了/完了の可視化

Sprint 合計

23h

（P0のみ）

23h

🏗️ Sprint 3：1,2の繋ぎこみ

🔢 推定合計：21h

#

タイトル

工数

時間

優先度

担当

備考

#B1

タスクAPI実装（作成/更新/完了）

XL

14h

🔴 P0

BE

OpenAPI + Echo + sqlc

#B2

フロント/API結合・承認導線接続

L

7h

🔴 P0

BE・FE

E2E確認込み

Sprint 合計

21h

（P0のみ）

21h

🏗️ Sprint 4：機能追加

🔢 推定合計：19h

#

タイトル

工数

時間

優先度

担当

備考

#B3

広報文章作成

L

7h

🟡 P1

FE

テンプレ文面生成

#A2,B4,C4

定例情報確認（Discord botサービス）

XL

12h

🟢 P2

FE・BE

日時/Meet URL確認 + リマインド

Sprint 合計

19h

（P0のみ）

0h

📊 工数サマリー

Sprint別工数

xychart-beta
    title "Sprint別 工数"
    x-axis ["S1", "S2", "S3", "S4"]
    y-axis "時間 (h)" 0 --> 30
    bar [17, 23, 21, 19]

優先度別内訳

pie title 優先度別 工数内訳
    "🔴 P0" : 51
    "🟡 P1" : 17
    "🟢 P2" : 12

🗓️ フェーズ分解テンプレート

flowchart TB
    subgraph MVP["🔴 MVPフェーズ"]
        A["Sprint 1\n17h"]
        B["Sprint 2\n23h"]
        C["Sprint 3\n21h"]
        TOT["合計 61h\n約1.1週間想定（3名）"]
        A --> B --> C --> TOT
    end

    subgraph EXT["🟡 拡張フェーズ"]
        EXT_TOT["Sprint 4\n19h\n約0.3週間想定（3名）"]
    end

    MVP --> EXT

⏱️ クリティカルパス可視化テンプレート

gantt
    title 主要Issue 工数
    dateFormat X
    axisFormat %s h

    section 基盤
    #001 OIDC基盤構築         :0, 7
    #002 DB設計・マイグレーション:0, 8

    section コア機能
    #010 API開発              :crit, 7, 14
    #011 UI開発               :crit, 7, 16
    #012 結合試験             :crit, 21, 7

📈 規模感サマリー表

区分

Issue数

工数合計

並行3名想定

並行4名想定

🔴 MVP

5件

51h

約0.5週間

約0.4週間

🟡 P1

3件

17h

約0.2週間

約0.1週間

🟢 P2

1件

12h

約0.1週間

約0.1週間

合計

9件

80h

約0.8週間

約0.5週間

🧮 見積もり計算式（コピー用）

総工数 ÷ (人数 × 1日8h × 週5日)

例：

80h ÷ (3人 × 40h/週) = 約0.7週間

※ レビュー・テスト込みなら ×1.3〜1.5 を推奨

🎯 実務で使う際の運用ルール（推奨）

1. 見積もり精度向上

UIはデザイン確定後に再見積

外部サービス連携（Discord/OIDC）はバッファ20〜30%確保

新規メンバーアサイン時は+30%補正

2. スプリント設計

MVPは P0のみ抽出

FE最大工数タスクは早期分解

API設計 → UI → 結合の順に並べる

3. 並行開発最適化

BE専任1名以上を確保

FEは画面実装と結合試験を分離

Infraは最初に完了させる

🔎 使い方まとめ

全Issueを書き出す

工数ラベルを割り当てる

P0のみ抽出 → MVP期間算出

1.3倍補正をかけて現実的スケジュール化

クリティカルパスをMermaidで可視化
