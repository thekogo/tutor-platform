import { T } from "@/lib/terms";
import { CourseSelectView } from "./CourseSelectView";

export const metadata = { title: `เลือก${T.course}` };

export default function CourseSelectPage() {
  return <CourseSelectView />;
}
