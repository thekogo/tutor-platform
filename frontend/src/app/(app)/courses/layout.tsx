import { Suspense } from "react";
import { CourseHeader } from "./CourseHeader";

export default function CourseLayout({ children }: { children: React.ReactNode }) {
  return (
    <div className="flex flex-col gap-6">
      <Suspense fallback={<div className="h-20" aria-hidden="true" />}>
        <CourseHeader />
      </Suspense>
      <div>{children}</div>
    </div>
  );
}
