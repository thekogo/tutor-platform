import Link from "next/link";
import { MasteryChip } from "@/components/MasteryChip";
import { RecommendationCard } from "@/components/RecommendationCard";
import { StartSessionButton } from "@/components/StartSessionButton";
import { EmptyState, PageHeader } from "@/components/StateViews";
import { getDefaultCourse, getMe } from "@/lib/api/queries";
import { serverApi, unwrap, unwrapOptional } from "@/lib/api/server";
import { MASTERY_LEVELS } from "@/lib/api/types";
import { formatTimeLimit } from "@/lib/countdown";
import { countByLevel } from "@/lib/graph-layout";
import { T } from "@/lib/terms";
import { btn, card, cn, percent } from "@/lib/ui";

export const metadata = { title: "หน้าหลัก" };

export default async function DashboardPage() {
  const [me, course] = await Promise.all([getMe(), getDefaultCourse()]);
  if (!course) {
    return (
      <EmptyState title={`ยังไม่มี${T.course}`}>
        เจ้าของระบบยังไม่ได้นำเข้าเนื้อหา กรุณากลับมาใหม่ภายหลัง
      </EmptyState>
    );
  }
  const api = await serverApi();
  const p = { params: { path: { courseId: course.id } } };
  const [recs, exams, mastery, graph, path] = await Promise.all([
    api.GET("/courses/{courseId}/recommendations", p).then(unwrap),
    api.GET("/courses/{courseId}/exams", p).then(unwrap),
    api.GET("/courses/{courseId}/mastery", p).then(unwrap),
    api.GET("/courses/{courseId}", p).then(unwrap),
    api.GET("/courses/{courseId}/learning-path", p).then(unwrapOptional),
  ]);
  const [top, ...others] = recs;
  const counts = countByLevel(
    graph.concepts.map((c) => c.id),
    mastery,
  );

  return (
    <div className="flex flex-col gap-8">
      <PageHeader title={`สวัสดี ${me.name}`} description={`${course.name}${course.goal ? ` · ${course.goal}` : ""}`} />

      {!me.placementDone && (
        <section data-testid="placement-prompt" className="rounded-xl border border-amber-200 bg-amber-50 p-5 shadow-sm" aria-labelledby="placement-h">
          <h2 id="placement-h" className="text-lg font-semibold">
            เริ่มต้นด้วย{T.placementTest} (ไม่บังคับ)
          </h2>
          <p className="mt-1 text-slate-700">
            ทำแบบทดสอบสั้น ๆ ที่ปรับตามคำตอบของคุณ เพื่อให้ระบบรู้{T.masteryLevel}เริ่มต้นและแนะนำได้แม่นยำขึ้น
            ข้ามได้ แล้วระบบจะเรียนรู้จาก{T.quiz}ที่คุณทำแทน
          </p>
          <div className="mt-3">
            <StartSessionButton request={{ kind: "placement", courseId: course.id }}>เริ่ม{T.placementTest}</StartSessionButton>
          </div>
        </section>
      )}

      {path && path.currentStepConceptId && (
        <section className={cn(card, "flex flex-wrap items-center justify-between gap-3")} aria-label={T.learningPath}>
          <p>
            คุณกำลังเรียนตาม{T.learningPath} ขั้นปัจจุบัน:{" "}
            <strong>{path.steps.find((s) => s.id === path.currentStepConceptId)?.name}</strong>
          </p>
          <Link href="/learning-path" className={btn.secondary}>
            ไปที่{T.learningPath}
          </Link>
        </section>
      )}

      <section aria-labelledby="rec-h">
        <h2 id="rec-h" className="sr-only">
          {T.recommendation}
        </h2>
        {top ? (
          <RecommendationCard recommendation={top} courseId={course.id} top />
        ) : (
          <EmptyState title={`ยังไม่มี${T.recommendation}ในตอนนี้`}>
            {mastery.some((m) => m.level !== "unassessed")
              ? "ยอดเยี่ยม! ไม่พบแนวคิดที่ยังอ่อน ลองทำชุดข้อสอบเพื่อตรวจสอบความพร้อม"
              : `ทำ${T.placementTest}หรือ${T.quiz}สักชุด เพื่อให้ระบบเริ่มวิเคราะห์จุดอ่อนของคุณ`}
          </EmptyState>
        )}
      </section>

      {others.length > 0 && (
        <section aria-labelledby="others-h">
          <h2 id="others-h" className="mb-3 text-lg font-semibold">
            {T.rootCauses}อื่น ๆ
          </h2>
          <div className="grid gap-4 lg:grid-cols-2">
            {others.map((r) => (
              <RecommendationCard key={`${r.kind}-${r.conceptId}-${r.targetConceptId}`} recommendation={r} courseId={course.id} headingLevel="h3" />
            ))}
          </div>
        </section>
      )}

      <div className="grid gap-6 lg:grid-cols-2">
        <section className={card} aria-labelledby="map-h">
          <h2 id="map-h" className="text-lg font-semibold">
            {T.masteryMap}
          </h2>
          <p className="mt-1 text-sm text-slate-600">
            สรุป{T.masteryLevel}จาก {graph.concepts.length} {T.concept}
          </p>
          <ul className="mt-3 grid grid-cols-2 gap-2">
            {MASTERY_LEVELS.map((l) => (
              <li key={l} className="flex items-center justify-between rounded-lg border border-slate-200 px-3 py-2">
                <MasteryChip level={l} />
                <span className="text-lg font-semibold tabular-nums">{counts[l]}</span>
              </li>
            ))}
          </ul>
          <Link href={`/courses/${course.id}/map`} className={cn(btn.secondary, "mt-4")}>
            เปิด{T.masteryMap}
          </Link>
        </section>

        <section className={card} aria-labelledby="exams-h">
          <h2 id="exams-h" className="text-lg font-semibold">
            {T.exam}
          </h2>
          <p className="mt-1 text-sm text-slate-600">จับเวลาเหมือนสอบจริง เฉลยจะแสดงหลังส่งข้อสอบ</p>
          {exams.length === 0 ? (
            <p className="mt-3 text-slate-600">ยังไม่มี{T.exam}</p>
          ) : (
            <ul className="mt-3 flex flex-col gap-3">
              {exams.map((e) => (
                <li key={e.id} className="flex flex-wrap items-center justify-between gap-2 rounded-lg border border-slate-200 p-3">
                  <div>
                    <p className="font-medium">{e.name}</p>
                    <p className="text-sm text-slate-600">
                      {e.questionCount} ข้อ · {formatTimeLimit(e.timeLimitSeconds)}
                      {e.lastScore != null && ` · คะแนนล่าสุด ${percent(e.lastScore)}`}
                    </p>
                  </div>
                  <StartSessionButton request={{ kind: "exam", courseId: course.id, examId: e.id }} variant="secondary">
                    เริ่มสอบ
                  </StartSessionButton>
                </li>
              ))}
            </ul>
          )}
        </section>
      </div>
    </div>
  );
}
