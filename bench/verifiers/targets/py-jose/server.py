"""Verifier target: python-jose (widely used, incl. by FastAPI tutorials).
Correct config: RS256 only, require iss, aud=rs-A; python-jose checks exp."""
import os, json, http.server, urllib.request
from jose import jwt

JWKS = os.environ.get("JWKS_URL", "http://localhost:8099/.well-known/jwks.json")
ISS = os.environ.get("ISSUER", "http://localhost:8099")
AUD = os.environ.get("AUDIENCE", "rs-A")
PORT = int(os.environ.get("PORT", "10008"))

jwks = json.load(urllib.request.urlopen(JWKS))


class H(http.server.BaseHTTPRequestHandler):
    def do_GET(self):
        tok = self.headers.get("Authorization", "").replace("Bearer ", "")
        try:
            jwt.decode(tok, jwks, algorithms=["RS256"], audience=AUD, issuer=ISS,
                       options={"require_exp": True})
            self.send_response(200); self.end_headers(); self.wfile.write(b"ok")
        except Exception as e:  # noqa: BLE001
            self.send_response(401); self.end_headers(); self.wfile.write(str(e).encode())

    def log_message(self, *a):
        pass


http.server.HTTPServer(("127.0.0.1", PORT), H).serve_forever()
