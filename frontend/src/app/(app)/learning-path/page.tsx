import Link from "next/link";
import { AbandonPathButton } from "@/components/AbandonPathButton";
import { MasteryChip } from "@/components/MasteryChip";
import { StartSessionButton } from "@/components/StartSessionButton";
import { EmptyState, PageHeader } from "@/components/StateViews";
import { getDefaultCourse } from "@/lib/api/queries";
import { serverApi, unwrapOptional } from "@/lib/api/server";
import { T } from "@/lib/terms";
import { btn, card, cn, formatDateTime } from "@/lib/ui";

export const metadata = { title: T.learningPath };

export default async function LearningPathPage() {
  const course = await getDefaultCourse();
  if (!course) return <EmptyState title={`ยังไม่มี${T.course}`} />;
  const api = await serverApi();
  const path = await api
    .GET("/courses/{courseId}/learning-path", { params: { path: { courseId: course.id } } })
    .then(unwrapOptional);

  if (!path) {
    return (
      <div>
        <PageHeader title={T.learningPath} />
        <EmptyState
          title={`ยังไม่มี${T.learningPath}ที่กำลังเรียน`}
          action={
            <Link href="/" className={btn.primary}>
              ดู{T.recommendation}ที่หน้าหลัก
            </Link>
          }
        >
          เริ่ม{T.learningPath}ได้จาก{T.recommendation} ระบบจะพาเรียนจาก{T.rootCause}ไล่ขึ้นไปทีละเรื่อง
        </EmptyState>
      </div>
    );
  }

  const done = !path.currentStepConceptId;
  const currentIndex = path.steps.findIndex((s) => s.id === path.currentStepConceptId);

  return (
    <div>
      <PageHeader
        title={T.learningPath}
        description={`เริ่มเมื่อ ${formatDateTime(path.startedAt)} · เรียนจาก${T.rootCause}ไล่ขึ้นไปทีละเรื่อง ข้ามเรื่องที่อยู่ระดับกำลังพัฒนาขึ้นไปแล้ว`}
        actions={<AbandonPathButton courseId={course.id} />}
      />
      {done && (
        <div className={cn(card, "mb-6 border-green-200 bg-green-50")} role="status">
          <p className="font-semibold text-green-800">🎉 จบ{T.learningPath}แล้ว!</p>
          <p className="text-sm text-green-900">กลับไปหน้าหลักเพื่อดู{T.recommendation}ถัดไป</p>
        </div>
      )}
      <ol className="flex flex-col gap-3" aria-label="ขั้นตอนของเส้นทางการเรียน">
        {path.steps.map((s, i) => {
          const current = s.id === path.currentStepConceptId;
          const passed = !done && currentIndex >= 0 ? i < currentIndex : done;
          return (
            <li
              key={s.id}
              aria-current={current ? "step" : undefined}
              className={cn(
                card,
                "flex flex-wrap items-center gap-4",
                current && "border-blue-400 ring-2 ring-blue-200",
                passed && !current && "opacity-75",
              )}
            >
              <span
                className={cn(
                  "flex size-9 shrink-0 items-center justify-center rounded-full text-sm font-bold",
                  current ? "bg-blue-700 text-white" : "bg-slate-200 text-slate-700",
                )}
                aria-hidden="true"
              >
                {i + 1}
              </span>
              <div className="min-w-0 flex-1">
                <p className="text-xs text-slate-500">
                  {i === 0 ? T.rootCause : i === path.steps.length - 1 ? "เป้าหมาย" : `ขั้นที่ ${i + 1}`}
                  {current && <strong className="ml-2 text-blue-800">· ขั้นปัจจุบัน</strong>}
                </p>
                <Link href={`/concepts/${s.id}`} className="font-medium text-slate-900 hover:underline">
                  {s.name}
                </Link>
              </div>
              {s.level && <MasteryChip level={s.level} />}
              {current && (
                <div className="flex gap-2">
                  <Link href={`/concepts/${s.id}`} className={btn.secondary}>
                    อ่าน{T.lesson}
                  </Link>
                  <StartSessionButton request={{ kind: "quiz", courseId: course.id, conceptId: s.id }}>
                    ทำ{T.quiz}
                  </StartSessionButton>
                </div>
              )}
            </li>
          );
        })}
      </ol>
    </div>
  );
}
