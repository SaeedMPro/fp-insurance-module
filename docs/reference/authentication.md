# Authentication

Two schemes, for two kinds of caller.

## Interactive users — JWT

```
POST /api/v1/auth/login
{ "username": "saeed.mazahery", "password": "Employee123!" }
```

Returns a signed token and the user record. Send it on every subsequent request:

```
Authorization: Bearer <token>
```

| | |
| --- | --- |
| Algorithm | HS256 |
| Claims carried | user id, username, role |
| Default lifetime | 8 hours (`JWT_TTL`) |
| Passwords | bcrypt; never returned by any endpoint |

The token carries the role, so the API can authorise a request without a database
round trip for the user on every call.

`GET /api/v1/auth/me` returns the current user — used by the frontend on load to
restore a session.

### What the frontend does with it

Stores the token, attaches it with a request interceptor, and logs out on any
`401`. Downloads go through the same client rather than a plain link, because
document downloads need the header too.

### Failure modes

| Response | Meaning |
| --- | --- |
| 401 `invalid username or password` | Wrong credentials, or the account is inactive |
| 401 `missing bearer token` | Header absent |
| 401 `invalid or expired token` | Signature failed or the token has expired |
| 403 `insufficient role for this action` | Authenticated, but the role is not allowed here |

## The parent system — API key

The integration endpoints use a header instead of a token:

```
X-API-Key: <key>
```

The value is hashed with SHA-256 and compared against active rows in
`integration_api_keys`; the plaintext key is never stored. These endpoints carry
no role and are outside the user model, because the caller is a system.

| Response | Meaning |
| --- | --- |
| 401 `missing X-API-Key header` | Header absent |
| 401 `invalid API key` | No matching active key |

The demo key is `dev-integration-key`. Being in a seed file, it is for
development only.

## Configuration that matters in production

The service **refuses to start** in production on a default or too-short
`JWT_SECRET`, or without an explicit `DATABASE_URL`. Misconfiguration of exactly
this kind is otherwise discovered by being exploited.

See [Configuration](configuration).
