"use client";

import Link from "next/link";
import { Loaded } from "@/components/Loaded";
import { EmptyState, PageHeader } from "@/components/StateViews";
import { api, unwrapOptional } from "@/lib/api/client";
import { getCourseGraph, getCourses, getMastery, getMe } from "@/lib/api/queries";
import { useLoad } from "@/lib/api/use-load";
import { MASTERY_LEVELS, type CourseGraph, type CourseSummary, type LearningPath, type Mastery } from "@/lib/api/types";
import { countByLevel } from "@/lib/graph-layout";
import { MASTERY_LEVEL_LABEL, T } from "@/lib/terms";
import { routes } from "@/lib/routes";
import { btn, card, cn } from "@/lib/ui";

/** Solid fills for the progress bar (Tailwind needs literal class names). */
const BAR_CLASS = { weak: "bg-level-weak", developing: "bg-level-developing", mastered: "bg-level-mastered" } as const;

export function CourseSelectView() {
  const state = useLoad("home", () => Promise.all([getMe(), getCourses()]));
  return <Loaded state={state}>{([me, courses]) => <CourseSelect name={me.name} courses={courses} />}</Loaded>;
}

function CourseSelect({ name, courses }: { name: string; courses: CourseSummary[] }) {
  return (
    <div className="flex flex-col gap-6">
      <PageHeader title={`สวัสดี ${name}`} description={`เลือก${T.course}ที่ต้องการเรียน ความคืบหน้าของแต่ละ${T.course}แยกจากกัน`} />
      {courses.length === 0 ? (
        <EmptyState title={`ยังไม่มี${T.course}`}>เจ้าของระบบยังไม่ได้นำเข้าเนื้อหา กรุณากลับมาใหม่ภายหลัง</EmptyState>
      ) : (
        <ul className="grid gap-5 md:grid-cols-2" aria-label={`${T.course}ทั้งหมด`}>
          {courses.map((c) => (
            <li key={c.id}>
              <CourseCard course={c} />
            </li>
          ))}
        </ul>
      )}
    </div>
  );
}

function CourseCard({ course }: { course: CourseSummary }) {
  const state = useLoad(`course-card:${course.id}`, () =>
    Promise.all([
      getCourseGraph(course.id),
      getMastery(course.id),
      api()
        .GET("/courses/{courseId}/learning-path", { params: { path: { courseId: course.id } } })
        .then(unwrapOptional),
    ]),
  );
  return (
    <Loaded state={state} loading={<div className={cn(card, "h-56 animate-pulse bg-slate-100")} aria-hidden="true" />}>
      {([graph, mastery, path]) => <CourseCardBody course={course} graph={graph} mastery={mastery} path={path} />}
    </Loaded>
  );
}

type CardData = {
  course: CourseSummary;
  graph: CourseGraph;
  mastery: Mastery[];
  path: LearningPath | null;
};

function CourseCardBody({ course, graph, mastery, path }: CardData) {
  const total = graph.concepts.length;
  const counts = countByLevel(
    graph.concepts.map((c) => c.id),
    mastery,
  );
  const started = counts.unassessed < total;
  const currentStep = path?.currentStepConceptId ? path.steps.find((s) => s.id === path.currentStepConceptId) : undefined;
  const headingId = `course-${course.id}`;

  return (
    <article data-testid="course-card" className={cn(card, "flex h-full flex-col gap-4")} aria-labelledby={headingId}>
      <div>
        <h2 id={headingId} className="text-xl font-semibold text-slate-900">
          {course.name}
        </h2>
        {course.goal && <p className="mt-1 text-sm text-slate-600">{course.goal}</p>}
        <p className="mt-2 text-sm text-slate-600">
          {graph.topics.length} {T.topic} · {total} {T.concept}
        </p>
      </div>

      {total > 0 && (
        <div>
          <div className="flex h-2.5 overflow-hidden rounded-full bg-slate-100" aria-hidden="true">
            {(["mastered", "developing", "weak"] as const).map((l) =>
              counts[l] > 0 ? <div key={l} className={BAR_CLASS[l]} style={{ width: `${(counts[l] / total) * 100}%` }} /> : null,
            )}
          </div>
          <p className="mt-2 text-xs text-slate-600">
            {started
              ? MASTERY_LEVELS.filter((l) => l !== "unassessed")
                  .map((l) => `${MASTERY_LEVEL_LABEL[l]} ${counts[l]}`)
                  .join(" · ") + ` · ${MASTERY_LEVEL_LABEL.unassessed} ${counts.unassessed}`
              : `ยังไม่ได้เริ่มเรียน${T.course}นี้`}
          </p>
        </div>
      )}

      <ul className="flex flex-col gap-1 text-sm text-slate-700">
        <li>
          {T.placementTest}: {course.placementDone ? "ทำแล้ว" : "ยังไม่ทำ"}
        </li>
        {currentStep && (
          <li>
            {T.learningPath}: <strong>{currentStep.name}</strong>
          </li>
        )}
      </ul>

      <div className="mt-auto">
        <Link href={routes.course(course.id)} className={btn.primary}>
          {started ? "เรียนต่อ" : "เริ่มเรียน"}
          <span className="sr-only"> {course.name}</span>
        </Link>
      </div>
    </article>
  );
}
