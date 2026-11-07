"use client";

import Link from "next/link";
import { usePathname } from "next/navigation";
import { useState } from "react";
import { api } from "@/lib/api/client";
import { T } from "@/lib/terms";
import { cn } from "@/lib/ui";

type Props = { courseId: string | null; isOwner: boolean; learnerName: string };

export function AppNav({ courseId, isOwner, learnerName }: Props) {
  const pathname = usePathname();
  const [open, setOpen] = useState(false);
  const links = [
    { href: "/", label: "หน้าหลัก" },
    ...(courseId ? [{ href: `/courses/${courseId}/map`, label: T.masteryMap }] : []),
    { href: "/learning-path", label: T.learningPath },
    { href: "/reports/formats", label: `รายงาน${T.questionFormat}` },
    ...(isOwner ? [{ href: "/admin", label: "จัดการระบบ" }] : []),
  ];
  const isActive = (href: string) => (href === "/" ? pathname === "/" : pathname.startsWith(href));

  async function logout() {
    await api().POST("/auth/logout");
    // Full reload so every Server Component drops the old session.
    // eslint-disable-next-line @next/next/no-location-assign-relative-destination
    window.location.assign("/login");
  }

  return (
    <nav aria-label="เมนูหลัก" className="flex flex-1 items-center justify-end gap-2 md:justify-between">
      <button
        type="button"
        className="rounded-lg border border-slate-300 px-3 py-1.5 text-sm md:hidden"
        aria-expanded={open}
        aria-controls="main-menu"
        onClick={() => setOpen((v) => !v)}
      >
        เมนู
      </button>
      <ul
        id="main-menu"
        className={cn(
          "absolute top-14 right-0 left-0 z-40 flex-col gap-1 border-b border-slate-200 bg-white p-3 shadow md:static md:flex md:flex-row md:border-0 md:p-0 md:shadow-none",
          open ? "flex" : "hidden",
        )}
      >
        {links.map((l) => (
          <li key={l.href}>
            <Link
              href={l.href}
              onClick={() => setOpen(false)}
              aria-current={isActive(l.href) ? "page" : undefined}
              className={cn(
                "block rounded-lg px-3 py-1.5 text-sm",
                isActive(l.href) ? "bg-blue-50 font-semibold text-blue-800" : "text-slate-700 hover:bg-slate-100",
              )}
            >
              {l.label}
            </Link>
          </li>
        ))}
      </ul>
      <div className="hidden items-center gap-3 md:flex">
        <span className="text-sm text-slate-600">{learnerName}</span>
        <button type="button" onClick={logout} className="rounded-lg px-3 py-1.5 text-sm text-slate-700 hover:bg-slate-100">
          ออกจากระบบ
        </button>
      </div>
      {open && (
        <button type="button" onClick={logout} className="sr-only focus:not-sr-only">
          ออกจากระบบ
        </button>
      )}
    </nav>
  );
}
