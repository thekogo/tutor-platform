import Link from "next/link";
import type { ConceptRef, Recommendation } from "@/lib/api/types";
import { T } from "@/lib/terms";
import { btn, card, cn } from "@/lib/ui";
import { MasteryChip } from "./MasteryChip";
import { StartLearningPathButton } from "./StartLearningPathButton";
import { StartSessionButton } from "./StartSessionButton";

type Props = {
  recommendation: Recommendation;
  courseId: string;
  /** The top Recommendation gets the prominent layout. */
  top?: boolean;
  headingLevel?: "h2" | "h3";
};

/** The reason path, ordered from the Root Cause up to the struggling Concept. */
export function reasonPath(rec: Recommendation): ConceptRef[] {
  if (rec.path.length > 0) return rec.path;
  // Defensive: the reason must always be visible even if the path is missing.
  return [{ id: rec.conceptId, name: rec.conceptName }];
}

export function ReasonPath({ recommendation }: { recommendation: Recommendation }) {
  const steps = reasonPath(recommendation);
  const lastIndex = steps.length - 1;
  return (
    <ol
      data-testid="recommendation-path"
      aria-label="เส้นทางเหตุผล จากต้นเหตุไปยังแนวคิดที่ติดขัด"
      className="flex flex-wrap items-stretch gap-x-1 gap-y-2"
    >
      {steps.map((step, i) => (
        <li key={step.id} className="flex items-center gap-1">
          <Link
            href={`/concepts/${step.id}`}
            className={cn(
              "flex flex-col gap-1 rounded-lg border bg-white px-3 py-2 text-sm hover:border-blue-400 hover:bg-blue-50",
              i === 0 ? "border-red-300 ring-1 ring-red-200" : "border-slate-200",
            )}
          >
            <span className="text-xs text-slate-500">
              {i === 0
                ? recommendation.kind === "diagnose"
                  ? "ต้องวัดก่อน"
                  : T.rootCause
                : i === lastIndex
                  ? "แนวคิดที่ติดขัด"
                  : `ขั้นที่ ${i + 1}`}
            </span>
            <span className="font-medium text-slate-900">{step.name}</span>
            {step.level && <MasteryChip level={step.level} className="self-start" />}
          </Link>
          {i < lastIndex && (
            <span aria-hidden="true" className="px-1 text-lg text-slate-400">
              →
            </span>
          )}
        </li>
      ))}
    </ol>
  );
}

export function RecommendationCard({ recommendation: rec, courseId, top = false, headingLevel = "h2" }: Props) {
  const steps = reasonPath(rec);
  const target = steps[steps.length - 1];
  const Heading = headingLevel;
  const isDiagnose = rec.kind === "diagnose";

  return (
    <article
      aria-labelledby={`rec-${rec.conceptId}`}
      data-testid={top ? "top-recommendation" : "recommendation"}
      className={cn(card, top && "border-blue-200 bg-gradient-to-br from-blue-50 to-white p-6")}
    >
      <p className="text-sm font-medium text-blue-800">
        {top ? `${T.recommendation}อันดับแรก` : T.recommendation}
        <span className="ml-2 text-slate-500">
          · ช่วยปลดล็อก {rec.unblocks} แนวคิดที่ยังอ่อน
        </span>
      </p>
      <Heading id={`rec-${rec.conceptId}`} className={cn("mt-1 font-bold text-slate-900", top ? "text-2xl" : "text-lg")}>
        {isDiagnose ? `วัดความเข้าใจเรื่อง “${rec.conceptName}” ก่อน` : `เรียน “${rec.conceptName}”`}
      </Heading>
      <p className="mt-2 text-slate-700">
        {isDiagnose ? (
          <>
            คุณยังติดขัดที่ <strong>{target.name}</strong> แต่เรายังไม่มีข้อมูลเรื่อง <strong>{rec.conceptName}</strong>{" "}
            ซึ่งเป็น{T.prerequisite} ทำ{T.diagnosticQuiz}สั้น ๆ เพื่อให้แนะนำได้แม่นขึ้น
          </>
        ) : steps.length > 1 ? (
          <>
            คุณยังติดขัดที่ <strong>{target.name}</strong> เพราะ{T.prerequisite}{" "}
            <strong>{rec.conceptName}</strong> ยังอ่อน แก้ที่{T.rootCause}ก่อนแล้วไล่ขึ้นมาตามเส้นทางนี้
          </>
        ) : (
          <>
            <strong>{rec.conceptName}</strong> ยังอ่อน และ{T.prerequisite}ของเรื่องนี้ไม่ได้อ่อน จึงควรเริ่มที่เรื่องนี้โดยตรง
          </>
        )}
      </p>

      <div className="mt-4">
        <h4 className="sr-only">เหตุผลของ{T.recommendation}</h4>
        <ReasonPath recommendation={rec} />
      </div>

      <div className="mt-5 flex flex-wrap gap-2">
        {isDiagnose ? (
          <StartSessionButton request={{ kind: "diagnostic", courseId, conceptId: rec.conceptId }}>
            ทำ{T.diagnosticQuiz}
          </StartSessionButton>
        ) : (
          <>
            {rec.targetConceptId && (
              <StartLearningPathButton
                courseId={courseId}
                rootConceptId={rec.conceptId}
                targetConceptId={rec.targetConceptId}
                variant={top ? "primary" : "secondary"}
              />
            )}
            <Link href={`/concepts/${rec.conceptId}`} className={btn.secondary}>
              อ่าน{T.lesson}
            </Link>
            <StartSessionButton request={{ kind: "quiz", courseId, conceptId: rec.conceptId }} variant="ghost">
              ทำ{T.quiz}
            </StartSessionButton>
          </>
        )}
      </div>
    </article>
  );
}
