BEGIN;

-- Each parent (including the root group) has its own sibling sort sequence.
ALTER TABLE public.menus
    ADD COLUMN IF NOT EXISTS sort_order INTEGER;

WITH ranked_menus AS (
    SELECT
        id,
        ROW_NUMBER() OVER (
            PARTITION BY parent_id
            ORDER BY created_at ASC NULLS LAST, id ASC
        )::INTEGER AS sort_order
    FROM public.menus
)
UPDATE public.menus AS menu
SET sort_order = ranked_menus.sort_order
FROM ranked_menus
WHERE menu.id = ranked_menus.id
  AND menu.sort_order IS NULL;

ALTER TABLE public.menus
    ALTER COLUMN sort_order SET NOT NULL;

CREATE UNIQUE INDEX IF NOT EXISTS menus_root_sort_order_unique_idx
    ON public.menus (sort_order)
    WHERE parent_id IS NULL;

CREATE UNIQUE INDEX IF NOT EXISTS menus_parent_sort_order_unique_idx
    ON public.menus (parent_id, sort_order)
    WHERE parent_id IS NOT NULL;

COMMIT;
