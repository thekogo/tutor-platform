import { render } from "@testing-library/react";
import { describe, expect, it } from "vitest";
import { Markdown, normalizeMath } from "./Markdown";

describe("Markdown", () => {
  it("renders inline and display math with KaTeX", () => {
    const { container } = render(<Markdown>{"ร้อยละ $15$ ของ $x$\n\n$$\\frac{45}{0.15} = 300$$"}</Markdown>);
    expect(container.querySelectorAll(".katex").length).toBeGreaterThanOrEqual(3);
    expect(container.querySelector(".katex-display")).not.toBeNull();
    // MathML is emitted for screen readers.
    expect(container.querySelector("math")).not.toBeNull();
    expect(container).toHaveTextContent("ร้อยละ");
  });

  it("does not render raw HTML", () => {
    const { container } = render(<Markdown>{'ก่อน <img src=x onerror="alert(1)"> <script>alert(1)</script> หลัง'}</Markdown>);
    expect(container.querySelector("img")).toBeNull();
    expect(container.querySelector("script")).toBeNull();
  });

  it("strips javascript: links", () => {
    const { container } = render(<Markdown>{"[คลิก](javascript:alert(1))"}</Markdown>);
    expect(container.querySelector("a")?.getAttribute("href") ?? "").not.toContain("javascript");
  });

  it("renders GFM tables", () => {
    const { container } = render(<Markdown>{"| a | b |\n|---|---|\n| 1 | 2 |"}</Markdown>);
    expect(container.querySelector("table")).not.toBeNull();
  });

  it("inline mode has no block wrapper", () => {
    const { container } = render(<Markdown inline>{"$x^2$"}</Markdown>);
    expect(container.querySelector("p")).toBeNull();
    expect(container.querySelector(".katex")).not.toBeNull();
  });

  it("renders bad TeX without throwing", () => {
    const { container } = render(<Markdown>{"$\\frac{1$"}</Markdown>);
    expect(container.textContent).toBeTruthy();
  });
});

describe("normalizeMath", () => {
  it("turns single-line $$…$$ into a display block", () => {
    expect(normalizeMath("ก่อน\n\n$$a+b$$\n\nหลัง")).toBe("ก่อน\n\n$$\na+b\n$$\n\nหลัง");
  });
  it("leaves inline $…$ and fenced blocks alone", () => {
    const md = "ค่า $x$ และ\n$$\nx^2\n$$";
    expect(normalizeMath(md)).toBe(md);
  });
});
