import { mockId } from "./ids";

export type MockChoice = { id: string; label: string; bodyMd: string; correct: boolean };
export type MockQuestion = {
  id: string;
  slug: string;
  stemMd: string;
  explanationMd: string;
  format: "standard" | "data-sufficiency" | "out-of-the-box";
  difficulty: 1 | 2 | 3;
  /** Primary Concept slug. */
  primary: string;
  /** All tagged Concept slugs (incl. primary). */
  concepts: string[];
  choices: MockChoice[];
  /** Exam Questions are kept out of Quizzes. */
  examOnly?: boolean;
};

type Def = Omit<MockQuestion, "id" | "choices"> & { choices: [string, boolean][] };

const LABELS = ["A", "B", "C", "D", "E"];

function q(def: Def): MockQuestion {
  const id = mockId("question", def.slug);
  return {
    ...def,
    id,
    choices: def.choices.map(([bodyMd, correct], i) => ({
      id: mockId("choice", `${def.slug}:${i}`),
      label: LABELS[i],
      bodyMd,
      correct,
    })),
  };
}

export const mockQuestions: MockQuestion[] = [
  q({
    slug: "percent-of-1",
    primary: "percent-of",
    concepts: ["percent-of", "percent-basics"],
    format: "standard",
    difficulty: 1,
    stemMd: "ร้อยละ $15$ ของจำนวนหนึ่งเท่ากับ $45$ จำนวนนั้นคือเท่าใด",
    choices: [["$300$", true], ["$675$", false], ["$30$", false], ["$450$", false]],
    explanationMd:
      "ให้จำนวนนั้นเป็น $x$\n\n$$0.15x = 45 \\Rightarrow x = \\dfrac{45}{0.15} = 300$$\n\nดังนั้นจำนวนนั้นคือ **300**",
  }),
  q({
    slug: "percent-change-1",
    primary: "percent-change",
    concepts: ["percent-change", "percent-of"],
    format: "standard",
    difficulty: 1,
    stemMd: "ราคาสินค้าเพิ่มจาก $80$ บาท เป็น $100$ บาท ราคาเพิ่มขึ้นร้อยละเท่าใด",
    choices: [["$20\\%$", false], ["$25\\%$", true], ["$80\\%$", false], ["$125\\%$", false]],
    explanationMd:
      "ร้อยละการเปลี่ยนแปลงคิดจาก **ค่าเดิม** เสมอ\n\n$$\\frac{100-80}{80}\\times 100\\% = \\frac{20}{80}\\times 100\\% = 25\\%$$\n\nข้อผิดพลาดที่พบบ่อยคือหารด้วย $100$ (ค่าใหม่) ซึ่งได้ $20\\%$",
  }),
  q({
    slug: "successive-percent-1",
    primary: "successive-percent",
    concepts: ["successive-percent", "percent-change"],
    format: "standard",
    difficulty: 2,
    stemMd:
      "ร้านค้าขึ้นราคาสินค้า $20\\%$ แล้วต่อมาลดราคาลง $20\\%$ ราคาสุดท้ายเทียบกับราคาเดิมเป็นอย่างไร",
    choices: [
      ["เท่าเดิม", false],
      ["ลดลง $4\\%$", true],
      ["เพิ่มขึ้น $4\\%$", false],
      ["ลดลง $2\\%$", false],
    ],
    explanationMd:
      "การเปลี่ยนแปลงร้อยละซ้อนกันต้อง **คูณ** ตัวคูณ ไม่ใช่บวกลบร้อยละตรง ๆ\n\n$$1.20 \\times 0.80 = 0.96$$\n\nราคาสุดท้ายเป็น $96\\%$ ของราคาเดิม คือ **ลดลง $4\\%$**",
  }),
  q({
    slug: "proportion-1",
    primary: "proportion",
    concepts: ["proportion", "linear-equations"],
    format: "standard",
    difficulty: 1,
    stemMd: "ถ้า $\\dfrac{x}{12} = \\dfrac{5}{8}$ แล้ว $x$ มีค่าเท่าใด",
    choices: [["$7.5$", true], ["$9.6$", false], ["$6$", false], ["$10$", false]],
    explanationMd: "คูณไขว้\n\n$$8x = 5 \\times 12 = 60 \\Rightarrow x = \\frac{60}{8} = 7.5$$",
  }),
  q({
    slug: "proportion-2",
    primary: "proportion",
    concepts: ["proportion", "ratio-basics"],
    format: "standard",
    difficulty: 2,
    stemMd:
      "ปากกา $3$ ด้ามราคา $45$ บาท ถ้าราคาต่อด้ามเท่ากัน ปากกา $8$ ด้ามจะราคาเท่าใด",
    choices: [["$110$ บาท", false], ["$115$ บาท", false], ["$120$ บาท", true], ["$135$ บาท", false]],
    explanationMd:
      "ตั้งสัดส่วน $\\dfrac{3}{45} = \\dfrac{8}{x}$\n\n$$3x = 360 \\Rightarrow x = 120$$\n\nหรือคิดราคาต่อด้าม $45 \\div 3 = 15$ บาท แล้ว $15 \\times 8 = 120$ บาท",
  }),
  q({
    slug: "remainders-1",
    primary: "remainders",
    concepts: ["remainders", "divisibility"],
    format: "standard",
    difficulty: 2,
    stemMd: "จำนวนเต็มบวก $n$ หารด้วย $6$ เหลือเศษ $4$ เศษจากการหาร $2n$ ด้วย $6$ คือเท่าใด",
    choices: [["$2$", true], ["$4$", false], ["$8$", false], ["$0$", false]],
    explanationMd:
      "เขียน $n = 6k + 4$\n\n$$2n = 12k + 8 = 6(2k+1) + 2$$\n\nดังนั้นเศษคือ **2** (เศษต้องน้อยกว่าตัวหารเสมอ จึงตอบ $8$ ไม่ได้)",
  }),
  q({
    slug: "unit-digit-1",
    primary: "unit-digit",
    concepts: ["unit-digit", "exponents", "remainders"],
    format: "standard",
    difficulty: 2,
    stemMd: "หลักหน่วยของ $3^{2026}$ คือเท่าใด",
    choices: [["$1$", false], ["$3$", false], ["$7$", false], ["$9$", true]],
    explanationMd:
      "หลักหน่วยของ $3^n$ วนเป็นรอบ $3, 9, 7, 1$ (รอบละ 4)\n\n$$2026 \\bmod 4 = 2$$\n\nจึงตรงกับตัวที่ 2 ของรอบ คือ **9**",
  }),
  q({
    slug: "quadratic-1",
    primary: "quadratic-equations",
    concepts: ["quadratic-equations", "factoring"],
    format: "standard",
    difficulty: 2,
    stemMd: "ผลบวกของรากทั้งหมดของสมการ $x^2 - 5x + 6 = 0$ เท่ากับเท่าใด",
    choices: [["$-5$", false], ["$5$", true], ["$6$", false], ["$1$", false]],
    explanationMd:
      "แยกตัวประกอบ $(x-2)(x-3) = 0$ ได้ $x = 2, 3$\n\nผลบวกของราก $= 2 + 3 = 5$ (ตรงกับ $-\\dfrac{b}{a} = 5$)",
  }),
  q({
    slug: "factoring-1",
    primary: "factoring",
    concepts: ["factoring", "algebraic-expressions"],
    format: "standard",
    difficulty: 1,
    stemMd: "ข้อใดเป็นการแยกตัวประกอบของ $x^2 - 9$",
    choices: [
      ["$(x-9)(x+1)$", false],
      ["$(x-3)^2$", false],
      ["$(x-3)(x+3)$", true],
      ["$x(x-9)$", false],
    ],
    explanationMd: "ผลต่างกำลังสอง $a^2 - b^2 = (a-b)(a+b)$\n\n$$x^2 - 9 = x^2 - 3^2 = (x-3)(x+3)$$",
  }),
  q({
    slug: "ds-linear-systems-1",
    primary: "linear-systems",
    concepts: ["linear-systems"],
    format: "data-sufficiency",
    difficulty: 2,
    stemMd: "จงหาค่าของ $x$\n\n1. $x + y = 10$\n2. $x - y = 2$",
    choices: [
      ["ข้อมูลข้อ 1 อย่างเดียวเพียงพอ", false],
      ["ข้อมูลข้อ 2 อย่างเดียวเพียงพอ", false],
      ["ต้องใช้ข้อมูลทั้งสองข้อร่วมกัน", true],
      ["ใช้ข้อมูลทั้งสองข้อก็ยังไม่เพียงพอ", false],
    ],
    explanationMd:
      "แต่ละข้อมีสองตัวแปรในสมการเดียว จึงหา $x$ ไม่ได้\n\nใช้ร่วมกัน: บวกสองสมการได้ $2x = 12 \\Rightarrow x = 6$",
  }),
  q({
    slug: "percent-basics-1",
    primary: "percent-basics",
    concepts: ["percent-basics", "fractions"],
    format: "standard",
    difficulty: 1,
    stemMd: "$\\dfrac{3}{8}$ คิดเป็นร้อยละเท่าใด",
    choices: [["$37.5\\%$", true], ["$38\\%$", false], ["$3.75\\%$", false], ["$26.7\\%$", false]],
    explanationMd: "$$\\frac{3}{8} \\times 100\\% = 37.5\\%$$",
  }),
  q({
    slug: "ootb-percent-1",
    primary: "percent-change",
    concepts: ["percent-change", "estimation"],
    format: "out-of-the-box",
    difficulty: 3,
    stemMd:
      "ด้านของสี่เหลี่ยมจัตุรัสยาวขึ้น $10\\%$ พื้นที่ของสี่เหลี่ยมจัตุรัสเพิ่มขึ้นร้อยละเท่าใด",
    choices: [["$10\\%$", false], ["$20\\%$", false], ["$21\\%$", true], ["$100\\%$", false]],
    explanationMd: "พื้นที่แปรผันตามกำลังสองของด้าน\n\n$$(1.1)^2 = 1.21$$\n\nพื้นที่เพิ่มขึ้น **$21\\%$**",
  }),

  // ── Exam-only Questions ────────────────────────────────────
  q({
    slug: "exam-mean-1",
    primary: "mean",
    concepts: ["mean"],
    format: "standard",
    difficulty: 1,
    examOnly: true,
    stemMd: "ค่าเฉลี่ยเลขคณิตของข้อมูล $4, 8, 10, 14$ คือเท่าใด",
    choices: [["$8$", false], ["$9$", true], ["$10$", false], ["$12$", false]],
    explanationMd: "$$\\bar{x} = \\frac{4+8+10+14}{4} = \\frac{36}{4} = 9$$",
  }),
  q({
    slug: "exam-speed-1",
    primary: "speed-distance-time",
    concepts: ["speed-distance-time", "unit-conversion"],
    format: "standard",
    difficulty: 1,
    examOnly: true,
    stemMd: "รถยนต์วิ่งด้วยอัตราเร็ว $60$ กม./ชม. เป็นเวลา $2$ ชั่วโมง $30$ นาที จะได้ระยะทางกี่กิโลเมตร",
    choices: [["$120$", false], ["$130$", false], ["$150$", true], ["$180$", false]],
    explanationMd: "$2$ ชั่วโมง $30$ นาที $= 2.5$ ชั่วโมง\n\n$$s = vt = 60 \\times 2.5 = 150 \\text{ กม.}$$",
  }),
  q({
    slug: "exam-comb-1",
    primary: "combinations",
    concepts: ["combinations"],
    format: "standard",
    difficulty: 2,
    examOnly: true,
    stemMd: "เลือกกรรมการ $2$ คนจากผู้สมัคร $6$ คน ได้กี่วิธี",
    choices: [["$12$", false], ["$15$", true], ["$30$", false], ["$36$", false]],
    explanationMd: "$$\\binom{6}{2} = \\frac{6!}{2!\\,4!} = \\frac{6 \\times 5}{2} = 15$$",
  }),
  q({
    slug: "exam-venn-1",
    primary: "venn-two-sets",
    concepts: ["venn-two-sets", "set-operations"],
    format: "standard",
    difficulty: 2,
    examOnly: true,
    stemMd:
      "นักศึกษา $40$ คน ชอบคณิตศาสตร์ $25$ คน ชอบภาษาอังกฤษ $20$ คน และชอบทั้งสองวิชา $10$ คน มีกี่คนที่ไม่ชอบทั้งสองวิชา",
    choices: [["$0$", false], ["$5$", true], ["$10$", false], ["$15$", false]],
    explanationMd: "$$n(A \\cup B) = 25 + 20 - 10 = 35$$\n\nไม่ชอบทั้งสองวิชา $= 40 - 35 = 5$ คน",
  }),
  q({
    slug: "exam-seq-1",
    primary: "sequence-patterns",
    concepts: ["sequence-patterns"],
    format: "out-of-the-box",
    difficulty: 2,
    examOnly: true,
    stemMd: "พจน์ถัดไปของลำดับ $2, 6, 12, 20, 30, \\ldots$ คือเท่าใด",
    choices: [["$40$", false], ["$42$", true], ["$44$", false], ["$45$", false]],
    explanationMd: "พจน์ที่ $n$ คือ $n(n+1)$: $1\\cdot2, 2\\cdot3, 3\\cdot4, \\ldots$\n\nพจน์ที่ 6 $= 6 \\times 7 = 42$",
  }),
];

export const questionById = new Map(mockQuestions.map((x) => [x.id, x]));
