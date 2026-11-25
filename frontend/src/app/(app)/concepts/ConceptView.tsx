"use client";

import Link from "next/link";
import { Markdown } from "@/components/Markdown";
import { MasteryChip } from "@/components/MasteryChip";
import { StartSessionButton } from "@/components/StartSessionButton";
import { EmptyState } from "@/components/StateViews";
import { NotFoundView } from "@/components/ErrorViews";
import { Loaded } from "@/components/Loaded";
import { api, unwrap } from "@/lib/api/client";
import { getCourse, getCourseGraph, getMastery } from "@/lib/api/queries";
import { useLoad } from "@/lib/api/use-load";
import type { ConceptRef } from "@/lib/api/types";
import { routes } from "@/lib/routes";
import { T } from "@/lib/terms";
import { useIdParam } from "@/lib/use-id-param";
import { btn, card, cn } from "@/lib/ui";

async function loadConcept(id: string) {
  const concept = await api().GET("/concepts/{conceptId}", { params: { path: { conceptId: id } } }).then(unwrap);
  const [course, graph, mastery] = await Promise.all([
    getCourse(concept.courseId),
    getCourseGraph(concept.courseId),
    getMastery(concept.courseId),
  ]);
  return { concept, course, graph, mastery };
}

export function ConceptView() {
  const id = useIdParam();
  const state = useLoad(id && `concept:${id}`, () => loadConcept(id!));
  if (!id) return <NotFoundView />;
  return <Loaded state={state}>{(d) => <ConceptPage {...d} />}</Loaded>;
}

function ConceptPage({ concept, course, graph, mastery }: Awaited<ReturnType<typeof loadConcept>>) {
  const levelOf = new Map(mastery.map((m) => [m.conceptId, m.level]));
  const nameOf = new Map(graph.concepts.map((c) => [c.id, c.name]));
  const toRef = (cid: string): ConceptRef => ({ id: cid, name: nameOf.get(cid) ?? "แนวคิด", level: levelOf.get(cid) });
  const prereqs = concept.prerequisiteIds.map(toRef);
  const dependents = concept.dependentIds.map(toRef);
  const canQuiz = (concept.activeQuestionCount ?? 1) > 0;

  return (
    <div className="grid gap-6 lg:grid-cols-[1fr_20rem]">
      <article className="min-w-0">
        <p className="text-sm text-slate-600">
          <Link href={routes.course(course.id)} className="text-blue-800 hover:underline">
            {course.name}
          </Link>
          <span aria-hidden="true"> / </span>
          {T.topic}: {concept.topicName ?? "-"}
        </p>
        <h1 className="mt-1 text-2xl font-bold">{concept.name}</h1>
        <div className="mt-2 flex flex-wrap items-center gap-2">
          <span className="text-sm text-slate-700">{T.masteryLevel}ของคุณ:</span>
          <MasteryChip level={concept.mastery.level} score={concept.mastery.score} size="md" />
        </div>

        <div className="mt-4 flex flex-wrap gap-2">
          {canQuiz ? (
            <StartSessionButton request={{ kind: "quiz", courseId: course.id, conceptId: concept.id }} testId="start-quiz">
              เริ่ม{T.quiz}เรื่องนี้
            </StartSessionButton>
          ) : (
            <p className="text-sm text-slate-600">ยังไม่มี{T.question}สำหรับ{T.concept}นี้ (แจ้งผู้ดูแลเนื้อหาแล้ว)</p>
          )}
          <Link href={routes.courseMap(course.id)} className={btn.secondary}>
            ดูบน{T.masteryMap}
          </Link>
        </div>

        <section className={cn(card, "mt-6")} aria-labelledby="lesson-h">
          <h2 id="lesson-h" className="mb-2 text-lg font-semibold">
            {T.lesson}
          </h2>
          {concept.lessonMd ? (
            <Markdown>{concept.lessonMd}</Markdown>
          ) : (
            <EmptyState title={`ยังไม่มี${T.lesson}สำหรับเรื่องนี้`}>ลองทำ{T.quiz}เพื่อฝึกไปก่อน แล้วอ่านคำอธิบายเฉลยหลังตอบแต่ละข้อ</EmptyState>
          )}
        </section>
      </article>

      <aside className="flex flex-col gap-4">
        <ConceptList title={T.prerequisite} empty="ไม่มี — เรื่องนี้เป็นเรื่องพื้นฐาน" items={prereqs} />
        <ConceptList title={T.dependents} empty="ยังไม่มีเรื่องที่ต่อยอด" items={dependents} />
      </aside>
    </div>
  );
}

function ConceptList({ title, items, empty }: { title: string; items: ConceptRef[]; empty: string }) {
  return (
    <section className={card} aria-label={title}>
      <h2 className="mb-2 font-semibold">{title}</h2>
      {items.length === 0 ? (
        <p className="text-sm text-slate-600">{empty}</p>
      ) : (
        <ul className="flex flex-col gap-2">
          {items.map((c) => (
            <li key={c.id} className="flex items-center justify-between gap-2 text-sm">
              <Link href={routes.concept(c.id)} className="text-blue-800 hover:underline">
                {c.name}
              </Link>
              <MasteryChip level={c.level ?? "unassessed"} />
            </li>
          ))}
        </ul>
      )}
    </section>
  );
}
