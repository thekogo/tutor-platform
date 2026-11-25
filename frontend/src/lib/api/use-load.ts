"use client";

import { useCallback, useEffect, useState } from "react";
import { onInvalidate } from "./client";

export type LoadState<T> =
  | { status: "loading" }
  | { status: "ok"; data: T }
  | { status: "error"; error: unknown };

/**
 * Client-side data loading for the static export. Runs `load` on mount and
 * whenever `key` changes; refetches in place (keeping the old data on screen)
 * after invalidate() or reload().
 */
export function useLoad<T>(key: string | null, load: () => Promise<T>): LoadState<T> & { reload: () => void } {
  const [version, setVersion] = useState(0);
  const [state, setState] = useState<{ key: string | null; value: LoadState<T> }>({ key, value: { status: "loading" } });
  const reload = useCallback(() => setVersion((v) => v + 1), []);

  useEffect(() => onInvalidate(reload), [reload]);

  useEffect(() => {
    if (key === null) return;
    let cancelled = false;
    load().then(
      (data) => !cancelled && setState({ key, value: { status: "ok", data } }),
      (error: unknown) => !cancelled && setState({ key, value: { status: "error", error } }),
    );
    return () => {
      cancelled = true;
    };
    // `load` is a fresh closure every render; `key` identifies what it fetches.
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [key, version]);

  const value: LoadState<T> = state.key === key ? state.value : { status: "loading" };
  return { ...value, reload };
}
