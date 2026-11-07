import { McpClient } from "./mcp";
import { COURSE_SLUG, MCP_TOKEN, MCP_URL } from "./env";

/**
 * Creates a few Questions on one Concept through MCP and runs them through the
 * independent verification pass so they become Active (ADR 0003).
 *
 * The tool argument shapes below are the e2e suite's ASSUMPTION about the MCP
 * contract; if internal/mcpserver differs, adjust ONLY this file.
 *
 *   create_questions { course, questions: [{ stem_md, explanation_md, choices: [{label, body_md, is_correct}],
 *                      primary_concept, concepts, difficulty, format, source }] }
 *     -> { question_ids: string[] }  (or { questions: [{id}] })
 *   list_questions_for_verification { course, limit } -> { questions: [{ id, stem_md, choices: [{label, body_md}] }] }
 *   submit_verification { question_id, chosen_label } -> { status }
 */
export async function seedQuizQuestions(conceptSlug: string, count = 3): Promise<void> {
  const mcp = new McpClient(MCP_URL, MCP_TOKEN);
  await mcp.connect();

  const runId = Date.now().toString(36);
  const questions = Array.from({ length: count }, (_, i) => {
    const a = 3 + i;
    const b = 4 + i;
    const answer = a * b;
    const body = (n: number) => `$${n}$`;
    return {
      stem_md: `[e2e ${runId}-${i}] จงหาค่าของ $${a} \\times ${b}$`,
      explanation_md: `$${a} \\times ${b} = ${answer}$`,
      choices: [
        { label: "A", body_md: body(answer), is_correct: true },
        { label: "B", body_md: body(answer + 1), is_correct: false },
        { label: "C", body_md: body(answer - 1), is_correct: false },
        { label: "D", body_md: body(a + b), is_correct: false },
      ],
      primary_concept: conceptSlug,
      concepts: [conceptSlug],
      difficulty: 1,
      format: "standard",
      source: "ai_generated",
    };
  });

  await mcp.callTool("create_questions", { course: COURSE_SLUG, questions });

  // Independent verifier: sees no answer key; we "solve" by recomputing a×b from the stem.
  const pending = await mcp.callTool<{
    questions: { id: string; stem_md: string; choices: { label: string; body_md: string }[] }[];
  }>("list_questions_for_verification", { course: COURSE_SLUG, limit: 100 });

  for (const q of pending.questions ?? []) {
    const m = /\[e2e [^\]]+\].*?(\d+) \\times (\d+)/.exec(q.stem_md);
    if (!m) continue; // not ours
    const answer = `$${Number(m[1]) * Number(m[2])}$`;
    const choice = q.choices.find((c) => c.body_md.trim() === answer);
    if (!choice) continue;
    await mcp.callTool("submit_verification", { question_id: q.id, chosen_label: choice.label });
  }
}
