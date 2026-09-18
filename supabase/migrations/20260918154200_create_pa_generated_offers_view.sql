-- =============================================================
-- Migration: Create PlayerAuctions Generated Offers View
-- =============================================================
CREATE OR REPLACE VIEW playerauctions_generated_offers AS
SELECT 
    s.id AS service_id,
    
    -- Calculates the actual unit quantity (10, 20, 30...)
    (s.base_quantity + (step_num - 1) * s.step_increment) AS final_quantity,
    
    -- Dynamically formats the time. If >= 24 hours, converts to "X Days". Otherwise "X Hours".
    CASE 
        WHEN s.time_scales = TRUE THEN
            CASE 
                WHEN (s.base_delivery_hours * step_num) >= 24 THEN ((s.base_delivery_hours * step_num) / 24)::TEXT || ' Days'
                ELSE (s.base_delivery_hours * step_num)::TEXT || ' Hours'
            END
        ELSE
            CASE 
                WHEN s.base_delivery_hours >= 24 THEN (s.base_delivery_hours / 24)::TEXT || ' Days'
                ELSE s.base_delivery_hours::TEXT || ' Hours'
            END
    END AS final_delivery_time,

    -- Dynamically builds the title with the quantity and correct time
    s.base_title || ' - ' || 
    (s.base_quantity + (step_num - 1) * s.step_increment)::TEXT || 'x ' || s.unit_name || 
    ' - ' || 
    (CASE 
        WHEN s.time_scales = TRUE THEN
            CASE WHEN (s.base_delivery_hours * step_num) >= 24 THEN ((s.base_delivery_hours * step_num) / 24)::TEXT || ' Days' ELSE (s.base_delivery_hours * step_num)::TEXT || ' Hours' END
        ELSE
            CASE WHEN s.base_delivery_hours >= 24 THEN (s.base_delivery_hours / 24)::TEXT || ' Days' ELSE s.base_delivery_hours::TEXT || ' Hours' END
    END) AS generated_title,
    
    -- Calculates the scaled price
    (s.base_price + ((step_num - 1) * (s.base_price * s.discount_modifier)))::NUMERIC(10,2) AS final_price,
    
    -- Aggregates categories
    STRING_AGG(c.name, ' | ') AS categories,

    -- Dynamic description linking based on unit_name, with fallback to generic
    COALESCE(
        (SELECT id FROM public.playerauctions_descriptions WHERE slug = LOWER(s.unit_name) LIMIT 1),
        (SELECT id FROM public.playerauctions_descriptions WHERE slug = 'general' LIMIT 1)
    ) AS description_id
    
FROM playerauctions_services s
CROSS JOIN generate_series(1, s.max_steps) AS step_num 
JOIN playerauctions_service_categories sc ON s.id = sc.service_id
JOIN playerauctions_categories c ON sc.category_id = c.id
GROUP BY 
    s.id, s.base_title, s.unit_name, s.base_quantity, s.step_increment, 
    s.base_price, s.discount_modifier, s.base_delivery_hours, s.time_scales, step_num
ORDER BY 
    s.id, final_quantity ASC;
