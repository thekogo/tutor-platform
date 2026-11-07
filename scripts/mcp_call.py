"""Call a tool on the platform's MCP server (streamable HTTP, JSON responses).

Usage:
  python3 scripts/mcp_call.py <tool> [args.json | -]      # args from file or stdin
  python3 scripts/mcp_call.py <tool> --json '{"course": "cu-best-math"}'

Env: MCP_URL (default http://localhost:8080/mcp), MCP_TOKEN (default: read from ./.env).
Prints the tool result (structured content if present, else text) as JSON.
Exit code 1 if the tool reports an error.
"""
import json
import os
import sys
import urllib.error
import urllib.request


def token():
    if os.environ.get("MCP_TOKEN"):
        return os.environ["MCP_TOKEN"]
    env = os.path.join(os.path.dirname(os.path.abspath(__file__)), "..", ".env")
    with open(env, encoding="utf-8") as f:
        for line in f:
            if line.startswith("MCP_TOKEN="):
                return line.split("=", 1)[1].strip()
    sys.exit("MCP_TOKEN not set and not found in .env")


def call(tool, args):
    body = json.dumps({"jsonrpc": "2.0", "id": 1, "method": "tools/call",
                       "params": {"name": tool, "arguments": args}}).encode()
    req = urllib.request.Request(
        os.environ.get("MCP_URL", "http://localhost:8080/mcp"), data=body, method="POST",
        headers={"Authorization": f"Bearer {token()}", "Content-Type": "application/json",
                 "Accept": "application/json, text/event-stream"})
    try:
        with urllib.request.urlopen(req, timeout=120) as r:
            resp = json.loads(r.read())
    except urllib.error.HTTPError as e:
        sys.exit(f"HTTP {e.code}: {e.read().decode(errors='replace')}")
    if "error" in resp:
        print(json.dumps(resp["error"], ensure_ascii=False, indent=2))
        sys.exit(1)
    res = resp["result"]
    out = res.get("structuredContent")
    if out is None:
        out = "\n".join(c.get("text", "") for c in res.get("content", []))
    print(json.dumps(out, ensure_ascii=False, indent=2) if not isinstance(out, str) else out)
    if res.get("isError"):
        sys.exit(1)


def main():
    if len(sys.argv) < 2:
        sys.exit(__doc__)
    tool, rest = sys.argv[1], sys.argv[2:]
    if rest[:1] == ["--json"]:
        args = json.loads(rest[1])
    elif rest and rest[0] != "-":
        with open(rest[0], encoding="utf-8") as f:
            args = json.load(f)
    elif rest:
        args = json.load(sys.stdin)
    else:
        args = {}
    call(tool, args)


if __name__ == "__main__":
    main()
