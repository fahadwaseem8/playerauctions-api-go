# PlayerAuctions API - Go

A Go-based background service and price-adjuster for integrating with the PlayerAuctions API.

---

## Getting Started

1. Copy `.env.example` to `.env` and fill in your credentials.
2. You can generate a new API key from the [PlayerAuctions API Key Management page](https://member.playerauctions.com/account/api-key-management).
3. The official documentation for the PlayerAuctions API can be found [here](https://support.playerauctions.com/hc/en-us/articles/41423203201945).

---

## Environment Variables

| Variable | Purpose |
|---|---|
| `SUPABASE_URL` | Your project's REST API URL (from Supabase dashboard → Project Settings → API) |
| `SUPABASE_KEY` | `service_role` secret key — used by the Go worker to read/write data |
| `PLAYERAUCTIONS_API_KEY` | PlayerAuctions seller API key |
| `PLAYERAUCTIONS_API_SECRET` | PlayerAuctions HMAC signing secret |

> **Note:** `SUPABASE_URL` and `SUPABASE_KEY` are needed by the **Go app at runtime**.
> `supabase link` (used by the CLI for migrations) is a separate thing — it authenticates
> via your Supabase account token and does **not** replace these env vars.

---

## Database — Supabase Migrations

### Prerequisites

```powershell
npm install -g supabase
```

### First-time setup (already done)

```powershell
supabase init           # creates supabase/ folder
supabase login          # authenticates via browser
supabase link           # links CLI to your remote Supabase project
```

### Applying existing migrations to a fresh project

```powershell
supabase db push
```

This runs all migration files in `supabase/migrations/` that haven't been applied yet.

### Adding a new migration

```powershell
# 1. Create a new timestamped migration file
supabase migration new <description>
# e.g. supabase migration new add_status_to_offers
# → creates: supabase/migrations/YYYYMMDDHHMMSS_add_status_to_offers.sql

# 2. Edit the generated file with your SQL changes

# 3. Push to remote
supabase db push
```

### Migration files

All migrations live in `supabase/migrations/` and are run in timestamp order.
Supabase tracks which ones have been applied so re-running `db push` is safe.

| Migration | Description |
|---|---|
| `20260912054918_create_offers_table.sql` | Initial `offers` table (title, price, category, game, is_scraped, created_at) |
