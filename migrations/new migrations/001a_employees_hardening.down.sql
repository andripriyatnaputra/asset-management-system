-- Hapus kolom audit
ALTER TABLE public.employees DROP COLUMN IF EXISTS created_at;
ALTER TABLE public.employees DROP COLUMN IF EXISTS updated_at;

-- Kembalikan email ke varchar
ALTER TABLE public.employees
  ALTER COLUMN email TYPE varchar(255);

-- Drop trigger function
DROP FUNCTION IF EXISTS public.set_timestamp();

-- Drop enum role (hati-hati kalau sudah dipakai banyak data!)
-- ALTER TABLE public.employees ALTER COLUMN role TYPE varchar(50);
-- DROP TYPE IF EXISTS employee_role;
