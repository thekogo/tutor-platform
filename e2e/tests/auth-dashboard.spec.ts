import { test, expect } from "@playwright/test";

test.describe("login and dashboard", () => {
  test("unauthenticated visitor is sent to the login page", async ({ browser }) => {
    const context = await browser.newContext({ storageState: { cookies: [], origins: [] } });
    const page = await context.newPage();
    await page.goto("/");
    await expect(page).toHaveURL(/\/login/);
    await expect(page.getByTestId("login-page")).toBeVisible();
    await context.close();
  });

  test("logged-in Learner sees a Recommendation or the Placement Test prompt", async ({ page }) => {
    await page.goto("/");
    await expect(page).not.toHaveURL(/\/login/);

    // A brand-new Learner has only Unassessed Concepts, so the dashboard shows either
    // the top Recommendation (often kind=diagnose) or the Placement Test prompt.
    const recommendation = page.getByTestId("top-recommendation");
    const placement = page.getByTestId("placement-prompt");
    await expect(recommendation.or(placement).first()).toBeVisible();

    if (await recommendation.isVisible()) {
      // A Recommendation is always shown with its reason path.
      await expect(recommendation.getByTestId("recommendation-path")).toBeVisible();
    }
  });
});
