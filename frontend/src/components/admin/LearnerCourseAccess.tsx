"use client";

import { useState } from "react";
import { api, errorMessage } from "@/lib/api/client";
import type { CourseSummary, Learner } from "@/lib/api/types";
import { T } from "@/lib/terms";

type Props = { learner: Learner; courses: CourseSummary[] };

/** Checkboxes for the Courses a Learner may study; each toggle saves immediately. */
export function LearnerCourseAccess({ learner, courses }: Props) {
  const [selected, setSelected] = useState<string[]>(learner.courseIds ?? []);
  const [pending, setPending] = useState(false);
  const [err, setErr] = useState<string | null>(null);

  if (learner.isOwner) return <span className="text-slate-600">ทุก{T.course}</span>;
  if (courses.length === 0) return <span className="text-slate-500">ยังไม่มี{T.course}</span>;

  async function toggle(courseId: string, on: boolean) {
    const prev = selected;
    const next = on ? [...prev, courseId] : prev.filter((id) => id !== courseId);
    setSelected(next);
    setPending(true);
    setErr(null);
    const { data, error } = await api().PUT("/admin/learners/{learnerId}/courses", {
      params: { path: { learnerId: learner.id } },
      body: { courseIds: next },
    });
    setPending(false);
    if (!data) {
      setSelected(prev);
      setErr(errorMessage(error, "บันทึกไม่สำเร็จ"));
      return;
    }
    setSelected(data.courseIds ?? next);
  }

  return (
    <fieldset aria-busy={pending} data-testid={`course-access-${learner.email}`}>
      <legend className="sr-only">
        {T.course}ที่ {learner.name} เข้าเรียนได้
      </legend>
      <div className="flex flex-col gap-1">
        {courses.map((c) => (
          <label key={c.id} className="flex items-center gap-2">
            <input
              type="checkbox"
              checked={selected.includes(c.id)}
              disabled={pending}
              onChange={(e) => toggle(c.id, e.currentTarget.checked)}
            />
            <span>{c.name}</span>
          </label>
        ))}
      </div>
      {err && (
        <p role="alert" className="mt-1 text-xs text-red-700">
          {err}
        </p>
      )}
    </fieldset>
  );
}
