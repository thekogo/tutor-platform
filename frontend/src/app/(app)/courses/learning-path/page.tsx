import { Suspense } from "react";
import { LoadingBlock } from "@/components/StateViews";
import { T } from "@/lib/terms";
import { LearningPathView } from "./LearningPathView";

export const metadata = { title: T.learningPath };

export default function Page() {
  return (
    <Suspense fallback={<LoadingBlock />}>
      <LearningPathView />
    </Suspense>
  );
}
