"""MCP-server-class verifier target: the official MCP Python SDK (`mcp`) bearer-auth
middleware in its DEFAULT configuration.

The base SDK ships no JWT verifier — it is bring-your-own — and its BearerAuthBackend
binds the token to this resource (RFC 8707 audience binding, the MCP spec's #1 rule)
only when `resource_server_url` is set, which defaults to None. This target stands up
the SDK's real middleware with a competent, standard BYO verifier (validates the RS256
signature, issuer, and expiry via PyJWT, and — following the MCP architecture — leaves
resource/audience binding to the framework) in that default config.

It measures: does an MCP server built on the reference SDK, wired the documented way,
reject a token minted for a different resource? With `resource_server_url` at its
default (None), the middleware performs no resource check, so a wrong-/absent-audience
token authenticates — token passthrough (MCP-01/02). Scored 200 (accepted) / 401.
"""
import os

import jwt
import uvicorn
from jwt import PyJWKClient
from mcp.server.auth.middleware.bearer_auth import BearerAuthBackend
from mcp.server.auth.provider import AccessToken, TokenVerifier
from starlette.applications import Starlette
from starlette.middleware import Middleware
from starlette.middleware.authentication import AuthenticationMiddleware
from starlette.requests import HTTPConnection
from starlette.responses import PlainTextResponse
from starlette.routing import Route

JWKS = os.environ.get("JWKS_URL", "http://localhost:8099/.well-known/jwks.json")
ISS = os.environ.get("ISSUER", "http://localhost:8099")
AUD = os.environ.get("AUDIENCE", "mcp-sdk-rs-A")
PORT = int(os.environ.get("PORT", "10012"))

_jwks = PyJWKClient(JWKS)


class Verifier(TokenVerifier):
    """A standard BYO verifier: RS256 signature + issuer + expiry (the universal
    JWT basics). It does NOT itself check the audience — per the MCP architecture
    that resource binding is the framework's job (BearerAuthBackend.resource_server_url).
    It still surfaces the token's aud as AccessToken.resource so a hardened config
    (resource_server_url set) would bind correctly."""

    async def verify_token(self, token: str):  # type: ignore[override]
        try:
            key = _jwks.get_signing_key_from_jwt(token).key
            claims = jwt.decode(
                token, key, algorithms=["RS256"], issuer=ISS,
                options={"verify_aud": False},  # resource binding delegated to the SDK
            )
        except Exception:  # noqa: BLE001 -- any failure is a rejection
            return None
        return AccessToken(
            token=token,
            client_id=str(claims.get("sub", "unknown")),
            scopes=[],
            expires_at=claims.get("exp"),
            resource=claims.get("aud"),
        )


# resource_server_url=None is the SDK default: no RFC 8707 resource binding.
backend = BearerAuthBackend(Verifier())


async def protected(request):
    conn = HTTPConnection(request.scope)
    if not conn.user.is_authenticated:
        return PlainTextResponse("denied", status_code=401)
    return PlainTextResponse("ok")


def _on_error(_conn, _exc):
    return PlainTextResponse("denied", status_code=401)


app = Starlette(
    routes=[Route("/", protected)],
    middleware=[Middleware(AuthenticationMiddleware, backend=backend, on_error=_on_error)],
)

if __name__ == "__main__":
    uvicorn.run(app, host="127.0.0.1", port=PORT, log_level="critical")
