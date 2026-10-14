-- Run after 20261014000000_use_base_delivery_days.sql.
-- Keep the existing 'general' description. If absent, keep the oldest description.
-- Other descriptions and per-offer links are removed: all offers share this content.
BEGIN;

LOCK TABLE public.playerauctions_descriptions, public.playerauctions_offers
    IN ACCESS EXCLUSIVE MODE;

ALTER TABLE public.playerauctions_offers DROP COLUMN description_id;

DELETE FROM public.playerauctions_descriptions
WHERE id NOT IN (
    SELECT id FROM public.playerauctions_descriptions
    ORDER BY (slug = 'general') DESC, created_at, id
    LIMIT 1
);

UPDATE public.playerauctions_descriptions
SET slug = 'general', name = 'Shared offer description';

INSERT INTO public.playerauctions_descriptions (slug, name, content)
SELECT 'general', 'Shared offer description', ''
WHERE NOT EXISTS (SELECT 1 FROM public.playerauctions_descriptions);

-- Together with the existing UNIQUE(slug), this permits at most one row.
ALTER TABLE public.playerauctions_descriptions
    ADD CONSTRAINT playerauctions_descriptions_shared_only CHECK (slug = 'general');

COMMENT ON TABLE public.playerauctions_descriptions
    IS 'Single shared HTML description for every offer. Edit the general row content.';
COMMENT ON COLUMN public.playerauctions_descriptions.slug
    IS 'Singleton key; must be general.';

-- Clean up display precision left by the delivery-days conversion.
UPDATE public.playerauctions_services
SET base_delivery_days = trim_scale(base_delivery_days);

NOTIFY pgrst, 'reload schema';
COMMIT;
