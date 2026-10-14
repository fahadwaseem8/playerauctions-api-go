# PlayerAuctions offers

Generate boosting offers from Supabase and preview or post them through the
PlayerAuctions Seller API. Run the app from the repo root with `go run .`.
Preview is the default; posting requires `-mode post -agree`.

## Setup

Use the Go version in `go.mod`. Copy `.env.example` to `.env` and fill in:

- `SUPABASE_URL` and `SUPABASE_KEY`: project URL and service-role key for generating offers.
- `PLAYERAUCTIONS_API_KEY` and `PLAYERAUCTIONS_API_SECRET`: required for PlayerAuctions metadata and posting; preview does not need them.

Existing environment variables take precedence over `.env`.

Find the real game and boosting category IDs using these read-only commands:

```powershell
go run . -mode games
go run . -mode categories -game-id 123
```

Replace `123` with the game's returned ID. The game must support `powerleveling`.
Copy `offers.config.example.json` to `offers.config.json`, then fill in:

- `gameId`: the PlayerAuctions game ID. The example's `0` is an invalid placeholder.
- `offerDuration`: listing lifetime in days: `3`, `7`, `14`, or `30`.
- `totalUnit`: available stock of each package; defaults to `10` in the example.
- `serviceCategories`: keys are your **Supabase service IDs**. Each entry supplies
  a PlayerAuctions `parentItemId` and, when applicable, an integer `itemId`.
  Omit `itemId` for a category without children. These IDs are different from
  Supabase category IDs. Add a mapping for each service you intend to generate.

The example maps service `1`; replace it with your own service IDs and real
category IDs. `offers.config.json` is local configuration and is ignored by Git.

## Preview

```powershell
go run .
go run . -service-id 1 -limit 2
go run . > offers.json
```

Convert the preview to `title | price | category` lines (overwrites `offers.txt`):

```powershell
go run ./cmd/offers-to-txt
```

Use `-input offers.json -output converted-offers.txt` to choose different files.
The converter preserves one line per JSON offer and uses Pokemon GO category names.

Preview fetches Supabase data and prints an array of API request bodies to stdout.
Progress and diagnostics go to stderr. It makes no PlayerAuctions requests.
`agreeCheck` stays false unless you pass `-agree` to accept the Secure Seller
Delivery Agreement. Category membership is verified against live metadata only
in post mode; preview performs local validation.

Each categorized service expands into `max_steps` packages. Quantity, pricing,
and delivery estimates come from the service settings. `base_delivery_days` accepts
fractional days (0.125 = 3 hours). Every offer uses the same HTML content from
the single `general` row in `playerauctions_descriptions`.
Services without category links are skipped, as in the original view.

The request shape uses `gameId`, `title`, `offerDesc`, `offerDuration`, `agreeCheck`,
`price`, `totalUnit`, `parentItemId`, and optional `itemId`. For example, a package
of 20 raids can have `totalUnit: 1`: one package is in stock, not 20 packages.
Delivery estimates appear in the title because this creation endpoint has no
`deliveryTime` field. A 36-hour service remains `36 Hours` rather than being
rounded down to one day.

## Post offers

Check account eligibility, review the preview, and then explicitly post:

```powershell
go run . -mode prevalidate
go run . -service-id 1 -limit 1 -agree
go run . -mode post -service-id 1 -limit 1 -agree
```

`-agree` means you accept PlayerAuctions' Secure Seller Delivery Agreement.
Before the first creation, the app validates the whole selected batch, checks
seller eligibility, confirms boosting support for the game, and verifies category
IDs. The prevalidation endpoint checks account eligibility; it does not validate
individual offer payloads or guarantee that creation will succeed.

Successful creations are written immediately as JSON Lines, including the offer ID,
URL, Supabase service ID, and package quantity. To retain them:

```powershell
go run . -mode post -agree > posted-offers.jsonl
```

This command posts **all** generated offers. Use `-service-id` and `-limit` to
restrict the batch. Ctrl+C cancels further work. Requests are spaced 19 seconds
apart for seller levels 0–3 and 10 seconds for levels 4–5. Earlier runs and other
clients may already have consumed hourly quota.

Creation stops on the first error without automatic retries. If a request times
out, check your listings before retrying: the server may have created it. Results
are not saved back to Supabase, and rerunning a batch can create duplicate offers.
There is no persistent resume or deduplication mechanism.

Use `go run . -help` for all options. `-config` selects a different mapping file.

## Code layout

| Location | Responsibility |
|---|---|
| `main.go` | Entry point and Ctrl+C cancellation |
| `internal/app/` | Commands, preparation, preflight checks, posting and pacing |
| `internal/config/` | Environment and offer configuration |
| `internal/offers/` | Service models, runtime generation and API payload mapping |
| `internal/supabase/` | Paginated reads of services and descriptions |
| `internal/playerauctions/` | HMAC signing, HTTP errors, metadata and boosting endpoints |
| `supabase/migrations/` | Existing database schema |

No database views, deployment services, background workers, or price adjuster
are required. The initial migration creates the final schema and a blank shared
description. Populate services, categories and category links, then edit the shared
description to generate offers. For a linked Supabase project, apply schema
migrations with `supabase db push`.

## Verification and contract references

```powershell
go test ./...
go vet ./...
```

Tests use local mock servers and cover signing the exact transmitted bytes,
response errors, metadata, pagination, runtime generation, payload mapping,
preview, preflight checks, pacing, and stopping after failed creation. They do not
establish acceptance by a live seller account.

The integration uses `https://seller-api.playerauctions.com` and the official
[`X-PA-*` headers and HMAC-SHA256 signing](https://support.playerauctions.com/hc/en-us/articles/58370185051929).
The signature input is the API key followed by the Unix timestamp in seconds and
exact request body bytes, with a lowercase hexadecimal result. GET calls sign an
empty body. Responses must have a successful HTTP status and `code: 10000`.

The overview mentions form encoding, but the specific
[boosting creation endpoint](https://support.playerauctions.com/hc/en-us/articles/58910046914457)
and [creation flow](https://support.playerauctions.com/hc/en-us/articles/59528735920281)
show JSON payloads; this client follows those endpoint-specific instructions.
It supports boosting creation and the metadata needed for it, not all Seller API
operations.

Other contract references:

- [Game list](https://support.playerauctions.com/hc/en-us/articles/58909225382297)
- [Boosting categories](https://support.playerauctions.com/hc/en-us/articles/58908718113945)
- [Creation prevalidation](https://support.playerauctions.com/hc/en-us/articles/58910892947737)
- [API errors](https://support.playerauctions.com/hc/en-us/articles/58908561192857)

## Fresh-project database setup

Apply the single migration `20260918150545_create_pa_services.sql` with
`supabase db push`. It creates the tables, indexes, policies, update trigger,
`base_delivery_days`, and the single shared description row directly.
Edit the `general` row's `content` in `playerauctions_descriptions`, then populate
services and category links before generating offers.
