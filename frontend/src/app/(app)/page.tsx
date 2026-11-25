import Link from "next/link";
import { EmptyState, PageHeader } from "@/components/StateViews";
import { getCourses, getMe } from "@/lib/api/queries";
import { serverApi, unwrap, unwrapOptional } from "@/lib/api/server";
import { MASTERY_LEVELS, type CourseSummary } from "@/lib/api/types";
import { countByLevel } from "@/lib/graph-layout";
import { MASTERY_LEVEL_LABEL, T } from "@/lib/terms";
import { btn, card, cn } from "@/lib/ui";

/** Solid fills for the progress bar (Tailwind needs literal class names). */
const BAR_CLASS = { weak: "bg-level-weak", developing: "bg-level-developing", mastered: "bg-level-mastered" } as const;

export const metadata = { title: `เลือก${T.course}` };

export default async function CourseSelectPage() {
  const [me, courses] = await Promise.all([getMe(), getCourses()]);

  return (
    <div className="flex flex-col gap-6">
      <PageHeader title={`สวัสดี ${me.name}`} description={`เลือก${T.course}ที่ต้องการเรียน ความคืบหน้าของแต่ละ${T.course}แยกจากกัน`} />
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

async function CourseCard({ course }: { course: CourseSummary }) {
  const api = await serverApi();
  const p = { params: { path: { courseId: course.id } } };
  const [graph, mastery, path] = await Promise.all([
    api.GET("/courses/{courseId}", p).then(unwrap),
    api.GET("/courses/{courseId}/mastery", p).then(unwrap),
    api.GET("/courses/{courseId}/learning-path", p).then(unwrapOptional),
  ]);
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
        <Link href={`/courses/${course.id}`} className={btn.primary}>
          {started ? "เรียนต่อ" : "เริ่มเรียน"}
          <span className="sr-only"> {course.name}</span>
        </Link>
      </div>
    </article>
  );
}
