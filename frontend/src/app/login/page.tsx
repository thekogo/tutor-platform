import { Suspense } from "react";
import { BrandLogo } from "@/components/BrandMark";
import { DevLoginForm } from "@/components/DevLoginForm";
import { GoogleLoginLink, LoginNotices } from "@/components/LoginNotices";
import { BRAND } from "@/lib/brand";
import { isDevLoginEnabled } from "@/lib/env";
import { T } from "@/lib/terms";

export const metadata = { title: "เข้าสู่ระบบ" };

export default function LoginPage() {
  return (
    <main data-testid="login-page" className="flex flex-1 items-center justify-center bg-gradient-to-b from-blue-50 to-slate-50 p-4">
      <div className="w-full max-w-md rounded-2xl border border-slate-200 bg-white p-8 shadow-sm">
        <h1 className="flex items-center gap-3 text-2xl font-bold text-blue-900">
          <BrandLogo className="size-10" />
          {BRAND.name}
        </h1>
        <p className="mt-2 text-slate-600">
          ติวด้วยกราฟความรู้ ระบบจะหา{T.rootCause}และแนะนำว่าควรเรียนอะไรต่อ
        </p>
        {/* Reads ?invite= and ?error= in the browser (static page). */}
        <Suspense fallback={<GoogleLoginLink />}>
          <LoginNotices />
        </Suspense>
        {isDevLoginEnabled && (
          <Suspense fallback={null}>
            <DevLoginForm />
          </Suspense>
        )}
      </div>
    </main>
  );
}
