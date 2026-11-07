import "server-only";
import { cache } from "react";
import { serverApi, unwrap } from "./server";

/** Current Learner (401 → /login). Deduplicated per request. */
export const getMe = cache(async () => {
  const api = await serverApi();
  return unwrap(await api.GET("/me"));
});

/** The Learner's Course. The platform currently serves one Course; we use the first. */
export const getDefaultCourse = cache(async () => {
  const api = await serverApi();
  const courses = unwrap(await api.GET("/courses"));
  return courses[0] ?? null;
});

export const getCourseGraph = cache(async (courseId: string) => {
  const api = await serverApi();
  return unwrap(await api.GET("/courses/{courseId}", { params: { path: { courseId } } }));
});
