# PlayerAuctions offer preview

A single Go program that reads Supabase services, categories, and descriptions,
generates offer variants at runtime, and prints JSON payloads. It does not create
database views or post offers to PlayerAuctions. The payload format is a preview;
live API posting is not implemented yet.

## Run

Copy `.env.example` to `.env` and set your Supabase project URL and service-role key.
With the Go version specified in `go.mod` installed, run:

```powershell
go run .
```

JSON goes to stdout and the preview count goes to stderr. To save the payloads:

```powershell
go run . > offers.json
```

`main.go` expands each categorized service into `max_steps` offers using its
quantity increment, price modifier, and delivery settings. Descriptions match the
lowercase unit name, falling back to the `general` description. If neither exists,
the description is empty. An empty database produces an empty JSON array.

## Migrations

`supabase/migrations/20260918150545_create_pa_services.sql` creates the tables,
indexes, triggers, and access policies. No insert/seed migration is currently
present in this checkout. Service, category, category-link, and description rows
must exist to generate populated previews.

For a linked Supabase project, apply migrations with:

```powershell
supabase db push
```

The former generated-offers view migration has been removed. Existing databases
are not changed by this repository cleanup; the program no longer uses that view.
No deployment services, background workers, or price adjuster are included.
