/** Milliseconds left until `deadline` (never negative). */
export function remainingMs(deadline: string | Date, now: number = Date.now()): number {
  const end = typeof deadline === "string" ? Date.parse(deadline) : deadline.getTime();
  if (Number.isNaN(end)) return 0;
  return Math.max(0, end - now);
}

/** "MM:SS", or "H:MM:SS" when an hour or more remains. Rounds up to whole seconds. */
export function formatCountdown(ms: number): string {
  const total = Math.max(0, Math.ceil(ms / 1000));
  const h = Math.floor(total / 3600);
  const m = Math.floor((total % 3600) / 60);
  const s = total % 60;
  const mm = String(m).padStart(2, "0");
  const ss = String(s).padStart(2, "0");
  return h > 0 ? `${h}:${mm}:${ss}` : `${mm}:${ss}`;
}

/** Thai spoken form for screen readers, e.g. "เหลือเวลา 4 นาที 5 วินาที". */
export function describeCountdown(ms: number): string {
  const total = Math.max(0, Math.ceil(ms / 1000));
  const h = Math.floor(total / 3600);
  const m = Math.floor((total % 3600) / 60);
  const s = total % 60;
  const parts: string[] = [];
  if (h) parts.push(`${h} ชั่วโมง`);
  if (m) parts.push(`${m} นาที`);
  if (s || parts.length === 0) parts.push(`${s} วินาที`);
  return `เหลือเวลา ${parts.join(" ")}`;
}

/** Threshold under which the timer is shown as urgent. */
export const URGENT_MS = 60_000;

/** Human-friendly duration for an Exam time limit, e.g. "1 ชั่วโมง 30 นาที". */
export function formatTimeLimit(seconds: number): string {
  const h = Math.floor(seconds / 3600);
  const m = Math.round((seconds % 3600) / 60);
  if (h && m) return `${h} ชั่วโมง ${m} นาที`;
  if (h) return `${h} ชั่วโมง`;
  return `${m} นาที`;
}
