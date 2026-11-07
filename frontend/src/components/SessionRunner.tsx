"use client";

import { useRouter } from "next/navigation";
import { useCallback, useEffect, useId, useRef, useState } from "react";
import { api, errorMessage } from "@/lib/api/client";
import type { AnswerResult, ServedQuestion, StudySession } from "@/lib/api/types";
import { SESSION_KIND_LABEL, T } from "@/lib/terms";
import { btn, card, cn } from "@/lib/ui";
import { Countdown } from "./Countdown";
import { FlagDialog } from "./FlagDialog";
import { Markdown } from "./Markdown";
import { MasteryChip } from "./MasteryChip";

type Phase = "loading" | "answering" | "submitting" | "feedback" | "finishing" | "error";

type Props = {
  session: StudySession;
  /** Context shown in the header, e.g. the Concept or Exam name. */
  title?: string;
};

/**
 * One Question at a time.
 * - Quiz / Diagnostic / Placement: immediate feedback, Explanation, Mastery changes, Flag.
 * - Exam: countdown to the server deadline, NO feedback until finish, auto-finish at deadline.
 */
export function SessionRunner({ session, title }: Props) {
  const router = useRouter();
  const isExam = session.kind === "exam";
  const [phase, setPhase] = useState<Phase>("loading");
  const [question, setQuestion] = useState<ServedQuestion | null>(null);
  const [selected, setSelected] = useState<string | null>(null);
  const [feedback, setFeedback] = useState<AnswerResult | null>(null);
  const [answered, setAnswered] = useState(session.answeredCount);
  const [err, setErr] = useState<string | null>(null);
  const [flagOpen, setFlagOpen] = useState(false);
  const [flagged, setFlagged] = useState<Set<string>>(new Set());
  const shownAt = useRef(0);
  const finishing = useRef(false);
  const headingRef = useRef<HTMLHeadingElement>(null);
  const feedbackRef = useRef<HTMLDivElement>(null);
  const flagButtonRef = useRef<HTMLButtonElement>(null);
  const groupName = useId();

  const finish = useCallback(async () => {
    if (finishing.current) return;
    finishing.current = true;
    setPhase("finishing");
    const { response, error } = await api().POST("/sessions/{sessionId}/finish", {
      params: { path: { sessionId: session.id } },
    });
    if (response.ok) {
      router.push(`/sessions/${session.id}/result`);
      return;
    }
    finishing.current = false;
    setErr(errorMessage(error, "ส่งไม่สำเร็จ กรุณาลองใหม่"));
    setPhase("error");
  }, [router, session.id]);

  const loadNext = useCallback(async () => {
    setPhase("loading");
    setSelected(null);
    setFeedback(null);
    setErr(null);
    const { data, response, error } = await api().GET("/sessions/{sessionId}/next", {
      params: { path: { sessionId: session.id } },
    });
    if (response.status === 204 || response.status === 410) {
      // No more Questions, or the Exam deadline passed / session already finished.
      await finish();
      return;
    }
    if (!data) {
      setErr(errorMessage(error, "โหลดโจทย์ไม่สำเร็จ"));
      setPhase("error");
      return;
    }
    setQuestion(data);
    shownAt.current = Date.now();
    setPhase("answering");
  }, [finish, session.id]);

  useEffect(() => {
    if (session.finishedAt) {
      router.replace(`/sessions/${session.id}/result`);
      return;
    }
    // Initial fetch; state updates happen after the request resolves.
    // eslint-disable-next-line react-hooks/set-state-in-effect
    void loadNext();
    // Run once per session.
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [session.id]);

  useEffect(() => {
    if (phase === "answering") headingRef.current?.focus();
    if (phase === "feedback") feedbackRef.current?.focus();
  }, [phase, question]);

  async function submit(e: React.FormEvent) {
    e.preventDefault();
    if (!question || !selected) return;
    setPhase("submitting");
    setErr(null);
    const { data, response, error } = await api().POST("/sessions/{sessionId}/answers", {
      params: { path: { sessionId: session.id } },
      body: { questionId: question.id, choiceId: selected, timeSpentMs: Math.max(0, Date.now() - shownAt.current) },
    });
    if (response.status === 410) {
      await finish();
      return;
    }
    if (!data) {
      setErr(errorMessage(error, "ส่งคำตอบไม่สำเร็จ กรุณาลองใหม่"));
      setPhase("answering");
      return;
    }
    setAnswered((n) => n + 1);
    if (isExam) {
      // Exam rule: never reveal correctness before the Exam is finished.
      await loadNext();
      return;
    }
    setFeedback(data);
    setPhase("feedback");
  }

  function confirmFinish() {
    const msg = isExam ? "ยืนยันส่งข้อสอบ? ข้อที่ยังไม่ตอบจะนับว่าผิด" : "จบชุดนี้และดูผลเลยหรือไม่?";
    if (window.confirm(msg)) void finish();
  }

  const total = session.totalQuestions ?? null;
  const position = question?.position ?? answered + 1;
  const locked = phase !== "answering";
  const showFeedback = !isExam && phase === "feedback" && feedback;

  return (
    <div className="mx-auto flex max-w-3xl flex-col gap-4">
      <header className="flex flex-wrap items-center justify-between gap-3">
        <div>
          <p className="text-sm font-medium text-blue-800">{SESSION_KIND_LABEL[session.kind]}</p>
          {title && <p className="text-lg font-semibold text-slate-900">{title}</p>}
          <p className="text-sm text-slate-600" aria-live="polite">
            {total ? `ข้อที่ ${Math.min(position, total)} จาก ${total}` : `ข้อที่ ${position}`}
            {isExam && " · จะเห็นเฉลยหลังส่งข้อสอบ"}
          </p>
        </div>
        <div className="flex items-center gap-2">
          {isExam && session.deadline && <Countdown deadline={session.deadline} onExpire={() => void finish()} />}
          <button type="button" className={btn.secondary} onClick={confirmFinish} disabled={phase === "finishing"}>
            {isExam ? "ส่งข้อสอบ" : "จบชุดนี้"}
          </button>
        </div>
      </header>

      {total ? (
        <div
          role="progressbar"
          aria-label="ความคืบหน้า"
          aria-valuemin={0}
          aria-valuemax={total}
          aria-valuenow={answered}
          className="h-2 overflow-hidden rounded-full bg-slate-200"
        >
          <div className="h-full bg-blue-600 transition-all" style={{ width: `${(answered / total) * 100}%` }} />
        </div>
      ) : null}

      {(phase === "loading" || phase === "finishing") && (
        <div className={card} role="status" aria-live="polite">
          {phase === "finishing" ? "กำลังสรุปผล…" : `กำลังโหลด${T.question}…`}
        </div>
      )}

      {phase === "error" && (
        <div className={cn(card, "border-red-200 bg-red-50")} role="alert">
          <p className="text-red-800">{err}</p>
          <button type="button" className={cn(btn.secondary, "mt-3")} onClick={() => void loadNext()}>
            ลองใหม่
          </button>
        </div>
      )}

      {question && (phase === "answering" || phase === "submitting" || phase === "feedback") && (
        <form onSubmit={submit} className={card} data-testid="question" aria-labelledby={`${groupName}-h`}>
          <h2 id={`${groupName}-h`} ref={headingRef} tabIndex={-1} className="sr-only">
            {T.question}ข้อที่ {question.position}
          </h2>
          <Markdown className="text-lg">{question.stemMd}</Markdown>

          <fieldset className="mt-4" disabled={locked}>
            <legend className="mb-2 text-sm font-medium text-slate-700">เลือกคำตอบ</legend>
            <div className="flex flex-col gap-2">
              {question.choices.map((c) => {
                const isChosen = selected === c.id;
                const isKey = showFeedback && feedback.correctChoiceId === c.id;
                const isWrongPick = showFeedback && isChosen && !isKey;
                return (
                  <label
                    key={c.id}
                    className={cn(
                      "flex cursor-pointer items-start gap-3 rounded-lg border p-3 transition-colors has-[:focus-visible]:ring-2 has-[:focus-visible]:ring-blue-500",
                      isKey
                        ? "border-green-500 bg-green-50"
                        : isWrongPick
                          ? "border-red-500 bg-red-50"
                          : isChosen
                            ? "border-blue-500 bg-blue-50"
                            : "border-slate-200 hover:bg-slate-50",
                      locked && "cursor-default",
                    )}
                  >
                    <input
                      type="radio"
                      name={groupName}
                      value={c.id}
                      checked={isChosen}
                      onChange={() => setSelected(c.id)}
                      className="mt-1 size-4 accent-blue-700"
                    />
                    <span className="font-semibold text-slate-700">{c.label}.</span>
                    <Markdown inline className="flex-1">
                      {c.bodyMd}
                    </Markdown>
                    {isKey && <span className="text-sm font-medium text-green-800">✓ คำตอบที่ถูก</span>}
                    {isWrongPick && <span className="text-sm font-medium text-red-800">✗ คำตอบของคุณ</span>}
                  </label>
                );
              })}
            </div>
          </fieldset>

          {err && (
            <p role="alert" className="mt-3 text-sm text-red-700">
              {err}
            </p>
          )}

          {phase !== "feedback" && (
            <div className="mt-4 flex justify-end">
              <button
                type="submit"
                className={btn.primary}
                disabled={!selected || phase === "submitting"}
                data-testid="submit-answer"
              >
                {phase === "submitting" ? "กำลังส่ง…" : isExam ? "บันทึกคำตอบ" : "ส่งคำตอบ"}
              </button>
            </div>
          )}

          {showFeedback && (
            <div
              ref={feedbackRef}
              tabIndex={-1}
              data-testid="answer-feedback"
              className={cn(
                "mt-5 rounded-lg border p-4",
                feedback.correct ? "border-green-300 bg-green-50" : "border-red-300 bg-red-50",
              )}
            >
              <p role="status" className={cn("text-lg font-bold", feedback.correct ? "text-green-800" : "text-red-800")}>
                {feedback.correct ? "✓ ถูกต้อง!" : "✗ ยังไม่ถูก"}
              </p>

              {feedback.explanationMd && (
                <section data-testid="explanation" className="mt-3">
                  <h3 className="font-semibold">{T.explanation}</h3>
                  <Markdown>{feedback.explanationMd}</Markdown>
                </section>
              )}

              {feedback.masteryChanges && feedback.masteryChanges.length > 0 && (
                <section className="mt-3">
                  <h3 className="font-semibold">{T.masteryLevel}ที่เปลี่ยนไป</h3>
                  <ul className="mt-1 flex flex-col gap-1">
                    {feedback.masteryChanges.map((m) => (
                      <li key={m.conceptId} className="flex flex-wrap items-center gap-2 text-sm">
                        <span>{m.conceptName}:</span>
                        <MasteryChip level={m.before} />
                        <span aria-label="เปลี่ยนเป็น">→</span>
                        <MasteryChip level={m.after} />
                        {m.before === m.after && <span className="text-slate-500">(คงเดิม)</span>}
                      </li>
                    ))}
                  </ul>
                </section>
              )}

              <div className="mt-4 flex flex-wrap items-center justify-between gap-2">
                {flagged.has(question.id) ? (
                  <span className="text-sm text-slate-600">แจ้งปัญหาโจทย์ข้อนี้แล้ว</span>
                ) : (
                  <button
                    ref={flagButtonRef}
                    type="button"
                    className={btn.ghost}
                    onClick={() => setFlagOpen(true)}
                    data-testid="flag-question"
                  >
                    ⚑ {T.flag}
                  </button>
                )}
                <button type="button" className={btn.primary} onClick={() => void loadNext()} data-testid="next-question">
                  {total && answered >= total ? "ดูผลลัพธ์" : "ข้อถัดไป"}
                </button>
              </div>
            </div>
          )}
        </form>
      )}

      {flagOpen && question && (
        <FlagDialog
          questionId={question.id}
          onClose={(ok) => {
            setFlagOpen(false);
            if (ok) setFlagged((s) => new Set(s).add(question.id));
            else flagButtonRef.current?.focus();
          }}
        />
      )}
    </div>
  );
}
