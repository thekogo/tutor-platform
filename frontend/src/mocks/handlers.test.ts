import { describe, expect, it } from "vitest";
import { api } from "@/lib/api/client";
import { COURSE_ID, EXAMS, conceptId, db } from "./db";

describe("mock handlers (contract behaviour)", () => {
  it("never returns an answer key before answering, and none for Exam answers", async () => {
    const { data: s } = await api().POST("/sessions", { body: { kind: "exam", courseId: COURSE_ID, examId: EXAMS[0].id } });
    const { data: q } = await api().GET("/sessions/{sessionId}/next", { params: { path: { sessionId: s!.id } } });
    expect(JSON.stringify(q)).not.toMatch(/correct|explanation/i);
    const { data: a } = await api().POST("/sessions/{sessionId}/answers", {
      params: { path: { sessionId: s!.id } },
      body: { questionId: q!.id, choiceId: q!.choices[0].id },
    });
    expect(a).toEqual({ recorded: true });
  });

  it("returns 401 without a session when auto-login is off", async () => {
    db().autoLogin = false;
    const { response } = await api().GET("/me");
    expect(response.status).toBe(401);
  });

  it("forbids admin endpoints for non-Owners", async () => {
    db().autoLogin = false;
    document.cookie = "";
    const { response } = await api().GET("/admin/learners", { headers: { cookie: `tp_session=${db().learners[1].id}` } });
    expect([401, 403]).toContain(response.status);
  });

  it("quizzes on a Concept serve its Questions", async () => {
    const { data: s } = await api().POST("/sessions", { body: { kind: "quiz", courseId: COURSE_ID, conceptId: conceptId("percent-of") } });
    expect(s!.totalQuestions).toBeGreaterThan(0);
    expect(s!.deadline ?? null).toBeNull();
  });
});
