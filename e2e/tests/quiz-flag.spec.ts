import { test, expect } from "@playwright/test";
import { loadFixtures } from "../lib/fixtures";

test("start a Quiz from a Concept page, answer, see the Explanation, Flag the Question", async ({
  page,
}) => {
  const { quizConceptId } = loadFixtures();
  expect(
    quizConceptId,
    "No Concept has Active Questions. E2E setup seeds them via MCP (lib/seed-questions.ts); check the setup log.",
  ).toBeTruthy();

  // Concept page: Lesson + start Quiz
  await page.goto(`/concepts/${quizConceptId}`);
  await expect(page.getByRole("heading", { level: 1 })).toBeVisible();
  await page.getByTestId("start-quiz").click();

  // Session page: one Question at a time
  await expect(page).toHaveURL(/\/sessions\/[0-9a-f-]{36}/);
  const question = page.getByTestId("question");
  await expect(question).toBeVisible();

  // Choices are exposed as radio buttons (role=radio) inside the question.
  const choices = question.getByRole("radio");
  await expect(choices.first()).toBeVisible();
  expect(await choices.count()).toBeGreaterThanOrEqual(2);
  await choices.first().check();
  await page.getByTestId("submit-answer").click();

  // Immediate feedback for Quizzes: correct/incorrect + Explanation
  await expect(page.getByTestId("answer-feedback")).toBeVisible();
  await expect(page.getByTestId("explanation")).toBeVisible();

  // Flag (only possible after answering)
  await page.getByTestId("flag-question").click();
  const dialog = page.getByRole("dialog");
  await expect(dialog).toBeVisible();
  await dialog.getByRole("textbox").fill("e2e: เฉลยไม่ถูกต้อง (automated test)");
  await dialog.getByTestId("flag-submit").click();
  await expect(page.getByTestId("flag-success")).toBeVisible();
});
