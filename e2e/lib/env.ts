import path from "node:path";

export const BASE_URL = process.env.E2E_BASE_URL ?? "http://localhost:3000";
export const MCP_URL = process.env.E2E_MCP_URL ?? "http://localhost:8080/mcp";
export const MCP_TOKEN = process.env.E2E_MCP_TOKEN ?? "e2e-mcp-token";
/** Set to "false" to skip creating Questions through MCP (e.g. when the seed already has some). */
export const SEED_QUESTIONS = (process.env.E2E_SEED_QUESTIONS ?? "true") !== "false";
export const COURSE_SLUG = process.env.E2E_COURSE_SLUG ?? "cu-best-math";
/** Concept that receives the e2e Questions. A root Concept (no Prerequisites). */
export const QUIZ_CONCEPT_SLUG = process.env.E2E_QUIZ_CONCEPT_SLUG ?? "integer-operations";

const AUTH_DIR = path.join(__dirname, "..", ".auth");
export const OWNER_STATE = path.join(AUTH_DIR, "owner.json");
export const LEARNER_STATE = path.join(AUTH_DIR, "learner.json");
export const FIXTURES_FILE = path.join(AUTH_DIR, "fixtures.json");

export const OWNER = { email: "owner@e2e.test", name: "E2E Owner" };
export const LEARNER = { email: "learner@e2e.test", name: "E2E Learner" };
