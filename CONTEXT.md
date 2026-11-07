# Tutor Platform

A tutoring platform that models a course as a knowledge graph, tracks where each learner is weak, and points them to the concept they should study next.

## Course structure

**Course**:
A body of material preparing a learner for one goal (e.g. CU-BEST Math), made of Topics and Concepts.

**Topic**:
A chapter-level grouping of Concepts that mirrors one knowledge item of the course outline (e.g. ร้อยละ เปอร์เซ็น); outline items that are really Question Formats (e.g. data sufficiency) are not Topics. Used for navigation; not scored and not part of the prerequisite graph.
_Avoid_: Outline item, chapter, unit

**Concept**:
A fine-grained skill or idea that a learner can be strong or weak at (e.g. percent change). The node of the knowledge graph; belongs to exactly one Topic, and so to exactly one Course — Concepts are never shared across Courses. Concepts are archived, never deleted.
_Avoid_: Node, skill, point

**Prerequisite**:
A directed relation saying Concept A must be understood before Concept B. May cross Topics. The only relation in the knowledge graph, and it never forms a cycle.
_Avoid_: Dependency, edge, link

## Assessment

**Question**:
A single gradable item, tagged with one or more Concepts it tests.

**Primary Concept**:
The one Concept among a Question's tags that the Question is mainly testing.

**Difficulty**:
A Content Admin's rating of how hard a Question is, on a 1–3 scale.

**Question Status**:
Where a Question is in its lifecycle: **Draft** (written, not yet verified), **Active** (verified and served), **Flagged** (a Learner reported a problem), or **Retired**. Only Active Questions are served and count toward Mastery.

**Flag**:
A Learner's report, made after answering, that a Question's answer key or Explanation is wrong. One Flag pulls the Question for every Learner.
_Avoid_: Report, dispute

**Source**:
Where a Question came from: AI-generated, or imported from a past paper or other material.
_Avoid_: Origin, provenance

**Explanation**:
The worked solution to a Question, shown to the Learner after they answer.
_Avoid_: Solution, answer key

**Question Format**:
The presentation style of a Question (e.g. standard, data sufficiency, out-of-the-box), independent of the Concepts it tests.
_Avoid_: Question type

**Quiz**:
A short, untimed, repeatable practice set focused on one Concept or Topic, with immediate feedback.
_Avoid_: Exercise, test

**Exam**:
A fixed, Content-Admin-authored timed paper spanning the whole Course or a large part of it, modelled on the real target exam. Its Questions are kept out of Quizzes.
_Avoid_: Mock, test

**Placement Test**:
An optional, short adaptive test a new Learner takes to seed Mastery across the Course.
_Avoid_: Pre-test, onboarding exam

**Answer**:
One Learner's recorded response to one Question at one moment. Every Answer is kept; Mastery is derived from them.
_Avoid_: Attempt, response, submission

**Diagnostic Quiz**:
A short Quiz the platform issues on an Unassessed Concept to measure it before making a Recommendation.

**Content Request**:
A record that a Concept needs more Questions or Lesson material (e.g. its Question pool is exhausted), for the Content Admin to fulfil. Raised per Concept, not per Learner.

## Learning

**Lesson**:
The teaching material attached to one Concept: an explanation and worked examples.
_Avoid_: Content, material, chapter

## Weakness tracking

**Mastery**:
A Learner's current estimated command of one Concept, as a score from 0 to 1. Evidence from a Question counts in full toward its Primary Concept and partially toward its other Concepts.
_Avoid_: Proficiency, skill level, score

**Mastery Level**:
The band a Mastery falls into: **Unassessed** (too little evidence to judge), **Weak**, **Developing**, or **Mastered**. Unassessed means "unknown", never "weak".

**Root Cause**:
The deepest Weak Concept on the Prerequisite path to a struggling Concept whose own Prerequisites are not Weak — the real gap to study first.
_Avoid_: Weak point, gap

**Recommendation**:
The platform's suggestion of which Concept a Learner should study next, always shown with its reason: the Prerequisite path from the struggling Concept back to the Root Cause. Advisory only — no Concept is ever locked.
_Avoid_: Next step, suggestion

**Learning Path**:
The ordered walk a Recommendation starts: from the Root Cause back up the Prerequisite path to the struggling Concept, one Concept at a time, skipping any already Developing or better.
_Avoid_: Study plan, remediation path

**Priority**:
How Recommendations are ranked when several Root Causes exist: by how many Weak Concepts fixing it unblocks, weighted by its Topic's Exam Weight.

**Exam Weight**:
A Content Admin's rating of how much a Topic matters on the real target exam.
_Avoid_: Topic weight, importance

**Mastery Map**:
The view of a Course's knowledge graph with each Concept colored by the Learner's Mastery Level.
_Avoid_: Skill tree, progress map

## People

**Learner**:
A person studying a Course, joined by invitation. Each Learner's Answers and Mastery are private to them and the Owner.
_Avoid_: Student, user

**Content Admin**:
The role that authors Courses, Concepts, Prerequisites, Lessons and Questions. Held by an AI agent, which generates content or imports it from outside sources — never by the Learner, so the Learner never sees Questions before answering them.
_Avoid_: Author, teacher, tutor

**Owner**:
The person who runs the platform: invites Learners, operates the Content Admin agent, and resolves Flags and Content Requests. Also a Learner.
_Avoid_: Admin, superuser
