/**
 * App URLs. The frontend is a static export, so an id travels in the query
 * string (`/courses?id=…`) instead of a dynamic path segment: nginx can only
 * serve pages that exist at build time. nginx 301s the old path-style URLs
 * (`/courses/<id>`, `/concepts/<id>`, …) to these.
 */
const withId = (path: string, id: string) => `${path}?id=${encodeURIComponent(id)}`;

export const routes = {
  home: "/",
  login: "/login",
  admin: "/admin",
  course: (id: string) => withId("/courses", id),
  courseMap: (id: string) => withId("/courses/map", id),
  courseLearningPath: (id: string) => withId("/courses/learning-path", id),
  courseFormatReport: (id: string) => withId("/courses/reports/formats", id),
  concept: (id: string) => withId("/concepts", id),
  session: (id: string) => withId("/sessions", id),
  sessionResult: (id: string) => withId("/sessions/result", id),
};
