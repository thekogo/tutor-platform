"use client";

import { useRouter } from "next/navigation";
import { useId, useState } from "react";
import { api, errorMessage } from "@/lib/api/client";
import { T } from "@/lib/terms";
import { btn } from "@/lib/ui";

type Props = { questionId?: string; withInput?: boolean };

export function RetireQuestionButton({ questionId, withInput = false }: Props) {
  const router = useRouter();
  const inputId = useId();
  const [value, setValue] = useState(questionId ?? "");
  const [pending, setPending] = useState(false);
  const [msg, setMsg] = useState<{ ok: boolean; text: string } | null>(null);

  async function retire(e?: React.FormEvent) {
    e?.preventDefault();
    const id = value.trim();
    if (!id) return;
    if (!window.confirm(`ยืนยันเลิกใช้${T.question}นี้? จะไม่ถูกเสิร์ฟให้ผู้เรียนอีก`)) return;
    setPending(true);
    const { response, error } = await api().POST("/admin/questions/{questionId}/retire", {
      params: { path: { questionId: id } },
    });
    setPending(false);
    if (response.ok) {
      setMsg({ ok: true, text: `เลิกใช้${T.question}แล้ว` });
      router.refresh();
    } else {
      setMsg({ ok: false, text: errorMessage(error, "ไม่สำเร็จ") });
    }
  }

  return (
    <form onSubmit={retire} className="flex flex-wrap items-end gap-2">
      {withInput && (
        <div className="flex flex-col">
          <label htmlFor={inputId} className="text-sm font-medium">
            รหัส{T.question}
          </label>
          <input
            id={inputId}
            value={value}
            onChange={(e) => setValue(e.target.value)}
            className="w-80 max-w-full rounded border border-slate-300 px-2 py-1 font-mono text-sm"
            placeholder="UUID"
            required
          />
        </div>
      )}
      <button type="submit" disabled={pending || !value.trim()} className={btn.danger}>
        {pending ? "กำลังดำเนินการ…" : `เลิกใช้${T.question}`}
      </button>
      {msg && (
        <span role={msg.ok ? "status" : "alert"} className={msg.ok ? "text-sm text-green-700" : "text-sm text-red-700"}>
          {msg.text}
        </span>
      )}
    </form>
  );
}
