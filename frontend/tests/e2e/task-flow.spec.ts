import { expect, test } from "@playwright/test";

import { resetState } from "./utils";

test.describe("タスクフロー", () => {
  test.beforeEach(async ({ page }) => {
    await resetState(page);
  });

  test("作成→Done→Approve まで完了できる", async ({ page }) => {
    const taskTitle = `E2E Task ${Date.now()}`;

    await page.getByRole("button", { name: "開く" }).click();
    await page.getByPlaceholder("タスク名").fill(taskTitle);
    await page.locator("#create-task-owner").selectOption("nakai");
    await page.getByRole("button", { name: "作成" }).click();

    const card = page.getByTestId("task-card").filter({ hasText: taskTitle });
    await expect(card).toBeVisible();

    const stateToggle = card.getByTestId("state-toggle-button");
    await stateToggle.click();
    await card.getByTestId("state-confirm-yes").click();
    await expect(stateToggle).toHaveText("Done");

    await card.locator("select").first().selectOption("kido");

    const approveButton = card.getByTestId("approve-button");
    await expect(approveButton).toBeEnabled();
    await approveButton.click();
    await card.getByTestId("approve-confirm-yes").click();
    await expect(approveButton).toBeDisabled();
    await expect(approveButton).toHaveText("Approved");
  });
});
