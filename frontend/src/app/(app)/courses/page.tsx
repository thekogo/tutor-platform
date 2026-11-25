import { Suspense } from "react";
import { LoadingBlock } from "@/components/StateViews";
import { CourseDashboardView } from "./CourseDashboardView";

export const metadata = { title: "ภาพรวมคอร์ส" };

export default function CourseDashboardPage() {
  return (
    <Suspense fallback={<LoadingBlock />}>
      <CourseDashboardView />
    </Suspense>
  );
}
