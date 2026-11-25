import { api, HttpError, unwrap } from "./client";

/** Current Learner (401 → /login). */
export async function getMe() {
  return unwrap(await api().GET("/me"));
}

/** Every Course, with the current Learner's per-Course Placement Test status. */
export async function getCourses() {
  return unwrap(await api().GET("/courses"));
}

/** One Course summary by id; HttpError(404) when it doesn't exist. */
export async function getCourse(courseId: string) {
  const course = (await getCourses()).find((c) => c.id === courseId);
  if (!course) throw new HttpError(404, `/courses/${courseId}`);
  return course;
}

export async function getCourseGraph(courseId: string) {
  return unwrap(await api().GET("/courses/{courseId}", { params: { path: { courseId } } }));
}

export async function getMastery(courseId: string) {
  return unwrap(await api().GET("/courses/{courseId}/mastery", { params: { path: { courseId } } }));
}
