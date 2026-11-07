import { expect, request as pwRequest, type APIRequestContext } from "@playwright/test";
import { BASE_URL } from "./env";

/** Minimal shapes from api/openapi.yaml used by the journeys. */
export interface CourseSummary {
  id: string;
  slug: string;
  name: string;
}
export interface Concept {
  id: string;
  slug: string;
  name: string;
  topicId: string;
  prerequisiteIds: string[];
}
export interface CourseGraph extends CourseSummary {
  topics: { id: string; slug: string; name: string }[];
  concepts: Concept[];
}
export interface ConceptDetail extends Concept {
  activeQuestionCount?: number;
}
export interface Learner {
  id: string;
  email: string;
  name: string;
  isOwner: boolean;
}

/**
 * API context that talks to the backend through the frontend's /api proxy, so the
 * session cookie lands on the web origin. Sends Origin because the backend rejects
 * state-changing requests without a matching one.
 */
export async function newApiContext(storageState?: string): Promise<APIRequestContext> {
  return pwRequest.newContext({
    baseURL: BASE_URL,
    extraHTTPHeaders: { Origin: new URL(BASE_URL).origin },
    storageState,
  });
}

export async function devLogin(api: APIRequestContext, who: { email: string; name: string }) {
  const res = await api.post("/api/auth/dev-login", { data: who });
  expect(
    res.status(),
    `dev-login failed (${res.status()}): is AUTH_DEV_LOGIN=true? body: ${await res.text()}`,
  ).toBe(204);
}

async function getJSON<T>(api: APIRequestContext, url: string): Promise<T> {
  const res = await api.get(url);
  expect(res.ok(), `GET ${url} -> ${res.status()}: ${await res.text()}`).toBeTruthy();
  return (await res.json()) as T;
}

export const getMe = (api: APIRequestContext) => getJSON<Learner>(api, "/api/me");

export async function getCourseBySlug(api: APIRequestContext, slug: string): Promise<CourseGraph> {
  const courses = await getJSON<CourseSummary[]>(api, "/api/courses");
  const course = courses.find((c) => c.slug === slug) ?? courses[0];
  expect(course, "no Course found: did `make seed` run?").toBeTruthy();
  return getJSON<CourseGraph>(api, `/api/courses/${course!.id}`);
}

export const getConcept = (api: APIRequestContext, id: string) =>
  getJSON<ConceptDetail>(api, `/api/concepts/${id}`);

/** First Concept (preferred slug first) that has Active Questions to quiz on. */
export async function findQuizzableConcept(
  api: APIRequestContext,
  course: CourseGraph,
  preferredSlug?: string,
): Promise<Concept | undefined> {
  const ordered = [...course.concepts].sort(
    (a, b) => Number(b.slug === preferredSlug) - Number(a.slug === preferredSlug),
  );
  for (const c of ordered) {
    const detail = await getConcept(api, c.id);
    if ((detail.activeQuestionCount ?? 0) > 0) return c;
  }
  return undefined;
}
