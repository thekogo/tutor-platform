"use client";

import { useSearchParams } from "next/navigation";

/** The `?id=` of the current page (see lib/routes.ts); null when absent. Needs a <Suspense> ancestor. */
export function useIdParam(): string | null {
  const id = useSearchParams().get("id");
  return id && id.trim() ? id : null;
}
