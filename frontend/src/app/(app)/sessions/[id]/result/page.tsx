import Link from "next/link";
import { redirect } from "next/navigation";
import { Markdown } from "@/components/Markdown";
import { RecommendationCard } from "@/components/RecommendationCard";
import { StartLearningPathButton } from "@/components/StartLearningPathButton";
import { EmptyState, PageHeader } from "@/components/StateViews";
import { serverApi, unwrap } from "@/lib/api/server";
import { SESSION_KIND_LABEL, T } from "@/lib/terms";
import { btn, card, cn, percent } from "@/lib/ui";

export const metadata = { title: "ผลลัพธ์" };

export default async function SessionResultPage({ params }: { params: Promise<{ id: string }> }) {
  const { id } = await params;
  const api = await serverApi();
  const res = await api.GET("/sessions/{sessionId}/result", { params: { path: { sessionId: id } } });
  if (res.response.status === 409) redirect(`/sessions/${id}`);
  const result = unwrap(res);
  const session = await api.GET("/sessions/{sessionId}", { params: { path: { sessionId: id } } }).then(unwrap);

  const total = result.total ?? result.answered;
  const score = total > 0 ? result.correct / total : 0;
  const topStudy = result.rootCauses.find((r) => r.kind === "study" && r.targetConceptId);

  return (
    <div className="flex flex-col gap-6">
      <PageHeader
        title={`ผลลัพธ์${SESSION_KIND_LABEL[result.kind]}`}
        actions={
          <Link href="/" className={btn.secondary}>
            กลับหน้าหลัก
          </Link>
        }
      />

      <section className={cn(card, "flex flex-wrap items-center gap-6")} aria-label="คะแนน">
        <div>
          <p className="text-sm text-slate-600">คะแนน</p>
          <p className="text-4xl font-bold tabular-nums">
            {result.correct}
            <span className="text-2xl text-slate-500"> / {total}</span>
          </p>
        </div>
        <div className="text-lg font-semibold">{percent(score)}</div>
        <p className="text-sm text-slate-600">
          ตอบ {result.answered} ข้อ{total > result.answered ? ` · ไม่ได้ตอบ ${total - result.answered} ข้อ` : ""}
        </p>
        {topStudy && (
          <div className="ml-auto">
            <StartLearningPathButton
              courseId={session.courseId}
              rootConceptId={topStudy.conceptId}
              targetConceptId={topStudy.targetConceptId!}
              label={`เริ่ม${T.learningPath}: ${topStudy.conceptName}`}
            />
          </div>
        )}
      </section>

      <section className={card} aria-labelledby="topic-h">
        <h2 id="topic-h" className="mb-3 text-lg font-semibold">
          ผลราย{T.topic}
        </h2>
        {result.topicBreakdown.length === 0 ? (
          <p className="text-slate-600">ไม่มีคำตอบในชุดนี้</p>
        ) : (
          <table className="w-full text-sm">
            <caption className="sr-only">จำนวนข้อที่ตอบถูกแยกตาม{T.topic}</caption>
            <thead>
              <tr className="border-b border-slate-200 text-left text-slate-600">
                <th scope="col" className="py-2">{T.topic}</th>
                <th scope="col" className="py-2 text-right">ถูก / ตอบ</th>
                <th scope="col" className="w-1/3 py-2">
                  <span className="sr-only">สัดส่วน</span>
                </th>
              </tr>
            </thead>
            <tbody>
              {result.topicBreakdown.map((t) => {
                const acc = t.answered ? t.correct / t.answered : 0;
                return (
                  <tr key={t.topicId} className="border-b border-slate-100">
                    <th scope="row" className="py-2 text-left font-normal">{t.topicName}</th>
                    <td className="py-2 text-right tabular-nums">
                      {t.correct} / {t.answered} ({percent(acc)})
                    </td>
                    <td className="py-2 pl-3">
                      <div className="h-2 rounded-full bg-slate-200" aria-hidden="true">
                        <div className="h-2 rounded-full bg-blue-600" style={{ width: percent(acc) }} />
                      </div>
                    </td>
                  </tr>
                );
              })}
            </tbody>
          </table>
        )}
      </section>

      <section aria-labelledby="rc-h">
        <h2 id="rc-h" className="mb-3 text-lg font-semibold">
          {T.rootCauses}
        </h2>
        {result.rootCauses.length === 0 ? (
          <EmptyState title="ไม่พบต้นเหตุของจุดอ่อนจากชุดนี้" />
        ) : (
          <div className="grid gap-4 lg:grid-cols-2">
            {result.rootCauses.map((r, i) => (
              <RecommendationCard key={`${r.kind}-${r.conceptId}-${r.targetConceptId}`} recommendation={r} courseId={session.courseId} top={i === 0} headingLevel="h3" />
            ))}
          </div>
        )}
      </section>

      <section aria-labelledby="review-h">
        <h2 id="review-h" className="mb-3 text-lg font-semibold">
          ทบทวนรายข้อ
        </h2>
        <ol className="flex flex-col gap-4">
          {result.review.map((r, i) => (
            <li key={r.questionId} className={cn(card, r.correct ? "border-green-200" : "border-red-200")}>
              <div className="mb-2 flex items-center justify-between gap-2">
                <span className="font-semibold">ข้อที่ {i + 1}</span>
                <span className={cn("rounded-full px-2 py-0.5 text-sm font-medium", r.correct ? "bg-green-100 text-green-800" : "bg-red-100 text-red-800")}>
                  {r.correct ? "✓ ถูก" : r.chosenChoiceId ? "✗ ผิด" : "– ไม่ได้ตอบ"}
                </span>
              </div>
              <Markdown>{r.stemMd}</Markdown>
              <details className="mt-3">
                <summary className="cursor-pointer text-sm font-medium text-blue-800">ดู{T.explanation}</summary>
                <Markdown className="mt-2">{r.explanationMd}</Markdown>
              </details>
            </li>
          ))}
        </ol>
      </section>
    </div>
  );
}
