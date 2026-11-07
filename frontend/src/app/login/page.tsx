import { Suspense } from "react";
import { DevLoginForm } from "@/components/DevLoginForm";
import { isApiMocking, isDevLoginEnabled } from "@/lib/env";
import { T } from "@/lib/terms";

export const metadata = { title: "เข้าสู่ระบบ" };

export default async function LoginPage({ searchParams }: { searchParams: Promise<Record<string, string | string[] | undefined>> }) {
  const sp = await searchParams;
  const invite = typeof sp.invite === "string" ? sp.invite : undefined;
  const error = typeof sp.error === "string" ? sp.error : undefined;
  const googleHref = `/api/auth/google/login${invite ? `?invite=${encodeURIComponent(invite)}` : ""}`;

  return (
    <main data-testid="login-page" className="flex flex-1 items-center justify-center bg-gradient-to-b from-blue-50 to-slate-50 p-4">
      <div className="w-full max-w-md rounded-2xl border border-slate-200 bg-white p-8 shadow-sm">
        <h1 className="text-2xl font-bold text-blue-900">ติวเตอร์ CU-BEST</h1>
        <p className="mt-2 text-slate-600">
          ติวคณิตศาสตร์ด้วยกราฟความรู้ ระบบจะหา{T.rootCause}และแนะนำว่าควรเรียนอะไรต่อ
        </p>
        {invite && (
          <p className="mt-4 rounded-lg bg-blue-50 p-3 text-sm text-blue-900" role="status">
            คุณได้รับ{T.invite} เข้าสู่ระบบด้วย Google เพื่อเริ่มใช้งาน
          </p>
        )}
        {error && (
          <p className="mt-4 rounded-lg bg-red-50 p-3 text-sm text-red-800" role="alert">
            เข้าสู่ระบบไม่สำเร็จ: ระบบนี้ใช้ได้เฉพาะผู้ที่ได้รับ{T.invite}
          </p>
        )}
        {/* Plain link: a full navigation so the backend can 302 to Google. */}
        <a
          href={googleHref}
          data-testid="google-login"
          className="mt-6 flex w-full items-center justify-center gap-3 rounded-lg border border-slate-300 bg-white px-4 py-2.5 font-medium text-slate-800 hover:bg-slate-50"
        >
          <svg aria-hidden="true" width="18" height="18" viewBox="0 0 48 48">
            <path fill="#FFC107" d="M43.6 20.5H42V20H24v8h11.3C33.7 32.7 29.2 36 24 36c-6.6 0-12-5.4-12-12s5.4-12 12-12c3.1 0 5.8 1.2 7.9 3.1l5.7-5.7C34 6.1 29.3 4 24 4 12.9 4 4 12.9 4 24s8.9 20 20 20 20-8.9 20-20c0-1.3-.1-2.4-.4-3.5z" />
            <path fill="#FF3D00" d="M6.3 14.7l6.6 4.8C14.7 15.1 19 12 24 12c3.1 0 5.8 1.2 7.9 3.1l5.7-5.7C34 6.1 29.3 4 24 4 16.3 4 9.7 8.3 6.3 14.7z" />
            <path fill="#4CAF50" d="M24 44c5.2 0 9.9-2 13.4-5.2l-6.2-5.2C29.2 35.1 26.7 36 24 36c-5.2 0-9.6-3.3-11.3-8l-6.5 5C9.5 39.6 16.2 44 24 44z" />
            <path fill="#1976D2" d="M43.6 20.5H42V20H24v8h11.3c-.8 2.2-2.2 4.2-4.1 5.6l6.2 5.2C37 39.2 44 34 44 24c0-1.3-.1-2.4-.4-3.5z" />
          </svg>
          เข้าสู่ระบบด้วย Google
        </a>
        {isApiMocking && (
          <p className="mt-2 text-center text-xs text-amber-700">โหมดจำลอง: ปุ่ม Google จะเข้าสู่ระบบเป็น{T.owner}ทันที</p>
        )}
        {isDevLoginEnabled && (
          <Suspense fallback={null}>
            <DevLoginForm />
          </Suspense>
        )}
      </div>
    </main>
  );
}
