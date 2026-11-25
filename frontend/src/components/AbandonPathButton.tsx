"use client";

import { useState } from "react";
import { api, invalidate } from "@/lib/api/client";
import { T } from "@/lib/terms";
import { btn } from "@/lib/ui";

export function AbandonPathButton({ courseId }: { courseId: string }) {
  const [pending, setPending] = useState(false);
  async function abandon() {
    if (!window.confirm(`เลิก${T.learningPath}นี้หรือไม่?`)) return;
    setPending(true);
    await api().DELETE("/courses/{courseId}/learning-path", { params: { path: { courseId } } });
    invalidate();
    setPending(false);
  }
  return (
    <button type="button" onClick={abandon} disabled={pending} className={btn.danger}>
      เลิก{T.learningPath}นี้
    </button>
  );
}
