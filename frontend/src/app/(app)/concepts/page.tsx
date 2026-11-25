import { Suspense } from "react";
import { LoadingBlock } from "@/components/StateViews";
import { ConceptView } from "./ConceptView";

export default function Page() {
  return (
    <Suspense fallback={<LoadingBlock />}>
      <ConceptView />
    </Suspense>
  );
}
