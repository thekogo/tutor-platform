"use client";

import Link from "next/link";
import { AppNav } from "@/components/AppNav";
import { BrandMark } from "@/components/BrandMark";
import { getMe } from "@/lib/api/queries";
import { useLoad } from "@/lib/api/use-load";
import { BRAND } from "@/lib/brand";

/** Site header. Loading /me here also sends a signed-out visitor to /login (401). */
export function AppHeader() {
  const me = useLoad("me", getMe);
  return (
    <header className="sticky top-0 z-30 border-b border-slate-200 bg-white/95 backdrop-blur">
      <div className="relative mx-auto flex h-14 max-w-7xl items-center gap-4 px-4">
        <Link href="/" aria-label={`${BRAND.name} — เลือกคอร์ส`}>
          <BrandMark />
        </Link>
        {me.status === "ok" && <AppNav isOwner={me.data.isOwner} learnerName={me.data.name} />}
      </div>
    </header>
  );
}
