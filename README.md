# Bitnob Cross-Chain Stablecoin Swap

Backend service for a Bitnob-powered cross-chain swap app. Users create a swap by choosing source and destination chains/assets, receive a Bitnob deposit address, and the backend handles deposit webhooks, route selection, trading, withdrawal, and completion status updates.

## Prerequisites

- Go 1.21+
- PostgreSQL
- Bitnob API credentials
- ngrok or a similar tunnel for local webhook testing

## Environment Setup

Copy the example environment file and fill in the values:

```bash
cp .env.example .env
```

Required keys:

- `BITNOB_CLIENT_ID`
- `BITNOB_CLIENT_SECRET`
- `BITNOB_BASE_URL`
- `DATABASE_URL`
- `BITNOB_WEBHOOK_SECRET`
- `PORT`
- `SWAP_FEE_USDT`
- `MIN_SWAP_AMOUNT`

`BITNOB_BASE_URL` is listed for completeness, but the backend config intentionally hardcodes Bitnob's base URL to `https://api.bitnob.com`.

## Database Setup

Create a PostgreSQL database and run the migration:

```bash
psql "$DATABASE_URL" -f migrations/001_init.sql
```

## Run The Backend

```bash
go run ./cmd/server
```

On startup, the server connects to PostgreSQL and calls `POST /api/whoami` on Bitnob. If credential verification fails, startup exits.

## Local Webhooks

Expose the backend with ngrok:

```bash
ngrok http 8080
```

Configure the Bitnob webhook URL as:

```text
https://your-ngrok-domain.ngrok-free.app/webhooks/bitnob
```

## API Routes

- `POST /swap` creates a swap, validates the destination address, generates a deposit address, saves the swap, and returns deposit instructions.
- `GET /swap/{id}` returns swap status and persisted swap fields for polling.
- `GET /prices` proxies Bitnob live trading prices.
- `POST /webhooks/bitnob` accepts Bitnob webhooks for `address.deposit.confirmed` and `transfer.success`.

## Notes

- All Bitnob request signing uses HMAC-SHA256 with `X-Auth-Client`, `X-Auth-Timestamp`, `X-Auth-Nonce`, and `X-Auth-Signature`.
- Quotes are ordered immediately and retried once if the quote expires before order placement.
- Withdrawal references use `swapID + "-withdrawal"` for idempotency.
- Amounts sent to Bitnob are encoded as integer strings at the API boundary.
