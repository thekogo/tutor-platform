import clsx, { type ClassValue } from "clsx";

export const cn = (...v: ClassValue[]) => clsx(v);

const base =
  "inline-flex items-center justify-center gap-2 rounded-lg px-4 py-2 text-sm font-medium transition-colors disabled:cursor-not-allowed disabled:opacity-60";

export const btn = {
  primary: cn(base, "bg-blue-700 text-white hover:bg-blue-800"),
  secondary: cn(base, "border border-slate-300 bg-white text-slate-800 hover:bg-slate-100"),
  ghost: cn(base, "text-blue-700 hover:bg-blue-50"),
  danger: cn(base, "border border-red-300 bg-white text-red-700 hover:bg-red-50"),
};

export const card = "rounded-xl border border-slate-200 bg-white p-5 shadow-sm";

export function percent(x: number): string {
  return `${Math.round(x * 100)}%`;
}

const dateFmt = new Intl.DateTimeFormat("th-TH", { dateStyle: "medium", timeStyle: "short" });
export function formatDateTime(iso: string): string {
  const d = new Date(iso);
  return Number.isNaN(d.getTime()) ? iso : dateFmt.format(d);
}

export function isPast(iso: string, now: number = Date.now()): boolean {
  return Date.parse(iso) < now;
}
