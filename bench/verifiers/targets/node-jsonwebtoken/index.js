// Verifier target: jsonwebtoken + jwks-rsa (the ubiquitous Node stack).
// Correct config: RS256 only, require iss, aud=rs-A; jsonwebtoken checks exp by default.
const http = require('http');
const jwt = require('jsonwebtoken');
const jwksClient = require('jwks-rsa');

const JWKS = process.env.JWKS_URL || 'http://localhost:8099/.well-known/jwks.json';
const ISS = process.env.ISSUER || 'http://localhost:8099';
const AUD = process.env.AUDIENCE || 'rs-A';
const PORT = process.env.PORT || 10003;

const client = jwksClient({ jwksUri: JWKS });
function getKey(header, cb) {
  client.getSigningKey(header.kid, (e, key) => {
    if (e) return cb(e);
    cb(null, key.getPublicKey());
  });
}

http.createServer((req, res) => {
  const auth = (req.headers.authorization || '').replace(/^Bearer /, '');
  jwt.verify(auth, getKey, { algorithms: ['RS256'], issuer: ISS, audience: AUD }, (err) => {
    if (err) { res.writeHead(401); res.end(err.message); return; }
    res.writeHead(200); res.end('ok');
  });
}).listen(PORT, () => console.log('node-jsonwebtoken on ' + PORT));
