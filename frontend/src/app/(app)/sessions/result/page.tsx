import { Suspense } from "react";
import { LoadingBlock } from "@/components/StateViews";
import { SessionResultView } from "./SessionResultView";

export const metadata = { title: "ผลลัพธ์" };

export default function Page() {
  return (
    <Suspense fallback={<LoadingBlock />}>
      <SessionResultView />
    </Suspense>
  );
}
