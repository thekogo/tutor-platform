/**
 * Deterministic UUID-shaped ids for mock data, derived from a namespace + slug,
 * so the server process and tests agree on ids without shared state.
 */
function fnv1a(input: string, seed: number): number {
  let h = seed >>> 0;
  for (let i = 0; i < input.length; i++) {
    h ^= input.charCodeAt(i);
    h = Math.imul(h, 0x01000193) >>> 0;
  }
  return h >>> 0;
}

export function mockId(namespace: string, slug: string): string {
  const key = `${namespace}:${slug}`;
  const hex = [0x811c9dc5, 0x1234567, 0x9e3779b9, 0x7f4a7c15]
    .map((seed) => fnv1a(key, seed).toString(16).padStart(8, "0"))
    .join("");
  // Shape as RFC 4122 v4-looking UUID.
  return `${hex.slice(0, 8)}-${hex.slice(8, 12)}-4${hex.slice(13, 16)}-a${hex.slice(17, 20)}-${hex.slice(20, 32)}`;
}

let counter = 0;
/** Fresh id for runtime-created records (sessions, flags, invites). */
export function newId(namespace: string): string {
  counter += 1;
  return mockId(namespace, `${Date.now()}-${counter}-${Math.random()}`);
}
