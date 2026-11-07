import { SessionRunner } from "@/components/SessionRunner";
import { serverApi, unwrap } from "@/lib/api/server";
import { SESSION_KIND_LABEL } from "@/lib/terms";

export const metadata = { title: "ทำโจทย์" };

export default async function SessionPage({ params }: { params: Promise<{ id: string }> }) {
  const { id } = await params;
  const api = await serverApi();
  const session = await api.GET("/sessions/{sessionId}", { params: { path: { sessionId: id } } }).then(unwrap);

  // Contract gap: StudySession has no Concept/Exam name, so resolve it for the header.
  let title: string | undefined;
  if (session.conceptId) {
    const c = await api.GET("/concepts/{conceptId}", { params: { path: { conceptId: session.conceptId } } });
    title = c.data?.name;
  } else if (session.examId) {
    const exams = await api.GET("/courses/{courseId}/exams", { params: { path: { courseId: session.courseId } } });
    title = exams.data?.find((e) => e.id === session.examId)?.name;
  }

  return <SessionRunner key={session.id} session={session} title={title ?? SESSION_KIND_LABEL[session.kind]} />;
}
