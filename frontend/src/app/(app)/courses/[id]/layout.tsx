import Link from "next/link";
import { CourseNav } from "@/components/CourseNav";
import { getCourse } from "@/lib/api/queries";
import { T } from "@/lib/terms";

export default async function CourseLayout({ children, params }: { children: React.ReactNode; params: Promise<{ id: string }> }) {
  const { id } = await params;
  const course = await getCourse(id);
  return (
    <div className="flex flex-col gap-6">
      <div className="flex flex-col gap-3">
        <p className="text-sm text-slate-600">
          <Link href="/" className="text-blue-800 hover:underline">
            {T.course}ทั้งหมด
          </Link>
          <span aria-hidden="true"> / </span>
          <span className="font-medium text-slate-900">{course.name}</span>
        </p>
        <CourseNav courseId={course.id} />
      </div>
      <div>{children}</div>
    </div>
  );
}
