import { BRAND } from "@/lib/brand";
import { cn } from "@/lib/ui";

/** Logo: three linked knowledge-graph nodes. Same drawing as app/icon.svg. */
export function BrandLogo({ className }: { className?: string }) {
  return (
    <svg viewBox="0 0 32 32" aria-hidden="true" className={cn("size-7 shrink-0", className)}>
      <rect width="32" height="32" rx="8" fill="#1d4ed8" />
      <path d="M10 22 16 10 22 22" fill="none" stroke="#bfdbfe" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round" />
      <circle cx="16" cy="10" r="3.5" fill="#fff" />
      <circle cx="10" cy="22" r="3.5" fill="#fff" />
      <circle cx="22" cy="22" r="3.5" fill="#86efac" />
    </svg>
  );
}

/** Logo + product name. */
export function BrandMark({ className }: { className?: string }) {
  return (
    <span className={cn("inline-flex items-center gap-2 font-bold whitespace-nowrap text-blue-800", className)}>
      <BrandLogo />
      {BRAND.name}
    </span>
  );
}
