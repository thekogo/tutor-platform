import fs from "node:fs";
import { FIXTURES_FILE } from "./env";

export interface Fixtures {
  courseId: string;
  conceptCount: number;
  quizConceptId: string | null;
}

/** Data discovered by the setup project (course id, quizzable Concept). */
export function loadFixtures(): Fixtures {
  if (!fs.existsSync(FIXTURES_FILE)) {
    throw new Error(`${FIXTURES_FILE} missing: run the "setup" project first (pnpm test runs it).`);
  }
  return JSON.parse(fs.readFileSync(FIXTURES_FILE, "utf8")) as Fixtures;
}
