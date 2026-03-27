import { NextResponse } from "next/server";

const API_BASE_URL = process.env.NEXT_PUBLIC_API_BASE_URL ?? "";
const CRON_SECRET = process.env.CRON_SECRET ?? "";

/**
 * Vercel Cron Job: 毎月1日に2ヶ月先のワークフロー期間を自動プロビジョニングする。
 *
 * vercel.json の cron 設定:
 *   "0 0 1 * *" (毎月1日 0:00 UTC)
 *
 * 例: 12月1日実行 → 2月分の Meeting(planned) + Announcement(draft) を生成
 */
export async function GET(request: Request) {
  // Vercel Cron の認証: CRON_SECRET ヘッダーで検証
  const authHeader = request.headers.get("authorization");
  if (authHeader !== `Bearer ${CRON_SECRET}`) {
    return NextResponse.json({ error: "Unauthorized" }, { status: 401 });
  }

  // 2ヶ月先の年月を計算
  const now = new Date();
  const target = new Date(now.getFullYear(), now.getMonth() + 2, 1);
  const year = target.getFullYear();
  const month = target.getMonth() + 1;

  try {
    const res = await fetch(`${API_BASE_URL}/internal/v1/workflow-periods/provision`, {
      method: "POST",
      headers: {
        "Content-Type": "application/json",
        "X-Cron-Secret": CRON_SECRET,
      },
      body: JSON.stringify({ year, month }),
    });

    if (!res.ok) {
      const body = await res.text();
      console.error(`Provision failed: ${res.status} ${body}`);
      return NextResponse.json(
        { error: "provision failed", status: res.status, detail: body },
        { status: 500 }
      );
    }

    const data = await res.json();
    console.log(`Provisioned workflow period ${year}/${String(month).padStart(2, "0")}`);
    return NextResponse.json({ ok: true, year, month, ...data });
  } catch (err) {
    console.error("Provision request failed:", err);
    return NextResponse.json({ error: "internal error" }, { status: 500 });
  }
}
