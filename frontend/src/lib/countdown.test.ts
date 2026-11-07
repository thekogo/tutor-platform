import { describe, expect, it } from "vitest";
import { describeCountdown, formatCountdown, formatTimeLimit, remainingMs } from "./countdown";

describe("remainingMs", () => {
  const deadline = "2026-11-07T10:00:00.000Z";
  const at = (iso: string) => Date.parse(iso);

  it("counts down to the server deadline", () => {
    expect(remainingMs(deadline, at("2026-11-07T09:58:30.000Z"))).toBe(90_000);
  });
  it("never goes negative after the deadline", () => {
    expect(remainingMs(deadline, at("2026-11-07T10:00:05.000Z"))).toBe(0);
  });
  it("treats an invalid deadline as expired", () => {
    expect(remainingMs("nope", 0)).toBe(0);
  });
  it("accepts Date objects", () => {
    expect(remainingMs(new Date(10_000), 4_000)).toBe(6_000);
  });
});

describe("formatCountdown", () => {
  it.each([
    [0, "00:00"],
    [999, "00:01"],
    [61_000, "01:01"],
    [600_000, "10:00"],
    [3_600_000, "1:00:00"],
    [5_430_000, "1:30:30"],
  ])("%i ms → %s", (ms, out) => {
    expect(formatCountdown(ms)).toBe(out);
  });
});

describe("describeCountdown", () => {
  it("speaks minutes and seconds in Thai", () => {
    expect(describeCountdown(245_000)).toBe("เหลือเวลา 4 นาที 5 วินาที");
  });
  it("says zero seconds when expired", () => {
    expect(describeCountdown(0)).toBe("เหลือเวลา 0 วินาที");
  });
});

describe("formatTimeLimit", () => {
  it("formats hours and minutes", () => {
    expect(formatTimeLimit(5400)).toBe("1 ชั่วโมง 30 นาที");
    expect(formatTimeLimit(3600)).toBe("1 ชั่วโมง");
    expect(formatTimeLimit(600)).toBe("10 นาที");
  });
});
