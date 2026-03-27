import { expect, test } from "@playwright/test";

import {
  DEFAULT_MEET_URL,
  DEFAULT_TEMPLATE,
  computeDefaultMeetingAt,
  readWorkflowState,
  resetState,
} from "./utils";

test.describe("定例 (workflow-store)", () => {
  test.beforeEach(async ({ page }) => {
    await resetState(page, "/meeting");
  });

  test("定例日時とMeetリンクがリロード後も保持される", async ({ page }) => {
    await page.locator("#meeting-datetime").click();
    await page.getByLabel("定例の時間").selectOption("10");
    await page.getByLabel("定例の分").selectOption("15");

    const nextMeetUrl = "https://meet.example.com/e2e";
    await page.getByLabel("Meetのリンク貼付").fill(nextMeetUrl);

    const savedBeforeReload = await readWorkflowState(page);
    const savedMeetingAt = savedBeforeReload?.meetingAt ?? "";
    expect(savedMeetingAt).not.toBe("");

    await page.reload();

    await expect(page.getByLabel("Meetのリンク貼付")).toHaveValue(nextMeetUrl);
    await expect.poll(async () => {
      const state = await readWorkflowState(page);
      return state?.meetingAt;
    }).toBe(savedMeetingAt);
  });

  test("リセットで定例情報が初期化される", async ({ page }) => {
    await page.locator("#meeting-datetime").click();
    await page.getByLabel("定例の時間").selectOption("9");
    await page.getByLabel("定例の分").selectOption("45");
    await page.getByLabel("Meetのリンク貼付").fill("https://reset.example.com");

    await page.getByTestId("reset-meeting").click();

    const expectedMeetingAt = computeDefaultMeetingAt();
    await expect.poll(async () => {
      const state = await readWorkflowState(page);
      return state?.meetingAt;
    }).toBe(expectedMeetingAt);

    await expect.poll(async () => {
      const state = await readWorkflowState(page);
      return state?.meetUrl;
    }).toBe(DEFAULT_MEET_URL);

    await expect(page.getByLabel("Meetのリンク貼付")).toHaveValue(DEFAULT_MEET_URL);
  });
});

test.describe("広報 (workflow-store)", () => {
  test.beforeEach(async ({ page }) => {
    await resetState(page, "/publicity");
  });

  test("テンプレートとチャンネル状態がリロード後も保持される", async ({ page }) => {
    const template = "Playwrightで更新するテンプレート";
    await page.locator("#publicity-template").fill(template);

    const xChannel = page.getByTestId("publicity-channel-x");
    await xChannel.getByRole("button", { name: "Doneにする" }).click();
    await xChannel.getByRole("button", { name: "はい" }).click();

    await page.reload();

    await expect(page.locator("#publicity-template")).toHaveValue(template);
    await expect(xChannel.getByText("Done")).toBeVisible();

    await expect.poll(async () => {
      const state = await readWorkflowState(page);
      const x = state?.channels?.find((channel: { id: string }) => channel.id === "x");
      return x?.state;
    }).toBe("done");
  });

  test("リセットで広報情報が初期化される", async ({ page }) => {
    const xChannel = page.getByTestId("publicity-channel-x");
    await xChannel.getByRole("button", { name: "Doneにする" }).click();
    await xChannel.getByRole("button", { name: "はい" }).click();

    await page.getByTestId("reset-publicity").click();

    await expect.poll(async () => {
      const state = await readWorkflowState(page);
      return state?.template;
    }).toBe(DEFAULT_TEMPLATE);

    await expect.poll(async () => {
      const state = await readWorkflowState(page);
      const channelStates = Object.fromEntries(
        (state?.channels ?? []).map((channel: { id: string; state: string }) => [channel.id, channel.state])
      );
      return channelStates;
    }).toEqual({ x: "in_progress", instagram: "in_progress", facebook: "done" });

    await expect(page.locator("#publicity-template")).toHaveValue(DEFAULT_TEMPLATE);
    await expect(xChannel.getByText("in progress")).toBeVisible();
    await expect(page.getByTestId("publicity-channel-instagram").getByText("in progress")).toBeVisible();
    await expect(page.getByTestId("publicity-channel-facebook").getByText("Done")).toBeVisible();
  });
});
