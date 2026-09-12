-- =============================================================
-- Migration: add_descriptions_table
-- Creates the descriptions table and links it to offers.
-- =============================================================

-- 1. Descriptions table — one row per description type
create table if not exists public.descriptions (
    id          uuid primary key default gen_random_uuid(),
    slug        text        not null unique,   -- e.g. 'general', 'raid-service', 'levelling'
    name        text        not null,          -- human-readable, e.g. 'General Pokemon GO Description'
    content     text        not null,          -- HTML content
    created_at  timestamptz not null default now(),
    updated_at  timestamptz not null default now()
);

-- Auto-update updated_at on every row change
create or replace function public.set_updated_at()
returns trigger language plpgsql as $$
begin
    new.updated_at = now();
    return new;
end;
$$;

create trigger descriptions_set_updated_at
    before update on public.descriptions
    for each row execute function public.set_updated_at();

-- 2. Add description_id FK to offers (nullable — not all offers need one)
alter table public.offers
    add column if not exists description_id uuid references public.descriptions(id) on delete set null;

-- 3. RLS for descriptions
alter table public.descriptions enable row level security;

create policy "Allow authenticated reads on descriptions"
    on public.descriptions
    for select
    to authenticated
    using (true);

create policy "Allow service role all access on descriptions"
    on public.descriptions
    for all
    to service_role
    using (true)
    with check (true);

-- 4. Comments
comment on table  public.descriptions               is 'Reusable HTML descriptions for offers, keyed by slug';
comment on column public.descriptions.id            is 'UUID primary key';
comment on column public.descriptions.slug          is 'Unique machine key, e.g. general, raid-service, levelling';
comment on column public.descriptions.name          is 'Human-readable label shown in admin/tooling';
comment on column public.descriptions.content       is 'HTML content of the description';
comment on column public.descriptions.created_at    is 'UTC timestamp when the row was inserted';
comment on column public.descriptions.updated_at    is 'UTC timestamp of last update (auto-managed)';
comment on column public.offers.description_id      is 'FK to descriptions — null means no description assigned yet';
