import { render, screen, waitFor, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, expect, it } from "vitest";
import { api } from "@/lib/api/client";
import type { Recommendation } from "@/lib/api/types";
import { COURSE_ID } from "@/mocks/db";
import { routerMock } from "@/test/setup";
import { RecommendationCard } from "./RecommendationCard";

async function fetchRecommendations(): Promise<Recommendation[]> {
  const { data } = await api().GET("/courses/{courseId}/recommendations", {
    params: { path: { courseId: COURSE_ID } },
  });
  return data!;
}

describe("RecommendationCard", () => {
  it("always shows the reason path from Root Cause to target with Mastery Level chips", async () => {
    const [top] = await fetchRecommendations();
    render(<RecommendationCard recommendation={top} courseId={COURSE_ID} top />);

    const card = screen.getByTestId("top-recommendation");
    const path = within(card).getByTestId("recommendation-path");
    const steps = within(path).getAllByRole("listitem");
    expect(steps).toHaveLength(top.path.length);
    expect(steps.length).toBeGreaterThan(2);

    // Ordered root → target.
    top.path.forEach((ref, i) => expect(steps[i]).toHaveTextContent(ref.name));
    expect(steps[0]).toHaveTextContent("ต้นเหตุของจุดอ่อน");
    expect(steps.at(-1)).toHaveTextContent("แนวคิดที่ติดขัด");

    // Every step carries a textual Mastery Level.
    for (const step of steps) expect(step).toHaveTextContent(/ระดับความเข้าใจ: (อ่อน|กำลังพัฒนา|เชี่ยวชาญ|ยังไม่ประเมิน)/);

    // Each step links to its Concept page.
    expect(within(steps[0]).getByRole("link")).toHaveAttribute("href", `/concepts/${top.conceptId}`);
  });

  it("still shows a reason when the path is empty", () => {
    const rec: Recommendation = { kind: "study", conceptId: "c1", conceptName: "สัดส่วน", path: [], unblocks: 0, priority: 0 };
    render(<RecommendationCard recommendation={rec} courseId={COURSE_ID} />);
    const path = screen.getByTestId("recommendation-path");
    expect(within(path).getAllByRole("listitem")).toHaveLength(1);
    expect(path).toHaveTextContent("สัดส่วน");
  });

  it("offers a Diagnostic Quiz for diagnose Recommendations", async () => {
    const recs = await fetchRecommendations();
    const diagnose = recs.find((r) => r.kind === "diagnose")!;
    render(<RecommendationCard recommendation={diagnose} courseId={COURSE_ID} />);
    expect(screen.getByText(/ต้องวัดก่อน/)).toBeInTheDocument();
    await userEvent.click(screen.getByRole("button", { name: "ทำควิซวินิจฉัย" }));
    await waitFor(() => expect(routerMock.push).toHaveBeenCalledWith(expect.stringMatching(/^\/sessions\//)));
  });

  it("starts the Learning Path from a study Recommendation", async () => {
    const [top] = await fetchRecommendations();
    render(<RecommendationCard recommendation={top} courseId={COURSE_ID} top />);
    await userEvent.click(screen.getByRole("button", { name: "เริ่มเส้นทางการเรียน" }));
    await waitFor(() => expect(routerMock.push).toHaveBeenCalledWith("/learning-path"));
  });
});
