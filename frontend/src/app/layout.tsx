import type { Metadata, Viewport } from "next";
import { IBM_Plex_Sans_Thai } from "next/font/google";
import "katex/dist/katex.min.css";
import "./globals.css";

const thai = IBM_Plex_Sans_Thai({
  variable: "--font-thai",
  subsets: ["thai", "latin"],
  weight: ["400", "500", "600", "700"],
  display: "swap",
});

export const metadata: Metadata = {
  title: { default: "ติวเตอร์ CU-BEST", template: "%s · ติวเตอร์ CU-BEST" },
  description: "ติวคณิตศาสตร์ CU-BEST ด้วยกราฟความรู้ ค้นหาจุดอ่อนและแนะนำสิ่งที่ควรเรียนต่อ",
};

export const viewport: Viewport = { width: "device-width", initialScale: 1 };

export default function RootLayout({ children }: { children: React.ReactNode }) {
  return (
    <html lang="th" className={thai.variable}>
      <body className="flex min-h-dvh flex-col font-sans">{children}</body>
    </html>
  );
}
