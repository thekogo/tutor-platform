"use client";

import { useEffect, useId, useRef, useState } from "react";
import { api, errorMessage } from "@/lib/api/client";
import type { HelpMessage } from "@/lib/api/types";
import { T } from "@/lib/terms";
import { btn, card, cn } from "@/lib/ui";
import { Markdown } from "./Markdown";

/** Server limits (api/openapi.yaml HelpRequest / app/help.go). */
const MAX_TURNS = 20;
const MAX_CHARS = 1000;

type Props = {
  sessionId: string;
  questionId: string;
};

/**
 * Tutor Chat for one Quiz Question. Stateless on the server: the whole
 * conversation is sent each time. The server adds the Question, answer key and
 * Explanation as confidential context; this component never sees them.
 * Mount with `key={questionId}` so each Question starts a fresh conversation.
 */
export function TutorChat({ sessionId, questionId }: Props) {
  const [open, setOpen] = useState(false);
  const [messages, setMessages] = useState<HelpMessage[]>([]);
  const [draft, setDraft] = useState("");
  const [pending, setPending] = useState(false);
  const [err, setErr] = useState<string | null>(null);
  const inputRef = useRef<HTMLTextAreaElement>(null);
  const logRef = useRef<HTMLDivElement>(null);
  const inputId = useId();
  const panelId = useId();

  useEffect(() => {
    if (open) inputRef.current?.focus();
  }, [open]);
  useEffect(() => {
    const el = logRef.current;
    if (el) el.scrollTop = el.scrollHeight;
  }, [messages, pending]);

  async function send(e?: React.FormEvent) {
    e?.preventDefault();
    const text = draft.trim();
    if (!text || pending) return;
    if (text.length > MAX_CHARS) {
      setErr(`ข้อความยาวเกินไป (ไม่เกิน ${MAX_CHARS} ตัวอักษร)`);
      return;
    }
    const next: HelpMessage[] = [...messages, { role: "user", content: text }];
    setMessages(next);
    setDraft("");
    setErr(null);
    setPending(true);
    const { data, error } = await api().POST("/sessions/{sessionId}/questions/{questionId}/help", {
      params: { path: { sessionId, questionId } },
      body: { messages: next.slice(-MAX_TURNS) },
    });
    setPending(false);
    if (data) {
      setMessages([...next, { role: "assistant", content: data.reply }]);
    } else {
      // Put the message back so the Learner can retry.
      setMessages(messages);
      setDraft(text);
      setErr(errorMessage(error, "ติวเตอร์ตอบไม่ได้ในตอนนี้ กรุณาลองใหม่"));
    }
    inputRef.current?.focus();
  }

  function onKeyDown(e: React.KeyboardEvent<HTMLTextAreaElement>) {
    if (e.key === "Enter" && !e.shiftKey && !e.nativeEvent.isComposing) {
      e.preventDefault();
      void send();
    }
  }

  if (!open) {
    return (
      <div className="flex justify-end">
        <button
          type="button"
          className={btn.ghost}
          onClick={() => setOpen(true)}
          aria-expanded={false}
          aria-controls={panelId}
          data-testid="tutor-chat-open"
        >
          💬 ถาม{T.tutorChat}
        </button>
      </div>
    );
  }

  return (
    <section id={panelId} className={cn(card, "flex flex-col gap-3")} aria-label={T.tutorChat} data-testid="tutor-chat">
      <div className="flex items-start justify-between gap-2">
        <div>
          <h3 className="font-semibold text-slate-900">💬 {T.tutorChat}</h3>
          <p className="text-sm text-slate-600">ช่วยชี้แนะทีละขั้น แต่จะไม่บอกคำตอบ ลองเล่าว่าคิดถึงไหนแล้ว</p>
        </div>
        <button type="button" className={btn.ghost} onClick={() => setOpen(false)} aria-expanded={true} aria-controls={panelId}>
          ซ่อน
        </button>
      </div>

      {(messages.length > 0 || pending) && (
        <div ref={logRef} role="log" aria-live="polite" className="flex max-h-96 flex-col gap-2 overflow-y-auto">
          {messages.map((m, i) => (
            <div
              key={i}
              data-testid={m.role === "user" ? "tutor-chat-user" : "tutor-chat-reply"}
              className={cn(
                "max-w-[85%] rounded-lg px-3 py-2 text-sm",
                m.role === "user" ? "self-end bg-blue-700 text-white" : "self-start bg-slate-100 text-slate-900",
              )}
            >
              {m.role === "user" ? <p className="whitespace-pre-wrap">{m.content}</p> : <Markdown>{m.content}</Markdown>}
            </div>
          ))}
          {pending && (
            <p role="status" className="self-start text-sm text-slate-500">
              ติวเตอร์กำลังคิด…
            </p>
          )}
        </div>
      )}

      {err && (
        <p role="alert" className="text-sm text-red-700">
          {err}
        </p>
      )}

      <form onSubmit={send} className="flex items-end gap-2">
        <label htmlFor={inputId} className="sr-only">
          ข้อความถึง{T.tutorChat}
        </label>
        <textarea
          id={inputId}
          ref={inputRef}
          value={draft}
          onChange={(e) => setDraft(e.target.value)}
          onKeyDown={onKeyDown}
          rows={2}
          maxLength={MAX_CHARS}
          placeholder="เช่น ไม่รู้จะเริ่มจากตรงไหน"
          className="flex-1 resize-y rounded-lg border border-slate-300 p-2 text-sm focus:outline-none focus:ring-2 focus:ring-blue-500"
          data-testid="tutor-chat-input"
        />
        <button type="submit" className={btn.primary} disabled={pending || !draft.trim()} data-testid="tutor-chat-send">
          ส่ง
        </button>
      </form>
    </section>
  );
}
