"use client";

import { useRouter } from "next/navigation";
import { useState } from "react";
import { api, errorMessage } from "@/lib/api/client";
import { T } from "@/lib/terms";
import { btn } from "@/lib/ui";

type Props = {
  courseId: string;
  rootConceptId: string;
  targetConceptId: string;
  variant?: keyof typeof btn;
  label?: string;
};

/** Starts a Learning Path from a Recommendation (replaces any active one). */
export function StartLearningPathButton({ courseId, rootConceptId, targetConceptId, variant = "primary", label }: Props) {
  const router = useRouter();
  const [pending, setPending] = useState(false);
  const [err, setErr] = useState<string | null>(null);

  async function start() {
    setPending(true);
    setErr(null);
    const { data, error } = await api().POST("/courses/{courseId}/learning-path", {
      params: { path: { courseId } },
      body: { rootConceptId, targetConceptId },
    });
    if (data) {
      router.push("/learning-path");
      router.refresh();
      return;
    }
    setPending(false);
    setErr(errorMessage(error, `เริ่ม${T.learningPath}ไม่สำเร็จ`));
  }

  return (
    <span className="inline-flex flex-col gap-1">
      <button type="button" onClick={start} disabled={pending} className={btn[variant]}>
        {pending ? "กำลังเริ่ม…" : (label ?? `เริ่ม${T.learningPath}`)}
      </button>
      {err && (
        <span role="alert" className="text-xs text-red-700">
          {err}
        </span>
      )}
    </span>
  );
}
