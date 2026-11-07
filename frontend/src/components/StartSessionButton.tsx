"use client";

import { useRouter } from "next/navigation";
import { useState } from "react";
import { api, errorMessage } from "@/lib/api/client";
import type { CreateSessionRequest } from "@/lib/api/types";
import { btn } from "@/lib/ui";

type Props = {
  request: CreateSessionRequest;
  children: React.ReactNode;
  variant?: keyof typeof btn;
  className?: string;
  testId?: string;
};

/** Starts a Quiz / Diagnostic Quiz / Placement Test / Exam and opens the session runner. */
export function StartSessionButton({ request, children, variant = "primary", className, testId }: Props) {
  const router = useRouter();
  const [pending, setPending] = useState(false);
  const [err, setErr] = useState<string | null>(null);

  async function start() {
    setPending(true);
    setErr(null);
    const { data, error } = await api().POST("/sessions", { body: request });
    if (data) {
      router.push(`/sessions/${data.id}`);
      return;
    }
    setPending(false);
    setErr(errorMessage(error, "เริ่มไม่สำเร็จ กรุณาลองใหม่"));
  }

  return (
    <span className="inline-flex flex-col gap-1">
      <button type="button" onClick={start} disabled={pending} className={className ?? btn[variant]} aria-busy={pending} data-testid={testId}>
        {pending ? "กำลังเริ่ม…" : children}
      </button>
      {err && (
        <span role="alert" className="text-xs text-red-700">
          {err}
        </span>
      )}
    </span>
  );
}
