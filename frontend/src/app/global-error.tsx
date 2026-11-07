"use client";

export default function GlobalError({ retry }: { error: unknown; retry: () => void }) {
  return (
    <html lang="th">
      <body style={{ fontFamily: "system-ui, sans-serif", padding: "2rem" }}>
        <h1>เกิดข้อผิดพลาดร้ายแรง</h1>
        <p>กรุณาลองใหม่อีกครั้ง</p>
        <button type="button" onClick={() => retry()}>
          ลองใหม่
        </button>
      </body>
    </html>
  );
}
