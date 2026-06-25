-- ============================================================
-- PRODUCTION CATCHUP MIGRATION
-- Dari: 14 tabel (versi awal)
-- Ke  : 63 tabel (versi current)
--
-- AMAN dijalankan berulang kali (idempotent).
-- Jalankan dengan: psql "$DATABASE_URL" -f V001__production_catchup.up.sql
-- ============================================================

BEGIN;

-- ════════════════════════════════════════════════════════════
-- STEP 1: Custom types
-- ════════════════════════════════════════════════════════════
DO $$ BEGIN
  CREATE TYPE public.asset_status AS ENUM (
    'in_stock', 'assigned', 'maintenance', 'retired', 'disposed'
  );
EXCEPTION WHEN duplicate_object THEN NULL;
END $$;

-- ════════════════════════════════════════════════════════════
-- STEP 2: Sequences untuk tabel baru
-- ════════════════════════════════════════════════════════════
CREATE SEQUENCE IF NOT EXISTS public.alerts_id_seq START 1;
CREATE SEQUENCE IF NOT EXISTS public.asset_history_id_seq START 1;
CREATE SEQUENCE IF NOT EXISTS public.audit_logs_id_seq START 1;
CREATE SEQUENCE IF NOT EXISTS public.budget_alerts_id_seq START 1;
CREATE SEQUENCE IF NOT EXISTS public.compliance_alerts_id_seq START 1;
CREATE SEQUENCE IF NOT EXISTS public.compliance_trend_id_seq START 1;
CREATE SEQUENCE IF NOT EXISTS public.cost_centers_id_seq START 1;
CREATE SEQUENCE IF NOT EXISTS public.email_logs_id_seq START 1;
CREATE SEQUENCE IF NOT EXISTS public.employee_trainings_id_seq START 1;
CREATE SEQUENCE IF NOT EXISTS public.governance_review_feedback_id_seq START 1;
CREATE SEQUENCE IF NOT EXISTS public.kg_edges_id_seq START 1;
CREATE SEQUENCE IF NOT EXISTS public.kg_nodes_id_seq START 1;
CREATE SEQUENCE IF NOT EXISTS public.locations_id_seq START 1;
CREATE SEQUENCE IF NOT EXISTS public.ml_calibration_models_id_seq START 1;
CREATE SEQUENCE IF NOT EXISTS public.role_delegations_id_seq START 1;
CREATE SEQUENCE IF NOT EXISTS public.services_id_seq START 1;
CREATE SEQUENCE IF NOT EXISTS public.sla_policies_id_seq START 1;
CREATE SEQUENCE IF NOT EXISTS public.ticket_attachments_id_seq START 1;
CREATE SEQUENCE IF NOT EXISTS public.ticket_categories_id_seq START 1;

-- ════════════════════════════════════════════════════════════
-- STEP 3: Rename software_licenses → licenses
--         (hanya jika belum direname)
-- ════════════════════════════════════════════════════════════
DO $$ BEGIN
  IF EXISTS (
    SELECT 1 FROM information_schema.tables
    WHERE table_schema = 'public' AND table_name = 'software_licenses'
  ) AND NOT EXISTS (
    SELECT 1 FROM information_schema.tables
    WHERE table_schema = 'public' AND table_name = 'licenses'
  ) THEN
    ALTER TABLE public.software_licenses RENAME TO licenses;
    RAISE NOTICE 'Renamed software_licenses → licenses';
  ELSE
    RAISE NOTICE 'Rename skipped (already done or licenses table exists)';
  END IF;
END $$;

-- Tambah kolom baru ke licenses (yang sebelumnya software_licenses)
ALTER TABLE public.licenses
  ADD COLUMN IF NOT EXISTS vendor               TEXT,
  ADD COLUMN IF NOT EXISTS publisher            TEXT,
  ADD COLUMN IF NOT EXISTS version              TEXT,
  ADD COLUMN IF NOT EXISTS license_type         TEXT,
  ADD COLUMN IF NOT EXISTS license_model        TEXT,
  ADD COLUMN IF NOT EXISTS contract_id          BIGINT,
  ADD COLUMN IF NOT EXISTS category             TEXT,
  ADD COLUMN IF NOT EXISTS metric               TEXT,
  ADD COLUMN IF NOT EXISTS maintenance_expiry   DATE,
  ADD COLUMN IF NOT EXISTS compliance_status    TEXT DEFAULT 'unknown',
  ADD COLUMN IF NOT EXISTS verification_date    TIMESTAMPTZ,
  ADD COLUMN IF NOT EXISTS updated_at           TIMESTAMPTZ DEFAULT now(),
  ADD COLUMN IF NOT EXISTS updated_by           BIGINT,
  ADD COLUMN IF NOT EXISTS entitlement_doc      TEXT,
  ADD COLUMN IF NOT EXISTS procurement_reference TEXT,
  ADD COLUMN IF NOT EXISTS budget_id            BIGINT,
  ADD COLUMN IF NOT EXISTS currency             VARCHAR(10) DEFAULT 'IDR',
  ADD COLUMN IF NOT EXISTS compliance_score     NUMERIC(5,2),
  ADD COLUMN IF NOT EXISTS created_by           BIGINT,
  ADD COLUMN IF NOT EXISTS document_hash        TEXT;

-- Tambah constraint compliance_status (jika belum ada)
DO $$ BEGIN
  ALTER TABLE public.licenses ADD CONSTRAINT licenses_compliance_status_check
    CHECK (compliance_status = ANY (ARRAY['compliant','non-compliant','unknown']));
EXCEPTION WHEN duplicate_object THEN NULL;
END $$;

-- ════════════════════════════════════════════════════════════
-- STEP 4: Tambah kolom ke tabel yang sudah ada
-- ════════════════════════════════════════════════════════════

-- assets
ALTER TABLE public.assets
  ADD COLUMN IF NOT EXISTS department_id        BIGINT,
  ADD COLUMN IF NOT EXISTS cost_center_id       BIGINT,
  ADD COLUMN IF NOT EXISTS location_id          BIGINT,
  ADD COLUMN IF NOT EXISTS purchase_cost        NUMERIC(14,2) DEFAULT 0,
  ADD COLUMN IF NOT EXISTS vendor               TEXT,
  ADD COLUMN IF NOT EXISTS warranty_expiry      DATE,
  ADD COLUMN IF NOT EXISTS useful_life_months   INT DEFAULT 36,
  ADD COLUMN IF NOT EXISTS depreciation_method  TEXT DEFAULT 'straight_line',
  ADD COLUMN IF NOT EXISTS salvage_value        NUMERIC(14,2) DEFAULT 0,
  ADD COLUMN IF NOT EXISTS serial_number        TEXT,
  ADD COLUMN IF NOT EXISTS asset_condition      TEXT DEFAULT 'good',
  ADD COLUMN IF NOT EXISTS acquisition_type     TEXT DEFAULT 'purchase',
  ADD COLUMN IF NOT EXISTS ownership_type       TEXT DEFAULT 'company_owned',
  ADD COLUMN IF NOT EXISTS disposal_date        DATE,
  ADD COLUMN IF NOT EXISTS disposed             BOOLEAN DEFAULT false,
  ADD COLUMN IF NOT EXISTS notes                TEXT,
  ADD COLUMN IF NOT EXISTS budget_id            BIGINT,
  ADD COLUMN IF NOT EXISTS contract_id          BIGINT,
  ADD COLUMN IF NOT EXISTS lifecycle_stage      VARCHAR(30) DEFAULT 'in_use',
  ADD COLUMN IF NOT EXISTS asset_criticality    VARCHAR(20),
  ADD COLUMN IF NOT EXISTS disposed_approved_by BIGINT,
  ADD COLUMN IF NOT EXISTS compliance_flag      BOOLEAN DEFAULT true,
  ADD COLUMN IF NOT EXISTS compliance_note      TEXT,
  ADD COLUMN IF NOT EXISTS verified_at          TIMESTAMPTZ,
  ADD COLUMN IF NOT EXISTS lifecycle_status     VARCHAR(20) DEFAULT 'active',
  ADD COLUMN IF NOT EXISTS asset_health_score   NUMERIC(5,2),
  ADD COLUMN IF NOT EXISTS created_by           BIGINT,
  ADD COLUMN IF NOT EXISTS updated_by           BIGINT,
  ADD COLUMN IF NOT EXISTS currency             VARCHAR(10) DEFAULT 'IDR',
  ADD COLUMN IF NOT EXISTS governance_score     NUMERIC(5,2) DEFAULT 0,
  ADD COLUMN IF NOT EXISTS updated_month        DATE;

-- employees
ALTER TABLE public.employees
  ADD COLUMN IF NOT EXISTS created_at   TIMESTAMPTZ DEFAULT now(),
  ADD COLUMN IF NOT EXISTS updated_at   TIMESTAMPTZ DEFAULT now(),
  ADD COLUMN IF NOT EXISTS last_login_at TIMESTAMPTZ;

-- departments
ALTER TABLE public.departments
  ADD COLUMN IF NOT EXISTS created_at   TIMESTAMPTZ DEFAULT now(),
  ADD COLUMN IF NOT EXISTS updated_at   TIMESTAMPTZ DEFAULT now(),
  ADD COLUMN IF NOT EXISTS deleted_at   TIMESTAMPTZ,
  ADD COLUMN IF NOT EXISTS manager_id   BIGINT,
  ADD COLUMN IF NOT EXISTS created_by   BIGINT,
  ADD COLUMN IF NOT EXISTS updated_by   BIGINT,
  ADD COLUMN IF NOT EXISTS cost_center_id BIGINT;

-- budgets
ALTER TABLE public.budgets
  ADD COLUMN IF NOT EXISTS category     VARCHAR(20) DEFAULT 'CAPEX',
  ADD COLUMN IF NOT EXISTS currency     VARCHAR(10) DEFAULT 'IDR',
  ADD COLUMN IF NOT EXISTS budget_year  INT,
  ADD COLUMN IF NOT EXISTS approved_by  BIGINT,
  ADD COLUMN IF NOT EXISTS used_amount  NUMERIC(15,2) DEFAULT 0,
  ADD COLUMN IF NOT EXISTS updated_at   TIMESTAMPTZ DEFAULT now(),
  ADD COLUMN IF NOT EXISTS cost_center_id BIGINT;

-- budget_transactions
ALTER TABLE public.budget_transactions
  ADD COLUMN IF NOT EXISTS contract_id    BIGINT,
  ADD COLUMN IF NOT EXISTS license_id     BIGINT,
  ADD COLUMN IF NOT EXISTS created_by     BIGINT,
  ADD COLUMN IF NOT EXISTS updated_at     TIMESTAMPTZ DEFAULT now(),
  ADD COLUMN IF NOT EXISTS entity_type    TEXT,
  ADD COLUMN IF NOT EXISTS entity_id      BIGINT,
  ADD COLUMN IF NOT EXISTS currency       VARCHAR(10) DEFAULT 'IDR',
  ADD COLUMN IF NOT EXISTS exchange_rate  NUMERIC(18,6),
  ADD COLUMN IF NOT EXISTS tax_amount     NUMERIC(18,2),
  ADD COLUMN IF NOT EXISTS category       VARCHAR(20),
  ADD COLUMN IF NOT EXISTS cost_center_id BIGINT;

-- asset_assignments
ALTER TABLE public.asset_assignments
  ADD COLUMN IF NOT EXISTS assigned_by_employee_id  BIGINT,
  ADD COLUMN IF NOT EXISTS returned_by_employee_id  BIGINT,
  ADD COLUMN IF NOT EXISTS status                   VARCHAR(20) DEFAULT 'active';

-- tickets (banyak kolom baru untuk SLA, ITSM)
ALTER TABLE public.tickets
  ADD COLUMN IF NOT EXISTS category_code          TEXT,
  ADD COLUMN IF NOT EXISTS service_code           TEXT,
  ADD COLUMN IF NOT EXISTS impact                 TEXT,
  ADD COLUMN IF NOT EXISTS urgency                TEXT,
  ADD COLUMN IF NOT EXISTS sla_policy_id          BIGINT,
  ADD COLUMN IF NOT EXISTS sla_due_at             TIMESTAMPTZ,
  ADD COLUMN IF NOT EXISTS sla_breached_at        TIMESTAMPTZ,
  ADD COLUMN IF NOT EXISTS response_due_at        TIMESTAMPTZ,
  ADD COLUMN IF NOT EXISTS last_assigned_at       TIMESTAMPTZ,
  ADD COLUMN IF NOT EXISTS last_assigned_by       BIGINT,
  ADD COLUMN IF NOT EXISTS updated_by             BIGINT,
  ADD COLUMN IF NOT EXISTS last_status_changed_at TIMESTAMPTZ,
  ADD COLUMN IF NOT EXISTS due_date               TIMESTAMPTZ,
  ADD COLUMN IF NOT EXISTS resolved_at            TIMESTAMPTZ,
  ADD COLUMN IF NOT EXISTS closed_at              TIMESTAMPTZ,
  ADD COLUMN IF NOT EXISTS category_tier          TEXT,
  ADD COLUMN IF NOT EXISTS linked_problem_id      BIGINT,
  ADD COLUMN IF NOT EXISTS escalation_level       INT DEFAULT 0,
  ADD COLUMN IF NOT EXISTS breach_flag            BOOLEAN DEFAULT false,
  ADD COLUMN IF NOT EXISTS compliance_flag        BOOLEAN DEFAULT false,
  ADD COLUMN IF NOT EXISTS compliance_score       NUMERIC(5,2),
  ADD COLUMN IF NOT EXISTS response_completed_at  TIMESTAMPTZ,
  ADD COLUMN IF NOT EXISTS response_time_minutes  INT,
  ADD COLUMN IF NOT EXISTS resolution_time_minutes INT,
  ADD COLUMN IF NOT EXISTS change_request_id      BIGINT,
  ADD COLUMN IF NOT EXISTS ticket_type            VARCHAR(30) DEFAULT 'incident';

-- ticket_comments
ALTER TABLE public.ticket_comments
  ADD COLUMN IF NOT EXISTS is_internal BOOLEAN DEFAULT false,
  ADD COLUMN IF NOT EXISTS updated_at  TIMESTAMPTZ DEFAULT now();

-- ════════════════════════════════════════════════════════════
-- STEP 5: Buat tabel baru (tanpa IF NOT EXISTS di init.sql asli)
--         Semua dibuat safe dengan IF NOT EXISTS di sini
-- ════════════════════════════════════════════════════════════

-- Prereq: cost_centers (direferensikan banyak tabel lain)
CREATE TABLE IF NOT EXISTS public.cost_centers (
  id         BIGINT NOT NULL DEFAULT nextval('public.cost_centers_id_seq'),
  code       TEXT   NOT NULL,
  name       TEXT   NOT NULL,
  deleted_at TIMESTAMPTZ,
  created_at TIMESTAMPTZ DEFAULT now(),
  updated_at TIMESTAMPTZ DEFAULT now(),
  created_by BIGINT,
  updated_by BIGINT,
  CONSTRAINT cost_centers_pkey PRIMARY KEY (id),
  CONSTRAINT cost_centers_code_key UNIQUE (code)
);

-- Prereq: locations
CREATE TABLE IF NOT EXISTS public.locations (
  id          BIGINT NOT NULL DEFAULT nextval('public.locations_id_seq'),
  site        TEXT   NOT NULL,
  building    TEXT,
  room        TEXT,
  description TEXT,
  status      VARCHAR(20) DEFAULT 'active',
  created_at  TIMESTAMPTZ DEFAULT now(),
  updated_at  TIMESTAMPTZ DEFAULT now(),
  created_by  BIGINT,
  updated_by  BIGINT,
  deleted_at  TIMESTAMPTZ,
  parent_id   BIGINT,
  CONSTRAINT locations_pkey PRIMARY KEY (id),
  CONSTRAINT chk_locations_parent_not_self CHECK (parent_id IS NULL OR parent_id <> id)
);

-- Prereq: services & ticket_categories (untuk FK di sla_policies & tickets)
CREATE TABLE IF NOT EXISTS public.services (
  id   BIGINT NOT NULL DEFAULT nextval('public.services_id_seq'),
  code TEXT   NOT NULL,
  name TEXT   NOT NULL,
  CONSTRAINT services_pkey PRIMARY KEY (id),
  CONSTRAINT services_code_key UNIQUE (code)
);

CREATE TABLE IF NOT EXISTS public.ticket_categories (
  id   BIGINT NOT NULL DEFAULT nextval('public.ticket_categories_id_seq'),
  code TEXT   NOT NULL,
  name TEXT   NOT NULL,
  CONSTRAINT ticket_categories_pkey PRIMARY KEY (id),
  CONSTRAINT ticket_categories_code_key UNIQUE (code)
);

-- Prereq: contracts (direferensikan licenses, assets)
CREATE TABLE IF NOT EXISTS public.contracts (
  id                       BIGINT NOT NULL GENERATED ALWAYS AS IDENTITY,
  contract_number          TEXT   NOT NULL,
  vendor                   TEXT,
  contract_type            TEXT,
  start_date               DATE   NOT NULL,
  end_date                 DATE,
  total_value              NUMERIC(15,2),
  currency                 VARCHAR(10) DEFAULT 'IDR',
  payment_terms            TEXT,
  contact_person           TEXT,
  contact_email            TEXT,
  attachment_url           TEXT,
  notes                    TEXT,
  status                   TEXT DEFAULT 'active',
  created_at               TIMESTAMPTZ DEFAULT now(),
  updated_at               TIMESTAMPTZ DEFAULT now(),
  updated_by               BIGINT,
  deleted_at               TIMESTAMPTZ,
  budget_id                BIGINT,
  renewal_date             DATE,
  termination_notice_days  INT DEFAULT 30,
  created_by               BIGINT,
  cost_center_id           BIGINT,
  CONSTRAINT contracts_pkey PRIMARY KEY (id),
  CONSTRAINT contracts_status_check CHECK (status = ANY (ARRAY['active','expired','terminated']))
);

-- sla_policies
CREATE TABLE IF NOT EXISTS public.sla_policies (
  id                      BIGINT NOT NULL DEFAULT nextval('public.sla_policies_id_seq'),
  name                    TEXT   NOT NULL,
  category_code           TEXT,
  service_code            TEXT,
  impact                  TEXT   NOT NULL,
  urgency                 TEXT   NOT NULL,
  resulting_priority      TEXT   NOT NULL,
  response_minutes        INT    NOT NULL,
  resolve_minutes         INT    NOT NULL,
  is_active               BOOLEAN DEFAULT true NOT NULL,
  compliance_score        NUMERIC(5,2) DEFAULT 100,
  legacy_compliance_score DOUBLE PRECISION,
  deleted_at              TIMESTAMPTZ,
  created_by              BIGINT,
  updated_by              BIGINT,
  created_at              TIMESTAMPTZ DEFAULT now(),
  updated_at              TIMESTAMPTZ DEFAULT now(),
  CONSTRAINT sla_policies_pkey PRIMARY KEY (id),
  CONSTRAINT unique_sla_combo UNIQUE (impact, urgency, category_code, service_code),
  CONSTRAINT sla_policies_impact_check CHECK (impact = ANY (ARRAY['Low','Medium','High'])),
  CONSTRAINT sla_policies_urgency_check CHECK (urgency = ANY (ARRAY['Low','Medium','High'])),
  CONSTRAINT sla_policies_resulting_priority_check CHECK (resulting_priority = ANY (ARRAY['Low','Medium','High','Critical']))
);

-- alerts
CREATE TABLE IF NOT EXISTS public.alerts (
  id              BIGINT NOT NULL DEFAULT nextval('public.alerts_id_seq'),
  message         TEXT   NOT NULL,
  severity        VARCHAR(20) DEFAULT 'info' NOT NULL,
  category        VARCHAR(50) DEFAULT 'system',
  acknowledged    BOOLEAN DEFAULT false,
  created_at      TIMESTAMPTZ DEFAULT now(),
  acknowledged_by BIGINT,
  asset_id        BIGINT,
  CONSTRAINT alerts_pkey PRIMARY KEY (id)
);

-- asset_history
CREATE TABLE IF NOT EXISTS public.asset_history (
  id                BIGINT NOT NULL DEFAULT nextval('public.asset_history_id_seq'),
  asset_id          BIGINT NOT NULL,
  action            TEXT   NOT NULL,
  detail            TEXT,
  actor_employee_id BIGINT,
  from_status       public.asset_status,
  to_status         public.asset_status,
  created_at        TIMESTAMPTZ DEFAULT now() NOT NULL,
  compliance_flag   BOOLEAN DEFAULT true,
  compliance_note   TEXT,
  hash              TEXT,
  CONSTRAINT asset_history_pkey PRIMARY KEY (id)
);

-- audit_logs
CREATE TABLE IF NOT EXISTS public.audit_logs (
  id           BIGINT NOT NULL DEFAULT nextval('public.audit_logs_id_seq'),
  actor_id     BIGINT,
  entity_name  VARCHAR(50) NOT NULL,
  entity_id    BIGINT,
  action       VARCHAR(100) NOT NULL,
  changes      JSONB,
  created_at   TIMESTAMPTZ DEFAULT now(),
  ip_address   TEXT,
  user_agent   TEXT,
  request_path TEXT,
  severity     VARCHAR(100),
  category     VARCHAR(50),
  hash         TEXT,
  prev_hash    TEXT,
  CONSTRAINT audit_logs_pkey PRIMARY KEY (id)
);

-- budget_alerts
CREATE TABLE IF NOT EXISTS public.budget_alerts (
  id         BIGINT      NOT NULL DEFAULT nextval('public.budget_alerts_id_seq'),
  budget_id  BIGINT      NOT NULL,
  usage_pct  NUMERIC(5,2) NOT NULL,
  alerted_at TIMESTAMPTZ  NOT NULL DEFAULT now(),
  CONSTRAINT budget_alerts_pkey PRIMARY KEY (id)
);

-- compliance_alerts
CREATE TABLE IF NOT EXISTS public.compliance_alerts (
  id         BIGINT NOT NULL DEFAULT nextval('public.compliance_alerts_id_seq'),
  message    TEXT   NOT NULL,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  CONSTRAINT compliance_alerts_pkey PRIMARY KEY (id)
);

-- compliance_trend
CREATE TABLE IF NOT EXISTS public.compliance_trend (
  id         BIGINT      NOT NULL DEFAULT nextval('public.compliance_trend_id_seq'),
  last_value NUMERIC(5,2) NOT NULL,
  created_at TIMESTAMPTZ  NOT NULL DEFAULT now(),
  CONSTRAINT compliance_trend_pkey PRIMARY KEY (id)
);

-- data_governance
CREATE TABLE IF NOT EXISTS public.data_governance (
  entity_name       VARCHAR(100) NOT NULL,
  owner_employee_id BIGINT,
  retention_period  INTERVAL DEFAULT '5 years',
  last_reviewed_at  TIMESTAMPTZ DEFAULT now(),
  notes             TEXT,
  CONSTRAINT data_governance_pkey PRIMARY KEY (entity_name)
);

-- email_logs
CREATE TABLE IF NOT EXISTS public.email_logs (
  id            BIGINT NOT NULL DEFAULT nextval('public.email_logs_id_seq'),
  recipient     TEXT   NOT NULL,
  subject       TEXT   NOT NULL,
  body_preview  TEXT,
  status        VARCHAR(20) DEFAULT 'SENT' NOT NULL,
  error_message TEXT,
  created_at    TIMESTAMPTZ NOT NULL DEFAULT now(),
  CONSTRAINT email_logs_pkey PRIMARY KEY (id)
);

-- employee_trainings
CREATE TABLE IF NOT EXISTS public.employee_trainings (
  id              BIGINT NOT NULL DEFAULT nextval('public.employee_trainings_id_seq'),
  employee_id     BIGINT NOT NULL,
  training_name   VARCHAR(255) NOT NULL,
  certificate_url TEXT,
  completed_at    DATE,
  created_at      TIMESTAMPTZ DEFAULT now(),
  CONSTRAINT employee_trainings_pkey PRIMARY KEY (id)
);

-- governance_review_feedback
CREATE TABLE IF NOT EXISTS public.governance_review_feedback (
  id                BIGINT       NOT NULL DEFAULT nextval('public.governance_review_feedback_id_seq'),
  asset_id          BIGINT       NOT NULL,
  reviewer_id       BIGINT,
  risk_index        NUMERIC(6,2) NOT NULL,
  system_note       TEXT,
  reviewer_comment  TEXT,
  reviewer_decision BOOLEAN,
  created_at        TIMESTAMPTZ DEFAULT now(),
  updated_at        TIMESTAMPTZ DEFAULT now(),
  CONSTRAINT governance_review_feedback_pkey PRIMARY KEY (id)
);

-- kg_nodes + kg_edges
CREATE TABLE IF NOT EXISTS public.kg_nodes (
  id          BIGINT NOT NULL DEFAULT nextval('public.kg_nodes_id_seq'),
  entity_type VARCHAR(30) NOT NULL,
  entity_id   BIGINT      NOT NULL,
  label       TEXT        NOT NULL,
  props       JSONB       NOT NULL DEFAULT '{}',
  created_at  TIMESTAMPTZ DEFAULT now(),
  updated_at  TIMESTAMPTZ DEFAULT now(),
  CONSTRAINT kg_nodes_pkey PRIMARY KEY (id),
  CONSTRAINT kg_nodes_entity_type_entity_id_key UNIQUE (entity_type, entity_id)
);

CREATE TABLE IF NOT EXISTS public.kg_edges (
  id          BIGINT NOT NULL DEFAULT nextval('public.kg_edges_id_seq'),
  src_node_id BIGINT NOT NULL,
  dst_node_id BIGINT NOT NULL,
  rel_type    VARCHAR(40) NOT NULL,
  weight      NUMERIC(6,3) DEFAULT 1,
  props       JSONB NOT NULL DEFAULT '{}',
  created_at  TIMESTAMPTZ DEFAULT now(),
  updated_at  TIMESTAMPTZ DEFAULT now(),
  CONSTRAINT kg_edges_pkey PRIMARY KEY (id)
);

-- ml_calibration_models
CREATE TABLE IF NOT EXISTS public.ml_calibration_models (
  id             BIGINT NOT NULL DEFAULT nextval('public.ml_calibration_models_id_seq'),
  model_name     VARCHAR(100) NOT NULL,
  last_trained_at TIMESTAMPTZ DEFAULT now(),
  total_samples  INT DEFAULT 0,
  avg_error      NUMERIC(6,3) DEFAULT 0,
  parameters     JSONB DEFAULT '{}',
  created_at     TIMESTAMPTZ DEFAULT now(),
  updated_at     TIMESTAMPTZ DEFAULT now(),
  CONSTRAINT ml_calibration_models_pkey PRIMARY KEY (id)
);

-- problems
CREATE TABLE IF NOT EXISTS public.problems (
  id                 BIGINT NOT NULL GENERATED ALWAYS AS IDENTITY,
  title              TEXT,
  description        TEXT,
  status             TEXT DEFAULT 'Open',
  created_at         TIMESTAMPTZ DEFAULT now(),
  priority           VARCHAR(20) DEFAULT 'Medium' NOT NULL,
  assigned_to        BIGINT,
  created_by         BIGINT,
  updated_by         BIGINT,
  root_cause         TEXT,
  workaround         TEXT,
  known_error        BOOLEAN DEFAULT false NOT NULL,
  permanent_solution TEXT,
  related_asset_id   BIGINT,
  updated_at         TIMESTAMPTZ DEFAULT now() NOT NULL,
  resolved_at        TIMESTAMPTZ,
  deleted_at         TIMESTAMPTZ,
  CONSTRAINT problems_pkey PRIMARY KEY (id),
  CONSTRAINT problems_status_check CHECK (status = ANY (ARRAY['Open','Under Investigation','Known Error','Resolved','Closed'])),
  CONSTRAINT problems_priority_check CHECK (priority = ANY (ARRAY['Low','Medium','High','Critical']))
);

-- role_delegations
CREATE TABLE IF NOT EXISTS public.role_delegations (
  id            BIGINT NOT NULL DEFAULT nextval('public.role_delegations_id_seq'),
  delegator_id  BIGINT,
  delegatee_id  BIGINT,
  role_override VARCHAR(50) NOT NULL,
  start_date    DATE NOT NULL,
  end_date      DATE NOT NULL,
  created_at    TIMESTAMPTZ DEFAULT now(),
  is_active     BOOLEAN DEFAULT true NOT NULL,
  reason        TEXT,
  revoked_at    TIMESTAMPTZ,
  revoked_by    BIGINT,
  CONSTRAINT role_delegations_pkey PRIMARY KEY (id)
);

-- ticket_attachments
CREATE TABLE IF NOT EXISTS public.ticket_attachments (
  id         BIGINT NOT NULL DEFAULT nextval('public.ticket_attachments_id_seq'),
  ticket_id  BIGINT NOT NULL,
  comment_id BIGINT,
  filename   TEXT NOT NULL,
  path       TEXT NOT NULL,
  mime_type  TEXT,
  size       BIGINT,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  deleted_at TIMESTAMP,
  CONSTRAINT ticket_attachments_pkey PRIMARY KEY (id)
);

-- ════════════════════════════════════════════════════════════
-- STEP 6: Tabel modul ITSM baru (semua IF NOT EXISTS)
-- ════════════════════════════════════════════════════════════

CREATE TABLE IF NOT EXISTS public.service_catalog (
  id                      BIGSERIAL PRIMARY KEY,
  code                    VARCHAR(50) NOT NULL UNIQUE,
  name                    VARCHAR(255) NOT NULL,
  category                VARCHAR(100),
  description             TEXT,
  sla_policy_id           BIGINT REFERENCES public.sla_policies(id) ON DELETE SET NULL,
  approval_required       BOOLEAN NOT NULL DEFAULT false,
  fulfillment_sla_minutes INT,
  is_active               BOOLEAN NOT NULL DEFAULT true,
  created_by              BIGINT REFERENCES public.employees(id),
  created_at              TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at              TIMESTAMPTZ NOT NULL DEFAULT now(),
  deleted_at              TIMESTAMPTZ
);

CREATE TABLE IF NOT EXISTS public.service_requests (
  id                 BIGSERIAL PRIMARY KEY,
  sr_number          VARCHAR(50) NOT NULL UNIQUE,
  service_catalog_id BIGINT REFERENCES public.service_catalog(id),
  subject            VARCHAR(255) NOT NULL,
  description        TEXT,
  status             VARCHAR(30) NOT NULL DEFAULT 'New',
  priority           VARCHAR(20) NOT NULL DEFAULT 'Medium',
  requested_by       BIGINT NOT NULL REFERENCES public.employees(id),
  assigned_to        BIGINT REFERENCES public.employees(id),
  department_id      BIGINT REFERENCES public.departments(id),
  fulfilled_at       TIMESTAMPTZ,
  created_at         TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at         TIMESTAMPTZ NOT NULL DEFAULT now(),
  deleted_at         TIMESTAMPTZ,
  category           VARCHAR(100),
  notes              TEXT
);

CREATE TABLE IF NOT EXISTS public.approval_workflows (
  id          BIGSERIAL PRIMARY KEY,
  entity_type VARCHAR(50) NOT NULL,
  entity_id   BIGINT NOT NULL,
  level       INT NOT NULL DEFAULT 1,
  approver_id BIGINT REFERENCES public.employees(id),
  status      VARCHAR(20) NOT NULL DEFAULT 'pending',
  decided_at  TIMESTAMPTZ,
  notes       TEXT,
  created_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS public.change_requests (
  id                   BIGSERIAL PRIMARY KEY,
  cr_number            VARCHAR(50) NOT NULL UNIQUE,
  title                VARCHAR(255) NOT NULL,
  description          TEXT,
  type                 VARCHAR(20) NOT NULL DEFAULT 'Normal',
  status               VARCHAR(30) NOT NULL DEFAULT 'Draft',
  risk_level           VARCHAR(20) NOT NULL DEFAULT 'Medium',
  impact_description   TEXT,
  rollback_plan        TEXT,
  change_window_start  TIMESTAMPTZ,
  change_window_end    TIMESTAMPTZ,
  planned_date         DATE,
  actual_date          DATE,
  created_by           BIGINT REFERENCES public.employees(id),
  approver_id          BIGINT REFERENCES public.employees(id),
  implemented_by       BIGINT REFERENCES public.employees(id),
  created_at           TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at           TIMESTAMPTZ NOT NULL DEFAULT now(),
  deleted_at           TIMESTAMPTZ
);

CREATE TABLE IF NOT EXISTS public.change_approvals (
  id           BIGSERIAL PRIMARY KEY,
  change_id    BIGINT NOT NULL REFERENCES public.change_requests(id) ON DELETE CASCADE,
  approver_id  BIGINT REFERENCES public.employees(id),
  decision     VARCHAR(20),
  notes        TEXT,
  decided_at   TIMESTAMPTZ,
  created_at   TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS public.change_tasks (
  id           BIGSERIAL PRIMARY KEY,
  change_id    BIGINT NOT NULL REFERENCES public.change_requests(id) ON DELETE CASCADE,
  title        VARCHAR(255) NOT NULL,
  description  TEXT,
  assigned_to  BIGINT REFERENCES public.employees(id),
  status       VARCHAR(20) NOT NULL DEFAULT 'pending',
  due_date     DATE,
  completed_at TIMESTAMPTZ,
  created_at   TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS public.problem_incidents (
  id         BIGSERIAL PRIMARY KEY,
  problem_id BIGINT NOT NULL REFERENCES public.problems(id) ON DELETE CASCADE,
  ticket_id  BIGINT NOT NULL REFERENCES public.tickets(id) ON DELETE CASCADE,
  linked_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS public.incident_postmortems (
  id              BIGSERIAL PRIMARY KEY,
  ticket_id       BIGINT REFERENCES public.tickets(id),
  problem_id      BIGINT REFERENCES public.problems(id),
  timeline        TEXT,
  root_cause      TEXT,
  contributing_factors TEXT,
  impact_summary  TEXT,
  action_items    JSONB DEFAULT '[]',
  reviewed_by     BIGINT REFERENCES public.employees(id),
  reviewed_at     TIMESTAMPTZ,
  created_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at      TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS public.escalation_rules (
  id              BIGSERIAL PRIMARY KEY,
  name            VARCHAR(255) NOT NULL,
  priority        VARCHAR(20) NOT NULL DEFAULT 'Medium',
  category_code   VARCHAR(50),
  breach_minutes  INT NOT NULL,
  escalate_to     BIGINT REFERENCES public.employees(id),
  notify_email    TEXT,
  is_active       BOOLEAN NOT NULL DEFAULT true,
  created_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at      TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS public.webhook_subscriptions (
  id         BIGSERIAL PRIMARY KEY,
  name       VARCHAR(255) NOT NULL,
  url        TEXT NOT NULL,
  events     TEXT[] NOT NULL DEFAULT '{}',
  secret     TEXT,
  is_active  BOOLEAN NOT NULL DEFAULT true,
  created_by BIGINT REFERENCES public.employees(id),
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS public.webhook_delivery_logs (
  id              BIGSERIAL PRIMARY KEY,
  subscription_id BIGINT NOT NULL REFERENCES public.webhook_subscriptions(id) ON DELETE CASCADE,
  event_type      VARCHAR(100) NOT NULL,
  payload         TEXT NOT NULL,
  status          VARCHAR(20) NOT NULL DEFAULT 'pending',
  response_code   INT,
  response_body   TEXT,
  attempt_count   INT NOT NULL DEFAULT 0,
  last_attempt_at TIMESTAMPTZ,
  created_at      TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS public.asset_qr_codes (
  id         BIGSERIAL PRIMARY KEY,
  asset_id   BIGINT NOT NULL REFERENCES public.assets(id) ON DELETE CASCADE,
  qr_data    TEXT NOT NULL,
  format     VARCHAR(20) NOT NULL DEFAULT 'qr',
  label_data JSONB,
  printed_at TIMESTAMPTZ,
  printed_by BIGINT REFERENCES public.employees(id),
  created_by BIGINT REFERENCES public.employees(id),
  created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS public.ldap_sync_configs (
  id          BIGSERIAL PRIMARY KEY,
  name        VARCHAR(255) NOT NULL,
  host        VARCHAR(255) NOT NULL,
  port        INT NOT NULL DEFAULT 389,
  use_tls     BOOLEAN NOT NULL DEFAULT false,
  base_dn     TEXT NOT NULL,
  bind_dn     TEXT NOT NULL,
  bind_password TEXT,
  user_filter TEXT DEFAULT '(objectClass=person)',
  field_map   JSONB DEFAULT '{"sAMAccountName":"username","cn":"name","mail":"email"}',
  is_active   BOOLEAN NOT NULL DEFAULT true,
  created_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS public.ldap_sync_logs (
  id           BIGSERIAL PRIMARY KEY,
  config_id    BIGINT NOT NULL REFERENCES public.ldap_sync_configs(id) ON DELETE CASCADE,
  status       VARCHAR(20) NOT NULL DEFAULT 'running',
  users_found  INT NOT NULL DEFAULT 0,
  users_synced INT NOT NULL DEFAULT 0,
  users_skipped INT NOT NULL DEFAULT 0,
  errors       JSONB DEFAULT '[]',
  started_at   TIMESTAMPTZ NOT NULL DEFAULT now(),
  finished_at  TIMESTAMPTZ,
  triggered_by BIGINT REFERENCES public.employees(id)
);

CREATE TABLE IF NOT EXISTS public.dr_plans (
  id            BIGSERIAL PRIMARY KEY,
  name          VARCHAR(255) NOT NULL,
  plan_type     VARCHAR(50) NOT NULL DEFAULT 'DR',
  status        VARCHAR(20) NOT NULL DEFAULT 'draft',
  description   TEXT,
  rto_minutes   INT,
  rpo_minutes   INT,
  owner_id      BIGINT REFERENCES public.employees(id),
  last_tested_at TIMESTAMPTZ,
  next_test_due  DATE,
  document_url   TEXT,
  created_at     TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at     TIMESTAMPTZ NOT NULL DEFAULT now(),
  deleted_at     TIMESTAMPTZ
);

CREATE TABLE IF NOT EXISTS public.dr_plan_steps (
  id          BIGSERIAL PRIMARY KEY,
  plan_id     BIGINT NOT NULL REFERENCES public.dr_plans(id) ON DELETE CASCADE,
  step_order  INT NOT NULL DEFAULT 1,
  title       VARCHAR(255) NOT NULL,
  description TEXT,
  responsible_id BIGINT REFERENCES public.employees(id),
  duration_minutes INT,
  created_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS public.dr_tests (
  id              BIGSERIAL PRIMARY KEY,
  plan_id         BIGINT NOT NULL REFERENCES public.dr_plans(id) ON DELETE CASCADE,
  test_type       VARCHAR(50) NOT NULL DEFAULT 'tabletop',
  status          VARCHAR(20) NOT NULL DEFAULT 'scheduled',
  scheduled_at    TIMESTAMPTZ,
  started_at      TIMESTAMPTZ,
  completed_at    TIMESTAMPTZ,
  actual_rto_minutes INT,
  actual_rpo_minutes INT,
  outcome         VARCHAR(20),
  summary         TEXT,
  conducted_by    BIGINT REFERENCES public.employees(id),
  created_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at      TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS public.dr_test_results (
  id          BIGSERIAL PRIMARY KEY,
  test_id     BIGINT NOT NULL REFERENCES public.dr_tests(id) ON DELETE CASCADE,
  step_id     BIGINT REFERENCES public.dr_plan_steps(id),
  result      VARCHAR(20) NOT NULL DEFAULT 'pass',
  notes       TEXT,
  recorded_by BIGINT REFERENCES public.employees(id),
  created_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS public.compliance_frameworks (
  id          BIGSERIAL PRIMARY KEY,
  code        VARCHAR(50) NOT NULL UNIQUE,
  name        VARCHAR(255) NOT NULL,
  description TEXT,
  version     VARCHAR(20),
  is_active   BOOLEAN NOT NULL DEFAULT true,
  created_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS public.compliance_controls (
  id             BIGSERIAL PRIMARY KEY,
  framework_id   BIGINT NOT NULL REFERENCES public.compliance_frameworks(id) ON DELETE CASCADE,
  control_code   VARCHAR(50) NOT NULL,
  title          VARCHAR(255) NOT NULL,
  description    TEXT,
  severity       VARCHAR(20) NOT NULL DEFAULT 'medium',
  is_active      BOOLEAN NOT NULL DEFAULT true,
  created_at     TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at     TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS public.compliance_evidence (
  id           BIGSERIAL PRIMARY KEY,
  control_id   BIGINT NOT NULL REFERENCES public.compliance_controls(id) ON DELETE CASCADE,
  entity_type  VARCHAR(50),
  entity_id    BIGINT,
  title        VARCHAR(255) NOT NULL,
  description  TEXT,
  file_url     TEXT,
  status       VARCHAR(20) NOT NULL DEFAULT 'active',
  submitted_by BIGINT REFERENCES public.employees(id),
  submitted_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  expires_at   TIMESTAMPTZ,
  created_at   TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at   TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS public.vendor_performance (
  id            BIGSERIAL PRIMARY KEY,
  vendor_name   VARCHAR(255) NOT NULL,
  contract_id   BIGINT REFERENCES public.contracts(id) ON DELETE SET NULL,
  period_start  DATE NOT NULL,
  period_end    DATE NOT NULL,
  sla_target    NUMERIC(5,2) NOT NULL DEFAULT 99.0,
  sla_actual    NUMERIC(5,2),
  incidents     INT DEFAULT 0,
  response_time_avg INT,
  quality_score NUMERIC(5,2),
  notes         TEXT,
  created_at    TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at    TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS public.service_availability (
  id            BIGSERIAL PRIMARY KEY,
  service_code  VARCHAR(50) NOT NULL,
  service_name  VARCHAR(255) NOT NULL,
  period_start  DATE NOT NULL,
  period_end    DATE NOT NULL,
  uptime_minutes INT NOT NULL DEFAULT 0,
  downtime_minutes INT NOT NULL DEFAULT 0,
  planned_downtime_minutes INT NOT NULL DEFAULT 0,
  availability_pct NUMERIC(7,4),
  incidents     INT DEFAULT 0,
  mttr_minutes  INT,
  notes         TEXT,
  created_at    TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at    TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS public.asset_specifications (
  id              BIGSERIAL PRIMARY KEY,
  asset_id        BIGINT NOT NULL REFERENCES public.assets(id) ON DELETE CASCADE,
  cpu             TEXT,
  ram_gb          NUMERIC(8,2),
  storage_gb      NUMERIC(10,2),
  storage_type    VARCHAR(20),
  os_name         TEXT,
  os_version      TEXT,
  screen_size     NUMERIC(5,2),
  resolution      VARCHAR(30),
  network_ports   JSONB DEFAULT '[]',
  power_watt      NUMERIC(8,2),
  weight_kg       NUMERIC(6,3),
  dimensions_cm   JSONB,
  color           VARCHAR(50),
  battery_capacity TEXT,
  custom_fields   JSONB DEFAULT '{}',
  created_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at      TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS public.software_usage_logs (
  id            BIGSERIAL PRIMARY KEY,
  license_id    BIGINT REFERENCES public.licenses(id) ON DELETE SET NULL,
  asset_id      BIGINT REFERENCES public.assets(id) ON DELETE SET NULL,
  employee_id   BIGINT REFERENCES public.employees(id) ON DELETE SET NULL,
  session_start TIMESTAMPTZ NOT NULL,
  session_end   TIMESTAMPTZ,
  duration_minutes INT,
  created_at    TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS public.asset_disposal_records (
  id                      BIGSERIAL PRIMARY KEY,
  asset_id                BIGINT NOT NULL REFERENCES public.assets(id) ON DELETE CASCADE,
  disposal_method         VARCHAR(100),
  date_disposed           DATE,
  data_wipe_completed     BOOLEAN DEFAULT false,
  data_wipe_method        VARCHAR(100),
  environmental_compliant BOOLEAN DEFAULT false,
  certificate_number      VARCHAR(100),
  certificate_url         TEXT,
  disposed_by             BIGINT REFERENCES public.employees(id),
  approved_by             BIGINT REFERENCES public.employees(id),
  notes                   TEXT,
  created_at              TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at              TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS public.notifications (
  id          BIGSERIAL PRIMARY KEY,
  user_id     BIGINT NOT NULL REFERENCES public.employees(id) ON DELETE CASCADE,
  type        VARCHAR(50)  NOT NULL,
  title       VARCHAR(255) NOT NULL,
  message     TEXT         NOT NULL,
  entity_type VARCHAR(50),
  entity_id   BIGINT,
  is_read     BOOLEAN      NOT NULL DEFAULT false,
  created_at  TIMESTAMPTZ  NOT NULL DEFAULT now()
);

-- ════════════════════════════════════════════════════════════
-- STEP 7: Indexes
-- ════════════════════════════════════════════════════════════

CREATE INDEX IF NOT EXISTS idx_alerts_acknowledged           ON public.alerts (acknowledged);
CREATE INDEX IF NOT EXISTS idx_audit_logs_category           ON public.audit_logs (category);
CREATE INDEX IF NOT EXISTS idx_audit_logs_created_at         ON public.audit_logs (created_at DESC);
CREATE INDEX IF NOT EXISTS idx_audit_logs_severity           ON public.audit_logs (severity);
CREATE UNIQUE INDEX IF NOT EXISTS uniq_asset_history_hash    ON public.asset_history (hash) WHERE hash IS NOT NULL;
CREATE INDEX IF NOT EXISTS idx_budget_alerts_budget_at       ON public.budget_alerts (budget_id, alerted_at DESC);
CREATE INDEX IF NOT EXISTS idx_compliance_alerts_created_at  ON public.compliance_alerts (created_at DESC);
CREATE INDEX IF NOT EXISTS idx_compliance_trend_created_at   ON public.compliance_trend (created_at DESC);
CREATE INDEX IF NOT EXISTS idx_contracts_number              ON public.contracts (lower(contract_number));
CREATE INDEX IF NOT EXISTS idx_contracts_status              ON public.contracts (status);
CREATE INDEX IF NOT EXISTS idx_contracts_vendor              ON public.contracts (lower(vendor));
CREATE INDEX IF NOT EXISTS idx_email_logs_created_at         ON public.email_logs (created_at DESC);
CREATE INDEX IF NOT EXISTS idx_employee_trainings_employee_id ON public.employee_trainings (employee_id);
CREATE INDEX IF NOT EXISTS idx_grf_asset                     ON public.governance_review_feedback (asset_id);
CREATE INDEX IF NOT EXISTS idx_grf_reviewer                  ON public.governance_review_feedback (reviewer_id);
CREATE INDEX IF NOT EXISTS idx_kg_edges_dst                  ON public.kg_edges (dst_node_id);
CREATE INDEX IF NOT EXISTS idx_kg_edges_rel                  ON public.kg_edges (rel_type);
CREATE INDEX IF NOT EXISTS idx_kg_edges_src                  ON public.kg_edges (src_node_id);
CREATE INDEX IF NOT EXISTS idx_kg_nodes_type                 ON public.kg_nodes (entity_type);
CREATE INDEX IF NOT EXISTS idx_licenses_compliance_status    ON public.licenses (compliance_status);
CREATE UNIQUE INDEX IF NOT EXISTS uq_locations_site          ON public.locations (COALESCE(parent_id, 0), site, COALESCE(building,''), COALESCE(room,'')) WHERE status = 'active';
CREATE INDEX IF NOT EXISTS idx_problems_status               ON public.problems (status);
CREATE INDEX IF NOT EXISTS idx_problems_assigned_to          ON public.problems (assigned_to);
CREATE INDEX IF NOT EXISTS idx_problems_known_error          ON public.problems (known_error) WHERE known_error = true;
CREATE INDEX IF NOT EXISTS idx_role_delegations_active       ON public.role_delegations (start_date, end_date);
CREATE INDEX IF NOT EXISTS idx_role_delegations_delegatee_id ON public.role_delegations (delegatee_id);
CREATE INDEX IF NOT EXISTS idx_role_delegations_is_active    ON public.role_delegations (is_active) WHERE is_active = true;
CREATE INDEX IF NOT EXISTS idx_ticket_attachments_comment_id ON public.ticket_attachments (comment_id);
CREATE INDEX IF NOT EXISTS idx_ticket_comments_ticket_id     ON public.ticket_comments (ticket_id);
CREATE INDEX IF NOT EXISTS idx_tickets_sla_due_at            ON public.tickets (sla_due_at);
CREATE INDEX IF NOT EXISTS idx_tickets_response_due_at       ON public.tickets (response_due_at);
CREATE INDEX IF NOT EXISTS idx_tickets_sla_policy_id         ON public.tickets (sla_policy_id);
CREATE INDEX IF NOT EXISTS idx_tickets_ticket_type           ON public.tickets (ticket_type);
CREATE INDEX IF NOT EXISTS idx_assets_department_id          ON public.assets (department_id);
CREATE INDEX IF NOT EXISTS idx_assets_location_id            ON public.assets (location_id);
CREATE INDEX IF NOT EXISTS idx_assets_lifecycle_stage        ON public.assets (lifecycle_stage);
CREATE INDEX IF NOT EXISTS idx_assets_deleted_at             ON public.assets (deleted_at);
CREATE INDEX IF NOT EXISTS idx_assets_contract_id            ON public.assets (contract_id);
CREATE INDEX IF NOT EXISTS idx_assets_compliance_flag        ON public.assets (compliance_flag);
CREATE INDEX IF NOT EXISTS idx_assets_governance_score       ON public.assets (governance_score);
CREATE INDEX IF NOT EXISTS idx_assets_asset_criticality      ON public.assets (asset_criticality);
CREATE INDEX IF NOT EXISTS idx_departments_manager_id        ON public.departments (manager_id);
CREATE INDEX IF NOT EXISTS idx_notif_user_id                 ON public.notifications (user_id);
CREATE INDEX IF NOT EXISTS idx_notif_is_read                 ON public.notifications (user_id, is_read) WHERE is_read = false;
CREATE INDEX IF NOT EXISTS idx_notif_created_at              ON public.notifications (created_at);
CREATE INDEX IF NOT EXISTS idx_sr_status                     ON public.service_requests (status);
CREATE INDEX IF NOT EXISTS idx_sr_requested_by               ON public.service_requests (requested_by);
CREATE INDEX IF NOT EXISTS idx_cr_status                     ON public.change_requests (status);
CREATE INDEX IF NOT EXISTS idx_cr_type                       ON public.change_requests (type);
CREATE INDEX IF NOT EXISTS idx_cr_deleted_at                 ON public.change_requests (deleted_at);
CREATE INDEX IF NOT EXISTS idx_drp_status                    ON public.dr_plans (status);
CREATE INDEX IF NOT EXISTS idx_drt_plan_id                   ON public.dr_tests (plan_id);
CREATE INDEX IF NOT EXISTS idx_cf_is_active                  ON public.compliance_frameworks (is_active);
CREATE INDEX IF NOT EXISTS idx_cc_framework_id               ON public.compliance_controls (framework_id);
CREATE INDEX IF NOT EXISTS idx_ce_control_id                 ON public.compliance_evidence (control_id);
CREATE INDEX IF NOT EXISTS idx_ce_status                     ON public.compliance_evidence (status);
CREATE INDEX IF NOT EXISTS idx_notif_user_id                 ON public.notifications (user_id);
CREATE INDEX IF NOT EXISTS idx_vp_vendor_name                ON public.vendor_performance (vendor_name);
CREATE INDEX IF NOT EXISTS idx_sa_service_code               ON public.service_availability (service_code);
CREATE INDEX IF NOT EXISTS idx_asset_spec_asset_id           ON public.asset_specifications (asset_id);
CREATE INDEX IF NOT EXISTS idx_wdl_subscription_id           ON public.webhook_delivery_logs (subscription_id);
CREATE INDEX IF NOT EXISTS idx_wdl_status                    ON public.webhook_delivery_logs (status);

-- ════════════════════════════════════════════════════════════
-- STEP 8: Foreign keys untuk tabel yang sudah ada (tickets, assets, dll.)
-- ════════════════════════════════════════════════════════════

DO $$ BEGIN
  ALTER TABLE public.tickets ADD CONSTRAINT fk_tickets_sla_policy
    FOREIGN KEY (sla_policy_id) REFERENCES public.sla_policies(id) ON DELETE SET NULL;
EXCEPTION WHEN duplicate_object THEN NULL;
END $$;

DO $$ BEGIN
  ALTER TABLE public.tickets ADD CONSTRAINT fk_tickets_change_request
    FOREIGN KEY (change_request_id) REFERENCES public.change_requests(id) ON DELETE SET NULL;
EXCEPTION WHEN duplicate_object THEN NULL;
END $$;

DO $$ BEGIN
  ALTER TABLE public.alerts ADD CONSTRAINT alerts_acknowledged_by_fkey
    FOREIGN KEY (acknowledged_by) REFERENCES public.employees(id);
EXCEPTION WHEN duplicate_object THEN NULL;
END $$;

DO $$ BEGIN
  ALTER TABLE public.alerts ADD CONSTRAINT alerts_asset_id_fkey
    FOREIGN KEY (asset_id) REFERENCES public.assets(id) ON DELETE SET NULL;
EXCEPTION WHEN duplicate_object THEN NULL;
END $$;

DO $$ BEGIN
  ALTER TABLE public.sla_policies ADD CONSTRAINT sla_policies_category_code_fkey
    FOREIGN KEY (category_code) REFERENCES public.ticket_categories(code) ON DELETE SET NULL;
EXCEPTION WHEN duplicate_object THEN NULL;
END $$;

DO $$ BEGIN
  ALTER TABLE public.sla_policies ADD CONSTRAINT sla_policies_service_code_fkey
    FOREIGN KEY (service_code) REFERENCES public.services(code) ON DELETE SET NULL;
EXCEPTION WHEN duplicate_object THEN NULL;
END $$;

DO $$ BEGIN
  ALTER TABLE public.licenses ADD CONSTRAINT fk_licenses_contract
    FOREIGN KEY (contract_id) REFERENCES public.contracts(id) ON UPDATE CASCADE ON DELETE SET NULL;
EXCEPTION WHEN duplicate_object THEN NULL;
END $$;

COMMIT;

-- ════════════════════════════════════════════════════════════
-- SELESAI
-- Verifikasi: SELECT count(*) FROM information_schema.tables
--             WHERE table_schema = 'public';
-- Expected: 63 rows
-- ════════════════════════════════════════════════════════════
