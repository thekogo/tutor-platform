"use client";

import Link from "next/link";
import { usePathname } from "next/navigation";
import { routes } from "@/lib/routes";
import { T } from "@/lib/terms";
import { cn } from "@/lib/ui";

/** Tabs for the pages of one Course. */
export function CourseNav({ courseId }: { courseId: string }) {
  const pathname = usePathname();
  const tabs = [
    { path: "/courses", href: routes.course(courseId), label: "ภาพรวม" },
    { path: "/courses/map", href: routes.courseMap(courseId), label: T.masteryMap },
    { path: "/courses/learning-path", href: routes.courseLearningPath(courseId), label: T.learningPath },
    { path: "/courses/reports/formats", href: routes.courseFormatReport(courseId), label: `รายงาน${T.questionFormat}` },
  ];
  const isActive = (path: string) => pathname.replace(/\/+$/, "") === path;

  return (
    <nav aria-label={`เมนู${T.course}`} className="-mx-4 overflow-x-auto px-4">
      <ul className="flex gap-1 border-b border-slate-200">
        {tabs.map((t) => (
          <li key={t.path}>
            <Link
              href={t.href}
              aria-current={isActive(t.path) ? "page" : undefined}
              className={cn(
                "-mb-px block border-b-2 px-3 py-2 text-sm whitespace-nowrap",
                isActive(t.path) ? "border-blue-700 font-semibold text-blue-800" : "border-transparent text-slate-600 hover:text-slate-900",
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
