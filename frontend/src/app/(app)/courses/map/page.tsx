import { Suspense } from "react";
import { LoadingBlock } from "@/components/StateViews";
import { T } from "@/lib/terms";
import { MasteryMapView } from "./MasteryMapView";

export const metadata = { title: T.masteryMap };

export default function Page() {
  return (
    <Suspense fallback={<LoadingBlock />}>
      <MasteryMapView />
    </Suspense>
  );
}
