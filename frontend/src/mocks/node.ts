import { setupServer } from "msw/node";
import { handlers } from "./handlers";

/** MSW server for Vitest (same handlers as mock mode). */
export const server = setupServer(...handlers);
