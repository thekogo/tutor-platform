import Link from "next/link";
import { AppNav } from "@/components/AppNav";
import { BrandMark } from "@/components/BrandMark";
import { BRAND } from "@/lib/brand";
import { getMe } from "@/lib/api/queries";

export default async function AppLayout({ children }: { children: React.ReactNode }) {
  const me = await getMe();
  return (
    <>
      <a href="#main" className="sr-only focus:not-sr-only focus:absolute focus:top-2 focus:left-2 focus:z-50 focus:rounded focus:bg-white focus:p-2">
        ข้ามไปยังเนื้อหาหลัก
      </a>
      <header className="sticky top-0 z-30 border-b border-slate-200 bg-white/95 backdrop-blur">
        <div className="relative mx-auto flex h-14 max-w-7xl items-center gap-4 px-4">
          <Link href="/" aria-label={`${BRAND.name} — เลือกคอร์ส`}>
            <BrandMark />
          </Link>
          <AppNav isOwner={me.isOwner} learnerName={me.name} />
        </div>
      </header>
      <main id="main" className="mx-auto w-full max-w-7xl flex-1 px-4 py-6">
        {children}
      </main>
    </>
  );
}
