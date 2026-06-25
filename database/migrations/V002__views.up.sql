-- Tambah semua views yang belum ada di production
-- Aman dijalankan ulang (CREATE OR REPLACE / DROP IF EXISTS + CREATE)

BEGIN;

-- Kolom yang hilang di production
ALTER TABLE public.software_installations
  ADD COLUMN IF NOT EXISTS removed_at TIMESTAMPTZ;

CREATE INDEX IF NOT EXISTS idx_software_installations_active
  ON public.software_installations (license_id) WHERE removed_at IS NULL;

-- 1. compliance_index
CREATE OR REPLACE VIEW public.compliance_index AS
 SELECT d.id AS department_id,
    d.name AS department_name,
    count(DISTINCT a.id) FILTER (WHERE ((a.status)::text = 'active'::text)) AS total_assets,
    count(DISTINCT t.id) FILTER (WHERE ((t.status)::text = 'Open'::text)) AS open_tickets,
    round((((1)::numeric - ((count(DISTINCT t.id) FILTER (WHERE ((t.status)::text = 'Open'::text)))::numeric / (NULLIF(count(DISTINCT a.id), 0))::numeric)) * (100)::numeric), 2) AS compliance_score
   FROM (((public.departments d
     LEFT JOIN public.employees e ON ((e.department_id = d.id)))
     LEFT JOIN public.assets a ON ((a.department_id = d.id)))
     LEFT JOIN public.tickets t ON ((t.assigned_to_employee_id = e.id)))
  GROUP BY d.id, d.name;

-- 2. sla_violation_report (materialized view)
CREATE MATERIALIZED VIEW IF NOT EXISTS public.sla_violation_report AS
 SELECT t.id AS ticket_id,
    t.subject,
    COALESCE(t.status, 'Unknown'::character varying) AS status,
    s.name AS policy_name,
    s.response_minutes,
    s.resolve_minutes,
    t.response_due_at,
    t.resolved_at,
    now() AS checked_at,
    GREATEST((EXTRACT(epoch FROM (now() - t.response_due_at)) / (3600)::numeric), (0)::numeric) AS overdue_response_hours,
        CASE
            WHEN (t.resolved_at IS NULL) THEN NULL::numeric
            WHEN (t.resolved_at > (t.response_due_at + ((COALESCE(s.resolve_minutes, 0))::double precision * '00:01:00'::interval))) THEN (EXTRACT(epoch FROM (t.resolved_at - (t.response_due_at + ((COALESCE(s.resolve_minutes, 0))::double precision * '00:01:00'::interval)))) / (3600)::numeric)
            ELSE (0)::numeric
        END AS overdue_resolution_hours
   FROM (public.tickets t
     LEFT JOIN public.sla_policies s ON ((t.sla_policy_id = s.id)))
  WHERE (((t.status)::text <> 'Resolved'::text) AND (now() > t.response_due_at))
  WITH NO DATA;

-- 3. sla_compliance_score
CREATE OR REPLACE VIEW public.sla_compliance_score AS
 SELECT d.id AS department_id,
    d.name AS department_name,
    count(t.id) AS total_tickets,
    count(t.id) FILTER (WHERE ((t.status)::text = 'Resolved'::text)) AS resolved_tickets,
    round((((count(t.id) FILTER (WHERE ((t.status)::text = 'Resolved'::text)))::numeric / (NULLIF(count(t.id), 0))::numeric) * (100)::numeric), 2) AS resolve_rate_pct,
    count(v.ticket_id) AS violations,
    round((((1)::numeric - ((count(v.ticket_id))::numeric / (NULLIF(count(t.id), 0))::numeric)) * (100)::numeric), 2) AS sla_compliance_pct
   FROM (((public.departments d
     LEFT JOIN public.employees e ON ((e.department_id = d.id)))
     LEFT JOIN public.tickets t ON ((t.assigned_to_employee_id = e.id)))
     LEFT JOIN public.sla_violation_report v ON ((v.ticket_id = t.id)))
  GROUP BY d.id, d.name;

-- 4. compliance_summary
CREATE OR REPLACE VIEW public.compliance_summary AS
 SELECT d.id AS department_id,
    d.name AS department_name,
    ci.compliance_score,
    sc.sla_compliance_pct,
    round(((ci.compliance_score + sc.sla_compliance_pct) / (2)::numeric), 2) AS total_compliance_index,
    count(a.id) AS total_assets,
    count(t.id) AS total_tickets,
    count(v.ticket_id) AS sla_violations,
    max(asess.completed_at) AS last_audit_date
   FROM (((((((public.departments d
     LEFT JOIN public.employees e ON ((e.department_id = d.id)))
     LEFT JOIN public.assets a ON ((a.department_id = d.id)))
     LEFT JOIN public.tickets t ON ((t.assigned_to_employee_id = e.id)))
     LEFT JOIN public.sla_violation_report v ON ((v.ticket_id = t.id)))
     LEFT JOIN public.compliance_index ci ON ((ci.department_id = d.id)))
     LEFT JOIN public.sla_compliance_score sc ON ((sc.department_id = d.id)))
     LEFT JOIN public.audit_sessions asess ON ((asess.created_by_employee_id = e.id)))
  GROUP BY d.id, d.name, ci.compliance_score, sc.sla_compliance_pct;

-- 5. v_budget_overview
CREATE OR REPLACE VIEW public.v_budget_overview AS
 SELECT b.id AS budget_id,
    b.name AS budget_name,
    b.category,
    'IDR'::character varying AS currency,
    b.total_amount,
    COALESCE(sum(bt.amount), (0)::numeric) AS realized_amount,
    (b.total_amount - COALESCE(sum(bt.amount), (0)::numeric)) AS remaining_amount,
    round(
        CASE
            WHEN (b.total_amount > (0)::numeric) THEN ((COALESCE(sum(bt.amount), (0)::numeric) / b.total_amount) * (100)::numeric)
            ELSE (0)::numeric
        END, 2) AS realization_percent,
        CASE
            WHEN (COALESCE(sum(bt.amount), (0)::numeric) > b.total_amount) THEN 'overspend'::text
            WHEN (COALESCE(sum(bt.amount), (0)::numeric) > (b.total_amount * 0.8)) THEN 'warning'::text
            ELSE 'ok'::text
        END AS status
   FROM (public.budgets b
     LEFT JOIN public.budget_transactions bt ON ((b.id = bt.budget_id)))
  WHERE (b.deleted_at IS NULL)
  GROUP BY b.id;

-- 6. v_security_audit
CREATE OR REPLACE VIEW public.v_security_audit AS
 SELECT a.id,
    a.entity_name,
    a.action,
    a.actor_id,
    e.name AS actor_name,
    a.request_path,
    a.created_at
   FROM (public.audit_logs a
     LEFT JOIN public.employees e ON ((e.id = a.actor_id)))
  WHERE (lower((a.action)::text) = ANY (ARRAY['login','logout','token_refresh','change_password','failed_login','get','post','put','delete']))
  ORDER BY a.created_at DESC;

-- 7. v_license_reconciliation
CREATE OR REPLACE VIEW public.v_license_reconciliation AS
SELECT
    l.id                                          AS license_id,
    l.name                                        AS license_name,
    l.license_type,
    l.license_model,
    l.total_seats                                 AS entitled_seats,
    COALESCE(
        (SELECT count(*) FROM public.software_installations asi
         WHERE asi.license_id = l.id AND asi.removed_at IS NULL), 0
    )::INT                                        AS installed_seats,
    l.total_seats - COALESCE(
        (SELECT count(*) FROM public.software_installations asi
         WHERE asi.license_id = l.id AND asi.removed_at IS NULL), 0
    )::INT                                        AS available_seats,
    CASE
        WHEN l.total_seats = COALESCE(
            (SELECT count(*) FROM public.software_installations asi
             WHERE asi.license_id = l.id AND asi.removed_at IS NULL), 0)
            THEN 'compliant'
        WHEN l.total_seats < COALESCE(
            (SELECT count(*) FROM public.software_installations asi
             WHERE asi.license_id = l.id AND asi.removed_at IS NULL), 0)
            THEN 'under_licensed'
        ELSE 'over_licensed'
    END                                           AS reconciliation_status,
    l.expiration_date,
    l.compliance_status,
    l.vendor,
    l.cost,
    l.currency,
    COALESCE(
        (SELECT count(DISTINCT sul.employee_id) FROM public.software_usage_logs sul
         WHERE sul.license_id = l.id AND sul.session_start >= now() - INTERVAL '90 days'), 0
    )::INT                                        AS active_users_90d,
    (SELECT max(sul.session_start) FROM public.software_usage_logs sul
     WHERE sul.license_id = l.id)                AS last_used_at
FROM public.licenses l
WHERE l.deleted_at IS NULL;

-- 8. v_asset_disposal_compliance
CREATE OR REPLACE VIEW public.v_asset_disposal_compliance AS
SELECT
    a.id                    AS asset_id,
    a.name                  AS asset_name,
    a.asset_tag,
    a.lifecycle_stage,
    d.id                    AS disposal_record_id,
    d.disposal_method,
    d.data_wipe_completed,
    d.environmental_compliant,
    d.certificate_number,
    d.date_disposed,
    auth.name               AS authorized_by,
    exec.name               AS executed_by,
    CASE
        WHEN d.id IS NULL                  THEN 'missing_record'
        WHEN NOT d.data_wipe_completed     THEN 'data_wipe_pending'
        WHEN NOT d.environmental_compliant THEN 'env_non_compliant'
        ELSE 'compliant'
    END                     AS compliance_status
FROM public.assets a
LEFT JOIN public.asset_disposal_records d   ON d.asset_id  = a.id
LEFT JOIN public.employees auth ON auth.id = d.approved_by
LEFT JOIN public.employees exec ON exec.id  = d.disposed_by
WHERE a.lifecycle_stage IN ('disposal_pending','disposal_approved','disposed')
   OR d.id IS NOT NULL;

COMMIT;
