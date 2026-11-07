"use client";

import { useEffect, useRef, useState } from "react";
import { describeCountdown, formatCountdown, remainingMs, URGENT_MS } from "@/lib/countdown";
import { cn } from "@/lib/ui";

type Props = {
  /** Server-issued deadline (ISO date-time). */
  deadline: string;
  onExpire: () => void;
  /** Injectable clock for tests. */
  now?: () => number;
};

/** Exam countdown to the server deadline; fires onExpire once at zero. */
export function Countdown({ deadline, onExpire, now = Date.now }: Props) {
  const [ms, setMs] = useState(() => remainingMs(deadline, now()));
  const expired = useRef(false);
  const onExpireRef = useRef(onExpire);
  useEffect(() => {
    onExpireRef.current = onExpire;
  }, [onExpire]);

  useEffect(() => {
    function tick() {
      const left = remainingMs(deadline, now());
      setMs(left);
      if (left <= 0 && !expired.current) {
        expired.current = true;
        onExpireRef.current();
      }
    }
    tick();
    const id = setInterval(tick, 250);
    return () => clearInterval(id);
  }, [deadline, now]);

  const urgent = ms <= URGENT_MS;
  return (
    <div
      className={cn(
        "inline-flex items-center gap-2 rounded-lg border px-3 py-1.5 font-mono text-lg tabular-nums",
        urgent ? "border-red-300 bg-red-50 text-red-800" : "border-slate-300 bg-white text-slate-800",
      )}
    >
      <span aria-hidden="true">⏱</span>
      <span role="timer" aria-label={describeCountdown(ms)} data-testid="countdown">
        {formatCountdown(ms)}
      </span>
      {urgent && ms > 0 && (
        <span role="alert" className="sr-only">
          เหลือเวลาไม่ถึง 1 นาที
        </span>
      )}
    </div>
  );
}
