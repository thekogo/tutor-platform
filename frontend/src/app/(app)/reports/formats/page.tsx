import { EmptyState, PageHeader } from "@/components/StateViews";
import { getDefaultCourse } from "@/lib/api/queries";
import { serverApi, unwrap } from "@/lib/api/server";
import { T } from "@/lib/terms";
import { card, cn, percent } from "@/lib/ui";

export const metadata = { title: `รายงาน${T.questionFormat}` };

export default async function FormatReportPage() {
  const course = await getDefaultCourse();
  if (!course) return <EmptyState title={`ยังไม่มี${T.course}`} />;
  const api = await serverApi();
  const stats = await api
    .GET("/courses/{courseId}/format-report", { params: { path: { courseId: course.id } } })
    .then(unwrap);
  const answeredAny = stats.some((s) => s.answered > 0);

  return (
    <div>
      <PageHeader
        title={`รายงาน${T.questionFormat}`}
        description={`ความแม่นยำแยกตาม${T.questionFormat} (เช่น Data sufficiency, คณิตคิดนอกกรอบ) ไม่ขึ้นกับ${T.concept}ที่ออกสอบ`}
      />
      {!answeredAny ? (
        <EmptyState title="ยังไม่มีข้อมูล">ทำ{T.quiz}หรือ{T.exam}ก่อน แล้วรายงานจะแสดงที่นี่</EmptyState>
      ) : (
        <div className={card}>
          <table className="w-full text-sm">
            <caption className="sr-only">ความแม่นยำแยกตาม{T.questionFormat}</caption>
            <thead>
              <tr className="border-b border-slate-200 text-left text-slate-600">
                <th scope="col" className="py-2">{T.questionFormat}</th>
                <th scope="col" className="py-2 text-right">ตอบ</th>
                <th scope="col" className="py-2 text-right">ถูก</th>
                <th scope="col" className="py-2 text-right">ความแม่นยำ</th>
                <th scope="col" className="w-1/3 py-2"><span className="sr-only">กราฟ</span></th>
              </tr>
            </thead>
            <tbody>
              {stats.map((s) => {
                const low = s.answered > 0 && s.accuracy < 0.5;
                return (
                  <tr key={s.format} className="border-b border-slate-100">
                    <th scope="row" className="py-3 text-left font-medium">{s.formatName}</th>
                    <td className="py-3 text-right tabular-nums">{s.answered}</td>
                    <td className="py-3 text-right tabular-nums">{s.correct}</td>
                    <td className={cn("py-3 text-right font-semibold tabular-nums", low && "text-red-700")}>
                      {s.answered ? percent(s.accuracy) : "–"}
                      {low && <span className="ml-1 text-xs font-normal">(ควรฝึกเพิ่ม)</span>}
                    </td>
                    <td className="py-3 pl-4">
                      <div className="h-2.5 rounded-full bg-slate-200" aria-hidden="true">
                        <div className={cn("h-2.5 rounded-full", low ? "bg-red-500" : "bg-blue-600")} style={{ width: percent(s.accuracy) }} />
                      </div>
                    </td>
                  </tr>
                );
              })}
            </tbody>
          </table>
        </div>
      )}
    </div>
  );
}
