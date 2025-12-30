-- indeks untuk soft delete filter
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_employees_deleted_at
  ON public.employees (deleted_at);

-- indeks untuk filter per departemen
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_employees_department_id
  ON public.employees (department_id);

-- trigram index untuk pencarian nama
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_employees_name_trgm
  ON public.employees USING gin (name gin_trgm_ops);

-- trigram index untuk email (opsional)
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_employees_email_trgm
  ON public.employees USING gin (email gin_trgm_ops);
