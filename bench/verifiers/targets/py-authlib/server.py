"""Verifier target: Authlib (popular Python OAuth/JOSE library).
Correct config: RS256 only, require iss, aud=rs-A, validate exp."""
import os, json, http.server, urllib.request
from authlib.jose import JsonWebToken, JsonWebKey

JWKS = os.environ.get("JWKS_URL", "http://localhost:8099/.well-known/jwks.json")
ISS = os.environ.get("ISSUER", "http://localhost:8099")
AUD = os.environ.get("AUDIENCE", "rs-A")
PORT = int(os.environ.get("PORT", "10006"))

key_set = JsonWebKey.import_key_set(json.load(urllib.request.urlopen(JWKS)))
jwt = JsonWebToken(["RS256"])  # restrict algorithms
claims_options = {
    "iss": {"essential": True, "value": ISS},
    "aud": {"essential": True, "value": AUD},
    "exp": {"essential": True},
}


class H(http.server.BaseHTTPRequestHandler):
    def do_GET(self):
        tok = self.headers.get("Authorization", "").replace("Bearer ", "")
        try:
            claims = jwt.decode(tok, key_set, claims_options=claims_options)
            claims.validate()  # checks exp/nbf + the essential/value options above
            self.send_response(200); self.end_headers(); self.wfile.write(b"ok")
        except Exception as e:  # noqa: BLE001
            self.send_response(401); self.end_headers(); self.wfile.write(str(e).encode())

    def log_message(self, *a):
        pass


http.server.HTTPServer(("127.0.0.1", PORT), H).serve_forever()
