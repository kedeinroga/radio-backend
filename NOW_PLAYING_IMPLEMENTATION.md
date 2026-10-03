# Implementación: "Sonando ahora" (Now Playing) vía metadata ICY

## Contexto

El sitio **rradio.online** es rechazado por Google AdSense por *"contenido sin valor / no original"*: el grueso del sitio son listados agregados del directorio **radio-browser** (mismo dataset que miles de sitios) y texto SEO plantilla. La estrategia para resolverlo **sin blog y manteniendo la estética minimalista** es convertir el directorio en contenido **único y propio** capturando el **metadata ICY de los streams** (la canción que suena), al estilo Last.fm.

**Hallazgo de arquitectura decisivo:** los usuarios **invitados reciben la URL directa del stream** (`internal/handlers/stream_session_handler.go:101-122`) y bypassean el proxy de audio; solo los autenticados pasan por `AudioProxyService`. Por tanto la captura pasiva en el proxy cubriría una minoría y **nada cuando Googlebot rastrea** (Google no escucha). **Por eso el mecanismo principal es un job de sondeo** que consulta las top-N estaciones populares de forma periódica e independiente de oyentes.

## Convenciones a respetar
Arquitectura hexagonal: `domain` (entidades + interfaces) → `services` → `handlers`; repositorios en `internal/repositories/postgres` con `database/sql` puro; jobs con `robfig/cron/v3`; logging `slog`; errores con `fmt.Errorf("...: %w", err)`; constructores `NewXxx`; `context.Context` como primer parámetro.

---

## 1. Migración — `migrations/000027_create_station_tracks.{up,down}.sql`
Tabla append-only para el historial de pistas (sigue la convención de `000021_create_stream_sessions`):

- `station_tracks`:
  - `id UUID PK DEFAULT gen_random_uuid()`
  - `station_id TEXT NOT NULL` (sin FK — las estaciones viven en Redis/radio-browser, igual que `stream_sessions`)
  - `raw_title TEXT NOT NULL`
  - `artist TEXT`
  - `title TEXT`
  - `source TEXT CHECK (source IN ('poll','proxy')) DEFAULT 'poll'`
  - `played_at TIMESTAMPTZ NOT NULL DEFAULT NOW()`
  - `created_at TIMESTAMPTZ DEFAULT NOW()`
- Índice: `idx_station_tracks_station_played ON station_tracks(station_id, played_at DESC)` — sirve tanto "now playing" (`LIMIT 1`) como "recent tracks".
- `.down.sql`: `DROP TABLE IF EXISTS station_tracks`.
- Dedup de títulos consecutivos: a nivel de aplicación (ver §4), **sin triggers** (consistente con la preferencia del repo).

## 2. Dominio — `internal/domain/now_playing.go`
- `StationTrack` struct (`ID`, `StationID`, `RawTitle`, `Artist`, `Title`, `Source`, `PlayedAt`).
- Interfaz `StationTrackRepository`:
  - `Insert(ctx, *StationTrack) error`
  - `GetLatest(ctx, stationID string) (*StationTrack, error)`
  - `GetRecent(ctx, stationID string, limit int) ([]*StationTrack, error)`
  - `DeleteOlderThan(ctx, age time.Duration) (int, error)`
- Función pura `ParseStreamTitle(raw string) (artist, title string)` — separa `"Artista - Canción"`; sin guion → todo va a `title`. **Unit-testeable** en `now_playing_test.go`.

## 3. Infra — `internal/infrastructure/icy/reader.go` (paquete nuevo, núcleo reutilizable)
`func FetchNowPlaying(ctx context.Context, client *http.Client, streamURL string) (string, error)`:
1. **Resolver playlists**: si la URL es `.pls`/`.m3u`/`.m3u8` o el `Content-Type` es de playlist, descargar, extraer la primera URL de stream real y reintentar. *(Hallazgo #1.)*
2. GET con header `Icy-MetaData: 1`.
3. Leer header de respuesta `icy-metaint` (N). Sin header → `ErrNoMetadata`.
4. Descartar N bytes de audio; leer 1 byte de longitud (`L`); leer `L*16` bytes del bloque de metadata.
5. Extraer `StreamTitle='...';` con regex; devolver el título.
6. Cerrar la conexión de inmediato. **Caps**: timeout de contexto, tope de bytes leídos (máx ~1× `metaint`, y `metaint` máx ~256KB) para acotar tiempo/ancho de banda.
- Errores tipados: `ErrNoMetadata`, `ErrPlaylistUnresolved`.

## 4. Servicio — `internal/services/now_playing_service.go`
- `NowPlayingService{ repo domain.StationTrackRepository, icyClient *http.Client, logger *slog.Logger }` + `NewNowPlayingService(...)`.
- `CaptureForStation(ctx, stationID, streamURL string) error`: `icy.FetchNowPlaying` → `domain.ParseStreamTitle` → comparar con `repo.GetLatest` (dedup: no insertar si el título no cambió) → `repo.Insert` con `source='poll'`.
- `GetNowPlaying(ctx, stationID)` y `GetRecentTracks(ctx, stationID, limit)` (con tope de `limit`).
- `CleanupOldTracks(ctx)` → `repo.DeleteOlderThan(cfg.RetentionDays)`.

## 5. Repositorio — `internal/repositories/postgres/station_track_repository.go`
`database/sql` puro siguiendo `stream_session_repository.go`: `QueryRowContext`/`QueryContext`/`Scan`, `sql.NullString` para `artist`/`title`, `sql.ErrNoRows → domain.ErrNotFound`, errores envueltos con `%w`.

## 6. Job de sondeo — `internal/jobs/now_playing_jobs.go`
- `NowPlayingJobs{ nowPlayingService, stationService, cfg, logger }`.
- `PollPopularStations()` (firma `func()`): contexto con timeout; top-N populares vía `stationService` (reutilizar `GetPopular`); recorrer con **worker pool acotado** (semáforo con canal buffered, `cfg.MaxConcurrency`) llamando `CaptureForStation`; loguear resumen (capturadas / sin-metadata / errores). *(Hallazgo #7.)*
- `CleanupOldTracks()` (firma `func()`): retención diaria.
- Registro en `cmd/server/jobs_init.go`: `"poll-now-playing"` cada `cfg.PollInterval` (def. 5 min), `"cleanup-old-tracks"` diario. **Requiere pasar `stationService` y `nowPlayingService` a `InitializeJobSystem`** (cambio de firma).

## 7. Handler + rutas — `internal/handlers/now_playing_handler.go`
- `GET /api/v1/stations/:id/now-playing` → última pista (o `204` si no hay).
- `GET /api/v1/stations/:id/recent-tracks?limit=10` → lista.
- DTOs JSON `snake_case` (`station_id`, `artist`, `title`, `raw_title`, `played_at`). Registrar en el grupo de rutas de estaciones (mismo middleware `X-Rradio-Secret`). `Cache-Control` corto para eficiencia de crawler.

## 8. Config + wiring
- `internal/config/config.go`: bloque `NowPlaying{ Enabled, PollInterval, TopStationsCount, MaxConcurrency, RetentionDays, FetchTimeout }` con defaults vía env.
- `cmd/server/main.go`: instanciar repo → service → handler → registrar rutas; `httpClient` dedicado para ICY (timeout corto).

---

## Hallazgos por resolver (backend)
1. **URLs playlist (.pls/.m3u)** — sin resolverlas, la captura falla para muchas estaciones. *(Incluido en §3.)*
2. **Invitados bypassean el proxy** — razón por la que el job es el mecanismo principal. *(Resuelto por diseño.)*
3. **`extractCountryCode` es un stub que devuelve `"XX"`** (`stream_session_service.go:384`) — geo-analítica incorrecta. Pre-existente.
4. **`validateAdImpression` sin implementar** (`stream_session_service.go:351-372`) — las impresiones de anuncios nunca se validan. Pre-existente; impacta ingresos/analítica.
5. **El proxy reenvía bytes ICY crudos al cliente** (`audio_proxy_service.go:125` pide `Icy-MetaData:1` pero no separa el metadata intercalado) — puede corromper audio en algunos clientes o desperdiciar bytes. Decidir: no pedir metadata en el proxy, o parsearlo y separarlo (habilitaría la captura pasiva fase 2).
6. **Crecimiento de `station_tracks`** — retención por job ahora; particionado más adelante (patrón `station_plays`).
7. **Tormenta de conexiones salientes** — acotada con worker pool + timeouts. *(Incluido en §6.)*
8. **Fase 2 opcional — enriquecimiento pasivo en el proxy**: reutilizar `icy.FetchNowPlaying`/parser dentro de `ProxyStream` para capturar `source='proxy'` en tiempo real de oyentes autenticados.

## Verificación
- Unit tests: `ParseStreamTitle` (dominio) y `icy.FetchNowPlaying` contra un `httptest.Server` que emita `icy-metaint` + `StreamTitle`.
- `go test ./...` y targets del `Makefile` (golangci-lint).
- `go run ./cmd/migrate -action up`; ejecutar el job manualmente una vez; `curl` a `/now-playing` y `/recent-tracks` con `X-Rradio-Secret`.

## Orden de ejecución
migración → dominio/parser → ICY reader → repo → service → job → handler/rutas → wiring → tests.
