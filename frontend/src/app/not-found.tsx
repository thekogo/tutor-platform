import Link from "next/link";
import { btn } from "@/lib/ui";

export default function NotFound() {
  return (
    <div className="mx-auto flex max-w-lg flex-1 flex-col items-center justify-center gap-4 p-8 text-center">
      <p className="text-5xl font-bold text-slate-300">404</p>
      <h1 className="text-2xl font-bold">ไม่พบหน้าที่ต้องการ</h1>
      <p className="text-slate-600">ลิงก์อาจไม่ถูกต้อง หรือเนื้อหานี้ถูกเก็บเข้าคลังแล้ว</p>
      <Link href="/" className={btn.primary}>
        กลับหน้าหลัก
      </Link>
    </div>
  );
}
