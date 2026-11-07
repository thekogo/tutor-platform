/**
 * In-memory mock database. Lives on globalThis so that, in mock mode, the
 * Next.js route handler (browser calls) and Server Components (in-process
 * calls) share one state even if bundled separately.
 */
import type {
  Concept,
  ContentRequest,
  CourseGraph,
  Flag,
  Invite,
  Learner,
  MasteryLevel,
  SessionKind,
} from "@/lib/api/types";
import { mockId } from "./ids";
import { LESSON_SLUGS } from "./lessons";
import { mockQuestions } from "./questions";
import { seedConcepts, seedCourse, seedFormats, seedTopics } from "./seed-graph";

export const COURSE_ID = mockId("course", seedCourse.id);
export const conceptId = (slug: string) => mockId("concept", slug);
export const topicId = (slug: string) => mockId("topic", slug);

export const courseGraph: CourseGraph = {
  id: COURSE_ID,
  slug: seedCourse.id,
  name: seedCourse.name,
  goal: seedCourse.goal,
  questionFormats: seedFormats,
  topics: seedTopics.map((t) => ({ id: topicId(t.slug), slug: t.slug, name: t.name, examWeight: t.examWeight })),
  concepts: seedConcepts.map(
    (c): Concept => ({
      id: conceptId(c.slug),
      slug: c.slug,
      name: c.name,
      topicId: topicId(c.topic),
      prerequisiteIds: c.prerequisites.map(conceptId),
      hasLesson: LESSON_SLUGS.has(c.slug),
    }),
  ),
};

export const conceptById = new Map(courseGraph.concepts.map((c) => [c.id, c]));
export const conceptBySlug = new Map(courseGraph.concepts.map((c) => [c.slug, c]));
export const topicById = new Map(courseGraph.topics.map((t) => [t.id, t]));

export const EXAMS = [
  {
    id: mockId("exam", "cu-best-mini-1"),
    name: "ชุดข้อสอบย่อย CU-BEST ชุดที่ 1",
    timeLimitSeconds: 10 * 60,
    questionSlugs: ["exam-mean-1", "exam-speed-1", "exam-comb-1", "exam-venn-1", "exam-seq-1"],
  },
  {
    id: mockId("exam", "cu-best-speed-1"),
    name: "ชุดข้อสอบจับเวลาเร็ว 2 นาที",
    timeLimitSeconds: 2 * 60,
    questionSlugs: ["exam-mean-1", "exam-comb-1", "exam-seq-1"],
  },
];

/** Initial Mastery Levels: a Learner who is shaky on proportions → percent. */
const SEED_LEVELS: Record<string, [MasteryLevel, number, number]> = {
  "integer-operations": ["mastered", 0.92, 9],
  fractions: ["mastered", 0.88, 8],
  decimals: ["mastered", 0.85, 6],
  "odd-even-sign": ["mastered", 0.9, 4],
  "set-basics": ["mastered", 0.95, 4],
  "set-operations": ["mastered", 0.86, 5],
  "read-table": ["mastered", 0.9, 4],
  "angles-lines": ["mastered", 0.84, 4],
  "ratio-basics": ["developing", 0.68, 5],
  "linear-equations": ["developing", 0.7, 6],
  "algebraic-expressions": ["developing", 0.66, 5],
  exponents: ["developing", 0.62, 4],
  "percent-basics": ["developing", 0.6, 4],
  divisibility: ["developing", 0.64, 4],
  "triangle-properties": ["developing", 0.7, 3],
  mean: ["developing", 0.72, 3],
  "read-bar-line": ["developing", 0.66, 3],
  "counting-principles": ["developing", 0.6, 3],
  roots: ["developing", 0.58, 3],
  proportion: ["weak", 0.32, 5],
  "percent-of": ["weak", 0.3, 4],
  "percent-change": ["weak", 0.25, 4],
  "successive-percent": ["weak", 0.2, 3],
  "profit-loss-discount": ["weak", 0.35, 3],
  remainders: ["weak", 0.3, 3],
  "unit-digit": ["weak", 0.22, 3],
  "quadratic-equations": ["weak", 0.34, 3],
};

/** Recommendations as (kind, Root Cause slug, target slug, path root→target, unblocks). */
export const SEED_RECOMMENDATIONS: {
  kind: "study" | "diagnose";
  root: string;
  target: string;
  path: string[];
  unblocks: number;
}[] = [
  {
    kind: "study",
    root: "proportion",
    target: "successive-percent",
    path: ["proportion", "percent-of", "percent-change", "successive-percent"],
    unblocks: 5,
  },
  {
    kind: "study",
    root: "remainders",
    target: "unit-digit",
    path: ["remainders", "unit-digit"],
    unblocks: 1,
  },
  {
    kind: "diagnose",
    root: "factoring",
    target: "quadratic-equations",
    path: ["factoring", "quadratic-equations"],
    unblocks: 1,
  },
];

export type MockMastery = { level: MasteryLevel; score: number; evidence: number };

export type MockSession = {
  id: string;
  learnerId: string;
  kind: SessionKind;
  conceptId: string | null;
  topicId: string | null;
  examId: string | null;
  questionIds: string[];
  /** questionId → chosen choiceId */
  answers: Map<string, string>;
  startedAt: string;
  deadline: string | null;
  finishedAt: string | null;
};

export type MockLearningPath = {
  id: string;
  rootConceptId: string;
  targetConceptId: string;
  path: string[];
  startedAt: string;
  completedAt: string | null;
};

export type MockState = {
  learners: Learner[];
  /** If true, requests without a session cookie act as the Owner (tests). */
  autoLogin: boolean;
  mastery: Map<string, MockMastery>;
  sessions: Map<string, MockSession>;
  learningPath: MockLearningPath | null;
  flags: Flag[];
  contentRequests: ContentRequest[];
  invites: Invite[];
  retired: Set<string>;
  formatBaseline: Record<string, { answered: number; correct: number }>;
};

export const OWNER_ID = mockId("learner", "owner");
export const LEARNER_ID = mockId("learner", "mintra");

function seedState(): MockState {
  const mastery = new Map<string, MockMastery>();
  for (const [slug, [level, score, evidence]] of Object.entries(SEED_LEVELS)) {
    mastery.set(conceptId(slug), { level, score, evidence });
  }
  const now = Date.now();
  const iso = (msAgo: number) => new Date(now - msAgo).toISOString();
  const day = 86_400_000;
  return {
    learners: [
      { id: OWNER_ID, email: "owner@example.com", name: "ปกรณ์ วงศ์ใหญ่", isOwner: true, placementDone: false },
      { id: LEARNER_ID, email: "mintra@example.com", name: "มินตรา ใจดี", isOwner: false, placementDone: true },
    ],
    autoLogin: false,
    mastery,
    sessions: new Map(),
    learningPath: null,
    flags: [
      {
        id: mockId("flag", "1"),
        questionId: mockQuestions[2].id,
        learnerName: "มินตรา ใจดี",
        reason: "คิดว่าเฉลยถูกแล้ว แต่คำอธิบายเฉลยบรรทัดที่สองพิมพ์ 0.80 เป็น 0.08",
        createdAt: iso(2 * day),
        resolvedAt: null,
      },
    ],
    contentRequests: [
      {
        id: mockId("cr", "1"),
        conceptId: conceptId("remainders"),
        conceptName: conceptBySlug.get("remainders")!.name,
        reason: "คลังโจทย์หมด: ผู้เรียนทำครบทุกข้อที่ใช้งานอยู่แล้ว",
        createdAt: iso(day),
      },
      {
        id: mockId("cr", "2"),
        conceptId: conceptId("factoring"),
        conceptName: conceptBySlug.get("factoring")!.name,
        reason: "ยังไม่มีบทเรียนสำหรับแนวคิดนี้",
        createdAt: iso(3 * day),
      },
    ],
    invites: [
      { id: mockId("invite", "1"), createdAt: iso(10 * day), expiresAt: iso(3 * day), usedBy: "มินตรา ใจดี" },
      { id: mockId("invite", "2"), createdAt: iso(day), expiresAt: new Date(now + 6 * day).toISOString(), usedBy: null },
    ],
    retired: new Set(),
    formatBaseline: {
      standard: { answered: 42, correct: 27 },
      "data-sufficiency": { answered: 12, correct: 5 },
      "out-of-the-box": { answered: 8, correct: 3 },
    },
  };
}

const g = globalThis as typeof globalThis & { __tpMockDb?: MockState };

export function db(): MockState {
  if (!g.__tpMockDb) g.__tpMockDb = seedState();
  return g.__tpMockDb;
}

export function resetMockDb(overrides: Partial<MockState> = {}): MockState {
  g.__tpMockDb = { ...seedState(), ...overrides };
  return g.__tpMockDb;
}
