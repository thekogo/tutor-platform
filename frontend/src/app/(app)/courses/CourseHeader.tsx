"use client";

import Link from "next/link";
import { CourseNav } from "@/components/CourseNav";
import { getCourse } from "@/lib/api/queries";
import { useLoad } from "@/lib/api/use-load";
import { T } from "@/lib/terms";
import { useIdParam } from "@/lib/use-id-param";

/** Breadcrumb + tabs shared by every page of one Course (`?id=` is the Course id). */
export function CourseHeader() {
  const id = useIdParam();
  const course = useLoad(id && `course:${id}`, () => getCourse(id!));
  return (
    <div className="flex flex-col gap-3">
      <p className="text-sm text-slate-600">
        <Link href="/" className="text-blue-800 hover:underline">
          {T.course}ทั้งหมด
        </Link>
        {course.status === "ok" && (
          <>
            <span aria-hidden="true"> / </span>
            <span className="font-medium text-slate-900">{course.data.name}</span>
          </>
        )}
      </p>
      {id && <CourseNav courseId={id} />}
    </div>
  );
}
