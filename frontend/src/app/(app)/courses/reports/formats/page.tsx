import { Suspense } from "react";
import { LoadingBlock } from "@/components/StateViews";
import { T } from "@/lib/terms";
import { FormatReportView } from "./FormatReportView";

export const metadata = { title: `รายงาน${T.questionFormat}` };

export default function Page() {
  return (
    <Suspense fallback={<LoadingBlock />}>
      <FormatReportView />
    </Suspense>
  );
}
