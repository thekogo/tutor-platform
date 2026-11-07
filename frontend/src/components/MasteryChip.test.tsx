import { render, screen } from "@testing-library/react";
import { describe, expect, it } from "vitest";
import { MASTERY_LEVELS } from "@/lib/api/types";
import { MasteryChip, MasteryLegend } from "./MasteryChip";

describe("MasteryChip", () => {
  it.each([
    ["unassessed", "ยังไม่ประเมิน"],
    ["weak", "อ่อน"],
    ["developing", "กำลังพัฒนา"],
    ["mastered", "เชี่ยวชาญ"],
  ] as const)("conveys %s with text, not colour alone", (level, label) => {
    const { container } = render(<MasteryChip level={level} />);
    const chip = container.firstElementChild as HTMLElement;
    // Accessible text includes the term and the level name.
    expect(chip).toHaveTextContent(`ระดับความเข้าใจ: ${label}`);
    // The decorative icon is hidden from assistive tech.
    expect(chip.querySelector('[aria-hidden="true"]')).not.toBeNull();
    expect(chip).toHaveAttribute("data-level", level);
  });

  it("shows the score as a percentage when assessed", () => {
    render(<MasteryChip level="developing" score={0.634} />);
    expect(screen.getByText("(63%)")).toBeInTheDocument();
  });

  it("never shows a score for Unassessed (unknown is not weak)", () => {
    render(<MasteryChip level="unassessed" score={0} />);
    expect(screen.queryByText(/%/)).toBeNull();
  });

  it("legend lists every level", () => {
    render(<MasteryLegend />);
    const list = screen.getByRole("list", { name: /ระดับความเข้าใจ/ });
    expect(list.querySelectorAll("li")).toHaveLength(MASTERY_LEVELS.length);
  });
});
