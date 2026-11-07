"use client";

import { useSearchParams } from "next/navigation";
import { useId, useState } from "react";
import { api, errorMessage } from "@/lib/api/client";
import { btn } from "@/lib/ui";

/** Local dev / e2e only (NEXT_PUBLIC_DEV_LOGIN=true; backend AUTH_DEV_LOGIN=true). */
export function DevLoginForm() {
  const sp = useSearchParams();
  const emailId = useId();
  const nameId = useId();
  const [email, setEmail] = useState("");
  const [name, setName] = useState("");
  const [pending, setPending] = useState(false);
  const [err, setErr] = useState<string | null>(null);

  async function submit(e: React.FormEvent) {
    e.preventDefault();
    setPending(true);
    setErr(null);
    const { response, error } = await api().POST("/auth/dev-login", {
      body: { email, ...(name ? { name } : {}) },
    });
    if (response.ok) {
      const next = sp.get("next");
      window.location.assign(next && next.startsWith("/") && !next.startsWith("//") ? next : "/");
      return;
    }
    setPending(false);
    setErr(errorMessage(error, "เข้าสู่ระบบไม่สำเร็จ"));
  }

  return (
    <form onSubmit={submit} className="mt-6 border-t border-dashed border-slate-300 pt-4" data-testid="dev-login-form">
      <p className="text-sm font-semibold text-amber-800">เข้าสู่ระบบสำหรับนักพัฒนา</p>
      <label htmlFor={emailId} className="mt-2 block text-sm font-medium">
        อีเมล
      </label>
      <input
        id={emailId}
        name="email"
        type="email"
        required
        autoComplete="email"
        value={email}
        onChange={(e) => setEmail(e.target.value)}
        className="mt-1 w-full rounded-lg border border-slate-300 px-3 py-2"
      />
      <label htmlFor={nameId} className="mt-2 block text-sm font-medium">
        ชื่อ (ไม่บังคับ)
      </label>
      <input
        id={nameId}
        name="name"
        value={name}
        onChange={(e) => setName(e.target.value)}
        className="mt-1 w-full rounded-lg border border-slate-300 px-3 py-2"
      />
      {err && (
        <p role="alert" className="mt-2 text-sm text-red-700">
          {err}
        </p>
      )}
      <button type="submit" disabled={pending} className={`${btn.secondary} mt-3 w-full`}>
        {pending ? "กำลังเข้าสู่ระบบ…" : "เข้าสู่ระบบ (dev)"}
      </button>
    </form>
  );
}
