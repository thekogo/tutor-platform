import { forbidden } from "next/navigation";
import { CreateInvite } from "@/components/admin/CreateInvite";
import { RetireQuestionButton } from "@/components/admin/RetireQuestionButton";
import { PageHeader } from "@/components/StateViews";
import { getMe } from "@/lib/api/queries";
import { serverApi, unwrap } from "@/lib/api/server";
import { T } from "@/lib/terms";
import { card, formatDateTime, isPast } from "@/lib/ui";

export const metadata = { title: "จัดการระบบ" };

export default async function AdminPage() {
  const me = await getMe();
  if (!me.isOwner) forbidden();
  const api = await serverApi();
  const [invites, learners, flags, requests] = await Promise.all([
    api.GET("/admin/invites").then(unwrap),
    api.GET("/admin/learners").then(unwrap),
    api.GET("/admin/flags", { params: { query: { open: true } } }).then(unwrap),
    api.GET("/admin/content-requests").then(unwrap),
  ]);

  return (
    <div data-testid="admin-page" className="flex flex-col gap-6">
      <PageHeader title="จัดการระบบ" description={`สำหรับ${T.owner}: เชิญ${T.learner} ตรวจ${T.flags} และ${T.contentRequest}`} />

      <section className={card} aria-labelledby="inv-h">
        <h2 id="inv-h" className="mb-3 text-lg font-semibold">{T.invite}</h2>
        <CreateInvite />
        {invites.length > 0 && (
          <table className="mt-4 w-full text-sm">
            <caption className="sr-only">{T.invite}ทั้งหมด</caption>
            <thead>
              <tr className="border-b border-slate-200 text-left text-slate-600">
                <th scope="col" className="py-2">สร้างเมื่อ</th>
                <th scope="col" className="py-2">หมดอายุ</th>
                <th scope="col" className="py-2">สถานะ</th>
              </tr>
            </thead>
            <tbody>
              {invites.map((i) => {
                const expired = isPast(i.expiresAt);
                return (
                  <tr key={i.id} className="border-b border-slate-100">
                    <td className="py-2">{formatDateTime(i.createdAt)}</td>
                    <td className="py-2">{formatDateTime(i.expiresAt)}</td>
                    <td className="py-2">{i.usedBy ? `ใช้แล้วโดย ${i.usedBy}` : expired ? "หมดอายุ" : "ยังไม่ถูกใช้"}</td>
                  </tr>
                );
              })}
            </tbody>
          </table>
        )}
      </section>

      <section className={card} aria-labelledby="flags-h">
        <h2 id="flags-h" className="mb-3 text-lg font-semibold">{T.flags}ที่ยังเปิดอยู่ ({flags.length})</h2>
        {flags.length === 0 ? (
          <p className="text-slate-600">ไม่มีรายการค้าง</p>
        ) : (
          <ul className="flex flex-col gap-3">
            {flags.map((f) => (
              <li key={f.id} className="rounded-lg border border-slate-200 p-3">
                <p className="text-sm text-slate-600">
                  {f.learnerName ?? T.learner} · {formatDateTime(f.createdAt)} · {T.question} <code className="text-xs">{f.questionId}</code>
                </p>
                <p className="mt-1 whitespace-pre-wrap">{f.reason}</p>
                <div className="mt-2">
                  <RetireQuestionButton questionId={f.questionId} />
                </div>
              </li>
            ))}
          </ul>
        )}
        <p className="mt-3 text-xs text-slate-500">การแก้ไขเฉลยและปิดรายการทำผ่านเอเจนต์ผู้ดูแลเนื้อหา (MCP)</p>
      </section>

      <section className={card} aria-labelledby="cr-h">
        <h2 id="cr-h" className="mb-3 text-lg font-semibold">{T.contentRequest} ({requests.length})</h2>
        {requests.length === 0 ? (
          <p className="text-slate-600">ไม่มีคำขอค้าง</p>
        ) : (
          <ul className="flex flex-col gap-2">
            {requests.map((r) => (
              <li key={r.id} className="rounded-lg border border-slate-200 p-3">
                <p className="font-medium">{r.conceptName}</p>
                <p className="text-sm text-slate-700">{r.reason}</p>
                <p className="text-xs text-slate-500">{formatDateTime(r.createdAt)}</p>
              </li>
            ))}
          </ul>
        )}
      </section>

      <section className={card} aria-labelledby="ln-h">
        <h2 id="ln-h" className="mb-3 text-lg font-semibold">{T.learners} ({learners.length})</h2>
        <table className="w-full text-sm">
          <caption className="sr-only">รายชื่อ{T.learners}</caption>
          <thead>
            <tr className="border-b border-slate-200 text-left text-slate-600">
              <th scope="col" className="py-2">ชื่อ</th>
              <th scope="col" className="py-2">อีเมล</th>
              <th scope="col" className="py-2">บทบาท</th>
              <th scope="col" className="py-2">{T.placementTest}</th>
            </tr>
          </thead>
          <tbody>
            {learners.map((l) => (
              <tr key={l.id} className="border-b border-slate-100">
                <td className="py-2">{l.name}</td>
                <td className="py-2 break-all">{l.email}</td>
                <td className="py-2">{l.isOwner ? T.owner : T.learner}</td>
                <td className="py-2">{l.placementDone ? "ทำแล้ว" : "ยังไม่ทำ"}</td>
              </tr>
            ))}
          </tbody>
        </table>
      </section>

      <section className={card} aria-labelledby="retire-h">
        <h2 id="retire-h" className="mb-3 text-lg font-semibold">เลิกใช้{T.question}</h2>
        <RetireQuestionButton withInput />
      </section>
    </div>
  );
}
