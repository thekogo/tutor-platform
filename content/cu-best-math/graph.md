# CU-BEST คณิตศาสตร์ — knowledge graph

Generated from `content/cu-best-math/graph.yaml` by `scripts/graph_check.py`. Do not edit by hand.

```mermaid
flowchart LR
  subgraph number-system["ระบบจำนวน"]
    integer-operations["การดำเนินการจำนวนเต็มและลำดับการคำนวณ"]
    fractions["เศษส่วน"]
    decimals["ทศนิยม"]
    estimation["การประมาณค่าและคำนวณเร็ว"]
    odd-even-sign["สมบัติคู่-คี่ และเครื่องหมายบวก-ลบ"]
    divisibility["การหารลงตัว ตัวประกอบ จำนวนเฉพาะ"]
    gcd-lcm["ห.ร.ม. และ ค.ร.น."]
    remainders["เศษจากการหาร"]
    exponents["เลขยกกำลัง"]
    roots["รากที่สองและกรณฑ์"]
    unit-digit["หลักหน่วยของเลขยกกำลัง"]
    absolute-value["ค่าสัมบูรณ์"]
  end
  subgraph equations-inequalities["สมการ และอสมการ"]
    algebraic-expressions["นิพจน์พีชคณิตและการจัดรูป"]
    factoring["การแยกตัวประกอบพหุนาม"]
    linear-equations["สมการเชิงเส้นตัวแปรเดียว"]
    word-problem-translation["แปลงโจทย์ปัญหาเป็นสมการ"]
    age-problems["โจทย์อายุ"]
    linear-systems["ระบบสมการเชิงเส้นสองตัวแปร"]
    quadratic-equations["สมการกำลังสอง"]
    exponential-equations["สมการเลขยกกำลัง"]
    linear-inequalities["อสมการเชิงเส้น"]
    quadratic-inequalities["อสมการกำลังสอง"]
    absolute-value-equations["สมการและอสมการค่าสัมบูรณ์"]
  end
  subgraph ratio-proportion["อัตราส่วนและสัดส่วน"]
    ratio-basics["อัตราส่วนและอัตราส่วนที่เท่ากัน"]
    proportion["สัดส่วนและการแก้สัดส่วน"]
    multi-ratio["อัตราส่วนของหลายจำนวน (a:b:c)"]
    dividing-in-ratio["การแบ่งตามอัตราส่วน"]
    ratio-change["อัตราส่วนเมื่อเพิ่มหรือลดปริมาณ"]
  end
  subgraph rule-of-three["บัญญัติไตรยางศ์"]
    direct-variation["การแปรผันตรง"]
    inverse-variation["การแปรผกผัน"]
    rule-of-three["บัญญัติไตรยางศ์ (ตรงและผกผัน)"]
  end
  subgraph percent["ร้อยละ เปอร์เซ็นต์"]
    percent-basics["ความหมายของร้อยละ และการแปลงเศษส่วน/ทศนิยม"]
    percent-of["การหาร้อยละของจำนวน และหาจำนวนเต็มย้อนกลับ"]
    percent-change["ร้อยละการเปลี่ยนแปลง (เพิ่ม/ลด)"]
    successive-percent["ร้อยละซ้อน (เปลี่ยนแปลงหลายครั้ง)"]
    profit-loss-discount["กำไร ขาดทุน ส่วนลด"]
    interest["ดอกเบี้ยเชิงเดี่ยวและทบต้น"]
  end
  subgraph geometry["เรขาคณิต"]
    angles-lines["มุม เส้นตรง และเส้นขนาน"]
    triangle-properties["สมบัติของสามเหลี่ยมและผลรวมมุมภายใน"]
    pythagorean-theorem["ทฤษฎีบทพีทาโกรัส"]
    special-right-triangles["สามเหลี่ยมมุมฉากพิเศษ (30-60-90, 45-45-90)"]
    similar-triangles["สามเหลี่ยมคล้าย"]
    polygons["รูปหลายเหลี่ยมและผลรวมมุมภายใน"]
    perimeter-area["เส้นรอบรูปและพื้นที่สามเหลี่ยม/สี่เหลี่ยม"]
    circles["วงกลม เส้นรอบวง พื้นที่ ส่วนโค้ง และเซกเตอร์"]
    circle-angles["มุมในวงกลมและเส้นสัมผัส"]
    composite-area["พื้นที่ส่วนที่แรเงาและรูปประกอบ"]
    volume-surface-area["ปริมาตรและพื้นที่ผิวทรงสามมิติ"]
  end
  subgraph analytic-geometry["เรขาคณิตวิเคราะห์"]
    coordinate-plane["ระบบพิกัดฉาก"]
    distance-midpoint["ระยะทางและจุดกึ่งกลางระหว่างสองจุด"]
    slope["ความชัน"]
    line-equation["สมการเส้นตรง"]
    parallel-perpendicular["เส้นขนานและเส้นตั้งฉาก"]
    line-intersection["จุดตัดของเส้นตรง"]
    area-on-plane["พื้นที่รูปหลายเหลี่ยมบนระนาบพิกัด"]
    circle-equation["สมการวงกลม"]
  end
  subgraph sequences-series["ลำดับ อนุกรม"]
    sequence-patterns["การหาแบบรูปของลำดับ"]
    arithmetic-sequence["ลำดับเลขคณิต"]
    geometric-sequence["ลำดับเรขาคณิต"]
    arithmetic-series["อนุกรมเลขคณิต"]
    geometric-series["อนุกรมเรขาคณิต"]
    infinite-geometric-series["อนุกรมเรขาคณิตอนันต์"]
  end
  subgraph statistics["สถิติ"]
    mean["ค่าเฉลี่ยเลขคณิต"]
    weighted-mean["ค่าเฉลี่ยถ่วงน้ำหนักและค่าเฉลี่ยรวม"]
    median-mode["มัธยฐานและฐานนิยม"]
    quartiles-percentiles["ควอร์ไทล์และเปอร์เซ็นไทล์"]
    dispersion["พิสัย ส่วนเบี่ยงเบนมาตรฐาน ความแปรปรวน"]
    data-transformation-effect["ผลของการเพิ่ม/ลด/คูณข้อมูลต่อค่าสถิติ"]
  end
  subgraph sets["เซต"]
    set-basics["เซต สมาชิก และสับเซต"]
    subset-count["จำนวนสับเซตและเพาเวอร์เซต"]
    set-operations["ยูเนียน อินเตอร์เซกชัน คอมพลีเมนต์ ผลต่าง"]
    venn-two-sets["แผนภาพเวนน์และจำนวนสมาชิก 2 เซต"]
    venn-three-sets["แผนภาพเวนน์และจำนวนสมาชิก 3 เซต"]
  end
  subgraph counting-probability["การนับ ความน่าจะเป็น"]
    counting-principles["หลักการบวกและหลักการคูณ"]
    factorial["แฟกทอเรียล"]
    permutations["การเรียงสับเปลี่ยน"]
    permutations-special["เรียงสับเปลี่ยนของซ้ำและเรียงเป็นวงกลม"]
    combinations["การจัดหมู่"]
    probability-basics["ความน่าจะเป็นพื้นฐาน"]
    probability-counting["ความน่าจะเป็นที่ใช้การจัดหมู่/เรียงสับเปลี่ยน"]
    complement-at-least["เหตุการณ์ตรงข้ามและโจทย์ "อย่างน้อย""]
    independent-events["เหตุการณ์อิสระ และการหยิบแบบคืน/ไม่คืน"]
  end
  subgraph mixtures["ของผสม"]
    concentration["ความเข้มข้นเป็นร้อยละ"]
    mixing-solutions["การผสมสารละลายสองชนิด"]
    dilution-evaporation["การเติมน้ำและการระเหย"]
    repeated-replacement["ตักออกแล้วเติมแทนซ้ำหลายครั้ง"]
    mixture-ratio["การผสมตามอัตราส่วน"]
  end
  subgraph motion["การเคลื่อนที่"]
    unit-conversion["การแปลงหน่วย (กม./ชม. ↔ ม./วินาที)"]
    speed-distance-time["ความเร็ว ระยะทาง เวลา"]
    average-speed["ความเร็วเฉลี่ย"]
    relative-motion["การเคลื่อนที่สวนทางและไล่ตาม"]
    train-problems["รถไฟผ่านคน สะพาน หรือรถไฟอีกขบวน"]
    stream-problems["ตามน้ำและทวนน้ำ"]
    circular-track["การวิ่งรอบสนาม"]
  end
  subgraph work["การทำงาน"]
    work-rate["อัตราการทำงาน (งานต่อหน่วยเวลา)"]
    man-days["คน-วัน และจำนวนคนกับเวลา"]
    combined-work["การทำงานร่วมกัน"]
    work-in-stages["เข้าหรือออกจากงานกลางคัน"]
    pipes-tanks["ท่อน้ำเข้าและท่อน้ำออก"]
  end
  subgraph tables-graphs["ตาราง กราฟ"]
    read-table["การอ่านข้อมูลจากตาราง"]
    read-bar-line["การอ่านกราฟแท่งและกราฟเส้น"]
    read-pie["การอ่านแผนภูมิวงกลม"]
    chart-percent-change["เปรียบเทียบและหาร้อยละการเปลี่ยนแปลงจากกราฟ"]
    chart-multi-source["รวมข้อมูลจากหลายตาราง/กราฟ"]
  end
  integer-operations --> fractions
  fractions --> decimals
  fractions --> estimation
  decimals --> estimation
  integer-operations --> odd-even-sign
  integer-operations --> divisibility
  divisibility --> gcd-lcm
  divisibility --> remainders
  integer-operations --> exponents
  fractions --> exponents
  exponents --> roots
  exponents --> unit-digit
  remainders --> unit-digit
  integer-operations --> absolute-value
  integer-operations --> algebraic-expressions
  fractions --> algebraic-expressions
  algebraic-expressions --> factoring
  algebraic-expressions --> linear-equations
  linear-equations --> word-problem-translation
  word-problem-translation --> age-problems
  linear-systems --> age-problems
  linear-equations --> linear-systems
  factoring --> quadratic-equations
  roots --> quadratic-equations
  exponents --> exponential-equations
  linear-equations --> exponential-equations
  linear-equations --> linear-inequalities
  quadratic-equations --> quadratic-inequalities
  linear-inequalities --> quadratic-inequalities
  absolute-value --> absolute-value-equations
  linear-inequalities --> absolute-value-equations
  fractions --> ratio-basics
  ratio-basics --> proportion
  linear-equations --> proportion
  ratio-basics --> multi-ratio
  gcd-lcm --> multi-ratio
  ratio-basics --> dividing-in-ratio
  proportion --> ratio-change
  word-problem-translation --> ratio-change
  proportion --> direct-variation
  direct-variation --> inverse-variation
  direct-variation --> rule-of-three
  inverse-variation --> rule-of-three
  fractions --> percent-basics
  decimals --> percent-basics
  percent-basics --> percent-of
  proportion --> percent-of
  percent-of --> percent-change
  percent-change --> successive-percent
  percent-change --> profit-loss-discount
  successive-percent --> interest
  exponents --> interest
  integer-operations --> angles-lines
  angles-lines --> triangle-properties
  roots --> pythagorean-theorem
  triangle-properties --> pythagorean-theorem
  pythagorean-theorem --> special-right-triangles
  triangle-properties --> similar-triangles
  proportion --> similar-triangles
  triangle-properties --> polygons
  triangle-properties --> perimeter-area
  fractions --> perimeter-area
  perimeter-area --> circles
  circles --> circle-angles
  angles-lines --> circle-angles
  perimeter-area --> composite-area
  circles --> composite-area
  perimeter-area --> volume-surface-area
  circles --> volume-surface-area
  integer-operations --> coordinate-plane
  coordinate-plane --> distance-midpoint
  pythagorean-theorem --> distance-midpoint
  coordinate-plane --> slope
  fractions --> slope
  slope --> line-equation
  linear-equations --> line-equation
  line-equation --> parallel-perpendicular
  line-equation --> line-intersection
  linear-systems --> line-intersection
  distance-midpoint --> area-on-plane
  line-intersection --> area-on-plane
  perimeter-area --> area-on-plane
  distance-midpoint --> circle-equation
  quadratic-equations --> circle-equation
  integer-operations --> sequence-patterns
  sequence-patterns --> arithmetic-sequence
  linear-equations --> arithmetic-sequence
  sequence-patterns --> geometric-sequence
  exponents --> geometric-sequence
  arithmetic-sequence --> arithmetic-series
  geometric-sequence --> geometric-series
  geometric-series --> infinite-geometric-series
  integer-operations --> mean
  fractions --> mean
  mean --> weighted-mean
  integer-operations --> median-mode
  median-mode --> quartiles-percentiles
  mean --> dispersion
  roots --> dispersion
  mean --> data-transformation-effect
  median-mode --> data-transformation-effect
  dispersion --> data-transformation-effect
  set-basics --> subset-count
  exponents --> subset-count
  set-basics --> set-operations
  set-operations --> venn-two-sets
  venn-two-sets --> venn-three-sets
  integer-operations --> counting-principles
  counting-principles --> factorial
  factorial --> permutations
  permutations --> permutations-special
  permutations --> combinations
  counting-principles --> probability-basics
  fractions --> probability-basics
  set-operations --> probability-basics
  probability-basics --> probability-counting
  combinations --> probability-counting
  probability-basics --> complement-at-least
  probability-basics --> independent-events
  percent-of --> concentration
  concentration --> mixing-solutions
  word-problem-translation --> mixing-solutions
  weighted-mean --> mixing-solutions
  concentration --> dilution-evaporation
  word-problem-translation --> dilution-evaporation
  mixing-solutions --> repeated-replacement
  successive-percent --> repeated-replacement
  multi-ratio --> mixture-ratio
  dividing-in-ratio --> mixture-ratio
  ratio-basics --> unit-conversion
  proportion --> speed-distance-time
  linear-equations --> speed-distance-time
  speed-distance-time --> average-speed
  speed-distance-time --> relative-motion
  word-problem-translation --> relative-motion
  relative-motion --> train-problems
  relative-motion --> stream-problems
  linear-systems --> stream-problems
  relative-motion --> circular-track
  gcd-lcm --> circular-track
  fractions --> work-rate
  inverse-variation --> work-rate
  rule-of-three --> man-days
  work-rate --> combined-work
  word-problem-translation --> combined-work
  combined-work --> work-in-stages
  combined-work --> pipes-tanks
  read-table --> read-bar-line
  read-table --> read-pie
  percent-of --> read-pie
  read-bar-line --> chart-percent-change
  percent-change --> chart-percent-change
  estimation --> chart-percent-change
  read-bar-line --> chart-multi-source
  read-pie --> chart-multi-source
  ratio-basics --> chart-multi-source
```
