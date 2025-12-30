-- =========================================================
-- 001a__employees_hardening.sql
-- =========================================================

CREATE EXTENSION IF NOT EXISTS citext;
CREATE EXTENSION IF NOT EXISTS pg_trgm;

DO $$
BEGIN
  CREATE TYPE employee_role AS ENUM ('super_admin','asset_manager','it_support','finance','employee');
EXCEPTION WHEN duplicate_object THEN
  NULL;
END$$;

ALTER TABLE public.employees
  ADD COLUMN IF NOT EXISTS created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  ADD COLUMN IF NOT EXISTS updated_at TIMESTAMPTZ NOT NULL DEFAULT now();

UPDATE public.employees
SET created_at = COALESCE(created_at, now()),
    updated_at = COALESCE(updated_at, now());

ALTER TABLE public.employees
  ALTER COLUMN email TYPE CITEXT;

ALTER TABLE public.employees
  ALTER COLUMN role SET DEFAULT 'employee';

UPDATE public.employees
SET role = 'employee'
WHERE role IS NULL
   OR role NOT IN ('super_admin','asset_manager','it_support','finance','employee');

ALTER TABLE public.employees
  ALTER COLUMN role TYPE employee_role
  USING role::employee_role;

ALTER TABLE public.employees
  DROP CONSTRAINT IF EXISTS employees_department_id_fkey;

ALTER TABLE public.employees
  ADD CONSTRAINT employees_department_id_fkey
  FOREIGN KEY (department_id)
  REFERENCES public.departments (id)
  ON UPDATE CASCADE
  ON DELETE SET NULL;

-- Buat fungsi trigger
CREATE OR REPLACE FUNCTION public.set_timestamp()
RETURNS trigger
LANGUAGE plpgsql
AS $$
BEGIN
  NEW.updated_at = now();
  RETURN NEW;
END;
$$;
