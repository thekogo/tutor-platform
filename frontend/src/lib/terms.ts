/**
 * Thai UI copy for CONTEXT.md glossary terms. Use these everywhere so one
 * domain term always maps to one Thai phrase.
 */
import type { MasteryLevel, SessionKind } from "@/lib/api/types";

export const T = {
  course: "คอร์ส",
  topic: "หัวข้อ",
  concept: "แนวคิด",
  prerequisite: "พื้นฐานที่ต้องรู้ก่อน",
  dependents: "แนวคิดที่ต่อยอดจากเรื่องนี้",
  question: "โจทย์",
  explanation: "คำอธิบายเฉลย",
  flag: "แจ้งปัญหาโจทย์",
  flags: "รายการแจ้งปัญหาโจทย์",
  questionFormat: "รูปแบบโจทย์",
  quiz: "ควิซ",
  exam: "ชุดข้อสอบ",
  placementTest: "แบบทดสอบวัดระดับ",
  diagnosticQuiz: "ควิซวินิจฉัย",
  answer: "คำตอบ",
  contentRequest: "คำขอเพิ่มเนื้อหา",
  lesson: "บทเรียน",
  mastery: "ความเข้าใจ",
  masteryLevel: "ระดับความเข้าใจ",
  rootCause: "ต้นเหตุของจุดอ่อน",
  rootCauses: "ต้นเหตุของจุดอ่อน",
  recommendation: "คำแนะนำ",
  learningPath: "เส้นทางการเรียน",
  priority: "ลำดับความสำคัญ",
  examWeight: "น้ำหนักในข้อสอบ",
  masteryMap: "แผนที่ความเข้าใจ",
  learner: "ผู้เรียน",
  learners: "ผู้เรียน",
  owner: "เจ้าของระบบ",
  invite: "คำเชิญ",
} as const;

export const MASTERY_LEVEL_LABEL: Record<MasteryLevel, string> = {
  unassessed: "ยังไม่ประเมิน",
  weak: "อ่อน",
  developing: "กำลังพัฒนา",
  mastered: "เชี่ยวชาญ",
};

export const SESSION_KIND_LABEL: Record<SessionKind, string> = {
  quiz: T.quiz,
  diagnostic: T.diagnosticQuiz,
  placement: T.placementTest,
  exam: T.exam,
};
