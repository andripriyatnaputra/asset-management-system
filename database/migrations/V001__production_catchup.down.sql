-- ROLLBACK: Production Catchup Migration
-- PERINGATAN: Drop semua tabel yang ditambahkan V001.
-- Jangan jalankan di production kecuali benar-benar diperlukan!

BEGIN;

-- Views
DROP VIEW IF EXISTS public.compliance_summary CASCADE;
DROP VIEW IF EXISTS public.compliance_index CASCADE;
DROP VIEW IF EXISTS public.sla_violation_report CASCADE;
DROP VIEW IF EXISTS public.sla_compliance_score CASCADE;
DROP VIEW IF EXISTS public.v_budget_overview CASCADE;
DROP VIEW IF EXISTS public.v_security_audit CASCADE;

-- Tabel modul baru (urutan terbalik karena FK)
DROP TABLE IF EXISTS public.notifications CASCADE;
DROP TABLE IF EXISTS public.asset_disposal_records CASCADE;
DROP TABLE IF EXISTS public.software_usage_logs CASCADE;
DROP TABLE IF EXISTS public.asset_specifications CASCADE;
DROP TABLE IF EXISTS public.service_availability CASCADE;
DROP TABLE IF EXISTS public.vendor_performance CASCADE;
DROP TABLE IF EXISTS public.compliance_evidence CASCADE;
DROP TABLE IF EXISTS public.compliance_controls CASCADE;
DROP TABLE IF EXISTS public.compliance_frameworks CASCADE;
DROP TABLE IF EXISTS public.dr_test_results CASCADE;
DROP TABLE IF EXISTS public.dr_tests CASCADE;
DROP TABLE IF EXISTS public.dr_plan_steps CASCADE;
DROP TABLE IF EXISTS public.dr_plans CASCADE;
DROP TABLE IF EXISTS public.ldap_sync_logs CASCADE;
DROP TABLE IF EXISTS public.ldap_sync_configs CASCADE;
DROP TABLE IF EXISTS public.asset_qr_codes CASCADE;
DROP TABLE IF EXISTS public.webhook_delivery_logs CASCADE;
DROP TABLE IF EXISTS public.webhook_subscriptions CASCADE;
DROP TABLE IF EXISTS public.escalation_rules CASCADE;
DROP TABLE IF EXISTS public.incident_postmortems CASCADE;
DROP TABLE IF EXISTS public.problem_incidents CASCADE;
DROP TABLE IF EXISTS public.change_tasks CASCADE;
DROP TABLE IF EXISTS public.change_approvals CASCADE;
DROP TABLE IF EXISTS public.change_requests CASCADE;
DROP TABLE IF EXISTS public.approval_workflows CASCADE;
DROP TABLE IF EXISTS public.service_requests CASCADE;
DROP TABLE IF EXISTS public.service_catalog CASCADE;
DROP TABLE IF EXISTS public.role_delegations CASCADE;
DROP TABLE IF EXISTS public.ticket_attachments CASCADE;
DROP TABLE IF EXISTS public.governance_review_feedback CASCADE;
DROP TABLE IF EXISTS public.ml_calibration_models CASCADE;
DROP TABLE IF EXISTS public.kg_edges CASCADE;
DROP TABLE IF EXISTS public.kg_nodes CASCADE;
DROP TABLE IF EXISTS public.employee_trainings CASCADE;
DROP TABLE IF EXISTS public.email_logs CASCADE;
DROP TABLE IF EXISTS public.data_governance CASCADE;
DROP TABLE IF EXISTS public.compliance_trend CASCADE;
DROP TABLE IF EXISTS public.compliance_alerts CASCADE;
DROP TABLE IF EXISTS public.budget_alerts CASCADE;
DROP TABLE IF EXISTS public.audit_logs CASCADE;
DROP TABLE IF EXISTS public.asset_history CASCADE;
DROP TABLE IF EXISTS public.alerts CASCADE;
DROP TABLE IF EXISTS public.sla_policies CASCADE;
DROP TABLE IF EXISTS public.ticket_categories CASCADE;
DROP TABLE IF EXISTS public.services CASCADE;
DROP TABLE IF EXISTS public.contracts CASCADE;
DROP TABLE IF EXISTS public.problems CASCADE;
DROP TABLE IF EXISTS public.locations CASCADE;
DROP TABLE IF EXISTS public.cost_centers CASCADE;

-- Rename balik licenses → software_licenses
DO $$ BEGIN
  IF EXISTS (
    SELECT 1 FROM information_schema.tables
    WHERE table_schema = 'public' AND table_name = 'licenses'
  ) AND NOT EXISTS (
    SELECT 1 FROM information_schema.tables
    WHERE table_schema = 'public' AND table_name = 'software_licenses'
  ) THEN
    ALTER TABLE public.licenses RENAME TO software_licenses;
  END IF;
END $$;

-- Drop custom type
DROP TYPE IF EXISTS public.asset_status;

-- Drop sequences
DROP SEQUENCE IF EXISTS public.alerts_id_seq;
DROP SEQUENCE IF EXISTS public.asset_history_id_seq;
DROP SEQUENCE IF EXISTS public.audit_logs_id_seq;
DROP SEQUENCE IF EXISTS public.budget_alerts_id_seq;
DROP SEQUENCE IF EXISTS public.compliance_alerts_id_seq;
DROP SEQUENCE IF EXISTS public.compliance_trend_id_seq;
DROP SEQUENCE IF EXISTS public.cost_centers_id_seq;
DROP SEQUENCE IF EXISTS public.email_logs_id_seq;
DROP SEQUENCE IF EXISTS public.employee_trainings_id_seq;
DROP SEQUENCE IF EXISTS public.governance_review_feedback_id_seq;
DROP SEQUENCE IF EXISTS public.kg_edges_id_seq;
DROP SEQUENCE IF EXISTS public.kg_nodes_id_seq;
DROP SEQUENCE IF EXISTS public.locations_id_seq;
DROP SEQUENCE IF EXISTS public.ml_calibration_models_id_seq;
DROP SEQUENCE IF EXISTS public.role_delegations_id_seq;
DROP SEQUENCE IF EXISTS public.services_id_seq;
DROP SEQUENCE IF EXISTS public.sla_policies_id_seq;
DROP SEQUENCE IF EXISTS public.ticket_attachments_id_seq;
DROP SEQUENCE IF EXISTS public.ticket_categories_id_seq;

COMMIT;
