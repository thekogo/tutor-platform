import { NextResponse, type NextRequest } from "next/server";

const SESSION_COOKIE = "tp_session";
const PUBLIC_PATHS = ["/login"];

/**
 * 1. /api/* → ${BACKEND_URL}/api/* (runtime rewrite, keeps the session cookie
 *    first-party). Skipped in mock mode, where src/app/api/[...path] answers.
 * 2. Pages: no session cookie → /login?next=… (the backend remains the
 *    authority; an invalid cookie still yields 401 → /login).
 */
export function proxy(request: NextRequest) {
  const { pathname, search } = request.nextUrl;

  if (pathname === "/api" || pathname.startsWith("/api/")) {
    if (process.env.NEXT_PUBLIC_API_MOCKING === "enabled") return NextResponse.next();
    const backend = (process.env.BACKEND_URL ?? "http://localhost:8080").replace(/\/+$/, "");
    return NextResponse.rewrite(new URL(`${pathname}${search}`, backend));
  }

  const isPublic = PUBLIC_PATHS.some((p) => pathname === p || pathname.startsWith(`${p}/`));
  if (!isPublic && !request.cookies.has(SESSION_COOKIE)) {
    const url = request.nextUrl.clone();
    url.pathname = "/login";
    url.search = pathname === "/" ? "" : `?next=${encodeURIComponent(pathname + search)}`;
    return NextResponse.redirect(url);
  }
  return NextResponse.next();
}

export const config = {
  matcher: ["/((?!_next/static|_next/image|favicon.ico|robots.txt|.*\\.(?:svg|png|jpg|jpeg|gif|webp|ico|woff2?)$).*)"],
};
