/**
 * MSW handlers implementing api/openapi.yaml against the in-memory mock db.
 * Used by mock mode (NEXT_PUBLIC_API_MOCKING=enabled) and by Vitest.
 */
import { http, HttpResponse, type JsonBodyType } from "msw";
import type {
  AnswerResult,
  ApiError,
  ConceptDetail,
  ConceptRef,
  CourseSummary,
  CreateSessionRequest,
  CreatedInvite,
  ExamSummary,
  FormatStat,
  LearningPath,
  Learner,
  Mastery,
  MasteryChange,
  MasteryLevel,
  Recommendation,
  ServedQuestion,
  SessionResult,
  StudySession,
} from "@/lib/api/types";
import { SESSION_COOKIE } from "@/lib/env";
import {
  COURSE_ID,
  EXAMS,
  OWNER_ID,
  SEED_RECOMMENDATIONS,
  conceptById,
  conceptBySlug,
  conceptId,
  courseGraph,
  db,
  topicById,
  type MockMastery,
  type MockSession,
} from "./db";
import { mockId, newId } from "./ids";
import { lessonFor } from "./lessons";
import { mockQuestions, questionById, type MockQuestion } from "./questions";

const api = (path: string) => `*/api${path}`;

/** Typed JSON response (body checked against the contract type T). */
function json<T>(body: T, init?: ResponseInit) {
  return HttpResponse.json(body as unknown as JsonBodyType, init);
}

function error(status: number, code: string, message: string) {
  return json<ApiError>({ code, message }, { status });
}

const unauthorized = () => error(401, "unauthorized", "กรุณาเข้าสู่ระบบ");
const notFound = (what = "ไม่พบข้อมูล") => error(404, "not_found", what);

function currentLearner(cookies: Record<string, string>): Learner | null {
  const state = db();
  const token = cookies[SESSION_COOKIE];
  if (token) return state.learners.find((l) => l.id === token) ?? null;
  if (state.autoLogin) return state.learners.find((l) => l.id === OWNER_ID) ?? null;
  return null;
}

function sessionCookie(learnerId: string): string {
  return `${SESSION_COOKIE}=${learnerId}; Path=/; HttpOnly; SameSite=Lax; Max-Age=2592000`;
}

// ── Mastery helpers ─────────────────────────────────────────

function masteryOf(id: string): MockMastery {
  return db().mastery.get(id) ?? { level: "unassessed", score: 0, evidence: 0 };
}

function toMastery(id: string): Mastery {
  const m = masteryOf(id);
  return { conceptId: id, score: m.score, evidence: m.evidence, level: m.level };
}

function ref(id: string): ConceptRef {
  return { id, name: conceptById.get(id)?.name ?? "", level: masteryOf(id).level };
}

function nextLevel(level: MasteryLevel, correct: boolean): MasteryLevel {
  if (correct) {
    return level === "developing" || level === "mastered" ? "mastered" : "developing";
  }
  if (level === "mastered") return "developing";
  return "weak";
}

const LEVEL_SCORE: Record<MasteryLevel, number> = { unassessed: 0, weak: 0.3, developing: 0.6, mastered: 0.88 };

function applyAnswer(question: MockQuestion, correct: boolean): MasteryChange[] {
  const id = conceptId(question.primary);
  const before = masteryOf(id);
  const after = nextLevel(before.level, correct);
  db().mastery.set(id, { level: after, score: LEVEL_SCORE[after], evidence: before.evidence + 1 });
  return [{ conceptId: id, conceptName: conceptById.get(id)?.name ?? "", before: before.level, after }];
}

// ── Recommendations ─────────────────────────────────────────

function recommendations(): Recommendation[] {
  return SEED_RECOMMENDATIONS.filter((r) => {
    const level = masteryOf(conceptId(r.root)).level;
    return r.kind === "diagnose" ? level === "unassessed" : level === "weak";
  })
    .map((r) => ({
      kind: r.kind,
      conceptId: conceptId(r.root),
      conceptName: conceptBySlug.get(r.root)?.name ?? r.root,
      targetConceptId: conceptId(r.target),
      path: r.path.map((slug) => ref(conceptId(slug))),
      unblocks: r.unblocks,
      priority: r.unblocks * 1.0,
    }))
    .sort((a, b) => b.priority - a.priority);
}

// ── Learning Path ───────────────────────────────────────────

function toLearningPath(): LearningPath | null {
  const lp = db().learningPath;
  if (!lp) return null;
  const current = lp.path.find((id) => {
    const level = masteryOf(id).level;
    return level === "weak" || level === "unassessed";
  });
  if (!current && !lp.completedAt) lp.completedAt = new Date().toISOString();
  return {
    id: lp.id,
    courseId: COURSE_ID,
    steps: lp.path.map(ref),
    currentStepConceptId: current ?? null,
    startedAt: lp.startedAt,
    completedAt: lp.completedAt,
  };
}

// ── Sessions ────────────────────────────────────────────────

function servable(q: MockQuestion) {
  return !q.examOnly && !db().retired.has(q.id) && !db().flags.some((f) => f.questionId === q.id && !f.resolvedAt);
}

function pickQuestions(req: CreateSessionRequest): string[] {
  const pool = mockQuestions.filter(servable);
  const want = req.kind === "diagnostic" ? 3 : 4;
  let picked: MockQuestion[] = [];
  if (req.conceptId) {
    const slug = conceptById.get(req.conceptId)?.slug;
    picked = pool.filter((q) => slug && q.concepts.includes(slug));
    // Pad with neighbouring Topic Questions so a Quiz is never empty.
    const topic = conceptById.get(req.conceptId)?.topicId;
    if (picked.length < want) {
      picked.push(
        ...pool.filter((q) => !picked.includes(q) && conceptBySlug.get(q.primary)?.topicId === topic),
      );
    }
  } else if (req.topicId) {
    picked = pool.filter((q) => conceptBySlug.get(q.primary)?.topicId === req.topicId);
  }
  if (picked.length < want) picked.push(...pool.filter((q) => !picked.includes(q)));
  return picked.slice(0, want).map((q) => q.id);
}

function toStudySession(s: MockSession): StudySession {
  return {
    id: s.id,
    kind: s.kind,
    courseId: COURSE_ID,
    conceptId: s.conceptId,
    topicId: s.topicId,
    examId: s.examId,
    totalQuestions: s.kind === "placement" ? null : s.questionIds.length,
    answeredCount: s.answers.size,
    startedAt: s.startedAt,
    deadline: s.deadline,
    finishedAt: s.finishedAt,
  };
}

function isExpired(s: MockSession) {
  return s.deadline !== null && Date.now() > Date.parse(s.deadline);
}

function finish(s: MockSession) {
  if (!s.finishedAt) {
    s.finishedAt = new Date().toISOString();
    if (s.kind === "placement") {
      const l = db().learners.find((x) => x.id === s.learnerId);
      if (l) l.placementDone = true;
    }
  }
}

function toResult(s: MockSession): SessionResult {
  const breakdown = new Map<string, { topicId: string; topicName: string; answered: number; correct: number }>();
  let correctCount = 0;
  const review = s.questionIds.map((qid) => {
    const q = questionById.get(qid)!;
    const chosen = s.answers.get(qid) ?? null;
    const key = q.choices.find((c) => c.correct)!;
    const correct = chosen === key.id;
    if (chosen) {
      const topic = topicById.get(conceptBySlug.get(q.primary)!.topicId)!;
      const row = breakdown.get(topic.id) ?? { topicId: topic.id, topicName: topic.name, answered: 0, correct: 0 };
      row.answered++;
      if (correct) {
        row.correct++;
        correctCount++;
      }
      breakdown.set(topic.id, row);
    }
    return { questionId: qid, stemMd: q.stemMd, chosenChoiceId: chosen, correctChoiceId: key.id, correct, explanationMd: q.explanationMd };
  });
  return {
    sessionId: s.id,
    kind: s.kind,
    answered: s.answers.size,
    correct: correctCount,
    total: s.questionIds.length,
    topicBreakdown: [...breakdown.values()],
    rootCauses: recommendations(),
    review,
  };
}

function ownSession(id: string, learner: Learner): MockSession | null {
  const s = db().sessions.get(id);
  return s && s.learnerId === learner.id ? s : null;
}

// ── Handlers ────────────────────────────────────────────────

export const handlers = [
  // auth
  http.get(api("/auth/google/login"), ({ request }) => {
    const invite = new URL(request.url).searchParams.get("invite");
    // Mock Google: log in straight away (Owner, or the sample Learner when an invite is carried).
    const learnerId = invite ? db().learners.find((l) => !l.isOwner)!.id : OWNER_ID;
    return new HttpResponse(null, { status: 302, headers: { Location: "/", "Set-Cookie": sessionCookie(learnerId) } });
  }),
  http.get(api("/auth/google/callback"), () => new HttpResponse(null, { status: 302, headers: { Location: "/" } })),
  http.post(api("/auth/dev-login"), async ({ request }) => {
    const body = (await request.json()) as { email?: string; name?: string };
    if (!body?.email) return error(422, "invalid", "ต้องระบุอีเมล");
    const state = db();
    let learner = state.learners.find((l) => l.email.toLowerCase() === body.email!.toLowerCase());
    if (!learner) {
      learner = { id: mockId("learner", body.email), email: body.email, name: body.name || body.email.split("@")[0], isOwner: false, placementDone: false };
      state.learners.push(learner);
    }
    return new HttpResponse(null, { status: 204, headers: { "Set-Cookie": sessionCookie(learner.id) } });
  }),
  http.post(api("/auth/logout"), () =>
    new HttpResponse(null, { status: 204, headers: { "Set-Cookie": `${SESSION_COOKIE}=; Path=/; HttpOnly; SameSite=Lax; Max-Age=0` } }),
  ),
  http.get(api("/me"), ({ cookies }) => {
    const me = currentLearner(cookies);
    return me ? json<Learner>(me) : unauthorized();
  }),

  // courses
  http.get(api("/courses"), ({ cookies }) => {
    if (!currentLearner(cookies)) return unauthorized();
    const { id, slug, name, goal } = courseGraph;
    return json<CourseSummary[]>([{ id, slug, name, goal }]);
  }),
  http.get(api("/courses/:courseId"), ({ cookies, params }) => {
    if (!currentLearner(cookies)) return unauthorized();
    return params.courseId === COURSE_ID ? json(courseGraph) : notFound("ไม่พบคอร์ส");
  }),
  http.get(api("/courses/:courseId/exams"), ({ cookies, params }) => {
    const me = currentLearner(cookies);
    if (!me) return unauthorized();
    if (params.courseId !== COURSE_ID) return notFound("ไม่พบคอร์ส");
    const sessions = [...db().sessions.values()].filter((s) => s.learnerId === me.id && s.kind === "exam" && s.finishedAt);
    return json<ExamSummary[]>(
      EXAMS.map((e) => {
        const last = sessions.filter((s) => s.examId === e.id).at(-1);
        return {
          id: e.id,
          name: e.name,
          timeLimitSeconds: e.timeLimitSeconds,
          questionCount: e.questionSlugs.length,
          lastScore: last ? toResult(last).correct / e.questionSlugs.length : null,
        };
      }),
    );
  }),
  http.get(api("/concepts/:conceptId"), ({ cookies, params }) => {
    if (!currentLearner(cookies)) return unauthorized();
    const c = conceptById.get(String(params.conceptId));
    if (!c) return notFound("ไม่พบแนวคิด");
    const detail: ConceptDetail = {
      ...c,
      topicName: topicById.get(c.topicId)?.name,
      lessonMd: lessonFor(c.slug),
      dependentIds: courseGraph.concepts.filter((d) => d.prerequisiteIds.includes(c.id)).map((d) => d.id),
      mastery: toMastery(c.id),
      activeQuestionCount: mockQuestions.filter((q) => servable(q) && q.concepts.includes(c.slug)).length,
    };
    return json(detail);
  }),

  // progress
  http.get(api("/courses/:courseId/mastery"), ({ cookies, params }) => {
    if (!currentLearner(cookies)) return unauthorized();
    if (params.courseId !== COURSE_ID) return notFound("ไม่พบคอร์ส");
    return json<Mastery[]>(courseGraph.concepts.map((c) => toMastery(c.id)));
  }),
  http.get(api("/courses/:courseId/recommendations"), ({ cookies, params }) => {
    if (!currentLearner(cookies)) return unauthorized();
    if (params.courseId !== COURSE_ID) return notFound("ไม่พบคอร์ส");
    return json(recommendations());
  }),
  http.get(api("/courses/:courseId/format-report"), ({ cookies, params }) => {
    const me = currentLearner(cookies);
    if (!me) return unauthorized();
    if (params.courseId !== COURSE_ID) return notFound("ไม่พบคอร์ส");
    const totals = structuredClone(db().formatBaseline);
    for (const s of db().sessions.values()) {
      if (s.learnerId !== me.id) continue;
      for (const [qid, choice] of s.answers) {
        const q = questionById.get(qid)!;
        const t = (totals[q.format] ??= { answered: 0, correct: 0 });
        t.answered++;
        if (q.choices.find((c) => c.id === choice)?.correct) t.correct++;
      }
    }
    return json<FormatStat[]>(
      courseGraph.questionFormats.map((f) => {
        const t = totals[f.slug] ?? { answered: 0, correct: 0 };
        return { format: f.slug, formatName: f.name, answered: t.answered, correct: t.correct, accuracy: t.answered ? t.correct / t.answered : 0 };
      }),
    );
  }),
  http.get(api("/courses/:courseId/learning-path"), ({ cookies }) => {
    if (!currentLearner(cookies)) return unauthorized();
    const lp = toLearningPath();
    return lp ? json(lp) : notFound("ยังไม่มีเส้นทางการเรียนที่กำลังทำอยู่");
  }),
  http.post(api("/courses/:courseId/learning-path"), async ({ cookies, request }) => {
    if (!currentLearner(cookies)) return unauthorized();
    const body = (await request.json()) as { rootConceptId?: string; targetConceptId?: string };
    if (!body.rootConceptId || !body.targetConceptId || !conceptById.has(body.rootConceptId) || !conceptById.has(body.targetConceptId)) {
      return error(422, "invalid", "ไม่พบแนวคิดต้นทางหรือปลายทาง");
    }
    const rec = SEED_RECOMMENDATIONS.find((r) => conceptId(r.root) === body.rootConceptId && conceptId(r.target) === body.targetConceptId);
    db().learningPath = {
      id: newId("learning-path"),
      rootConceptId: body.rootConceptId,
      targetConceptId: body.targetConceptId,
      path: rec ? rec.path.map(conceptId) : [...new Set([body.rootConceptId, body.targetConceptId])],
      startedAt: new Date().toISOString(),
      completedAt: null,
    };
    return json(toLearningPath(), { status: 201 });
  }),
  http.delete(api("/courses/:courseId/learning-path"), ({ cookies }) => {
    if (!currentLearner(cookies)) return unauthorized();
    db().learningPath = null;
    return new HttpResponse(null, { status: 204 });
  }),

  // sessions
  http.post(api("/sessions"), async ({ cookies, request }) => {
    const me = currentLearner(cookies);
    if (!me) return unauthorized();
    const body = (await request.json()) as CreateSessionRequest;
    if (body.courseId !== COURSE_ID) return error(422, "invalid", "ไม่พบคอร์ส");
    if (body.kind === "diagnostic" && !body.conceptId) return error(422, "invalid", "ควิซวินิจฉัยต้องระบุแนวคิด");
    if (body.kind === "quiz" && !body.conceptId && !body.topicId) return error(422, "invalid", "ควิซต้องระบุแนวคิดหรือหัวข้อ");
    const now = new Date();
    let questionIds: string[];
    let deadline: string | null = null;
    if (body.kind === "exam") {
      const exam = EXAMS.find((e) => e.id === body.examId);
      if (!exam) return error(422, "invalid", "ไม่พบชุดข้อสอบ");
      questionIds = exam.questionSlugs.map((s) => mockId("question", s));
      deadline = new Date(now.getTime() + exam.timeLimitSeconds * 1000).toISOString();
    } else {
      questionIds = pickQuestions(body);
    }
    const session: MockSession = {
      id: newId("session"),
      learnerId: me.id,
      kind: body.kind,
      conceptId: body.conceptId ?? null,
      topicId: body.topicId ?? null,
      examId: body.examId ?? null,
      questionIds,
      answers: new Map(),
      startedAt: now.toISOString(),
      deadline,
      finishedAt: null,
    };
    db().sessions.set(session.id, session);
    return json<StudySession>(toStudySession(session), { status: 201 });
  }),
  http.get(api("/sessions/:sessionId"), ({ cookies, params }) => {
    const me = currentLearner(cookies);
    if (!me) return unauthorized();
    const s = ownSession(String(params.sessionId), me);
    return s ? json<StudySession>(toStudySession(s)) : notFound("ไม่พบชุดคำถามนี้");
  }),
  http.get(api("/sessions/:sessionId/next"), ({ cookies, params }) => {
    const me = currentLearner(cookies);
    if (!me) return unauthorized();
    const s = ownSession(String(params.sessionId), me);
    if (!s) return notFound("ไม่พบชุดคำถามนี้");
    if (s.finishedAt) return error(410, "finished", "ชุดคำถามนี้จบแล้ว");
    if (isExpired(s)) return error(410, "deadline_passed", "หมดเวลาสอบแล้ว");
    const idx = s.questionIds.findIndex((id) => !s.answers.has(id));
    if (idx < 0) return new HttpResponse(null, { status: 204 });
    const q = questionById.get(s.questionIds[idx])!;
    return json<ServedQuestion>({
      id: q.id,
      stemMd: q.stemMd,
      format: q.format,
      difficulty: q.difficulty,
      position: idx + 1,
      choices: q.choices.map(({ id, label, bodyMd }) => ({ id, label, bodyMd })),
    });
  }),
  http.post(api("/sessions/:sessionId/answers"), async ({ cookies, params, request }) => {
    const me = currentLearner(cookies);
    if (!me) return unauthorized();
    const s = ownSession(String(params.sessionId), me);
    if (!s) return notFound("ไม่พบชุดคำถามนี้");
    if (s.finishedAt) return error(410, "finished", "ชุดคำถามนี้จบแล้ว");
    if (isExpired(s)) return error(410, "deadline_passed", "หมดเวลาสอบแล้ว");
    const body = (await request.json()) as { questionId: string; choiceId: string };
    const q = questionById.get(body.questionId);
    if (!q || !s.questionIds.includes(q.id)) return error(409, "not_in_session", "โจทย์นี้ไม่อยู่ในชุดคำถาม");
    if (s.answers.has(q.id)) return error(409, "already_answered", "ตอบโจทย์ข้อนี้ไปแล้ว");
    const choice = q.choices.find((c) => c.id === body.choiceId);
    if (!choice) return error(409, "invalid_choice", "ไม่พบตัวเลือกนี้");
    s.answers.set(q.id, choice.id);
    const changes = applyAnswer(q, choice.correct);
    if (s.kind === "exam") return json<AnswerResult>({ recorded: true });
    return json<AnswerResult>({
      recorded: true,
      correct: choice.correct,
      correctChoiceId: q.choices.find((c) => c.correct)!.id,
      explanationMd: q.explanationMd,
      masteryChanges: changes,
    });
  }),
  http.post(api("/sessions/:sessionId/finish"), ({ cookies, params }) => {
    const me = currentLearner(cookies);
    if (!me) return unauthorized();
    const s = ownSession(String(params.sessionId), me);
    if (!s) return notFound("ไม่พบชุดคำถามนี้");
    finish(s);
    return json(toResult(s));
  }),
  http.get(api("/sessions/:sessionId/result"), ({ cookies, params }) => {
    const me = currentLearner(cookies);
    if (!me) return unauthorized();
    const s = ownSession(String(params.sessionId), me);
    if (!s) return notFound("ไม่พบชุดคำถามนี้");
    if (!s.finishedAt) return error(409, "not_finished", "ชุดคำถามนี้ยังไม่จบ");
    return json(toResult(s));
  }),

  // questions
  http.post(api("/questions/:questionId/flags"), async ({ cookies, params, request }) => {
    const me = currentLearner(cookies);
    if (!me) return unauthorized();
    const qid = String(params.questionId);
    const answered = [...db().sessions.values()].some((s) => s.learnerId === me.id && s.answers.has(qid));
    if (!answered) return error(403, "not_answered", "แจ้งปัญหาได้เฉพาะโจทย์ที่ตอบแล้ว");
    const body = (await request.json()) as { reason?: string };
    db().flags.push({ id: newId("flag"), questionId: qid, learnerName: me.name, reason: (body.reason ?? "").slice(0, 2000), createdAt: new Date().toISOString(), resolvedAt: null });
    return new HttpResponse(null, { status: 201 });
  }),

  // admin
  http.get(api("/admin/invites"), ({ cookies }) => ownerOnly(cookies) ?? json(db().invites)),
  http.post(api("/admin/invites"), ({ cookies, request }) => {
    const denied = ownerOnly(cookies);
    if (denied) return denied;
    const token = newId("invite-token").replace(/-/g, "");
    const now = Date.now();
    const invite: CreatedInvite = {
      id: newId("invite"),
      createdAt: new Date(now).toISOString(),
      expiresAt: new Date(now + 7 * 86_400_000).toISOString(),
      usedBy: null,
      url: `${new URL(request.url).origin}/login?invite=${token}`,
    };
    const { url: _url, ...stored } = invite;
    void _url;
    db().invites.unshift(stored);
    return json(invite, { status: 201 });
  }),
  http.get(api("/admin/learners"), ({ cookies }) => ownerOnly(cookies) ?? json(db().learners)),
  http.get(api("/admin/flags"), ({ cookies, request }) => {
    const denied = ownerOnly(cookies);
    if (denied) return denied;
    const open = new URL(request.url).searchParams.get("open") !== "false";
    return json(db().flags.filter((f) => (open ? !f.resolvedAt : true)));
  }),
  http.get(api("/admin/content-requests"), ({ cookies }) => ownerOnly(cookies) ?? json(db().contentRequests)),
  http.post(api("/admin/questions/:questionId/retire"), ({ cookies, params }) => {
    const denied = ownerOnly(cookies);
    if (denied) return denied;
    const qid = String(params.questionId);
    db().retired.add(qid);
    for (const f of db().flags) if (f.questionId === qid && !f.resolvedAt) f.resolvedAt = new Date().toISOString();
    return new HttpResponse(null, { status: 204 });
  }),
];

function ownerOnly(cookies: Record<string, string>) {
  const me = currentLearner(cookies);
  if (!me) return unauthorized();
  if (!me.isOwner) return error(403, "forbidden", "เฉพาะเจ้าของระบบเท่านั้น");
  return null;
}
