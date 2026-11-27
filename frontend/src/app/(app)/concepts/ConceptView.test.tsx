import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { http, HttpResponse } from "msw";
import { describe, expect, it, vi } from "vitest";
import { COURSE_ID, conceptId } from "@/mocks/db";
import { server } from "@/mocks/node";
import { ConceptView } from "./ConceptView";

const id = vi.hoisted(() => ({ current: "" }));
vi.mock("@/lib/use-id-param", () => ({ useIdParam: () => id.current }));

describe("ConceptView — Tutor Chat", () => {
  it("lets the Learner ask the tutor about the Concept", async () => {
    id.current = conceptId("percent-of");
    const user = userEvent.setup();
    let sent: unknown;
    server.use(
      http.post("*/api/concepts/:id/help", async ({ params, request }) => {
        expect(params.id).toBe(id.current);
        sent = await request.json();
        return HttpResponse.json({ reply: "เปอร์เซ็นต์คือ **ส่วนต่อร้อย**" });
      }),
    );
    render(<ConceptView />);

    await user.click(await screen.findByTestId("tutor-chat-open"));
    await user.type(screen.getByTestId("tutor-chat-input"), "ขอตัวอย่างเพิ่ม{Enter}");

    expect(await screen.findByTestId("tutor-chat-reply")).toHaveTextContent("เปอร์เซ็นต์คือ ส่วนต่อร้อย");
    expect(sent).toEqual({ messages: [{ role: "user", content: "ขอตัวอย่างเพิ่ม" }] });

    // "New chat" clears the conversation; the next message is sent without the old history.
    await user.click(screen.getByTestId("tutor-chat-new"));
    expect(screen.queryByTestId("tutor-chat-reply")).toBeNull();
    expect(screen.queryByTestId("tutor-chat-new")).toBeNull();
    const input = screen.getByTestId("tutor-chat-input");
    expect(input).toHaveFocus();
    await user.type(input, "อีกเรื่อง{Enter}");
    await screen.findByTestId("tutor-chat-reply");
    expect(sent).toEqual({ messages: [{ role: "user", content: "อีกเรื่อง" }] });
  });

  it("expands to a large dialog and back without losing the conversation", async () => {
    id.current = conceptId("percent-of");
    const user = userEvent.setup();
    server.use(http.post("*/api/concepts/:id/help", () => HttpResponse.json({ reply: "คำอธิบาย" })));
    render(<ConceptView />);
    await user.click(await screen.findByTestId("tutor-chat-open"));
    await user.type(screen.getByTestId("tutor-chat-input"), "ถาม{Enter}");
    await screen.findByTestId("tutor-chat-reply");

    await user.click(screen.getByTestId("tutor-chat-expand"));
    const dialog = screen.getByRole("dialog", { name: /ติวเตอร์ AI/ });
    expect(dialog).toHaveAttribute("aria-modal", "true");
    expect(screen.getByTestId("tutor-chat-reply")).toHaveTextContent("คำอธิบาย");
    expect(screen.getByTestId("tutor-chat-input")).toHaveFocus();
    expect(document.body.style.overflow).toBe("hidden");

    await user.keyboard("{Escape}");
    expect(screen.queryByRole("dialog")).toBeNull();
    expect(screen.getByTestId("tutor-chat-reply")).toHaveTextContent("คำอธิบาย");
    expect(document.body.style.overflow).toBe("");

    // Clicking the backdrop shrinks too; clicking inside the dialog does not.
    await user.click(screen.getByTestId("tutor-chat-expand"));
    await user.click(screen.getByTestId("tutor-chat-reply"));
    expect(screen.getByRole("dialog")).toBeInTheDocument();
    await user.click(screen.getByTestId("tutor-chat-backdrop"));
    expect(screen.queryByRole("dialog")).toBeNull();
  });

  it("drops a reply that arrives after starting a new chat", async () => {
    id.current = conceptId("percent-of");
    const user = userEvent.setup();
    let release!: () => void;
    const gate = new Promise<void>((r) => (release = r));
    server.use(
      http.post("*/api/concepts/:id/help", async () => {
        await gate;
        return HttpResponse.json({ reply: "คำตอบเก่า" });
      }),
    );
    render(<ConceptView />);
    await user.click(await screen.findByTestId("tutor-chat-open"));
    await user.type(screen.getByTestId("tutor-chat-input"), "ถาม{Enter}");
    await user.click(await screen.findByTestId("tutor-chat-new"));
    release();
    await new Promise((r) => setTimeout(r, 50));
    expect(screen.queryByText("คำตอบเก่า")).toBeNull();
    expect(screen.queryByTestId("tutor-chat-user")).toBeNull();
  });

  it("is hidden when the server does not offer it", async () => {
    id.current = conceptId("percent-of");
    server.use(
      http.get("*/api/concepts/:id", () =>
        HttpResponse.json({
          id: id.current,
          slug: "percent-of",
          name: "ร้อยละของจำนวน",
          topicId: "t",
          prerequisiteIds: [],
          courseId: COURSE_ID,
          dependentIds: [],
          mastery: { conceptId: id.current, score: 0, evidence: 0, level: "unassessed" },
          helpAvailable: false,
        }),
      ),
    );
    render(<ConceptView />);
    await screen.findByRole("heading", { level: 1 });
    expect(screen.queryByTestId("tutor-chat-open")).toBeNull();
  });
});
