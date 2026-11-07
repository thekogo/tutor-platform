import fs from "node:fs/promises";
import path from "node:path";
import { test as setup, expect } from "@playwright/test";
import { devLogin, findQuizzableConcept, getCourseBySlug, getMe, newApiContext } from "../lib/api";
import {
  COURSE_SLUG,
  FIXTURES_FILE,
  LEARNER,
  LEARNER_STATE,
  OWNER,
  OWNER_STATE,
  QUIZ_CONCEPT_SLUG,
  SEED_QUESTIONS,
} from "../lib/env";
import { seedQuizQuestions } from "../lib/seed-questions";
import type { Fixtures } from "../lib/fixtures";

setup("log in Owner and Learner, seed Questions", async () => {
  await fs.mkdir(path.dirname(OWNER_STATE), { recursive: true });

  // Order matters: on a fresh database the first Learner to log in becomes the Owner.
  const owner = await newApiContext();
  await devLogin(owner, OWNER);
  const ownerMe = await getMe(owner);
  expect(
    ownerMe.isOwner,
    `${OWNER.email} is not the Owner — the e2e database must be fresh (make e2e-down)`,
  ).toBe(true);
  await owner.storageState({ path: OWNER_STATE });

  const learner = await newApiContext();
  await devLogin(learner, LEARNER);
  expect((await getMe(learner)).isOwner).toBe(false);
  await learner.storageState({ path: LEARNER_STATE });

  const course = await getCourseBySlug(learner, COURSE_SLUG);
  expect(course.concepts.length, "Course has no Concepts: did `make seed` run?").toBeGreaterThan(0);

  if (SEED_QUESTIONS) {
    try {
      await seedQuizQuestions(QUIZ_CONCEPT_SLUG);
    } catch (err) {
      // Don't fail setup here: the quiz journey fails with a clearer message if no
      // Concept ends up with Active Questions.
      console.warn(`[e2e setup] seeding Questions via MCP failed: ${String(err)}`);
    }
  }

  const quizConcept = await findQuizzableConcept(learner, course, QUIZ_CONCEPT_SLUG);
  const fixtures: Fixtures = {
    courseId: course.id,
    conceptCount: course.concepts.length,
    quizConceptId: quizConcept?.id ?? null,
  };
  await fs.writeFile(FIXTURES_FILE, JSON.stringify(fixtures, null, 2));

  await owner.dispose();
  await learner.dispose();
});
