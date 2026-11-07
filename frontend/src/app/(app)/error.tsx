"use client";

import Link from "next/link";
import { useEffect } from "react";
import { btn, card } from "@/lib/ui";

export default function AppError({ error, retry }: { error: unknown; retry: () => void }) {
  useEffect(() => {
    console.error(error);
  }, [error]);
  const digest =
    error && typeof error === "object" && "digest" in error ? String((error as { digest?: unknown }).digest ?? "") : "";
  return (
    <div className={card} role="alert">
      <h1 className="text-xl font-bold text-red-800">เกิดข้อผิดพลาด</h1>
      <p className="mt-2 text-slate-700">โหลดข้อมูลไม่สำเร็จ อาจเป็นเพราะเซิร์ฟเวอร์ไม่ตอบสนอง กรุณาลองใหม่อีกครั้ง</p>
      {digest && <p className="mt-1 text-xs text-slate-500">รหัสอ้างอิง: {digest}</p>}
      <div className="mt-4 flex gap-2">
        <button type="button" onClick={() => retry()} className={btn.primary}>
          ลองใหม่
        </button>
        <Link href="/" className={btn.secondary}>
          กลับหน้าหลัก
        </Link>
      </div>
    </div>
  );
}
