-- =============================================================
-- PlayerAuctions Automation Suite — Supabase Schema
-- Table: offers
-- =============================================================

create table if not exists public.offers (
    id          uuid primary key default gen_random_uuid(),
    title       text           not null,
    price       numeric(10, 2) not null check (price >= 0),
    category    text           not null,
    game        text           not null default 'Pokemon Go',
    is_scraped  boolean        not null default false,
    created_at  timestamptz    not null default now()
);

-- Index for fast lookups by game
create index if not exists offers_game_idx on public.offers (game);

-- Index for time-ordered queries
create index if not exists offers_created_at_idx on public.offers (created_at desc);

-- Enable Row Level Security (recommended for Supabase)
alter table public.offers enable row level security;

-- Policy: allow all authenticated reads (adjust as needed)
create policy "Allow authenticated reads"
    on public.offers
    for select
    to authenticated
    using (true);

-- Policy: allow service role full access (for the Go worker)
create policy "Allow service role all access"
    on public.offers
    for all
    to service_role
    using (true)
    with check (true);

-- =============================================================
-- Comments
-- =============================================================
comment on table  public.offers               is 'Offers from PlayerAuctions — some scraped automatically, some added manually';
comment on column public.offers.id            is 'UUID primary key';
comment on column public.offers.title         is 'Offer listing title';
comment on column public.offers.price         is 'Offer price in USD';
comment on column public.offers.category      is 'Product category (e.g. Accounts, Items, Currency)';
comment on column public.offers.game          is 'Game the offer belongs to — defaults to Pokemon Go';
comment on column public.offers.is_scraped    is 'True if the offer was pulled automatically from the PA API; false if added manually';
comment on column public.offers.created_at    is 'UTC timestamp when the row was inserted';
