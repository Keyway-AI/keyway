"""MCP-server-class verifier target: FastMCP's JWTVerifier — the batteries-included
implementation of the MCP authorization spec's OAuth resource-server checks (verify
the RS256 signature against the AS JWKS, require the issuer, and require the token's
`aud` to be THIS server's resource identifier). This is the component that decides the
SoK's headline MCP threat: a token minted for another resource (wrong/absent `aud`)
must be rejected, never passed through.

As with the library targets, we invoke the SDK's real verifier behind a tiny HTTP
handler and score 200 (accepted) / 401 (rejected), isolating the auth decision from the
MCP JSON-RPC protocol layer. A single persistent event loop serves the async verifier.
"""
import asyncio
import http.server
import os
import threading

from fastmcp.server.auth.providers.jwt import JWTVerifier

JWKS = os.environ.get("JWKS_URL", "http://localhost:8099/.well-known/jwks.json")
ISS = os.environ.get("ISSUER", "http://localhost:8099")
AUD = os.environ.get("AUDIENCE", "mcp-rs-A")
PORT = int(os.environ.get("PORT", "10011"))

# ssrf_safe=False: the eval's JWKS is a localhost test issuer, which FastMCP's SSRF
# guard would otherwise refuse. This only affects where keys are fetched from, not the
# verification logic under test.
verifier = JWTVerifier(jwks_uri=JWKS, issuer=ISS, audience=AUD, algorithm="RS256", ssrf_safe=False)

_loop = asyncio.new_event_loop()
threading.Thread(target=_loop.run_forever, daemon=True).start()


def verify(token: str):
    fut = asyncio.run_coroutine_threadsafe(verifier.verify_token(token), _loop)
    return fut.result(timeout=5)


class H(http.server.BaseHTTPRequestHandler):
    def do_GET(self):
        tok = self.headers.get("Authorization", "").replace("Bearer ", "")
        try:
            access = verify(tok)
        except Exception:  # noqa: BLE001 -- any verifier error is a rejection
            access = None
        if access is not None:
            self.send_response(200)
            self.end_headers()
            self.wfile.write(b"ok")
        else:
            self.send_response(401)
            self.end_headers()
            self.wfile.write(b"denied")

    def log_message(self, *a):
        pass


http.server.HTTPServer(("127.0.0.1", PORT), H).serve_forever()
