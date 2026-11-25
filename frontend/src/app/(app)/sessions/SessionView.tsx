"use client";

import { NotFoundView } from "@/components/ErrorViews";
import { Loaded } from "@/components/Loaded";
import { SessionRunner } from "@/components/SessionRunner";
import { api, unwrap } from "@/lib/api/client";
import { useLoad } from "@/lib/api/use-load";
import { SESSION_KIND_LABEL } from "@/lib/terms";
import { useIdParam } from "@/lib/use-id-param";

async function loadSession(id: string) {
  const session = await api().GET("/sessions/{sessionId}", { params: { path: { sessionId: id } } }).then(unwrap);

  // Contract gap: StudySession has no Concept/Exam name, so resolve it for the header.
  let title: string | undefined;
  if (session.conceptId) {
    const c = await api().GET("/concepts/{conceptId}", { params: { path: { conceptId: session.conceptId } } });
    title = c.data?.name;
  } else if (session.examId) {
    const exams = await api().GET("/courses/{courseId}/exams", { params: { path: { courseId: session.courseId } } });
    title = exams.data?.find((e) => e.id === session.examId)?.name;
  }
  return { session, title: title ?? SESSION_KIND_LABEL[session.kind] };
}

export function SessionView() {
  const id = useIdParam();
  // Loaded once per session id: the runner owns the state from then on (no refetch on invalidate).
  const state = useLoad(id && `session:${id}`, () => loadSession(id!));
  if (!id) return <NotFoundView />;
  return <Loaded state={state}>{({ session, title }) => <SessionRunner key={session.id} session={session} title={title} />}</Loaded>;
}
