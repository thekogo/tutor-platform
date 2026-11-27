"use client";

import { useEffect, useId, useRef, useState } from "react";
import { errorMessage } from "@/lib/api/client";
import type { HelpMessage, HelpReply } from "@/lib/api/types";
import { T } from "@/lib/terms";
import { btn, card, cn } from "@/lib/ui";
import { Markdown } from "./Markdown";

/** Server limits (api/openapi.yaml HelpRequest / app/help.go). */
const MAX_TURNS = 20;
const MAX_CHARS = 1000;

/** Sends the conversation (oldest first, last turn the Learner's) to a Tutor Chat endpoint. */
export type AskTutor = (messages: HelpMessage[]) => Promise<{ data?: HelpReply; error?: unknown }>;

type Props = {
  ask: AskTutor;
  /** Button label while collapsed. */
  openLabel?: string;
  /** One line under the heading explaining what the tutor will do. */
  intro?: string;
  placeholder?: string;
  /** Collapsed state as a card with a full-width primary button, for sidebars. */
  prominent?: boolean;
};

/**
 * Tutor Chat panel. Stateless on the server: the whole conversation is sent
 * each time and the server adds its own context (a Question with its key, or a
 * Concept's Lesson); this component never sees it. Mount with a `key` per
 * subject so each one starts a fresh conversation.
 */
export function TutorChat({
  ask,
  openLabel = `ถาม${T.tutorChat}`,
  intro = "ช่วยชี้แนะทีละขั้น แต่จะไม่บอกคำตอบ ลองเล่าว่าคิดถึงไหนแล้ว",
  placeholder = "เช่น ไม่รู้จะเริ่มจากตรงไหน",
  prominent = false,
}: Props) {
  const [open, setOpen] = useState(false);
  const [messages, setMessages] = useState<HelpMessage[]>([]);
  const [draft, setDraft] = useState("");
  const [pending, setPending] = useState(false);
  const [err, setErr] = useState<string | null>(null);
  const inputRef = useRef<HTMLTextAreaElement>(null);
  const logRef = useRef<HTMLDivElement>(null);
  /** Bumped by "new chat" so a reply still in flight for the old conversation is dropped. */
  const convo = useRef(0);
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
    const mine = convo.current;
    const { data, error } = await ask(next.slice(-MAX_TURNS));
    if (mine !== convo.current) return;
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

  function newChat() {
    convo.current++;
    setMessages([]);
    setDraft("");
    setErr(null);
    setPending(false);
    inputRef.current?.focus();
  }

  function onKeyDown(e: React.KeyboardEvent<HTMLTextAreaElement>) {
    if (e.key === "Enter" && !e.shiftKey && !e.nativeEvent.isComposing) {
      e.preventDefault();
      void send();
    }
  }

  if (!open) {
    const button = (
      <button
        type="button"
        className={prominent ? cn(btn.primary, "w-full") : btn.ghost}
        onClick={() => setOpen(true)}
        aria-expanded={false}
        aria-controls={panelId}
        data-testid="tutor-chat-open"
      >
        💬 {openLabel}
      </button>
    );
    return prominent ? (
      <section className="flex flex-col gap-2 rounded-xl border border-blue-200 bg-blue-50 p-5 shadow-sm" aria-label={T.tutorChat}>
        <h2 className="font-semibold text-slate-900">{T.tutorChat}</h2>
        <p className="text-sm text-slate-700">{intro}</p>
        {button}
      </section>
    ) : (
      <div className="flex justify-end">{button}</div>
    );
  }

  return (
    <section id={panelId} className={cn(card, "flex flex-col gap-3")} aria-label={T.tutorChat} data-testid="tutor-chat">
      <div className="flex items-start justify-between gap-2">
        <div>
          <h3 className="font-semibold text-slate-900">💬 {T.tutorChat}</h3>
          <p className="text-sm text-slate-600">{intro}</p>
        </div>
        <div className="flex shrink-0 gap-1">
          {(messages.length > 0 || pending) && (
            <button type="button" className={btn.ghost} onClick={newChat} data-testid="tutor-chat-new">
              แชทใหม่
            </button>
          )}
          <button type="button" className={btn.ghost} onClick={() => setOpen(false)} aria-expanded={true} aria-controls={panelId}>
            ซ่อน
          </button>
        </div>
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
          placeholder={placeholder}
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
