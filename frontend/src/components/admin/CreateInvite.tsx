"use client";

import { useId, useState } from "react";
import { api, errorMessage, invalidate } from "@/lib/api/client";
import type { CourseSummary } from "@/lib/api/types";
import { T } from "@/lib/terms";
import { btn, formatDateTime } from "@/lib/ui";

export function CreateInvite({ courses }: { courses: CourseSummary[] }) {
  const inputId = useId();
  const [courseIds, setCourseIds] = useState<string[]>([]);
  const [url, setUrl] = useState<string | null>(null);
  const [expiresAt, setExpiresAt] = useState<string | null>(null);
  const [pending, setPending] = useState(false);
  const [copied, setCopied] = useState(false);
  const [err, setErr] = useState<string | null>(null);

  async function create() {
    setPending(true);
    setErr(null);
    setCopied(false);
    const { data, error } = await api().POST("/admin/invites", { body: { courseIds } });
    setPending(false);
    if (!data) {
      setErr(errorMessage(error, `สร้าง${T.invite}ไม่สำเร็จ`));
      return;
    }
    setUrl(data.url);
    setExpiresAt(data.expiresAt);
    invalidate();
  }

  async function copy() {
    if (!url) return;
    try {
      await navigator.clipboard.writeText(url);
      setCopied(true);
    } catch {
      setErr("คัดลอกไม่สำเร็จ กรุณาเลือกข้อความแล้วคัดลอกเอง");
    }
  }

  return (
    <div>
      {courses.length > 0 && (
        <fieldset className="mb-3">
          <legend className="text-sm font-medium">{T.course}ที่{T.learner}ใหม่จะเข้าเรียนได้</legend>
          <div className="mt-1 flex flex-wrap gap-x-4 gap-y-1">
            {courses.map((c) => (
              <label key={c.id} className="flex items-center gap-2 text-sm">
                <input
                  type="checkbox"
                  checked={courseIds.includes(c.id)}
                  onChange={(e) => {
                    const on = e.currentTarget.checked;
                    setCourseIds((prev) => (on ? [...prev, c.id] : prev.filter((id) => id !== c.id)));
                  }}
                  data-testid={`invite-course-${c.slug}`}
                />
                <span>{c.name}</span>
              </label>
            ))}
          </div>
          {courseIds.length === 0 && (
            <p className="mt-1 text-xs text-amber-700">ยังไม่ได้เลือก{T.course}: {T.learner}จะยังเข้าเรียนไม่ได้จนกว่าจะกำหนดสิทธิ์ภายหลัง</p>
          )}
        </fieldset>
      )}
      <button type="button" onClick={create} disabled={pending} className={btn.primary} data-testid="create-invite">
        {pending ? "กำลังสร้าง…" : `สร้างลิงก์${T.invite}`}
      </button>
      {url && (
        <div className="mt-3 rounded-lg border border-blue-200 bg-blue-50 p-3">
          <label htmlFor={inputId} className="text-sm font-medium">
            ลิงก์{T.invite} (แสดงครั้งเดียว{expiresAt ? ` · หมดอายุ ${formatDateTime(expiresAt)}` : ""})
          </label>
          <div className="mt-1 flex gap-2">
            <input id={inputId} readOnly value={url} className="min-w-0 flex-1 rounded border border-slate-300 bg-white px-2 py-1 text-sm" onFocus={(e) => e.currentTarget.select()} data-testid="invite-url" />
            <button type="button" onClick={copy} className={btn.secondary}>
              {copied ? "คัดลอกแล้ว ✓" : "คัดลอก"}
            </button>
          </div>
          <span className="sr-only" role="status">
            {copied ? "คัดลอกลิงก์แล้ว" : ""}
          </span>
        </div>
      )}
      {err && (
        <p role="alert" className="mt-2 text-sm text-red-700">
          {err}
        </p>
      )}
    </div>
  );
}
