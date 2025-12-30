-- 1) Enumerasi status
DO $$
BEGIN
  IF NOT EXISTS (SELECT 1 FROM pg_type WHERE typname = 'asset_status') THEN
    CREATE TYPE asset_status AS ENUM ('in_stock','assigned','maintenance','retired','disposed');
  END IF;
END$$;

-- 2) Tabel referensi lokasi & cost center (opsional tapi dianjurkan)
CREATE TABLE IF NOT EXISTS cost_centers (
  id BIGSERIAL PRIMARY KEY,
  code TEXT UNIQUE NOT NULL,
  name TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS locations (
  id BIGSERIAL PRIMARY KEY,
  site TEXT NOT NULL,
  building TEXT,
  room TEXT
);

-- 3) Kolom standar aset
ALTER TABLE assets
  ADD COLUMN IF NOT EXISTS status asset_status DEFAULT 'in_stock' NOT NULL,
  ADD COLUMN IF NOT EXISTS department_id BIGINT REFERENCES departments(id) ON DELETE SET NULL,
  ADD COLUMN IF NOT EXISTS cost_center_id BIGINT REFERENCES cost_centers(id) ON DELETE SET NULL,
  ADD COLUMN IF NOT EXISTS location_id BIGINT REFERENCES locations(id) ON DELETE SET NULL,
  ADD COLUMN IF NOT EXISTS purchase_date DATE,
  ADD COLUMN IF NOT EXISTS purchase_cost NUMERIC(14,2),
  ADD COLUMN IF NOT EXISTS vendor TEXT,
  ADD COLUMN IF NOT EXISTS warranty_expiry DATE,
  ADD COLUMN IF NOT EXISTS useful_life_months INT,
  ADD COLUMN IF NOT EXISTS depreciation_method TEXT,         -- 'straight_line','declining',...
  ADD COLUMN IF NOT EXISTS salvage_value NUMERIC(14,2);

-- 4) History wajib untuk audit trail
CREATE TABLE IF NOT EXISTS asset_history (
  id BIGSERIAL PRIMARY KEY,
  asset_id BIGINT NOT NULL REFERENCES assets(id) ON DELETE CASCADE,
  action TEXT NOT NULL,              -- 'created','updated','assigned','returned','maintenance','retired','disposed'
  detail TEXT,
  actor_employee_id BIGINT REFERENCES employees(id),
  from_status asset_status,
  to_status asset_status,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- 5) Pemetaan software terpasang (jika belum)
DO $$
DECLARE
  has_licenses BOOLEAN;
  has_software_licenses BOOLEAN;
BEGIN
  SELECT EXISTS (
    SELECT 1 FROM information_schema.tables
    WHERE table_schema = 'public' AND table_name = 'licenses'
  ) INTO has_licenses;

  SELECT EXISTS (
    SELECT 1 FROM information_schema.tables
    WHERE table_schema = 'public' AND table_name = 'software_licenses'
  ) INTO has_software_licenses;

  IF has_licenses THEN
    EXECUTE $SQL$
      CREATE TABLE IF NOT EXISTS asset_software_installs (
        id BIGSERIAL PRIMARY KEY,
        asset_id BIGINT NOT NULL REFERENCES assets(id) ON DELETE CASCADE,
        license_id BIGINT NOT NULL REFERENCES licenses(id) ON DELETE CASCADE,
        installed_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
        installed_by_employee_id BIGINT REFERENCES employees(id)
      );
    $SQL$;
  ELSIF has_software_licenses THEN
    EXECUTE $SQL$
      CREATE TABLE IF NOT EXISTS asset_software_installs (
        id BIGSERIAL PRIMARY KEY,
        asset_id BIGINT NOT NULL REFERENCES assets(id) ON DELETE CASCADE,
        license_id BIGINT NOT NULL REFERENCES software_licenses(id) ON DELETE CASCADE,
        installed_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
        installed_by_employee_id BIGINT REFERENCES employees(id)
      );
    $SQL$;
  ELSE
    RAISE NOTICE 'licenses/software_licenses table not found; creating asset_software_installs WITHOUT FK. Add FK later.';
    EXECUTE $SQL$
      CREATE TABLE IF NOT EXISTS asset_software_installs (
        id BIGSERIAL PRIMARY KEY,
        asset_id BIGINT NOT NULL REFERENCES assets(id) ON DELETE CASCADE,
        license_id BIGINT NOT NULL,
        installed_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
        installed_by_employee_id BIGINT REFERENCES employees(id)
      );
    $SQL$;
  END IF;
END$$;

