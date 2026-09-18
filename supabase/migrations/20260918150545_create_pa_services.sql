-- =============================================================
-- PlayerAuctions Automation Suite — Initial Schema Setup
-- Includes Descriptions, Offers, Services, and Categories
-- =============================================================

-- =============================================================
-- 1. Descriptions Table (playerauctions_descriptions)
-- =============================================================
create table if not exists public.playerauctions_descriptions (
    id          uuid primary key default gen_random_uuid(),
    slug        text        not null unique,   -- e.g. 'general', 'raid-service', 'levelling'
    name        text        not null,          -- human-readable, e.g. 'General Pokemon GO Description'
    content     text        not null,          -- HTML content
    created_at  timestamptz not null default now(),
    updated_at  timestamptz not null default now()
);

-- Auto-update updated_at on every row change
create or replace function public.playerauctions_set_updated_at()
returns trigger language plpgsql as $$
begin
    new.updated_at = now();
    return new;
end;
$$;

create trigger playerauctions_descriptions_set_updated_at
    before update on public.playerauctions_descriptions
    for each row execute function public.playerauctions_set_updated_at();

-- RLS for descriptions
alter table public.playerauctions_descriptions enable row level security;

create policy "Allow authenticated reads on descriptions"
    on public.playerauctions_descriptions
    for select
    to authenticated
    using (true);

create policy "Allow service role all access on descriptions"
    on public.playerauctions_descriptions
    for all
    to service_role
    using (true)
    with check (true);

-- =============================================================
-- 2. Offers Table (playerauctions_offers)
-- =============================================================
create table if not exists public.playerauctions_offers (
    id          uuid primary key default gen_random_uuid(),
    title       text           not null,
    price       numeric(10, 2) not null check (price >= 0),
    categories  text[]         not null default '{}',
    game        text           not null default 'Pokemon Go',
    is_scraped  boolean        not null default false,
    created_at  timestamptz    not null default now(),
    description_id uuid references public.playerauctions_descriptions(id) on delete set null
);

create index if not exists playerauctions_offers_game_idx on public.playerauctions_offers (game);
create index if not exists playerauctions_offers_created_at_idx on public.playerauctions_offers (created_at desc);
create index if not exists playerauctions_offers_categories_idx on public.playerauctions_offers using gin (categories);

-- Enable Row Level Security
alter table public.playerauctions_offers enable row level security;

create policy "Allow authenticated reads on offers"
    on public.playerauctions_offers
    for select
    to authenticated
    using (true);

create policy "Allow service role all access on offers"
    on public.playerauctions_offers
    for all
    to service_role
    using (true)
    with check (true);


-- =============================================================
-- 3. PlayerAuctions Services & Categories
-- =============================================================

-- Drop existing tables
DROP TABLE IF EXISTS playerauctions_service_categories CASCADE;
DROP TABLE IF EXISTS playerauctions_services CASCADE;
DROP TABLE IF EXISTS playerauctions_categories CASCADE;

-- Create Categories
CREATE TABLE playerauctions_categories (
    id SERIAL PRIMARY KEY,
    name VARCHAR(100) UNIQUE NOT NULL
);

-- Create Services (Now with Dynamic Time)
CREATE TABLE playerauctions_services (
    id SERIAL PRIMARY KEY,
    base_title TEXT NOT NULL,           
    unit_name VARCHAR(50) NOT NULL,     
    base_quantity INT NOT NULL,         
    step_increment INT NOT NULL,        
    max_steps INT NOT NULL,             
    base_price NUMERIC(10, 2) NOT NULL, 
    discount_modifier NUMERIC(5, 2) NOT NULL DEFAULT 1.00,
    base_delivery_hours INT NOT NULL,   -- Store in hours (e.g., 24 for 1 day)
    time_scales BOOLEAN DEFAULT TRUE    -- TRUE = multiply time by step, FALSE = time stays fixed
);

-- Create Junction Table
CREATE TABLE playerauctions_service_categories (
    service_id INT REFERENCES playerauctions_services(id) ON DELETE CASCADE,
    category_id INT REFERENCES playerauctions_categories(id) ON DELETE CASCADE,
    PRIMARY KEY (service_id, category_id)
);

-- =============================================================
-- 3.5 Enable RLS for Services & Categories
-- =============================================================
alter table public.playerauctions_categories enable row level security;
alter table public.playerauctions_services enable row level security;
alter table public.playerauctions_service_categories enable row level security;

create policy "Allow authenticated reads on categories"
    on public.playerauctions_categories for select to authenticated using (true);
create policy "Allow service role all access on categories"
    on public.playerauctions_categories for all to service_role using (true) with check (true);

create policy "Allow authenticated reads on services"
    on public.playerauctions_services for select to authenticated using (true);
create policy "Allow service role all access on services"
    on public.playerauctions_services for all to service_role using (true) with check (true);

create policy "Allow authenticated reads on service_categories"
    on public.playerauctions_service_categories for select to authenticated using (true);
create policy "Allow service role all access on service_categories"
    on public.playerauctions_service_categories for all to service_role using (true) with check (true);


-- =============================================================
-- 4. Comments for All Tables
-- =============================================================

-- Comments for playerauctions_descriptions
comment on table  public.playerauctions_descriptions               is 'Reusable HTML descriptions for offers, keyed by slug';
comment on column public.playerauctions_descriptions.id            is 'UUID primary key';
comment on column public.playerauctions_descriptions.slug          is 'Unique machine key, e.g. general, raid-service, levelling';
comment on column public.playerauctions_descriptions.name          is 'Human-readable label shown in admin/tooling';
comment on column public.playerauctions_descriptions.content       is 'HTML content of the description';
comment on column public.playerauctions_descriptions.created_at    is 'UTC timestamp when the row was inserted';
comment on column public.playerauctions_descriptions.updated_at    is 'UTC timestamp of last update (auto-managed)';

-- Comments for playerauctions_offers
comment on table  public.playerauctions_offers               is 'Offers from PlayerAuctions — some scraped automatically, some added manually';
comment on column public.playerauctions_offers.id            is 'UUID primary key';
comment on column public.playerauctions_offers.title         is 'Offer listing title';
comment on column public.playerauctions_offers.price         is 'Offer price in USD';
comment on column public.playerauctions_offers.categories    is 'Array of categories this offer appears under (e.g. {"Levelling","Pokemon Catching"})';
comment on column public.playerauctions_offers.game          is 'Game the offer belongs to — defaults to Pokemon Go';
comment on column public.playerauctions_offers.is_scraped    is 'True if the offer was pulled automatically from the PA API; false if added manually';
comment on column public.playerauctions_offers.created_at    is 'UTC timestamp when the row was inserted';
comment on column public.playerauctions_offers.description_id      is 'FK to descriptions — null means no description assigned yet';

-- Comments for playerauctions_categories
comment on table  public.playerauctions_categories               is 'Categories for classifying PlayerAuctions services';
comment on column public.playerauctions_categories.id            is 'Auto-incrementing primary key';
comment on column public.playerauctions_categories.name          is 'Unique name of the category';

-- Comments for playerauctions_services
comment on table  public.playerauctions_services                 is 'Configurable base services for generating dynamic PlayerAuctions listings';
comment on column public.playerauctions_services.id              is 'Auto-incrementing primary key';
comment on column public.playerauctions_services.base_title      is 'The base title template for the service';
comment on column public.playerauctions_services.unit_name       is 'The unit type being sold, e.g., Raids, Catches, KM, Days';
comment on column public.playerauctions_services.base_quantity   is 'The starting quantity at step 1';
comment on column public.playerauctions_services.step_increment  is 'How much the quantity increases per step';
comment on column public.playerauctions_services.max_steps       is 'The maximum number of steps allowed for this service';
comment on column public.playerauctions_services.base_price      is 'The base price at step 1';
comment on column public.playerauctions_services.discount_modifier is 'Modifier to apply to the price increment per step (e.g., 0.95 means 5% discount on added items)';
comment on column public.playerauctions_services.base_delivery_hours is 'The base delivery time in hours';
comment on column public.playerauctions_services.time_scales     is 'If TRUE, delivery time scales with steps. If FALSE, delivery time is fixed.';

-- Comments for playerauctions_service_categories
comment on table  public.playerauctions_service_categories               is 'Junction table linking services to categories';
comment on column public.playerauctions_service_categories.service_id    is 'Reference to playerauctions_services';
comment on column public.playerauctions_service_categories.category_id   is 'Reference to playerauctions_categories';



