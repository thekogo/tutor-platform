import { test, expect } from "@playwright/test";
import { newApiContext } from "../lib/api";
import { LEARNER_STATE, OWNER_STATE } from "../lib/env";

test.describe("/admin is Owner-only", () => {
  test("non-owner Learner cannot see the admin page", async ({ page }) => {
    const response = await page.goto("/admin");

    // Either redirected away, or a 403/404 page; never the admin UI.
    await expect(page.getByTestId("admin-page")).toHaveCount(0);
    const redirected = !new URL(page.url()).pathname.startsWith("/admin");
    const denied = response !== null && [403, 404].includes(response.status());
    const forbiddenShown = await page.getByTestId("forbidden").isVisible();
    expect(redirected || denied || forbiddenShown, `landed on ${page.url()}`).toBe(true);
  });

  test("non-owner Learner gets 403 from admin API", async () => {
    const api = await newApiContext(LEARNER_STATE);
    for (const path of ["/api/admin/learners", "/api/admin/invites", "/api/admin/flags"]) {
      const res = await api.get(path);
      expect(res.status(), path).toBe(403);
    }
    expect((await api.post("/api/admin/invites")).status()).toBe(403);
    await api.dispose();
  });

  test.describe("Owner (control)", () => {
    test.use({ storageState: OWNER_STATE });

    test("Owner can open the admin page", async ({ page }) => {
      await page.goto("/admin");
      await expect(page.getByTestId("admin-page")).toBeVisible();
    });
  });
});
