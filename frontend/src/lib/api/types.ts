import type { components } from "./schema";

type S = components["schemas"];

export type Learner = S["Learner"];
export type Invite = S["Invite"];
export type CreatedInvite = Invite & { url: string };
export type CourseSummary = S["CourseSummary"];
export type CourseGraph = S["CourseGraph"];
export type Topic = S["Topic"];
export type Concept = S["Concept"];
export type ConceptDetail = S["ConceptDetail"];
export type QuestionFormat = S["QuestionFormat"];
export type MasteryLevel = S["MasteryLevel"];
export type Mastery = S["Mastery"];
export type Recommendation = S["Recommendation"];
export type ConceptRef = S["ConceptRef"];
export type LearningPath = S["LearningPath"];
export type FormatStat = S["FormatStat"];
export type ExamSummary = S["ExamSummary"];
export type SessionKind = S["SessionKind"];
export type CreateSessionRequest = S["CreateSessionRequest"];
export type StudySession = S["StudySession"];
export type ServedQuestion = S["ServedQuestion"];
export type AnswerResult = S["AnswerResult"];
export type MasteryChange = NonNullable<AnswerResult["masteryChanges"]>[number];
export type SessionResult = S["SessionResult"];
export type ReviewItem = SessionResult["review"][number];
export type TopicBreakdown = SessionResult["topicBreakdown"][number];
export type Flag = S["Flag"];
export type ContentRequest = S["ContentRequest"];
export type ApiError = S["Error"];

export const MASTERY_LEVELS: readonly MasteryLevel[] = [
  "unassessed",
  "weak",
  "developing",
  "mastered",
] as const;
