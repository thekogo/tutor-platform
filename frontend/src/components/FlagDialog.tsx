"use client";

import { useEffect, useId, useRef, useState } from "react";
import { api, errorMessage } from "@/lib/api/client";
import { T } from "@/lib/terms";
import { btn } from "@/lib/ui";

type Props = {
  questionId: string;
  onClose: (flagged: boolean) => void;
};

/** Modal for flagging an answered Question (answer key or Explanation is wrong). */
export function FlagDialog({ questionId, onClose }: Props) {
  const titleId = useId();
  const reasonId = useId();
  const [reason, setReason] = useState("");
  const [pending, setPending] = useState(false);
  const [err, setErr] = useState<string | null>(null);
  const [done, setDone] = useState(false);
  const textRef = useRef<HTMLTextAreaElement>(null);
  const closeRef = useRef<HTMLButtonElement>(null);

  useEffect(() => {
    textRef.current?.focus();
  }, []);
  useEffect(() => {
    if (done) closeRef.current?.focus();
  }, [done]);

  useEffect(() => {
    function onKey(e: KeyboardEvent) {
      if (e.key === "Escape") onClose(done);
    }
    document.addEventListener("keydown", onKey);
    return () => document.removeEventListener("keydown", onKey);
  }, [done, onClose]);

  async function submit(e: React.FormEvent) {
    e.preventDefault();
    if (!reason.trim()) {
      setErr("กรุณาอธิบายปัญหาที่พบ");
      return;
    }
    setPending(true);
    setErr(null);
    const { error, response } = await api().POST("/questions/{questionId}/flags", {
      params: { path: { questionId } },
      body: { reason: reason.trim() },
    });
    setPending(false);
    if (response.ok) setDone(true);
    else setErr(errorMessage(error, "ส่งไม่สำเร็จ กรุณาลองใหม่"));
  }

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center bg-slate-900/50 p-4">
      <div
        role="dialog"
        aria-modal="true"
        aria-labelledby={titleId}
        className="w-full max-w-lg rounded-xl bg-white p-6 shadow-xl"
      >
        <h2 id={titleId} className="text-lg font-bold">
          {T.flag}
        </h2>
        {done ? (
          <div>
            <p data-testid="flag-success" role="status" className="mt-3 rounded-lg bg-green-50 p-3 text-green-800">
              ขอบคุณ! ส่งเรื่องแล้ว โจทย์นี้จะถูกพักไว้จนกว่าจะตรวจสอบเสร็จ
            </p>
            <div className="mt-4 flex justify-end">
              <button ref={closeRef} type="button" className={btn.primary} onClick={() => onClose(true)}>
                ปิด
              </button>
            </div>
          </div>
        ) : (
          <form onSubmit={submit} noValidate>
            <p className="mt-1 text-sm text-slate-600">
              แจ้งเมื่อคิดว่าเฉลยหรือ{T.explanation}ไม่ถูกต้อง โจทย์จะถูกพักไว้สำหรับผู้เรียนทุกคนจนกว่าจะแก้ไข
            </p>
            <label htmlFor={reasonId} className="mt-4 block text-sm font-medium">
              ปัญหาที่พบ
            </label>
            <textarea
              id={reasonId}
              ref={textRef}
              value={reason}
              onChange={(e) => setReason(e.target.value)}
              maxLength={2000}
              rows={4}
              required
              aria-invalid={err ? true : undefined}
              aria-describedby={err ? `${reasonId}-err` : undefined}
              className="mt-1 w-full rounded-lg border border-slate-300 p-2"
              placeholder="เช่น เฉลยควรเป็นข้อ C เพราะ…"
            />
            {err && (
              <p id={`${reasonId}-err`} role="alert" className="mt-1 text-sm text-red-700">
                {err}
              </p>
            )}
            <div className="mt-4 flex justify-end gap-2">
              <button type="button" className={btn.secondary} onClick={() => onClose(false)}>
                ยกเลิก
              </button>
              <button type="submit" className={btn.primary} disabled={pending} data-testid="flag-submit">
                {pending ? "กำลังส่ง…" : "ส่ง"}
              </button>
            </div>
          </form>
        )}
      </div>
    </div>
  );
}
