"use client";

import Link from "next/link";
import { useEffect, type ReactNode } from "react";
import { HttpError } from "@/lib/api/client";
import type { LoadState } from "@/lib/api/use-load";
import { btn, card } from "@/lib/ui";
import { ForbiddenView, NotFoundView } from "./ErrorViews";
import { LoadingBlock } from "./StateViews";

type Props<T> = {
  state: LoadState<T> & { reload: () => void };
  children: (data: T) => ReactNode;
  loading?: ReactNode;
};

/** Renders a useLoad() result: skeleton, 403/404 pages, a retryable error, or the data. */
export function Loaded<T>({ state, children, loading }: Props<T>) {
  if (state.status === "ok") return <>{children(state.data)}</>;
  if (state.status === "loading") return <>{loading ?? <LoadingBlock />}</>;
  const status = state.error instanceof HttpError ? state.error.status : 0;
  // 401: the API client is already navigating to /login.
  if (status === 401) return <>{loading ?? <LoadingBlock />}</>;
  if (status === 403) return <ForbiddenView />;
  if (status === 404) return <NotFoundView />;
  return <LoadError error={state.error} retry={state.reload} />;
}

function LoadError({ error, retry }: { error: unknown; retry: () => void }) {
  useEffect(() => {
    console.error(error);
  }, [error]);
  return (
    <div className={card} role="alert">
      <h1 className="text-xl font-bold text-red-800">เกิดข้อผิดพลาด</h1>
      <p className="mt-2 text-slate-700">โหลดข้อมูลไม่สำเร็จ อาจเป็นเพราะเซิร์ฟเวอร์ไม่ตอบสนอง กรุณาลองใหม่อีกครั้ง</p>
      <div className="mt-4 flex gap-2">
        <button type="button" onClick={retry} className={btn.primary}>
          ลองใหม่
        </button>
        <Link href="/" className={btn.secondary}>
          กลับหน้าหลัก
        </Link>
      </div>
    </div>
  );
}
