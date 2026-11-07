/**
 * Tiny MCP client over Streamable HTTP (JSON-RPC 2.0), just enough for e2e setup to
 * act as the Owner's Content Admin agent: initialize, then tools/call.
 * Handles both `application/json` and `text/event-stream` responses.
 */
const PROTOCOL_VERSION = "2025-06-18";

interface JsonRpcResponse {
  jsonrpc: "2.0";
  id?: number | string;
  result?: unknown;
  error?: { code: number; message: string; data?: unknown };
}

export interface ToolResult {
  content?: { type: string; text?: string }[];
  structuredContent?: unknown;
  isError?: boolean;
}

export class McpClient {
  private sessionId: string | undefined;
  private nextId = 1;

  constructor(
    private readonly url: string,
    private readonly token: string,
  ) {}

  private async post(body: object): Promise<JsonRpcResponse | undefined> {
    const headers: Record<string, string> = {
      "Content-Type": "application/json",
      Accept: "application/json, text/event-stream",
      Authorization: `Bearer ${this.token}`,
      "MCP-Protocol-Version": PROTOCOL_VERSION,
    };
    if (this.sessionId) headers["Mcp-Session-Id"] = this.sessionId;

    const res = await fetch(this.url, { method: "POST", headers, body: JSON.stringify(body) });
    const sid = res.headers.get("mcp-session-id");
    if (sid) this.sessionId = sid;
    if (res.status === 202 || res.status === 204) return undefined;
    const text = await res.text();
    if (!res.ok) throw new Error(`MCP HTTP ${res.status}: ${text}`);

    const id = (body as { id?: number }).id;
    if ((res.headers.get("content-type") ?? "").includes("text/event-stream")) {
      for (const line of text.split(/\r?\n/)) {
        if (!line.startsWith("data:")) continue;
        const msg = JSON.parse(line.slice(5).trim()) as JsonRpcResponse;
        if (msg.id === id) return msg;
      }
      throw new Error(`MCP: no response for id ${id} in event stream`);
    }
    return JSON.parse(text) as JsonRpcResponse;
  }

  private async rpc(method: string, params: object): Promise<unknown> {
    const msg = await this.post({ jsonrpc: "2.0", id: this.nextId++, method, params });
    if (!msg) throw new Error(`MCP ${method}: empty response`);
    if (msg.error) throw new Error(`MCP ${method}: ${msg.error.code} ${msg.error.message}`);
    return msg.result;
  }

  async connect(): Promise<void> {
    await this.rpc("initialize", {
      protocolVersion: PROTOCOL_VERSION,
      capabilities: {},
      clientInfo: { name: "tutor-platform-e2e", version: "0.0.0" },
    });
    await this.post({ jsonrpc: "2.0", method: "notifications/initialized" });
  }

  /** Calls a tool and returns its structured result (or parsed JSON text content). */
  async callTool<T = unknown>(name: string, args: object): Promise<T> {
    const result = (await this.rpc("tools/call", { name, arguments: args })) as ToolResult;
    const text = result.content?.map((c) => c.text ?? "").join("\n") ?? "";
    if (result.isError) throw new Error(`MCP tool ${name} failed: ${text}`);
    if (result.structuredContent !== undefined) return result.structuredContent as T;
    try {
      return JSON.parse(text) as T;
    } catch {
      return text as T;
    }
  }
}
