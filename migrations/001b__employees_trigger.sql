-- =========================================================
-- 001b__employees_trigger.sql
-- =========================================================

DROP TRIGGER IF EXISTS set_timestamp_employees ON public.employees;

CREATE TRIGGER set_timestamp_employees
BEFORE UPDATE ON public.employees
FOR EACH ROW
EXECUTE FUNCTION public.set_timestamp();
