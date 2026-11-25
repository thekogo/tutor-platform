"use client";

import Link from "next/link";
import { MasteryChip } from "@/components/MasteryChip";
import { MasteryMap } from "@/components/MasteryMap";
import { EmptyState, PageHeader } from "@/components/StateViews";
import { NotFoundView } from "@/components/ErrorViews";
import { Loaded } from "@/components/Loaded";
import { getCourseGraph, getMastery } from "@/lib/api/queries";
import { useLoad } from "@/lib/api/use-load";
import type { CourseGraph, Mastery } from "@/lib/api/types";
import { masteryByConcept } from "@/lib/graph-layout";
import { routes } from "@/lib/routes";
import { T } from "@/lib/terms";
import { useIdParam } from "@/lib/use-id-param";

export function MasteryMapView() {
  const id = useIdParam();
  const state = useLoad(id && `map:${id}`, () => Promise.all([getCourseGraph(id!), getMastery(id!)]));
  if (!id) return <NotFoundView />;
  return <Loaded state={state}>{([course, mastery]) => <MasteryMapPage course={course} mastery={mastery} />}</Loaded>;
}

function MasteryMapPage({ course, mastery }: { course: CourseGraph; mastery: Mastery[] }) {
  const byConcept = masteryByConcept(mastery);

  return (
    <div>
      <PageHeader
        title={T.masteryMap}
        description={`${course.name} · แต่ละกล่องคือ${T.concept} ลูกศรชี้จาก${T.prerequisite}ไปยังเรื่องที่ต่อยอด คลิกเพื่อเปิดบทเรียน`}
      />
      {course.concepts.length === 0 ? (
        <EmptyState title={`คอร์สนี้ยังไม่มี${T.concept}`} />
      ) : (
        <>
          <MasteryMap course={course} mastery={mastery} />
          <details className="mt-6 rounded-xl border border-slate-200 bg-white p-4">
            <summary className="cursor-pointer font-medium">ดู{T.masteryMap}แบบรายการ (เหมาะกับมือถือและโปรแกรมอ่านหน้าจอ)</summary>
            <div className="mt-4 grid gap-6 md:grid-cols-2 xl:grid-cols-3">
              {course.topics.map((t) => {
                const concepts = course.concepts.filter((c) => c.topicId === t.id);
                if (concepts.length === 0) return null;
                return (
                  <section key={t.id} aria-labelledby={`t-${t.id}`}>
                    <h2 id={`t-${t.id}`} className="mb-2 font-semibold">
                      {t.name}
                    </h2>
                    <ul className="flex flex-col gap-1">
                      {concepts.map((c) => {
                        const m = byConcept.get(c.id);
                        return (
                          <li key={c.id} className="flex items-center justify-between gap-2 text-sm">
                            <Link href={routes.concept(c.id)} className="text-blue-800 hover:underline">
                              {c.name}
                            </Link>
                            <MasteryChip level={m?.level ?? "unassessed"} />
                          </li>
                        );
                      })}
                    </ul>
                  </section>
                );
              })}
            </div>
          </details>
        </>
      )}
    </div>
  );
}
