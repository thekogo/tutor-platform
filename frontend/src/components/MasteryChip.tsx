import type { MasteryLevel } from "@/lib/api/types";
import { MASTERY_LEVEL_LABEL, T } from "@/lib/terms";
import { cn } from "@/lib/ui";

/** Icon per level so the level is readable without colour (paired with text). */
export const LEVEL_ICON: Record<MasteryLevel, string> = {
  unassessed: "?",
  weak: "!",
  developing: "◐",
  mastered: "✓",
};

export const LEVEL_CLASS: Record<MasteryLevel, string> = {
  unassessed: "bg-level-unassessed-bg text-level-unassessed border-level-unassessed/40 border-dashed",
  weak: "bg-level-weak-bg text-level-weak border-level-weak/40",
  developing: "bg-level-developing-bg text-level-developing border-level-developing/40",
  mastered: "bg-level-mastered-bg text-level-mastered border-level-mastered/40",
};

type Props = {
  level: MasteryLevel;
  /** Optional Mastery score 0–1, shown as a percentage. */
  score?: number | null;
  size?: "sm" | "md";
  className?: string;
};

export function MasteryChip({ level, score, size = "sm", className }: Props) {
  const label = MASTERY_LEVEL_LABEL[level];
  return (
    <span
      data-level={level}
      className={cn(
        "inline-flex shrink-0 items-center gap-1 rounded-full border font-medium whitespace-nowrap",
        size === "sm" ? "px-2 py-0.5 text-xs" : "px-3 py-1 text-sm",
        LEVEL_CLASS[level],
        className,
      )}
    >
      <span aria-hidden="true" className="font-bold">
        {LEVEL_ICON[level]}
      </span>
      <span className="sr-only">{T.masteryLevel}: </span>
      <span>{label}</span>
      {score != null && level !== "unassessed" && (
        <span className="opacity-80">({Math.round(score * 100)}%)</span>
      )}
    </span>
  );
}

export function MasteryLegend({ className }: { className?: string }) {
  const levels: MasteryLevel[] = ["weak", "developing", "mastered", "unassessed"];
  return (
    <div className={cn("flex flex-wrap items-center gap-2 text-sm", className)}>
      <span className="font-medium text-slate-700">{T.masteryLevel}:</span>
      <ul className="flex flex-wrap gap-2" aria-label={`คำอธิบาย${T.masteryLevel}`}>
        {levels.map((l) => (
          <li key={l}>
            <MasteryChip level={l} />
          </li>
        ))}
      </ul>
      <span className="text-xs text-slate-500">(ยังไม่ประเมิน = ยังไม่มีข้อมูล ไม่ได้แปลว่าอ่อน)</span>
    </div>
  );
}
