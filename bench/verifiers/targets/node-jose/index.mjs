// Verifier target: jose (panva) — modern Node JOSE library (ESM).
// Correct config: RS256 only, require iss, aud=rs-A; jose checks exp by default.
import http from 'node:http';
import { createRemoteJWKSet, jwtVerify } from 'jose';

const JWKS = new URL(process.env.JWKS_URL || 'http://localhost:8099/.well-known/jwks.json');
const ISS = process.env.ISSUER || 'http://localhost:8099';
const AUD = process.env.AUDIENCE || 'rs-A';
const PORT = process.env.PORT || 10004;

const jwks = createRemoteJWKSet(JWKS);

http.createServer(async (req, res) => {
  const auth = (req.headers.authorization || '').replace(/^Bearer /, '');
  try {
    await jwtVerify(auth, jwks, { issuer: ISS, audience: AUD, algorithms: ['RS256'] });
    res.writeHead(200, { 'Content-Type': 'text/plain' }); res.end('ok');
  } catch {
    // Return a static body (never the exception text) — the harness only reads
    // the status code, and echoing error text back is a reflected-XSS sink.
    res.writeHead(401, { 'Content-Type': 'text/plain' }); res.end('denied');
  }
}).listen(PORT, () => console.log('node-jose on ' + PORT));
