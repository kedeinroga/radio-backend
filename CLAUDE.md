# CLAUDE.md — radio-backend

Go (1.24) + Gin REST API for a radio streaming app: JWT auth, station search/playback
(via Radio Browser), favorites, analytics, i18n, an ad/campaign subsystem, premium
subscriptions (Stripe), and admin tooling. Deployed to Cloud Run.

## Commands

```bash
make run                # swag generate + go run (dev). Reads .env (SERVER_ENV=development)
make dev                # live reload (air)
make test               # unit tests + coverage
make test-quick         # tests, no coverage (faster)
make lint               # golangci-lint
make fmt                # gofmt
make swagger-generate   # regenerate docs/ from annotations (also runs inside `make run`)
make migrate-up         # apply DB migrations
make migrate-create NAME=create_x_table
```

- Server runs on `:8080` (`SERVER_PORT`). Health: `GET /health`.
- Requires Postgres + Redis (see `docker-compose.yml`, `.env.example`).

## Architecture (layered, dependencies point inward)

```
cmd/server/main.go         → wiring / DI (constructs services, handlers, routers)
internal/domain            → entities, domain errors, repository INTERFACES (no deps)
internal/services          → business logic; returns domain types or *domain.DomainError
internal/repositories      → impls: postgres/, redis/, radiobrowser/
internal/handlers          → HTTP handlers + Swagger annotations + response DTOs
internal/server            → routers (router.go + *_routes.go), docs.go (Scalar)
internal/middleware        → auth, rate limiting, CORS, security headers, i18n, shared-secret
internal/infrastructure    → cache, crypto, database, icy, jwt, logger
internal/i18n              → language parsing/support (es, en, fr, de)
internal/jobs              → background jobs
```

Routes are registered across **5 files**: `router.go` (core/auth/stations/analytics/
favorites/seo/translations/admin), `ad_routes.go`, `now_playing_routes.go`,
`premium_routes.go`, `stream_routes.go`.

## API docs (Scalar)

- UI served at **`GET /docs`**, spec at `GET /docs/openapi.json` — see `internal/server/docs.go`.
- **Dev only**: `registerDocs()` returns early when `isProduction`. Nothing is exposed in prod.
- Generated from swaggo annotations into `docs/` (`docs.go`, `swagger.json`, `swagger.yaml`).
  `make run` regenerates them on every start, so new annotated endpoints appear automatically.
- `/docs` overrides the global CSP to allow the Scalar CDN (`cdn.jsdelivr.net`).

## Response & error conventions (IMPORTANT for annotations)

Helpers live in `internal/handlers/response.go`; shared DTOs in `internal/handlers/dto_responses.go`.

Three distinct error shapes exist — document each faithfully:

| Shape | Produced by | DTO to use in `@Failure` |
|---|---|---|
| `{"error":{"code","message","field?}}` | handlers via `RespondWithError` / `RespondWithDomainError` | `ErrorResponse` |
| `{"error":"text"}` (flat) | auth middlewares (`auth.go`, `ad_auth.go`), `SharedSecretAuth` | `SimpleErrorResponse` |
| `{"error":"text"}` (flat) | ads/tracking handlers that call `c.JSON` directly | `SimpleErrorResponse` |

Rules of thumb:
- `401`/`403` on routes behind `Required()`/`AdminOnly()`/`SharedSecretAuth` → `SimpleErrorResponse`
  (middleware aborts before the handler; any in-handler 401 check is unreachable).
- Handler-level `400/404/409/500` → `ErrorResponse`.
- Some endpoints wrap success as `{"success":true,"data":...}` (analytics, translations,
  guest-rate-limit) — these have dedicated DTOs, do NOT use the bare type.
- `domain.DomainError` codes map to HTTP status in `mapDomainErrorToHTTPStatus` (response.go):
  e.g. `USER_ALREADY_EXISTS`→409, `*_NOT_FOUND`→404, `INVALID_CREDENTIALS`→401,
  `VALIDATION_ERROR`→400. Document the status the mapping actually yields.

When adding/changing an endpoint:
1. Add swaggo annotations (`@Summary`, `@Tags`, `@Param`, `@Success`, `@Failure`, `@Router`).
2. Trace every `c.JSON` / `RespondWith*` in the body and document each status with the
   correct DTO above. Create a DTO in the handler file if the shape is new — avoid
   `map[string]interface{}` / `gin.H` (they render as opaque objects).
3. `example:` tags on a shared DTO show the same example everywhere it's referenced — keep
   them neutral/illustrative.

## Conventions

- Errors: business logic returns `*domain.DomainError`; handlers translate via
  `RespondWithDomainError`. New shared error codes go in `internal/domain/errors.go`.
- Auth: RSA-signed JWT (keys in `keys/`, `make generate-keys`). `BearerAuth` security scheme.
  Public station/SEO routes also require the `X-Rradio-Secret` header (`SharedSecret`).
- i18n: language detected by middleware; passed through services via `lang` param.
- Comments/identifiers mix Spanish and English — match the surrounding file.
- After editing handler annotations, run `make swagger-generate` and confirm a clean build.

## Don't

- Don't expose `/docs` (or any spec) in production — keep the `isProduction` gate.
- Don't reintroduce a `success` field into `ErrorResponse` — the real error envelope has none.
- Don't commit/push unless asked.
