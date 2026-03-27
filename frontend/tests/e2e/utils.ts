import { Page } from "@playwright/test";

export const DEFAULT_MEET_URL = "https://meet.google.com/abc-defg-hij";
export const DEFAULT_TEMPLATE =
  "【イベント告知】4/10(金) 20:00から定例を開催します。参加URLはプロフィールから確認できます。初参加の方も歓迎です。";

export function computeDefaultMeetingAt() {
  const nextMeeting = new Date();
  nextMeeting.setDate(nextMeeting.getDate() + 12);
  nextMeeting.setHours(20, 0, 0, 0);

  const offset = nextMeeting.getTimezoneOffset() * 60_000;
  return new Date(nextMeeting.getTime() - offset).toISOString().slice(0, 16);
}

export async function resetState(page: Page, path = "/") {
  await page.goto(path);
  await page.evaluate(() => {
    window.localStorage.clear();
  });
  await page.reload();
}

export async function readWorkflowState(page: Page) {
  return page.evaluate(() => {
    const stored = window.localStorage.getItem("workflow-store-v1");
    if (!stored) return null;

    try {
      const parsed = JSON.parse(stored);
      return parsed.state ?? null;
    } catch {
      return null;
    }
  });
}
