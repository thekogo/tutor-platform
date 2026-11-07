import Link from "next/link";
import { T } from "@/lib/terms";
import { btn } from "@/lib/ui";

export default function Forbidden() {
  return (
    <div data-testid="forbidden" className="mx-auto flex max-w-lg flex-1 flex-col items-center justify-center gap-4 p-8 text-center">
      <p className="text-5xl font-bold text-slate-300">403</p>
      <h1 className="text-2xl font-bold">ไม่มีสิทธิ์เข้าถึงหน้านี้</h1>
      <p className="text-slate-600">หน้านี้สำหรับ{T.owner}เท่านั้น</p>
      <Link href="/" className={btn.primary}>
        กลับหน้าหลัก
      </Link>
    </div>
  );
}
