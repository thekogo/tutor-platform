import { Suspense } from "react";
import { LoadingBlock } from "@/components/StateViews";
import { SessionView } from "./SessionView";

export const metadata = { title: "ทำโจทย์" };

export default function Page() {
  return (
    <Suspense fallback={<LoadingBlock />}>
      <SessionView />
    </Suspense>
  );
}
