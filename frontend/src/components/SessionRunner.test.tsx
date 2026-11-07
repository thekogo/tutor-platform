import { act, render, screen, waitFor, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { http, HttpResponse } from "msw";
import { describe, expect, it, vi } from "vitest";
import { api } from "@/lib/api/client";
import type { CreateSessionRequest, StudySession } from "@/lib/api/types";
import { COURSE_ID, EXAMS, conceptId } from "@/mocks/db";
import { server } from "@/mocks/node";
import { routerMock } from "@/test/setup";
import { SessionRunner } from "./SessionRunner";

async function createSession(body: Omit<CreateSessionRequest, "courseId">): Promise<StudySession> {
  const { data } = await api().POST("/sessions", { body: { courseId: COURSE_ID, ...body } });
  return data!;
}

async function answerFirstChoice(user: ReturnType<typeof userEvent.setup>) {
  const question = await screen.findByTestId("question");
  const radios = within(question).getAllByRole("radio");
  await user.click(radios[0]);
  await user.click(screen.getByTestId("submit-answer"));
}

describe("SessionRunner — Quiz", () => {
  it("shows immediate feedback, Explanation and Mastery changes after answering", async () => {
    const user = userEvent.setup();
    const session = await createSession({ kind: "quiz", conceptId: conceptId("percent-of") });
    render(<SessionRunner session={session} />);

    const question = await screen.findByTestId("question");
    expect(screen.getByTestId("submit-answer")).toBeDisabled();
    expect(within(question).getAllByRole("radio").length).toBeGreaterThanOrEqual(2);
    // Math in the stem is typeset.
    expect(question.querySelector(".katex")).not.toBeNull();

    await answerFirstChoice(user);

    const feedback = await screen.findByTestId("answer-feedback");
    expect(feedback).toHaveTextContent(/ถูกต้อง|ยังไม่ถูก/);
    expect(within(feedback).getByTestId("explanation")).toHaveTextContent("คำอธิบายเฉลย");
    expect(feedback).toHaveTextContent("ระดับความเข้าใจที่เปลี่ยนไป");
    expect(screen.getByText("✓ คำตอบที่ถูก")).toBeInTheDocument();
    // Choices are locked once answered.
    for (const r of within(screen.getByTestId("question")).getAllByRole("radio")) expect(r).toBeDisabled();
  });

  it("lets the Learner flag the answered Question", async () => {
    const user = userEvent.setup();
    const session = await createSession({ kind: "quiz", conceptId: conceptId("percent-of") });
    render(<SessionRunner session={session} />);
    await answerFirstChoice(user);

    await user.click(await screen.findByTestId("flag-question"));
    const dialog = screen.getByRole("dialog", { name: "แจ้งปัญหาโจทย์" });
    const textbox = within(dialog).getByRole("textbox");
    expect(textbox).toHaveFocus();

    // Empty reason is rejected client-side.
    await user.click(within(dialog).getByTestId("flag-submit"));
    expect(within(dialog).getByRole("alert")).toHaveTextContent("กรุณาอธิบายปัญหา");

    await user.type(textbox, "เฉลยน่าจะผิด");
    await user.click(within(dialog).getByTestId("flag-submit"));
    expect(await within(dialog).findByTestId("flag-success")).toBeInTheDocument();

    await user.click(within(dialog).getByRole("button", { name: "ปิด" }));
    expect(screen.queryByRole("dialog")).toBeNull();
    expect(screen.getByText("แจ้งปัญหาโจทย์ข้อนี้แล้ว")).toBeInTheDocument();
  });

  it("goes to the result page after the last Question", async () => {
    const user = userEvent.setup();
    const session = await createSession({ kind: "diagnostic", conceptId: conceptId("factoring") });
    render(<SessionRunner session={session} />);
    for (let i = 0; i < session.totalQuestions!; i++) {
      await answerFirstChoice(user);
      await user.click(await screen.findByTestId("next-question"));
    }
    await waitFor(() => expect(routerMock.push).toHaveBeenCalledWith(`/sessions/${session.id}/result`));
  });
});

describe("SessionRunner — Exam", () => {
  it("never shows feedback before the Exam is finished", async () => {
    const user = userEvent.setup();
    const session = await createSession({ kind: "exam", examId: EXAMS[0].id });
    render(<SessionRunner session={session} />);

    expect(screen.getByRole("timer")).toBeInTheDocument();
    await answerFirstChoice(user);

    // Moves straight on to Question 2 with no feedback.
    await waitFor(() => expect(screen.getByText(/ข้อที่ 2 จาก/)).toBeInTheDocument());
    await screen.findByTestId("question");
    expect(screen.queryByTestId("answer-feedback")).toBeNull();
    expect(screen.queryByTestId("explanation")).toBeNull();
    expect(screen.queryByTestId("flag-question")).toBeNull();
  });

  it("ignores feedback fields even if the server sends them for an Exam", async () => {
    const user = userEvent.setup();
    const session = await createSession({ kind: "exam", examId: EXAMS[0].id });
    server.use(
      http.post("*/api/sessions/:id/answers", () =>
        HttpResponse.json({ recorded: true, correct: true, correctChoiceId: "x", explanationMd: "LEAK" }),
      ),
    );
    render(<SessionRunner session={session} />);
    await answerFirstChoice(user);
    await waitFor(() => expect(screen.queryByText("LEAK")).toBeNull());
    expect(screen.queryByTestId("answer-feedback")).toBeNull();
  });

  it("auto-finishes at the server deadline", async () => {
    vi.useFakeTimers({ shouldAdvanceTime: true });
    try {
      const session = await createSession({ kind: "exam", examId: EXAMS[1].id });
      render(<SessionRunner session={session} />);
      await screen.findByTestId("question");
      expect(routerMock.push).not.toHaveBeenCalled();
      await act(async () => {
        vi.advanceTimersByTime(EXAMS[1].timeLimitSeconds * 1000 + 1000);
      });
      await waitFor(() => expect(routerMock.push).toHaveBeenCalledWith(`/sessions/${session.id}/result`));
    } finally {
      vi.useRealTimers();
    }
  });
});
