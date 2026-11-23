import "server-only";
import { notFound } from "next/navigation";
import { cache } from "react";
import { serverApi, unwrap } from "./server";

/** Current Learner (401 → /login). Deduplicated per request. */
export const getMe = cache(async () => {
  const api = await serverApi();
  return unwrap(await api.GET("/me"));
});

/** Every Course, with the current Learner's per-Course Placement Test status. */
export const getCourses = cache(async () => {
  const api = await serverApi();
  return unwrap(await api.GET("/courses"));
});

/** One Course summary by id; 404s when it doesn't exist. */
export const getCourse = cache(async (courseId: string) => {
  const course = (await getCourses()).find((c) => c.id === courseId);
  if (!course) notFound();
  return course;
});

export const getCourseGraph = cache(async (courseId: string) => {
  const api = await serverApi();
  return unwrap(await api.GET("/courses/{courseId}", { params: { path: { courseId } } }));
});
