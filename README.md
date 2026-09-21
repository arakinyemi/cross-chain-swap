# Swap — non-KYC cross-chain swap interface

A FixedFloat-style swap interface: pick a pair, paste a payout address, send coins
to the deposit address you get back, watch the order finish. No account, no sign-up,
no identity documents, nothing to log into. An order is a URL.

The Bitnob-powered exchange engine (deposit addresses, routing, trading, payout)
runs behind it in the same binary.

```text
/            exchange form   — pair pickers, live rate, recipient address
/order/{id}  order page      — deposit address + QR, status timeline, summary
```

## Stack

| Layer | What |
| --- | --- |
| Interface | React 19 + TypeScript, built with Vite, in `web/` |
| Server | Go — API, Bitnob integration, QR rendering, serves the built bundle |

The bundle is embedded with `go:embed`, so a release is a single binary with no
static files to deploy beside it.

## Try it in one command

Demo mode stubs the exchange and keeps orders in memory, so the interface runs with
no credentials, no database and no webhook tunnel. Orders walk the real status
machine on a timer.

```bash
npm --prefix web ci && npm --prefix web run build
DEMO=1 go run ./cmd/server
# open http://localhost:8080
```

No real funds move in demo mode.

## Frontend development

Run Vite against the Go API for hot reload. Vite proxies `/assets`, `/quote`,
`/swap`, `/prices` and `/qr` to port 8080, so both halves stay live:

```bash
DEMO=1 go run ./cmd/server      # terminal 1 — API on :8080
npm --prefix web run dev        # terminal 2 — interface on :5173
```

| Command | Purpose |
| --- | --- |
| `npm --prefix web run dev` | Vite dev server with API proxy |
| `npm --prefix web run build` | Type-check and build into `internal/ui/dist` |
| `npm --prefix web run typecheck` | Types only |

`internal/ui/dist` is generated and gitignored. `go build` still succeeds without
it — the server then serves a page telling you to build the bundle.

## Run it for real

### Prerequisites

- Go 1.22+
- PostgreSQL
- Bitnob API credentials
- ngrok or a similar tunnel for webhooks

### Environment

```bash
cp .env.example .env
```

| Key | Purpose |
| --- | --- |
| `BITNOB_CLIENT_ID` | Bitnob API client ID |
| `BITNOB_CLIENT_SECRET` | Bitnob API secret, used for HMAC signing |
| `BITNOB_WEBHOOK_SECRET` | Verifies inbound Bitnob webhooks |
| `DATABASE_URL` | PostgreSQL connection string |
| `PORT` | Listen port (default `8080`) |
| `SWAP_FEE_USDT` | Flat service fee (default `2`) |
| `MIN_SWAP_AMOUNT` | Minimum order size (default `10`) |
| `DEMO` | `1` to run the interface without Bitnob or Postgres |

`BITNOB_BASE_URL` is listed in `.env.example` for completeness; the config
hardcodes `https://api.bitnob.com`.

### Database

```bash
psql "$DATABASE_URL" -f migrations/001_init.sql
```

### Start

```bash
go run ./cmd/server
```

On startup the server connects to PostgreSQL and calls `POST /api/whoami` on Bitnob.
If credential verification fails, startup exits.

### Webhooks

```bash
ngrok http 8080
```

Point the Bitnob webhook at `https://your-domain.ngrok-free.app/webhooks/bitnob`.
Without it an order sits at `awaiting_deposit` forever, because the deposit webhook
is what starts the trades.

## What the interface talks to

| Route | Purpose |
| --- | --- |
| `GET /assets` | Asset + network catalog, minimum, fee — drives the pickers |
| `POST /quote` | Prices a pair with no side effects, for the live "You get" line |
| `POST /swap` | Creates the order and returns the deposit address |
| `GET /swap/{id}` | Order status, polled by the order page every 5s |
| `GET /qr?data=` | PNG QR code for a deposit address |
| `GET /prices` | Raw Bitnob price board |
| `POST /webhooks/bitnob` | `address.deposit.confirmed`, `transfer.success` |

## Supported pairs

Defined in `internal/swap/catalog.go`, which is the single source of truth for both
the pickers and the API's validation — the interface reads it from `GET /assets`
rather than hardcoding a list:

| Asset | Networks |
| --- | --- |
| USDT | Ethereum, Tron, BSC, Polygon, Solana, Arbitrum |
| USDC | Ethereum, Solana, Polygon, BSC, Stellar, Arbitrum |
| BTC | Bitcoin |

## Layout

```text
web/src            React interface
  api.ts           typed client for the Go API
  types.ts         mirrors the server's JSON
  components/      ExchangeForm, OrderPage, AssetPicker, StatusTimeline, CopyField
  hooks/           useCatalog, useQuote, useOrder, useRoute
cmd/server         HTTP routes and handlers
internal/ui        embeds the built bundle, renders QR codes
internal/swap      order lifecycle, routing, pricing, asset catalog
internal/bitnob    signed Bitnob API client
internal/db        PostgreSQL store
internal/demo      stub exchange + in-memory store for DEMO=1
internal/webhook   Bitnob webhook handling
```

`web/src/types.ts` mirrors the server's JSON by hand. When a handler in
`cmd/server/main.go` changes shape, change it there too.

## Notes

- Rates are float, not locked. The order page says so; the final amount is settled
  at the market rate when the deposit confirms.
- Bitnob signing is HMAC-SHA256 over `X-Auth-Client`, `X-Auth-Timestamp`,
  `X-Auth-Nonce`, `X-Auth-Signature`.
- Quotes are ordered immediately and retried once if the quote expires first.
- Withdrawal references use `swapID + "-withdrawal"` for idempotency.
- Amounts are encoded as integer strings at the Bitnob boundary.
- No personal data is collected, but an order row holds the deposit and payout
  addresses. That is a real privacy surface: set a retention policy before running
  this in public.
