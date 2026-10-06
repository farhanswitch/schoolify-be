BEGIN;

ALTER TABLE public.users RENAME COLUMN created TO created_at;
ALTER TABLE public.users RENAME COLUMN updated TO updated_at;

-- Preserve the existing Asia/Jakarta wall-clock values while changing from timestamptz.
ALTER TABLE public.users
    ALTER COLUMN created_at TYPE timestamp without time zone
    USING created_at AT TIME ZONE current_setting('TimeZone');

ALTER TABLE public.users
    ALTER COLUMN updated_at TYPE timestamp without time zone
    USING updated_at AT TIME ZONE current_setting('TimeZone');

CREATE OR REPLACE FUNCTION public.set_updated_at()
RETURNS trigger
LANGUAGE plpgsql
AS $function$
BEGIN
    NEW.updated_at = CURRENT_TIMESTAMP;
    RETURN NEW;
END;
$function$;

COMMIT;
