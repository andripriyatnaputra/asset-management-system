-- 1) Master kategori & layanan (aman jika sudah ada)
CREATE TABLE IF NOT EXISTS ticket_categories (
  id BIGSERIAL PRIMARY KEY,
  code TEXT UNIQUE NOT NULL,
  name TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS services (
  id BIGSERIAL PRIMARY KEY,
  code TEXT UNIQUE NOT NULL,
  name TEXT NOT NULL
);

-- 2) SLA policies
CREATE TABLE IF NOT EXISTS sla_policies (
  id BIGSERIAL PRIMARY KEY,
  name TEXT NOT NULL,
  category_code TEXT REFERENCES ticket_categories(code) ON DELETE SET NULL,
  service_code  TEXT REFERENCES services(code) ON DELETE SET NULL,
  impact TEXT NOT NULL CHECK (impact IN ('Low','Medium','High')),
  urgency TEXT NOT NULL CHECK (urgency IN ('Low','Medium','High')),
  resulting_priority TEXT NOT NULL CHECK (resulting_priority IN ('Low','Medium','High','Critical')),
  response_minutes INT NOT NULL,
  resolve_minutes  INT NOT NULL,
  is_active BOOLEAN NOT NULL DEFAULT TRUE
);

-- 3) Kolom tambahan di tickets
ALTER TABLE tickets
  ADD COLUMN IF NOT EXISTS category_code TEXT REFERENCES ticket_categories(code) ON DELETE SET NULL,
  ADD COLUMN IF NOT EXISTS service_code  TEXT REFERENCES services(code) ON DELETE SET NULL,
  ADD COLUMN IF NOT EXISTS impact TEXT CHECK (impact IN ('Low','Medium','High')),
  ADD COLUMN IF NOT EXISTS urgency TEXT CHECK (urgency IN ('Low','Medium','High')),
  ADD COLUMN IF NOT EXISTS sla_policy_id BIGINT REFERENCES sla_policies(id) ON DELETE SET NULL,
  ADD COLUMN IF NOT EXISTS sla_due_at TIMESTAMPTZ,
  ADD COLUMN IF NOT EXISTS sla_breached_at TIMESTAMPTZ;

-- 4) Lampiran komentar
CREATE TABLE IF NOT EXISTS ticket_attachments (
  id BIGSERIAL PRIMARY KEY,
  ticket_id BIGINT NOT NULL REFERENCES tickets(id) ON DELETE CASCADE,
  comment_id BIGINT REFERENCES ticket_comments(id) ON DELETE CASCADE,
  filename TEXT NOT NULL,
  path TEXT NOT NULL,
  mime_type TEXT,
  size BIGINT,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Seed minimal (opsional)
INSERT INTO ticket_categories (code,name) VALUES
('INCIDENT','Incident'),('REQUEST','Service Request')
ON CONFLICT (code) DO NOTHING;

INSERT INTO services (code,name) VALUES
('EMAIL','Email Service'),('NETWORK','Network'),('ENDPOINT','End-user Device')
ON CONFLICT (code) DO NOTHING;

INSERT INTO sla_policies (name,category_code,service_code,impact,urgency,resulting_priority,response_minutes,resolve_minutes, is_active)
VALUES
('Network Major', 'INCIDENT', 'NETWORK', 'High', 'High', 'Critical', 15, 240, TRUE),
('Network Minor', 'INCIDENT', 'NETWORK', 'Medium', 'Low', 'Medium', 240, 2880, TRUE)
ON CONFLICT DO NOTHING;
