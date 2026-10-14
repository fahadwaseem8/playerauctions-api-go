-- Run once in the Supabase SQL Editor, after the initial schema migration.
-- Converts existing durations rather than reinterpreting hours as days:
-- 24 hours -> 1 day; 264 hours -> 11 days; 3 hours -> 0.125 days.
-- NUMERIC allows fractional days for services delivered in less than a day.
-- The updated Go generator and seed-offers.sql expect base_delivery_days.

BEGIN;

ALTER TABLE public.playerauctions_services
    RENAME COLUMN base_delivery_hours TO base_delivery_days;

ALTER TABLE public.playerauctions_services
    ALTER COLUMN base_delivery_days TYPE numeric
    USING (base_delivery_days::numeric / 24);

COMMENT ON COLUMN public.playerauctions_services.base_delivery_days
    IS 'Base delivery duration in days; fractional days are allowed. Multiplied by the step number when time_scales is true.';

-- Refresh the Supabase REST API schema cache after the transaction commits.
NOTIFY pgrst, 'reload schema';

COMMIT;
