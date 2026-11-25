/** Show the email dev-login form on /login (inlined at build time). */
export const isDevLoginEnabled = process.env.NEXT_PUBLIC_DEV_LOGIN === "true";
