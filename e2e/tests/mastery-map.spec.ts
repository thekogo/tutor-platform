import { test, expect } from "@playwright/test";
import { loadFixtures } from "../lib/fixtures";

test("Mastery Map renders a node for every Concept", async ({ page }) => {
  const { courseId, conceptCount } = loadFixtures();

  await page.goto(`/courses/${courseId}/map`);
  const map = page.getByTestId("mastery-map");
  await expect(map).toBeVisible();

  // @xyflow/react renders each node as .react-flow__node; Concept nodes carry
  // data-testid="concept-node" and data-level=<MasteryLevel>.
  const conceptNodes = map.getByTestId("concept-node");
  await expect(conceptNodes.first()).toBeVisible();
  await expect(conceptNodes).toHaveCount(conceptCount);

  const level = await conceptNodes.first().getAttribute("data-level");
  expect(["unassessed", "weak", "developing", "mastered"]).toContain(level);
});
