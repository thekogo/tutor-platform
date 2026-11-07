// Generated from content/cu-best-math/graph.yaml (seed data for mocks). Do not edit by hand.
// Regenerate if the seed graph changes substantially.

export const seedCourse = {"id": "cu-best-math", "name": "CU-BEST คณิตศาสตร์", "goal": "สอบเข้า ป.โท จุฬาฯ (CU-BEST)"} as const;

export const seedFormats: { slug: string; name: string }[] = [
  {
    "slug": "standard",
    "name": "โจทย์ปกติ"
  },
  {
    "slug": "data-sufficiency",
    "name": "Data sufficiency"
  },
  {
    "slug": "out-of-the-box",
    "name": "คณิตคิดนอกกรอบ"
  }
];

export const seedTopics: { slug: string; name: string; examWeight: number | null }[] = [
  {
    "slug": "number-system",
    "name": "ระบบจำนวน",
    "examWeight": null
  },
  {
    "slug": "equations-inequalities",
    "name": "สมการ และอสมการ",
    "examWeight": null
  },
  {
    "slug": "ratio-proportion",
    "name": "อัตราส่วนและสัดส่วน",
    "examWeight": null
  },
  {
    "slug": "rule-of-three",
    "name": "บัญญัติไตรยางศ์",
    "examWeight": null
  },
  {
    "slug": "percent",
    "name": "ร้อยละ เปอร์เซ็นต์",
    "examWeight": null
  },
  {
    "slug": "geometry",
    "name": "เรขาคณิต",
    "examWeight": null
  },
  {
    "slug": "analytic-geometry",
    "name": "เรขาคณิตวิเคราะห์",
    "examWeight": null
  },
  {
    "slug": "sequences-series",
    "name": "ลำดับ อนุกรม",
    "examWeight": null
  },
  {
    "slug": "statistics",
    "name": "สถิติ",
    "examWeight": null
  },
  {
    "slug": "sets",
    "name": "เซต",
    "examWeight": null
  },
  {
    "slug": "counting-probability",
    "name": "การนับ ความน่าจะเป็น",
    "examWeight": null
  },
  {
    "slug": "mixtures",
    "name": "ของผสม",
    "examWeight": null
  },
  {
    "slug": "motion",
    "name": "การเคลื่อนที่",
    "examWeight": null
  },
  {
    "slug": "work",
    "name": "การทำงาน",
    "examWeight": null
  },
  {
    "slug": "tables-graphs",
    "name": "ตาราง กราฟ",
    "examWeight": null
  }
];

export const seedConcepts: { slug: string; topic: string; name: string; prerequisites: string[] }[] = [
  {
    "slug": "integer-operations",
    "topic": "number-system",
    "name": "การดำเนินการจำนวนเต็มและลำดับการคำนวณ",
    "prerequisites": []
  },
  {
    "slug": "fractions",
    "topic": "number-system",
    "name": "เศษส่วน",
    "prerequisites": [
      "integer-operations"
    ]
  },
  {
    "slug": "decimals",
    "topic": "number-system",
    "name": "ทศนิยม",
    "prerequisites": [
      "fractions"
    ]
  },
  {
    "slug": "estimation",
    "topic": "number-system",
    "name": "การประมาณค่าและคำนวณเร็ว",
    "prerequisites": [
      "fractions",
      "decimals"
    ]
  },
  {
    "slug": "odd-even-sign",
    "topic": "number-system",
    "name": "สมบัติคู่-คี่ และเครื่องหมายบวก-ลบ",
    "prerequisites": [
      "integer-operations"
    ]
  },
  {
    "slug": "divisibility",
    "topic": "number-system",
    "name": "การหารลงตัว ตัวประกอบ จำนวนเฉพาะ",
    "prerequisites": [
      "integer-operations"
    ]
  },
  {
    "slug": "gcd-lcm",
    "topic": "number-system",
    "name": "ห.ร.ม. และ ค.ร.น.",
    "prerequisites": [
      "divisibility"
    ]
  },
  {
    "slug": "remainders",
    "topic": "number-system",
    "name": "เศษจากการหาร",
    "prerequisites": [
      "divisibility"
    ]
  },
  {
    "slug": "exponents",
    "topic": "number-system",
    "name": "เลขยกกำลัง",
    "prerequisites": [
      "integer-operations",
      "fractions"
    ]
  },
  {
    "slug": "roots",
    "topic": "number-system",
    "name": "รากที่สองและกรณฑ์",
    "prerequisites": [
      "exponents"
    ]
  },
  {
    "slug": "unit-digit",
    "topic": "number-system",
    "name": "หลักหน่วยของเลขยกกำลัง",
    "prerequisites": [
      "exponents",
      "remainders"
    ]
  },
  {
    "slug": "absolute-value",
    "topic": "number-system",
    "name": "ค่าสัมบูรณ์",
    "prerequisites": [
      "integer-operations"
    ]
  },
  {
    "slug": "algebraic-expressions",
    "topic": "equations-inequalities",
    "name": "นิพจน์พีชคณิตและการจัดรูป",
    "prerequisites": [
      "integer-operations",
      "fractions"
    ]
  },
  {
    "slug": "factoring",
    "topic": "equations-inequalities",
    "name": "การแยกตัวประกอบพหุนาม",
    "prerequisites": [
      "algebraic-expressions"
    ]
  },
  {
    "slug": "linear-equations",
    "topic": "equations-inequalities",
    "name": "สมการเชิงเส้นตัวแปรเดียว",
    "prerequisites": [
      "algebraic-expressions"
    ]
  },
  {
    "slug": "word-problem-translation",
    "topic": "equations-inequalities",
    "name": "แปลงโจทย์ปัญหาเป็นสมการ",
    "prerequisites": [
      "linear-equations"
    ]
  },
  {
    "slug": "age-problems",
    "topic": "equations-inequalities",
    "name": "โจทย์อายุ",
    "prerequisites": [
      "word-problem-translation",
      "linear-systems"
    ]
  },
  {
    "slug": "linear-systems",
    "topic": "equations-inequalities",
    "name": "ระบบสมการเชิงเส้นสองตัวแปร",
    "prerequisites": [
      "linear-equations"
    ]
  },
  {
    "slug": "quadratic-equations",
    "topic": "equations-inequalities",
    "name": "สมการกำลังสอง",
    "prerequisites": [
      "factoring",
      "roots"
    ]
  },
  {
    "slug": "exponential-equations",
    "topic": "equations-inequalities",
    "name": "สมการเลขยกกำลัง",
    "prerequisites": [
      "exponents",
      "linear-equations"
    ]
  },
  {
    "slug": "linear-inequalities",
    "topic": "equations-inequalities",
    "name": "อสมการเชิงเส้น",
    "prerequisites": [
      "linear-equations"
    ]
  },
  {
    "slug": "quadratic-inequalities",
    "topic": "equations-inequalities",
    "name": "อสมการกำลังสอง",
    "prerequisites": [
      "quadratic-equations",
      "linear-inequalities"
    ]
  },
  {
    "slug": "absolute-value-equations",
    "topic": "equations-inequalities",
    "name": "สมการและอสมการค่าสัมบูรณ์",
    "prerequisites": [
      "absolute-value",
      "linear-inequalities"
    ]
  },
  {
    "slug": "ratio-basics",
    "topic": "ratio-proportion",
    "name": "อัตราส่วนและอัตราส่วนที่เท่ากัน",
    "prerequisites": [
      "fractions"
    ]
  },
  {
    "slug": "proportion",
    "topic": "ratio-proportion",
    "name": "สัดส่วนและการแก้สัดส่วน",
    "prerequisites": [
      "ratio-basics",
      "linear-equations"
    ]
  },
  {
    "slug": "multi-ratio",
    "topic": "ratio-proportion",
    "name": "อัตราส่วนของหลายจำนวน (a:b:c)",
    "prerequisites": [
      "ratio-basics",
      "gcd-lcm"
    ]
  },
  {
    "slug": "dividing-in-ratio",
    "topic": "ratio-proportion",
    "name": "การแบ่งตามอัตราส่วน",
    "prerequisites": [
      "ratio-basics"
    ]
  },
  {
    "slug": "ratio-change",
    "topic": "ratio-proportion",
    "name": "อัตราส่วนเมื่อเพิ่มหรือลดปริมาณ",
    "prerequisites": [
      "proportion",
      "word-problem-translation"
    ]
  },
  {
    "slug": "direct-variation",
    "topic": "rule-of-three",
    "name": "การแปรผันตรง",
    "prerequisites": [
      "proportion"
    ]
  },
  {
    "slug": "inverse-variation",
    "topic": "rule-of-three",
    "name": "การแปรผกผัน",
    "prerequisites": [
      "direct-variation"
    ]
  },
  {
    "slug": "rule-of-three",
    "topic": "rule-of-three",
    "name": "บัญญัติไตรยางศ์ (ตรงและผกผัน)",
    "prerequisites": [
      "direct-variation",
      "inverse-variation"
    ]
  },
  {
    "slug": "percent-basics",
    "topic": "percent",
    "name": "ความหมายของร้อยละ และการแปลงเศษส่วน/ทศนิยม",
    "prerequisites": [
      "fractions",
      "decimals"
    ]
  },
  {
    "slug": "percent-of",
    "topic": "percent",
    "name": "การหาร้อยละของจำนวน และหาจำนวนเต็มย้อนกลับ",
    "prerequisites": [
      "percent-basics",
      "proportion"
    ]
  },
  {
    "slug": "percent-change",
    "topic": "percent",
    "name": "ร้อยละการเปลี่ยนแปลง (เพิ่ม/ลด)",
    "prerequisites": [
      "percent-of"
    ]
  },
  {
    "slug": "successive-percent",
    "topic": "percent",
    "name": "ร้อยละซ้อน (เปลี่ยนแปลงหลายครั้ง)",
    "prerequisites": [
      "percent-change"
    ]
  },
  {
    "slug": "profit-loss-discount",
    "topic": "percent",
    "name": "กำไร ขาดทุน ส่วนลด",
    "prerequisites": [
      "percent-change"
    ]
  },
  {
    "slug": "interest",
    "topic": "percent",
    "name": "ดอกเบี้ยเชิงเดี่ยวและทบต้น",
    "prerequisites": [
      "successive-percent",
      "exponents"
    ]
  },
  {
    "slug": "angles-lines",
    "topic": "geometry",
    "name": "มุม เส้นตรง และเส้นขนาน",
    "prerequisites": [
      "integer-operations"
    ]
  },
  {
    "slug": "triangle-properties",
    "topic": "geometry",
    "name": "สมบัติของสามเหลี่ยมและผลรวมมุมภายใน",
    "prerequisites": [
      "angles-lines"
    ]
  },
  {
    "slug": "pythagorean-theorem",
    "topic": "geometry",
    "name": "ทฤษฎีบทพีทาโกรัส",
    "prerequisites": [
      "roots",
      "triangle-properties"
    ]
  },
  {
    "slug": "special-right-triangles",
    "topic": "geometry",
    "name": "สามเหลี่ยมมุมฉากพิเศษ (30-60-90, 45-45-90)",
    "prerequisites": [
      "pythagorean-theorem"
    ]
  },
  {
    "slug": "similar-triangles",
    "topic": "geometry",
    "name": "สามเหลี่ยมคล้าย",
    "prerequisites": [
      "triangle-properties",
      "proportion"
    ]
  },
  {
    "slug": "polygons",
    "topic": "geometry",
    "name": "รูปหลายเหลี่ยมและผลรวมมุมภายใน",
    "prerequisites": [
      "triangle-properties"
    ]
  },
  {
    "slug": "perimeter-area",
    "topic": "geometry",
    "name": "เส้นรอบรูปและพื้นที่สามเหลี่ยม/สี่เหลี่ยม",
    "prerequisites": [
      "triangle-properties",
      "fractions"
    ]
  },
  {
    "slug": "circles",
    "topic": "geometry",
    "name": "วงกลม เส้นรอบวง พื้นที่ ส่วนโค้ง และเซกเตอร์",
    "prerequisites": [
      "perimeter-area"
    ]
  },
  {
    "slug": "circle-angles",
    "topic": "geometry",
    "name": "มุมในวงกลมและเส้นสัมผัส",
    "prerequisites": [
      "circles",
      "angles-lines"
    ]
  },
  {
    "slug": "composite-area",
    "topic": "geometry",
    "name": "พื้นที่ส่วนที่แรเงาและรูปประกอบ",
    "prerequisites": [
      "perimeter-area",
      "circles"
    ]
  },
  {
    "slug": "volume-surface-area",
    "topic": "geometry",
    "name": "ปริมาตรและพื้นที่ผิวทรงสามมิติ",
    "prerequisites": [
      "perimeter-area",
      "circles"
    ]
  },
  {
    "slug": "coordinate-plane",
    "topic": "analytic-geometry",
    "name": "ระบบพิกัดฉาก",
    "prerequisites": [
      "integer-operations"
    ]
  },
  {
    "slug": "distance-midpoint",
    "topic": "analytic-geometry",
    "name": "ระยะทางและจุดกึ่งกลางระหว่างสองจุด",
    "prerequisites": [
      "coordinate-plane",
      "pythagorean-theorem"
    ]
  },
  {
    "slug": "slope",
    "topic": "analytic-geometry",
    "name": "ความชัน",
    "prerequisites": [
      "coordinate-plane",
      "fractions"
    ]
  },
  {
    "slug": "line-equation",
    "topic": "analytic-geometry",
    "name": "สมการเส้นตรง",
    "prerequisites": [
      "slope",
      "linear-equations"
    ]
  },
  {
    "slug": "parallel-perpendicular",
    "topic": "analytic-geometry",
    "name": "เส้นขนานและเส้นตั้งฉาก",
    "prerequisites": [
      "line-equation"
    ]
  },
  {
    "slug": "line-intersection",
    "topic": "analytic-geometry",
    "name": "จุดตัดของเส้นตรง",
    "prerequisites": [
      "line-equation",
      "linear-systems"
    ]
  },
  {
    "slug": "area-on-plane",
    "topic": "analytic-geometry",
    "name": "พื้นที่รูปหลายเหลี่ยมบนระนาบพิกัด",
    "prerequisites": [
      "distance-midpoint",
      "line-intersection",
      "perimeter-area"
    ]
  },
  {
    "slug": "circle-equation",
    "topic": "analytic-geometry",
    "name": "สมการวงกลม",
    "prerequisites": [
      "distance-midpoint",
      "quadratic-equations"
    ]
  },
  {
    "slug": "sequence-patterns",
    "topic": "sequences-series",
    "name": "การหาแบบรูปของลำดับ",
    "prerequisites": [
      "integer-operations"
    ]
  },
  {
    "slug": "arithmetic-sequence",
    "topic": "sequences-series",
    "name": "ลำดับเลขคณิต",
    "prerequisites": [
      "sequence-patterns",
      "linear-equations"
    ]
  },
  {
    "slug": "geometric-sequence",
    "topic": "sequences-series",
    "name": "ลำดับเรขาคณิต",
    "prerequisites": [
      "sequence-patterns",
      "exponents"
    ]
  },
  {
    "slug": "arithmetic-series",
    "topic": "sequences-series",
    "name": "อนุกรมเลขคณิต",
    "prerequisites": [
      "arithmetic-sequence"
    ]
  },
  {
    "slug": "geometric-series",
    "topic": "sequences-series",
    "name": "อนุกรมเรขาคณิต",
    "prerequisites": [
      "geometric-sequence"
    ]
  },
  {
    "slug": "infinite-geometric-series",
    "topic": "sequences-series",
    "name": "อนุกรมเรขาคณิตอนันต์",
    "prerequisites": [
      "geometric-series"
    ]
  },
  {
    "slug": "mean",
    "topic": "statistics",
    "name": "ค่าเฉลี่ยเลขคณิต",
    "prerequisites": [
      "integer-operations",
      "fractions"
    ]
  },
  {
    "slug": "weighted-mean",
    "topic": "statistics",
    "name": "ค่าเฉลี่ยถ่วงน้ำหนักและค่าเฉลี่ยรวม",
    "prerequisites": [
      "mean"
    ]
  },
  {
    "slug": "median-mode",
    "topic": "statistics",
    "name": "มัธยฐานและฐานนิยม",
    "prerequisites": [
      "integer-operations"
    ]
  },
  {
    "slug": "quartiles-percentiles",
    "topic": "statistics",
    "name": "ควอร์ไทล์และเปอร์เซ็นไทล์",
    "prerequisites": [
      "median-mode"
    ]
  },
  {
    "slug": "dispersion",
    "topic": "statistics",
    "name": "พิสัย ส่วนเบี่ยงเบนมาตรฐาน ความแปรปรวน",
    "prerequisites": [
      "mean",
      "roots"
    ]
  },
  {
    "slug": "data-transformation-effect",
    "topic": "statistics",
    "name": "ผลของการเพิ่ม/ลด/คูณข้อมูลต่อค่าสถิติ",
    "prerequisites": [
      "mean",
      "median-mode",
      "dispersion"
    ]
  },
  {
    "slug": "set-basics",
    "topic": "sets",
    "name": "เซต สมาชิก และสับเซต",
    "prerequisites": []
  },
  {
    "slug": "subset-count",
    "topic": "sets",
    "name": "จำนวนสับเซตและเพาเวอร์เซต",
    "prerequisites": [
      "set-basics",
      "exponents"
    ]
  },
  {
    "slug": "set-operations",
    "topic": "sets",
    "name": "ยูเนียน อินเตอร์เซกชัน คอมพลีเมนต์ ผลต่าง",
    "prerequisites": [
      "set-basics"
    ]
  },
  {
    "slug": "venn-two-sets",
    "topic": "sets",
    "name": "แผนภาพเวนน์และจำนวนสมาชิก 2 เซต",
    "prerequisites": [
      "set-operations"
    ]
  },
  {
    "slug": "venn-three-sets",
    "topic": "sets",
    "name": "แผนภาพเวนน์และจำนวนสมาชิก 3 เซต",
    "prerequisites": [
      "venn-two-sets"
    ]
  },
  {
    "slug": "counting-principles",
    "topic": "counting-probability",
    "name": "หลักการบวกและหลักการคูณ",
    "prerequisites": [
      "integer-operations"
    ]
  },
  {
    "slug": "factorial",
    "topic": "counting-probability",
    "name": "แฟกทอเรียล",
    "prerequisites": [
      "counting-principles"
    ]
  },
  {
    "slug": "permutations",
    "topic": "counting-probability",
    "name": "การเรียงสับเปลี่ยน",
    "prerequisites": [
      "factorial"
    ]
  },
  {
    "slug": "permutations-special",
    "topic": "counting-probability",
    "name": "เรียงสับเปลี่ยนของซ้ำและเรียงเป็นวงกลม",
    "prerequisites": [
      "permutations"
    ]
  },
  {
    "slug": "combinations",
    "topic": "counting-probability",
    "name": "การจัดหมู่",
    "prerequisites": [
      "permutations"
    ]
  },
  {
    "slug": "probability-basics",
    "topic": "counting-probability",
    "name": "ความน่าจะเป็นพื้นฐาน",
    "prerequisites": [
      "counting-principles",
      "fractions",
      "set-operations"
    ]
  },
  {
    "slug": "probability-counting",
    "topic": "counting-probability",
    "name": "ความน่าจะเป็นที่ใช้การจัดหมู่/เรียงสับเปลี่ยน",
    "prerequisites": [
      "probability-basics",
      "combinations"
    ]
  },
  {
    "slug": "complement-at-least",
    "topic": "counting-probability",
    "name": "เหตุการณ์ตรงข้ามและโจทย์ \"อย่างน้อย\"",
    "prerequisites": [
      "probability-basics"
    ]
  },
  {
    "slug": "independent-events",
    "topic": "counting-probability",
    "name": "เหตุการณ์อิสระ และการหยิบแบบคืน/ไม่คืน",
    "prerequisites": [
      "probability-basics"
    ]
  },
  {
    "slug": "concentration",
    "topic": "mixtures",
    "name": "ความเข้มข้นเป็นร้อยละ",
    "prerequisites": [
      "percent-of"
    ]
  },
  {
    "slug": "mixing-solutions",
    "topic": "mixtures",
    "name": "การผสมสารละลายสองชนิด",
    "prerequisites": [
      "concentration",
      "word-problem-translation",
      "weighted-mean"
    ]
  },
  {
    "slug": "dilution-evaporation",
    "topic": "mixtures",
    "name": "การเติมน้ำและการระเหย",
    "prerequisites": [
      "concentration",
      "word-problem-translation"
    ]
  },
  {
    "slug": "repeated-replacement",
    "topic": "mixtures",
    "name": "ตักออกแล้วเติมแทนซ้ำหลายครั้ง",
    "prerequisites": [
      "mixing-solutions",
      "successive-percent"
    ]
  },
  {
    "slug": "mixture-ratio",
    "topic": "mixtures",
    "name": "การผสมตามอัตราส่วน",
    "prerequisites": [
      "multi-ratio",
      "dividing-in-ratio"
    ]
  },
  {
    "slug": "unit-conversion",
    "topic": "motion",
    "name": "การแปลงหน่วย (กม./ชม. ↔ ม./วินาที)",
    "prerequisites": [
      "ratio-basics"
    ]
  },
  {
    "slug": "speed-distance-time",
    "topic": "motion",
    "name": "ความเร็ว ระยะทาง เวลา",
    "prerequisites": [
      "proportion",
      "linear-equations"
    ]
  },
  {
    "slug": "average-speed",
    "topic": "motion",
    "name": "ความเร็วเฉลี่ย",
    "prerequisites": [
      "speed-distance-time"
    ]
  },
  {
    "slug": "relative-motion",
    "topic": "motion",
    "name": "การเคลื่อนที่สวนทางและไล่ตาม",
    "prerequisites": [
      "speed-distance-time",
      "word-problem-translation"
    ]
  },
  {
    "slug": "train-problems",
    "topic": "motion",
    "name": "รถไฟผ่านคน สะพาน หรือรถไฟอีกขบวน",
    "prerequisites": [
      "relative-motion"
    ]
  },
  {
    "slug": "stream-problems",
    "topic": "motion",
    "name": "ตามน้ำและทวนน้ำ",
    "prerequisites": [
      "relative-motion",
      "linear-systems"
    ]
  },
  {
    "slug": "circular-track",
    "topic": "motion",
    "name": "การวิ่งรอบสนาม",
    "prerequisites": [
      "relative-motion",
      "gcd-lcm"
    ]
  },
  {
    "slug": "work-rate",
    "topic": "work",
    "name": "อัตราการทำงาน (งานต่อหน่วยเวลา)",
    "prerequisites": [
      "fractions",
      "inverse-variation"
    ]
  },
  {
    "slug": "man-days",
    "topic": "work",
    "name": "คน-วัน และจำนวนคนกับเวลา",
    "prerequisites": [
      "rule-of-three"
    ]
  },
  {
    "slug": "combined-work",
    "topic": "work",
    "name": "การทำงานร่วมกัน",
    "prerequisites": [
      "work-rate",
      "word-problem-translation"
    ]
  },
  {
    "slug": "work-in-stages",
    "topic": "work",
    "name": "เข้าหรือออกจากงานกลางคัน",
    "prerequisites": [
      "combined-work"
    ]
  },
  {
    "slug": "pipes-tanks",
    "topic": "work",
    "name": "ท่อน้ำเข้าและท่อน้ำออก",
    "prerequisites": [
      "combined-work"
    ]
  },
  {
    "slug": "read-table",
    "topic": "tables-graphs",
    "name": "การอ่านข้อมูลจากตาราง",
    "prerequisites": []
  },
  {
    "slug": "read-bar-line",
    "topic": "tables-graphs",
    "name": "การอ่านกราฟแท่งและกราฟเส้น",
    "prerequisites": [
      "read-table"
    ]
  },
  {
    "slug": "read-pie",
    "topic": "tables-graphs",
    "name": "การอ่านแผนภูมิวงกลม",
    "prerequisites": [
      "read-table",
      "percent-of"
    ]
  },
  {
    "slug": "chart-percent-change",
    "topic": "tables-graphs",
    "name": "เปรียบเทียบและหาร้อยละการเปลี่ยนแปลงจากกราฟ",
    "prerequisites": [
      "read-bar-line",
      "percent-change",
      "estimation"
    ]
  },
  {
    "slug": "chart-multi-source",
    "topic": "tables-graphs",
    "name": "รวมข้อมูลจากหลายตาราง/กราฟ",
    "prerequisites": [
      "read-bar-line",
      "read-pie",
      "ratio-basics"
    ]
  }
];

