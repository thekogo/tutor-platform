"use client";

import Link from "next/link";
import { usePathname } from "next/navigation";
import { T } from "@/lib/terms";
import { cn } from "@/lib/ui";

/** Tabs for the pages of one Course. */
export function CourseNav({ courseId }: { courseId: string }) {
  const pathname = usePathname();
  const base = `/courses/${courseId}`;
  const tabs = [
    { href: base, label: "ภาพรวม" },
    { href: `${base}/map`, label: T.masteryMap },
    { href: `${base}/learning-path`, label: T.learningPath },
    { href: `${base}/reports/formats`, label: `รายงาน${T.questionFormat}` },
  ];
  const isActive = (href: string) => (href === base ? pathname === base : pathname.startsWith(href));

  return (
    <nav aria-label={`เมนู${T.course}`} className="-mx-4 overflow-x-auto px-4">
      <ul className="flex gap-1 border-b border-slate-200">
        {tabs.map((t) => (
          <li key={t.href}>
            <Link
              href={t.href}
              aria-current={isActive(t.href) ? "page" : undefined}
              className={cn(
                "-mb-px block border-b-2 px-3 py-2 text-sm whitespace-nowrap",
                isActive(t.href) ? "border-blue-700 font-semibold text-blue-800" : "border-transparent text-slate-600 hover:text-slate-900",
              )}
            >
              {t.label}
            </Link>
          </li>
        ))}
      </ul>
    </nav>
  );
}
