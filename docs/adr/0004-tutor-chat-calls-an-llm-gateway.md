# The Tutor Chat calls an LLM through a generic OpenAI-compatible gateway

Learners get stuck on Questions and want a hint before answering. We added a Tutor Chat: the Go server calls an LLM directly, which partly reverses the "platform never calls an LLM" stance behind ADR 0003. Content authoring still goes only through the Owner's agent over MCP; this is the one runtime LLM call.

The server, not the browser, assembles the prompt: it loads the Question's stem, choices, answer key and Explanation itself and passes them to the model as confidential context, with rules to coach one step at a time and never reveal the key, the final answer or the Explanation. The key and Explanation therefore never reach the browser before the Learner answers, which keeps the API guarantee that they are only sent after answering. The client sends only the conversation text, so the chat is stateless and nothing new is stored.

We talk to any OpenAI-compatible `/chat/completions` endpoint (configured by `AI_BASE_URL`, `AI_API_KEY`, `AI_MODEL`, plus `AI_EXTRA_BODY` for provider-specific options) instead of one vendor's SDK. The Owner currently uses Z.AI, and can switch to OpenRouter, LiteLLM, Cloudflare AI Gateway or another provider by changing environment variables only.

Consequences:
- **Withholding the answer depends on the prompt.** A determined Learner may still get the model to leak the answer through prompt injection. That is acceptable for a self-study tool, where the Learner only cheats themselves. The client-supplied history (including "assistant" turns) is untrusted for the same reason.
- **Quizzes only.** Exams model the real test, and Diagnostic Quizzes and Placement Tests exist to measure Mastery, so hints there would corrupt the evidence. Quiz Answers given after using the Tutor Chat still count toward Mastery in full. We may later record hint use and down-weight those Answers.
- **Cost and latency** are capped by a per-Learner in-memory rate limit and a per-call timeout kept under the 30s API/proxy timeout. The limit resets on restart, which is fine for a single binary.
