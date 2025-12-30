--
-- PostgreSQL database dump
--

-- Dumped from database version 15.13
-- Dumped by pg_dump version 15.13

SET statement_timeout = 0;
SET lock_timeout = 0;
SET idle_in_transaction_session_timeout = 0;
SET client_encoding = 'UTF8';
SET standard_conforming_strings = on;
SELECT pg_catalog.set_config('search_path', '', false);
SET check_function_bodies = false;
SET xmloption = content;
SET client_min_messages = warning;
SET row_security = off;

--
-- Name: cdc; Type: SCHEMA; Schema: -; Owner: admin
--

CREATE SCHEMA cdc;


ALTER SCHEMA cdc OWNER TO admin;

--
-- Name: pgcrypto; Type: EXTENSION; Schema: -; Owner: -
--

CREATE EXTENSION IF NOT EXISTS pgcrypto WITH SCHEMA public;


--
-- Name: EXTENSION pgcrypto; Type: COMMENT; Schema: -; Owner: 
--

COMMENT ON EXTENSION pgcrypto IS 'cryptographic functions';


--
-- Name: asset_status; Type: TYPE; Schema: public; Owner: admin
--

CREATE TYPE public.asset_status AS ENUM (
    'in_stock',
    'assigned',
    'maintenance',
    'retired',
    'disposed'
);


ALTER TYPE public.asset_status OWNER TO admin;

--
-- Name: kg_upsert_edge(bigint, bigint, text, jsonb, numeric); Type: FUNCTION; Schema: public; Owner: admin
--

CREATE FUNCTION public.kg_upsert_edge(p_src bigint, p_dst bigint, p_rel text, p_props jsonb, p_weight numeric) RETURNS bigint
    LANGUAGE plpgsql
    AS $$
DECLARE eid BIGINT;
BEGIN
  INSERT INTO kg_edges(src_node_id, dst_node_id, rel_type, props, weight)
  VALUES (p_src, p_dst, p_rel, COALESCE(p_props,'{}'::jsonb), COALESCE(p_weight,1))
  ON CONFLICT DO NOTHING
  RETURNING id INTO eid;

  IF eid IS NULL THEN
    UPDATE kg_edges
       SET props = COALESCE(p_props,'{}'::jsonb),
           weight= COALESCE(p_weight,1),
           updated_at=NOW()
     WHERE src_node_id=p_src AND dst_node_id=p_dst AND rel_type=p_rel
     RETURNING id INTO eid;
  END IF;
  RETURN eid;
END;
$$;


ALTER FUNCTION public.kg_upsert_edge(p_src bigint, p_dst bigint, p_rel text, p_props jsonb, p_weight numeric) OWNER TO admin;

--
-- Name: kg_upsert_node(text, bigint, text, jsonb); Type: FUNCTION; Schema: public; Owner: admin
--

CREATE FUNCTION public.kg_upsert_node(p_type text, p_id bigint, p_label text, p_props jsonb) RETURNS bigint
    LANGUAGE plpgsql
    AS $$
DECLARE nid BIGINT;
BEGIN
  INSERT INTO kg_nodes(entity_type, entity_id, label, props)
  VALUES (p_type, p_id, p_label, COALESCE(p_props,'{}'::jsonb))
  ON CONFLICT (entity_type, entity_id)
  DO UPDATE SET label=EXCLUDED.label, props=EXCLUDED.props, updated_at=NOW()
  RETURNING id INTO nid;
  RETURN nid;
END;
$$;


ALTER FUNCTION public.kg_upsert_node(p_type text, p_id bigint, p_label text, p_props jsonb) OWNER TO admin;

--
-- Name: recalc_budget_used_amount(bigint); Type: FUNCTION; Schema: public; Owner: admin
--

CREATE FUNCTION public.recalc_budget_used_amount(p_budget_id bigint) RETURNS void
    LANGUAGE plpgsql
    AS $$
BEGIN
  UPDATE public.budgets b
  SET used_amount = COALESCE((
      SELECT SUM(bt.amount)
      FROM public.budget_transactions bt
      WHERE bt.budget_id = p_budget_id
  ), 0)
  WHERE b.id = p_budget_id;
END$$;


ALTER FUNCTION public.recalc_budget_used_amount(p_budget_id bigint) OWNER TO admin;

--
-- Name: set_timestamp(); Type: FUNCTION; Schema: public; Owner: admin
--

CREATE FUNCTION public.set_timestamp() RETURNS trigger
    LANGUAGE plpgsql
    AS $$
BEGIN
  NEW.updated_at = NOW();
  RETURN NEW;
END;
$$;


ALTER FUNCTION public.set_timestamp() OWNER TO admin;

--
-- Name: set_updated_month(); Type: FUNCTION; Schema: public; Owner: admin
--

CREATE FUNCTION public.set_updated_month() RETURNS trigger
    LANGUAGE plpgsql
    AS $$
BEGIN
  NEW.updated_month := date_trunc('month', NEW.updated_at)::date;
  RETURN NEW;
END;
$$;


ALTER FUNCTION public.set_updated_month() OWNER TO admin;

--
-- Name: trg_assets_cdc(); Type: FUNCTION; Schema: public; Owner: admin
--

CREATE FUNCTION public.trg_assets_cdc() RETURNS trigger
    LANGUAGE plpgsql
    AS $$
DECLARE actor_id BIGINT;
BEGIN
  actor_id := NULLIF(current_setting('app.current_user_id', true),'')::BIGINT;

  IF TG_OP='INSERT' THEN
    INSERT INTO cdc.asset_changes(asset_id,operation,changes,changed_by)
    VALUES(NEW.id,'INSERT',jsonb_build_object('new',to_jsonb(NEW)),actor_id);
  ELSIF TG_OP='UPDATE' THEN
    INSERT INTO cdc.asset_changes(asset_id,operation,changes,changed_by)
    VALUES(NEW.id,'UPDATE',jsonb_build_object('old',to_jsonb(OLD),'new',to_jsonb(NEW)),actor_id);
  ELSIF TG_OP='DELETE' THEN
    INSERT INTO cdc.asset_changes(asset_id,operation,changes,changed_by)
    VALUES(OLD.id,'DELETE',jsonb_build_object('old',to_jsonb(OLD)),actor_id);
  END IF;
  RETURN NEW;
END;
$$;


ALTER FUNCTION public.trg_assets_cdc() OWNER TO admin;

--
-- Name: trg_audit_chain(); Type: FUNCTION; Schema: public; Owner: admin
--

CREATE FUNCTION public.trg_audit_chain() RETURNS trigger
    LANGUAGE plpgsql
    AS $$
DECLARE last_hash TEXT;
BEGIN
  SELECT hash INTO last_hash FROM audit_logs ORDER BY id DESC LIMIT 1;
  NEW.prev_hash := last_hash;
  NEW.hash := encode(
    digest(
      coalesce(NEW.id::text,'')||
      coalesce(NEW.actor_id::text,'')||
      coalesce(NEW.entity_name,'')||
      coalesce(NEW.action,'')||
      coalesce(NEW.created_at::text,'')||
      coalesce(last_hash,''),'sha256'
    ),'hex');
  RETURN NEW;
END;
$$;


ALTER FUNCTION public.trg_audit_chain() OWNER TO admin;

--
-- Name: trg_budget_tx_sync(); Type: FUNCTION; Schema: public; Owner: admin
--

CREATE FUNCTION public.trg_budget_tx_sync() RETURNS trigger
    LANGUAGE plpgsql
    AS $$
BEGIN
  IF (TG_OP = 'INSERT') THEN
    PERFORM public.recalc_budget_used_amount(NEW.budget_id);
  ELSIF (TG_OP = 'UPDATE') THEN
    -- Recalc untuk budget lama & baru jika pindah budget_id
    IF NEW.budget_id IS DISTINCT FROM OLD.budget_id THEN
      PERFORM public.recalc_budget_used_amount(OLD.budget_id);
    END IF;
    PERFORM public.recalc_budget_used_amount(NEW.budget_id);
  ELSIF (TG_OP = 'DELETE') THEN
    PERFORM public.recalc_budget_used_amount(OLD.budget_id);
  END IF;
  RETURN NULL;
END$$;


ALTER FUNCTION public.trg_budget_tx_sync() OWNER TO admin;

SET default_tablespace = '';

SET default_table_access_method = heap;

--
-- Name: asset_changes; Type: TABLE; Schema: cdc; Owner: admin
--

CREATE TABLE cdc.asset_changes (
    id bigint NOT NULL,
    asset_id bigint,
    operation character varying(10),
    changed_by bigint,
    changed_at timestamp with time zone DEFAULT now(),
    changes jsonb
);


ALTER TABLE cdc.asset_changes OWNER TO admin;

--
-- Name: asset_changes_id_seq; Type: SEQUENCE; Schema: cdc; Owner: admin
--

CREATE SEQUENCE cdc.asset_changes_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


ALTER TABLE cdc.asset_changes_id_seq OWNER TO admin;

--
-- Name: asset_changes_id_seq; Type: SEQUENCE OWNED BY; Schema: cdc; Owner: admin
--

ALTER SEQUENCE cdc.asset_changes_id_seq OWNED BY cdc.asset_changes.id;


--
-- Name: alerts; Type: TABLE; Schema: public; Owner: admin
--

CREATE TABLE public.alerts (
    id bigint NOT NULL,
    message text NOT NULL,
    severity character varying(20) DEFAULT 'info'::character varying NOT NULL,
    category character varying(50) DEFAULT 'system'::character varying,
    acknowledged boolean DEFAULT false,
    created_at timestamp with time zone DEFAULT now(),
    acknowledged_by bigint,
    asset_id bigint
);


ALTER TABLE public.alerts OWNER TO admin;

--
-- Name: alerts_id_seq; Type: SEQUENCE; Schema: public; Owner: admin
--

CREATE SEQUENCE public.alerts_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


ALTER TABLE public.alerts_id_seq OWNER TO admin;

--
-- Name: alerts_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: admin
--

ALTER SEQUENCE public.alerts_id_seq OWNED BY public.alerts.id;


--
-- Name: asset_assignments; Type: TABLE; Schema: public; Owner: admin
--

CREATE TABLE public.asset_assignments (
    id bigint NOT NULL,
    asset_id bigint NOT NULL,
    employee_id bigint NOT NULL,
    assigned_at timestamp with time zone DEFAULT now() NOT NULL,
    returned_at timestamp with time zone,
    notes text,
    assigned_by_employee_id bigint,
    returned_by_employee_id bigint
);


ALTER TABLE public.asset_assignments OWNER TO admin;

--
-- Name: asset_assignments_id_seq; Type: SEQUENCE; Schema: public; Owner: admin
--

CREATE SEQUENCE public.asset_assignments_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


ALTER TABLE public.asset_assignments_id_seq OWNER TO admin;

--
-- Name: asset_assignments_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: admin
--

ALTER SEQUENCE public.asset_assignments_id_seq OWNED BY public.asset_assignments.id;


--
-- Name: asset_history; Type: TABLE; Schema: public; Owner: admin
--

CREATE TABLE public.asset_history (
    id bigint NOT NULL,
    asset_id bigint NOT NULL,
    action text NOT NULL,
    detail text,
    actor_employee_id bigint,
    from_status public.asset_status,
    to_status public.asset_status,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    compliance_flag boolean DEFAULT true,
    compliance_note text,
    hash text
);


ALTER TABLE public.asset_history OWNER TO admin;

--
-- Name: asset_history_id_seq; Type: SEQUENCE; Schema: public; Owner: admin
--

CREATE SEQUENCE public.asset_history_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


ALTER TABLE public.asset_history_id_seq OWNER TO admin;

--
-- Name: asset_history_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: admin
--

ALTER SEQUENCE public.asset_history_id_seq OWNED BY public.asset_history.id;


--
-- Name: asset_maintenance_logs; Type: TABLE; Schema: public; Owner: admin
--

CREATE TABLE public.asset_maintenance_logs (
    id bigint NOT NULL,
    asset_id bigint NOT NULL,
    ticket_id bigint,
    log_type character varying(50) NOT NULL,
    description text NOT NULL,
    cost numeric(15,2) DEFAULT 0,
    log_date date DEFAULT CURRENT_DATE NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    performed_by_employee_id bigint,
    vendor text
);


ALTER TABLE public.asset_maintenance_logs OWNER TO admin;

--
-- Name: asset_maintenance_logs_id_seq; Type: SEQUENCE; Schema: public; Owner: admin
--

CREATE SEQUENCE public.asset_maintenance_logs_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


ALTER TABLE public.asset_maintenance_logs_id_seq OWNER TO admin;

--
-- Name: asset_maintenance_logs_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: admin
--

ALTER SEQUENCE public.asset_maintenance_logs_id_seq OWNED BY public.asset_maintenance_logs.id;


--
-- Name: asset_types; Type: TABLE; Schema: public; Owner: admin
--

CREATE TABLE public.asset_types (
    id bigint NOT NULL,
    name character varying(255) NOT NULL
);


ALTER TABLE public.asset_types OWNER TO admin;

--
-- Name: asset_types_id_seq; Type: SEQUENCE; Schema: public; Owner: admin
--

CREATE SEQUENCE public.asset_types_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


ALTER TABLE public.asset_types_id_seq OWNER TO admin;

--
-- Name: asset_types_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: admin
--

ALTER SEQUENCE public.asset_types_id_seq OWNED BY public.asset_types.id;


--
-- Name: assets; Type: TABLE; Schema: public; Owner: admin
--

CREATE TABLE public.assets (
    id bigint NOT NULL,
    name character varying(255) NOT NULL,
    asset_tag character varying(100) NOT NULL,
    status character varying(50) DEFAULT 'in_stock'::character varying NOT NULL,
    asset_type_id bigint,
    purchase_date date,
    initial_price numeric(15,2) DEFAULT 0,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL,
    deleted_at timestamp with time zone,
    department_id bigint,
    cost_center_id bigint,
    location_id bigint,
    purchase_cost numeric(14,2) DEFAULT 0,
    vendor text,
    warranty_expiry date,
    useful_life_months integer DEFAULT 36,
    depreciation_method text DEFAULT 'straight_line'::text,
    salvage_value numeric(14,2) DEFAULT 0,
    serial_number text,
    asset_condition text DEFAULT 'good'::text,
    acquisition_type text DEFAULT 'purchase'::text,
    ownership_type text DEFAULT 'company_owned'::text,
    disposal_date date,
    disposed boolean DEFAULT false,
    notes text,
    budget_id bigint,
    contract_id bigint,
    lifecycle_stage character varying(30) DEFAULT 'in_use'::character varying,
    asset_criticality character varying(20),
    disposed_approved_by bigint,
    compliance_flag boolean DEFAULT true,
    compliance_note text,
    verified_at timestamp with time zone,
    lifecycle_status character varying(20) DEFAULT 'active'::character varying,
    asset_health_score numeric(5,2),
    created_by bigint,
    updated_by bigint,
    currency character varying(10) DEFAULT 'IDR'::character varying,
    governance_score numeric(5,2) DEFAULT 0,
    updated_month date
);


ALTER TABLE public.assets OWNER TO admin;

--
-- Name: assets_id_seq; Type: SEQUENCE; Schema: public; Owner: admin
--

CREATE SEQUENCE public.assets_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


ALTER TABLE public.assets_id_seq OWNER TO admin;

--
-- Name: assets_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: admin
--

ALTER SEQUENCE public.assets_id_seq OWNED BY public.assets.id;


--
-- Name: audit_logs; Type: TABLE; Schema: public; Owner: admin
--

CREATE TABLE public.audit_logs (
    id bigint NOT NULL,
    actor_id bigint,
    entity_name character varying(50) NOT NULL,
    entity_id bigint,
    action character varying(100) NOT NULL,
    changes jsonb,
    created_at timestamp with time zone DEFAULT now(),
    ip_address text,
    user_agent text,
    request_path text,
    severity character varying(100),
    category character varying(50),
    hash text,
    prev_hash text
);


ALTER TABLE public.audit_logs OWNER TO admin;

--
-- Name: audit_logs_id_seq; Type: SEQUENCE; Schema: public; Owner: admin
--

CREATE SEQUENCE public.audit_logs_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


ALTER TABLE public.audit_logs_id_seq OWNER TO admin;

--
-- Name: audit_logs_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: admin
--

ALTER SEQUENCE public.audit_logs_id_seq OWNED BY public.audit_logs.id;


--
-- Name: audit_sessions; Type: TABLE; Schema: public; Owner: admin
--

CREATE TABLE public.audit_sessions (
    id bigint NOT NULL,
    name character varying(255) NOT NULL,
    status character varying(50) DEFAULT 'In Progress'::character varying NOT NULL,
    created_by_employee_id bigint NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    completed_at timestamp with time zone
);


ALTER TABLE public.audit_sessions OWNER TO admin;

--
-- Name: audit_sessions_id_seq; Type: SEQUENCE; Schema: public; Owner: admin
--

CREATE SEQUENCE public.audit_sessions_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


ALTER TABLE public.audit_sessions_id_seq OWNER TO admin;

--
-- Name: audit_sessions_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: admin
--

ALTER SEQUENCE public.audit_sessions_id_seq OWNED BY public.audit_sessions.id;


--
-- Name: audited_assets; Type: TABLE; Schema: public; Owner: admin
--

CREATE TABLE public.audited_assets (
    id bigint NOT NULL,
    session_id bigint NOT NULL,
    asset_id bigint NOT NULL,
    status character varying(50) DEFAULT 'Missing'::character varying NOT NULL,
    found_at timestamp with time zone,
    notes text
);


ALTER TABLE public.audited_assets OWNER TO admin;

--
-- Name: audited_assets_id_seq; Type: SEQUENCE; Schema: public; Owner: admin
--

CREATE SEQUENCE public.audited_assets_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


ALTER TABLE public.audited_assets_id_seq OWNER TO admin;

--
-- Name: audited_assets_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: admin
--

ALTER SEQUENCE public.audited_assets_id_seq OWNED BY public.audited_assets.id;


--
-- Name: budget_alerts; Type: TABLE; Schema: public; Owner: admin
--

CREATE TABLE public.budget_alerts (
    id bigint NOT NULL,
    budget_id bigint NOT NULL,
    usage_pct numeric(5,2) NOT NULL,
    alerted_at timestamp with time zone DEFAULT now() NOT NULL
);


ALTER TABLE public.budget_alerts OWNER TO admin;

--
-- Name: budget_alerts_id_seq; Type: SEQUENCE; Schema: public; Owner: admin
--

CREATE SEQUENCE public.budget_alerts_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


ALTER TABLE public.budget_alerts_id_seq OWNER TO admin;

--
-- Name: budget_alerts_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: admin
--

ALTER SEQUENCE public.budget_alerts_id_seq OWNED BY public.budget_alerts.id;


--
-- Name: budget_transactions; Type: TABLE; Schema: public; Owner: admin
--

CREATE TABLE public.budget_transactions (
    id bigint NOT NULL,
    budget_id bigint NOT NULL,
    contract_id bigint,
    license_id bigint,
    asset_id bigint,
    amount numeric(18,2) NOT NULL,
    transaction_date timestamp with time zone DEFAULT now() NOT NULL,
    notes text,
    created_by bigint,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL,
    entity_type text,
    entity_id bigint,
    currency character varying(10),
    exchange_rate numeric(18,6),
    tax_amount numeric(18,2),
    category character varying(20),
    cost_center_id bigint,
    CONSTRAINT budget_transactions_amount_check CHECK ((amount <> (0)::numeric))
);


ALTER TABLE public.budget_transactions OWNER TO admin;

--
-- Name: TABLE budget_transactions; Type: COMMENT; Schema: public; Owner: admin
--

COMMENT ON TABLE public.budget_transactions IS 'Transaksi realisasi anggaran (CAPEX/OPEX) untuk Contract, License, dan Asset — sesuai ISO/IEC 19770-10:2025 Grade A.';


--
-- Name: budget_transactions_id_seq; Type: SEQUENCE; Schema: public; Owner: admin
--

CREATE SEQUENCE public.budget_transactions_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


ALTER TABLE public.budget_transactions_id_seq OWNER TO admin;

--
-- Name: budget_transactions_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: admin
--

ALTER SEQUENCE public.budget_transactions_id_seq OWNED BY public.budget_transactions.id;


--
-- Name: budgets; Type: TABLE; Schema: public; Owner: admin
--

CREATE TABLE public.budgets (
    id bigint NOT NULL,
    name character varying(255) NOT NULL,
    department_id bigint,
    start_date date NOT NULL,
    end_date date NOT NULL,
    total_amount numeric(15,2) DEFAULT 0 NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    deleted_at timestamp with time zone,
    category character varying(20) DEFAULT 'CAPEX'::character varying,
    currency character varying(10) DEFAULT 'IDR'::character varying,
    budget_year integer,
    approved_by bigint,
    used_amount numeric(15,2) DEFAULT 0 NOT NULL,
    updated_at timestamp with time zone DEFAULT now(),
    cost_center_id bigint
);


ALTER TABLE public.budgets OWNER TO admin;

--
-- Name: budgets_id_seq; Type: SEQUENCE; Schema: public; Owner: admin
--

CREATE SEQUENCE public.budgets_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


ALTER TABLE public.budgets_id_seq OWNER TO admin;

--
-- Name: budgets_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: admin
--

ALTER SEQUENCE public.budgets_id_seq OWNED BY public.budgets.id;


--
-- Name: compliance_alerts; Type: TABLE; Schema: public; Owner: admin
--

CREATE TABLE public.compliance_alerts (
    id bigint NOT NULL,
    message text NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL
);


ALTER TABLE public.compliance_alerts OWNER TO admin;

--
-- Name: compliance_alerts_id_seq; Type: SEQUENCE; Schema: public; Owner: admin
--

CREATE SEQUENCE public.compliance_alerts_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


ALTER TABLE public.compliance_alerts_id_seq OWNER TO admin;

--
-- Name: compliance_alerts_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: admin
--

ALTER SEQUENCE public.compliance_alerts_id_seq OWNED BY public.compliance_alerts.id;


--
-- Name: departments; Type: TABLE; Schema: public; Owner: admin
--

CREATE TABLE public.departments (
    id bigint NOT NULL,
    name character varying(255) NOT NULL,
    created_at timestamp with time zone DEFAULT now(),
    updated_at timestamp with time zone DEFAULT now(),
    deleted_at timestamp with time zone,
    manager_id bigint,
    created_by bigint,
    updated_by bigint,
    cost_center_id bigint
);


ALTER TABLE public.departments OWNER TO admin;

--
-- Name: employees; Type: TABLE; Schema: public; Owner: admin
--

CREATE TABLE public.employees (
    id bigint NOT NULL,
    employee_nik character varying(50) NOT NULL,
    name character varying(255) NOT NULL,
    email character varying(255) NOT NULL,
    department_id bigint,
    password_hash character varying(255) NOT NULL,
    role character varying(50) DEFAULT 'employee'::character varying NOT NULL,
    deleted_at timestamp with time zone,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL,
    last_login_at timestamp with time zone,
    CONSTRAINT chk_role_dept CHECK ((((role)::text = 'super_admin'::text) OR (department_id IS NOT NULL)))
);


ALTER TABLE public.employees OWNER TO admin;

--
-- Name: tickets; Type: TABLE; Schema: public; Owner: admin
--

CREATE TABLE public.tickets (
    id bigint NOT NULL,
    subject character varying(255) NOT NULL,
    description text,
    status character varying(50) DEFAULT 'Open'::character varying NOT NULL,
    priority character varying(50) DEFAULT 'Medium'::character varying NOT NULL,
    created_by_employee_id bigint NOT NULL,
    assigned_to_employee_id bigint,
    related_asset_id bigint,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL,
    deleted_at timestamp with time zone,
    category_code text,
    service_code text,
    impact text,
    urgency text,
    sla_policy_id bigint,
    sla_due_at timestamp with time zone,
    sla_breached_at timestamp with time zone,
    response_due_at timestamp with time zone,
    last_assigned_at timestamp with time zone,
    last_assigned_by bigint,
    updated_by bigint,
    last_status_changed_at timestamp with time zone,
    due_date timestamp with time zone,
    resolved_at timestamp with time zone,
    closed_at timestamp with time zone,
    category_tier text,
    linked_problem_id bigint,
    escalation_level integer DEFAULT 0,
    breach_flag boolean DEFAULT false,
    compliance_flag boolean DEFAULT false,
    compliance_score numeric(5,2),
    response_completed_at timestamp with time zone,
    response_time_minutes integer,
    resolution_time_minutes integer,
    CONSTRAINT tickets_impact_check CHECK ((impact = ANY (ARRAY['Low'::text, 'Medium'::text, 'High'::text]))),
    CONSTRAINT tickets_urgency_check CHECK ((urgency = ANY (ARRAY['Low'::text, 'Medium'::text, 'High'::text])))
);


ALTER TABLE public.tickets OWNER TO admin;

--
-- Name: compliance_index; Type: VIEW; Schema: public; Owner: admin
--

CREATE VIEW public.compliance_index AS
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


ALTER TABLE public.compliance_index OWNER TO admin;

--
-- Name: VIEW compliance_index; Type: COMMENT; Schema: public; Owner: admin
--

COMMENT ON VIEW public.compliance_index IS 'KPI kepatuhan ITAM/ITSM per departemen (via assigned_to_employee_id).';


--
-- Name: sla_policies; Type: TABLE; Schema: public; Owner: admin
--

CREATE TABLE public.sla_policies (
    id bigint NOT NULL,
    name text NOT NULL,
    category_code text,
    service_code text,
    impact text NOT NULL,
    urgency text NOT NULL,
    resulting_priority text NOT NULL,
    response_minutes integer NOT NULL,
    resolve_minutes integer NOT NULL,
    is_active boolean DEFAULT true NOT NULL,
    compliance_score numeric(5,2) DEFAULT 100,
    legacy_compliance_score double precision,
    deleted_at timestamp with time zone,
    created_by bigint,
    updated_by bigint,
    created_at timestamp with time zone DEFAULT now(),
    updated_at timestamp with time zone DEFAULT now(),
    CONSTRAINT sla_policies_impact_check CHECK ((impact = ANY (ARRAY['Low'::text, 'Medium'::text, 'High'::text]))),
    CONSTRAINT sla_policies_resulting_priority_check CHECK ((resulting_priority = ANY (ARRAY['Low'::text, 'Medium'::text, 'High'::text, 'Critical'::text]))),
    CONSTRAINT sla_policies_urgency_check CHECK ((urgency = ANY (ARRAY['Low'::text, 'Medium'::text, 'High'::text])))
);


ALTER TABLE public.sla_policies OWNER TO admin;

--
-- Name: sla_violation_report; Type: MATERIALIZED VIEW; Schema: public; Owner: admin
--

CREATE MATERIALIZED VIEW public.sla_violation_report AS
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


ALTER TABLE public.sla_violation_report OWNER TO admin;

--
-- Name: MATERIALIZED VIEW sla_violation_report; Type: COMMENT; Schema: public; Owner: admin
--

COMMENT ON MATERIALIZED VIEW public.sla_violation_report IS 'Laporan tiket yang melewati batas waktu respons/resolusi (SLA violation).';


--
-- Name: sla_compliance_score; Type: VIEW; Schema: public; Owner: admin
--

CREATE VIEW public.sla_compliance_score AS
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


ALTER TABLE public.sla_compliance_score OWNER TO admin;

--
-- Name: VIEW sla_compliance_score; Type: COMMENT; Schema: public; Owner: admin
--

COMMENT ON VIEW public.sla_compliance_score IS 'SLA compliance KPI per departemen (via assigned_to_employee_id).';


--
-- Name: compliance_summary; Type: VIEW; Schema: public; Owner: admin
--

CREATE VIEW public.compliance_summary AS
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


ALTER TABLE public.compliance_summary OWNER TO admin;

--
-- Name: VIEW compliance_summary; Type: COMMENT; Schema: public; Owner: admin
--

COMMENT ON VIEW public.compliance_summary IS 'Ringkasan kepatuhan gabungan (Asset + Ticket + SLA + Audit) per departemen via assigned_to_employee_id dan tanggal audit terakhir.';


--
-- Name: compliance_trend; Type: TABLE; Schema: public; Owner: admin
--

CREATE TABLE public.compliance_trend (
    id bigint NOT NULL,
    last_value numeric(5,2) NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL
);


ALTER TABLE public.compliance_trend OWNER TO admin;

--
-- Name: compliance_trend_id_seq; Type: SEQUENCE; Schema: public; Owner: admin
--

CREATE SEQUENCE public.compliance_trend_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


ALTER TABLE public.compliance_trend_id_seq OWNER TO admin;

--
-- Name: compliance_trend_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: admin
--

ALTER SEQUENCE public.compliance_trend_id_seq OWNED BY public.compliance_trend.id;


--
-- Name: contracts; Type: TABLE; Schema: public; Owner: admin
--

CREATE TABLE public.contracts (
    id bigint NOT NULL,
    contract_number text NOT NULL,
    vendor text,
    contract_type text,
    start_date date NOT NULL,
    end_date date,
    total_value numeric(15,2),
    currency character varying(10) DEFAULT 'IDR'::character varying,
    payment_terms text,
    contact_person text,
    contact_email text,
    attachment_url text,
    notes text,
    status text DEFAULT 'active'::text,
    created_at timestamp with time zone DEFAULT now(),
    updated_at timestamp with time zone DEFAULT now(),
    updated_by bigint,
    deleted_at timestamp with time zone,
    budget_id bigint,
    renewal_date date,
    termination_notice_days integer DEFAULT 30,
    created_by bigint,
    cost_center_id bigint,
    CONSTRAINT contracts_status_check CHECK ((status = ANY (ARRAY['active'::text, 'expired'::text, 'terminated'::text])))
);


ALTER TABLE public.contracts OWNER TO admin;

--
-- Name: contracts_id_seq; Type: SEQUENCE; Schema: public; Owner: admin
--

ALTER TABLE public.contracts ALTER COLUMN id ADD GENERATED ALWAYS AS IDENTITY (
    SEQUENCE NAME public.contracts_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1
);


--
-- Name: cost_centers; Type: TABLE; Schema: public; Owner: admin
--

CREATE TABLE public.cost_centers (
    id bigint NOT NULL,
    code text NOT NULL,
    name text NOT NULL,
    deleted_at timestamp with time zone,
    created_at timestamp with time zone DEFAULT now(),
    updated_at timestamp with time zone DEFAULT now(),
    created_by bigint,
    updated_by bigint
);


ALTER TABLE public.cost_centers OWNER TO admin;

--
-- Name: COLUMN cost_centers.deleted_at; Type: COMMENT; Schema: public; Owner: admin
--

COMMENT ON COLUMN public.cost_centers.deleted_at IS 'Soft delete timestamp';


--
-- Name: cost_centers_id_seq; Type: SEQUENCE; Schema: public; Owner: admin
--

CREATE SEQUENCE public.cost_centers_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


ALTER TABLE public.cost_centers_id_seq OWNER TO admin;

--
-- Name: cost_centers_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: admin
--

ALTER SEQUENCE public.cost_centers_id_seq OWNED BY public.cost_centers.id;


--
-- Name: data_governance; Type: TABLE; Schema: public; Owner: admin
--

CREATE TABLE public.data_governance (
    entity_name character varying(100) NOT NULL,
    owner_employee_id bigint,
    retention_period interval DEFAULT '5 years'::interval,
    last_reviewed_at timestamp with time zone DEFAULT now(),
    notes text
);


ALTER TABLE public.data_governance OWNER TO admin;

--
-- Name: TABLE data_governance; Type: COMMENT; Schema: public; Owner: admin
--

COMMENT ON TABLE public.data_governance IS 'Menetapkan pemilik data & masa retensi (ISO 19770-10 A.5 & A.8).';


--
-- Name: departments_id_seq; Type: SEQUENCE; Schema: public; Owner: admin
--

CREATE SEQUENCE public.departments_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


ALTER TABLE public.departments_id_seq OWNER TO admin;

--
-- Name: departments_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: admin
--

ALTER SEQUENCE public.departments_id_seq OWNED BY public.departments.id;


--
-- Name: email_logs; Type: TABLE; Schema: public; Owner: admin
--

CREATE TABLE public.email_logs (
    id bigint NOT NULL,
    recipient text NOT NULL,
    subject text NOT NULL,
    body_preview text,
    status character varying(20) DEFAULT 'SENT'::character varying NOT NULL,
    error_message text,
    created_at timestamp with time zone DEFAULT now() NOT NULL
);


ALTER TABLE public.email_logs OWNER TO admin;

--
-- Name: email_logs_id_seq; Type: SEQUENCE; Schema: public; Owner: admin
--

CREATE SEQUENCE public.email_logs_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


ALTER TABLE public.email_logs_id_seq OWNER TO admin;

--
-- Name: email_logs_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: admin
--

ALTER SEQUENCE public.email_logs_id_seq OWNED BY public.email_logs.id;


--
-- Name: employee_trainings; Type: TABLE; Schema: public; Owner: admin
--

CREATE TABLE public.employee_trainings (
    id bigint NOT NULL,
    employee_id bigint NOT NULL,
    training_name character varying(255) NOT NULL,
    certificate_url text,
    completed_at date,
    created_at timestamp with time zone DEFAULT now()
);


ALTER TABLE public.employee_trainings OWNER TO admin;

--
-- Name: employee_trainings_id_seq; Type: SEQUENCE; Schema: public; Owner: admin
--

CREATE SEQUENCE public.employee_trainings_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


ALTER TABLE public.employee_trainings_id_seq OWNER TO admin;

--
-- Name: employee_trainings_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: admin
--

ALTER SEQUENCE public.employee_trainings_id_seq OWNED BY public.employee_trainings.id;


--
-- Name: employees_id_seq; Type: SEQUENCE; Schema: public; Owner: admin
--

CREATE SEQUENCE public.employees_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


ALTER TABLE public.employees_id_seq OWNER TO admin;

--
-- Name: employees_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: admin
--

ALTER SEQUENCE public.employees_id_seq OWNED BY public.employees.id;


--
-- Name: governance_review_feedback; Type: TABLE; Schema: public; Owner: admin
--

CREATE TABLE public.governance_review_feedback (
    id bigint NOT NULL,
    asset_id bigint NOT NULL,
    reviewer_id bigint,
    risk_index numeric(6,2) NOT NULL,
    system_note text,
    reviewer_comment text,
    reviewer_decision boolean,
    created_at timestamp with time zone DEFAULT now(),
    updated_at timestamp with time zone DEFAULT now()
);


ALTER TABLE public.governance_review_feedback OWNER TO admin;

--
-- Name: governance_review_feedback_id_seq; Type: SEQUENCE; Schema: public; Owner: admin
--

CREATE SEQUENCE public.governance_review_feedback_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


ALTER TABLE public.governance_review_feedback_id_seq OWNER TO admin;

--
-- Name: governance_review_feedback_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: admin
--

ALTER SEQUENCE public.governance_review_feedback_id_seq OWNED BY public.governance_review_feedback.id;


--
-- Name: kg_edges; Type: TABLE; Schema: public; Owner: admin
--

CREATE TABLE public.kg_edges (
    id bigint NOT NULL,
    src_node_id bigint NOT NULL,
    dst_node_id bigint NOT NULL,
    rel_type character varying(40) NOT NULL,
    weight numeric(6,3) DEFAULT 1,
    props jsonb DEFAULT '{}'::jsonb NOT NULL,
    created_at timestamp with time zone DEFAULT now(),
    updated_at timestamp with time zone DEFAULT now()
);


ALTER TABLE public.kg_edges OWNER TO admin;

--
-- Name: kg_edges_id_seq; Type: SEQUENCE; Schema: public; Owner: admin
--

CREATE SEQUENCE public.kg_edges_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


ALTER TABLE public.kg_edges_id_seq OWNER TO admin;

--
-- Name: kg_edges_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: admin
--

ALTER SEQUENCE public.kg_edges_id_seq OWNED BY public.kg_edges.id;


--
-- Name: kg_nodes; Type: TABLE; Schema: public; Owner: admin
--

CREATE TABLE public.kg_nodes (
    id bigint NOT NULL,
    entity_type character varying(30) NOT NULL,
    entity_id bigint NOT NULL,
    label text NOT NULL,
    props jsonb DEFAULT '{}'::jsonb NOT NULL,
    created_at timestamp with time zone DEFAULT now(),
    updated_at timestamp with time zone DEFAULT now()
);


ALTER TABLE public.kg_nodes OWNER TO admin;

--
-- Name: kg_nodes_id_seq; Type: SEQUENCE; Schema: public; Owner: admin
--

CREATE SEQUENCE public.kg_nodes_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


ALTER TABLE public.kg_nodes_id_seq OWNER TO admin;

--
-- Name: kg_nodes_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: admin
--

ALTER SEQUENCE public.kg_nodes_id_seq OWNED BY public.kg_nodes.id;


--
-- Name: licenses; Type: TABLE; Schema: public; Owner: admin
--

CREATE TABLE public.licenses (
    id bigint NOT NULL,
    name character varying(255) NOT NULL,
    license_key character varying(255),
    total_seats integer NOT NULL,
    purchase_date date,
    expiration_date date,
    cost numeric(15,2),
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    deleted_at timestamp with time zone,
    vendor text,
    publisher text,
    version text,
    license_type text,
    license_model text,
    contract_id bigint,
    category text,
    metric text,
    maintenance_expiry date,
    compliance_status text DEFAULT 'unknown'::text,
    verification_date timestamp with time zone,
    updated_at timestamp with time zone DEFAULT now(),
    updated_by bigint,
    entitlement_doc text,
    procurement_reference text,
    budget_id bigint,
    currency character varying(10) DEFAULT 'IDR'::character varying,
    compliance_score numeric(5,2),
    created_by bigint,
    document_hash text,
    CONSTRAINT licenses_compliance_status_check CHECK ((compliance_status = ANY (ARRAY['compliant'::text, 'non-compliant'::text, 'unknown'::text])))
);


ALTER TABLE public.licenses OWNER TO admin;

--
-- Name: locations; Type: TABLE; Schema: public; Owner: admin
--

CREATE TABLE public.locations (
    id bigint NOT NULL,
    site text NOT NULL,
    building text,
    room text,
    description text,
    status character varying(20) DEFAULT 'active'::character varying,
    created_at timestamp with time zone DEFAULT now(),
    updated_at timestamp with time zone DEFAULT now(),
    created_by bigint,
    updated_by bigint,
    deleted_at timestamp with time zone
);


ALTER TABLE public.locations OWNER TO admin;

--
-- Name: locations_id_seq; Type: SEQUENCE; Schema: public; Owner: admin
--

CREATE SEQUENCE public.locations_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


ALTER TABLE public.locations_id_seq OWNER TO admin;

--
-- Name: locations_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: admin
--

ALTER SEQUENCE public.locations_id_seq OWNED BY public.locations.id;


--
-- Name: ml_calibration_models; Type: TABLE; Schema: public; Owner: admin
--

CREATE TABLE public.ml_calibration_models (
    id bigint NOT NULL,
    model_name character varying(100) NOT NULL,
    last_trained_at timestamp with time zone DEFAULT now(),
    total_samples integer DEFAULT 0,
    avg_error numeric(6,3) DEFAULT 0,
    parameters jsonb DEFAULT '{}'::jsonb,
    created_at timestamp with time zone DEFAULT now(),
    updated_at timestamp with time zone DEFAULT now()
);


ALTER TABLE public.ml_calibration_models OWNER TO admin;

--
-- Name: ml_calibration_models_id_seq; Type: SEQUENCE; Schema: public; Owner: admin
--

CREATE SEQUENCE public.ml_calibration_models_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


ALTER TABLE public.ml_calibration_models_id_seq OWNER TO admin;

--
-- Name: ml_calibration_models_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: admin
--

ALTER SEQUENCE public.ml_calibration_models_id_seq OWNED BY public.ml_calibration_models.id;


--
-- Name: problems; Type: TABLE; Schema: public; Owner: admin
--

CREATE TABLE public.problems (
    id bigint NOT NULL,
    title text,
    description text,
    status text DEFAULT 'Open'::text,
    created_at timestamp with time zone DEFAULT now()
);


ALTER TABLE public.problems OWNER TO admin;

--
-- Name: problems_id_seq; Type: SEQUENCE; Schema: public; Owner: admin
--

ALTER TABLE public.problems ALTER COLUMN id ADD GENERATED ALWAYS AS IDENTITY (
    SEQUENCE NAME public.problems_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1
);


--
-- Name: role_delegations; Type: TABLE; Schema: public; Owner: admin
--

CREATE TABLE public.role_delegations (
    id bigint NOT NULL,
    delegator_id bigint,
    delegatee_id bigint,
    role_override character varying(50) NOT NULL,
    start_date date NOT NULL,
    end_date date NOT NULL,
    created_at timestamp with time zone DEFAULT now()
);


ALTER TABLE public.role_delegations OWNER TO admin;

--
-- Name: role_delegations_id_seq; Type: SEQUENCE; Schema: public; Owner: admin
--

CREATE SEQUENCE public.role_delegations_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


ALTER TABLE public.role_delegations_id_seq OWNER TO admin;

--
-- Name: role_delegations_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: admin
--

ALTER SEQUENCE public.role_delegations_id_seq OWNED BY public.role_delegations.id;


--
-- Name: services; Type: TABLE; Schema: public; Owner: admin
--

CREATE TABLE public.services (
    id bigint NOT NULL,
    code text NOT NULL,
    name text NOT NULL
);


ALTER TABLE public.services OWNER TO admin;

--
-- Name: services_id_seq; Type: SEQUENCE; Schema: public; Owner: admin
--

CREATE SEQUENCE public.services_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


ALTER TABLE public.services_id_seq OWNER TO admin;

--
-- Name: services_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: admin
--

ALTER SEQUENCE public.services_id_seq OWNED BY public.services.id;


--
-- Name: sla_policies_id_seq; Type: SEQUENCE; Schema: public; Owner: admin
--

CREATE SEQUENCE public.sla_policies_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


ALTER TABLE public.sla_policies_id_seq OWNER TO admin;

--
-- Name: sla_policies_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: admin
--

ALTER SEQUENCE public.sla_policies_id_seq OWNED BY public.sla_policies.id;


--
-- Name: software_installations; Type: TABLE; Schema: public; Owner: admin
--

CREATE TABLE public.software_installations (
    id bigint NOT NULL,
    asset_id bigint NOT NULL,
    license_id bigint NOT NULL,
    installation_date timestamp with time zone DEFAULT now() NOT NULL,
    notes text,
    removed_at timestamp with time zone
);


ALTER TABLE public.software_installations OWNER TO admin;

--
-- Name: software_installations_id_seq; Type: SEQUENCE; Schema: public; Owner: admin
--

CREATE SEQUENCE public.software_installations_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


ALTER TABLE public.software_installations_id_seq OWNER TO admin;

--
-- Name: software_installations_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: admin
--

ALTER SEQUENCE public.software_installations_id_seq OWNED BY public.software_installations.id;


--
-- Name: software_licenses_id_seq; Type: SEQUENCE; Schema: public; Owner: admin
--

CREATE SEQUENCE public.software_licenses_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


ALTER TABLE public.software_licenses_id_seq OWNER TO admin;

--
-- Name: software_licenses_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: admin
--

ALTER SEQUENCE public.software_licenses_id_seq OWNED BY public.licenses.id;


--
-- Name: ticket_attachments; Type: TABLE; Schema: public; Owner: admin
--

CREATE TABLE public.ticket_attachments (
    id bigint NOT NULL,
    ticket_id bigint NOT NULL,
    comment_id bigint,
    filename text NOT NULL,
    path text NOT NULL,
    mime_type text,
    size bigint,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    deleted_at timestamp without time zone
);


ALTER TABLE public.ticket_attachments OWNER TO admin;

--
-- Name: ticket_attachments_id_seq; Type: SEQUENCE; Schema: public; Owner: admin
--

CREATE SEQUENCE public.ticket_attachments_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


ALTER TABLE public.ticket_attachments_id_seq OWNER TO admin;

--
-- Name: ticket_attachments_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: admin
--

ALTER SEQUENCE public.ticket_attachments_id_seq OWNED BY public.ticket_attachments.id;


--
-- Name: ticket_categories; Type: TABLE; Schema: public; Owner: admin
--

CREATE TABLE public.ticket_categories (
    id bigint NOT NULL,
    code text NOT NULL,
    name text NOT NULL
);


ALTER TABLE public.ticket_categories OWNER TO admin;

--
-- Name: ticket_categories_id_seq; Type: SEQUENCE; Schema: public; Owner: admin
--

CREATE SEQUENCE public.ticket_categories_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


ALTER TABLE public.ticket_categories_id_seq OWNER TO admin;

--
-- Name: ticket_categories_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: admin
--

ALTER SEQUENCE public.ticket_categories_id_seq OWNED BY public.ticket_categories.id;


--
-- Name: ticket_comments; Type: TABLE; Schema: public; Owner: admin
--

CREATE TABLE public.ticket_comments (
    id bigint NOT NULL,
    ticket_id bigint NOT NULL,
    employee_id bigint NOT NULL,
    comment text NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    deleted_at timestamp without time zone
);


ALTER TABLE public.ticket_comments OWNER TO admin;

--
-- Name: ticket_comments_id_seq; Type: SEQUENCE; Schema: public; Owner: admin
--

CREATE SEQUENCE public.ticket_comments_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


ALTER TABLE public.ticket_comments_id_seq OWNER TO admin;

--
-- Name: ticket_comments_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: admin
--

ALTER SEQUENCE public.ticket_comments_id_seq OWNED BY public.ticket_comments.id;


--
-- Name: tickets_id_seq; Type: SEQUENCE; Schema: public; Owner: admin
--

CREATE SEQUENCE public.tickets_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


ALTER TABLE public.tickets_id_seq OWNER TO admin;

--
-- Name: tickets_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: admin
--

ALTER SEQUENCE public.tickets_id_seq OWNED BY public.tickets.id;


--
-- Name: v_budget_overview; Type: VIEW; Schema: public; Owner: admin
--

CREATE VIEW public.v_budget_overview AS
SELECT
    NULL::bigint AS budget_id,
    NULL::character varying(255) AS budget_name,
    NULL::character varying(20) AS category,
    NULL::character varying AS currency,
    NULL::numeric(15,2) AS total_amount,
    NULL::numeric AS realized_amount,
    NULL::numeric AS remaining_amount,
    NULL::numeric AS realization_percent,
    NULL::text AS status;


ALTER TABLE public.v_budget_overview OWNER TO admin;

--
-- Name: v_security_audit; Type: VIEW; Schema: public; Owner: admin
--

CREATE VIEW public.v_security_audit AS
 SELECT a.id,
    a.entity_name,
    a.action,
    a.actor_id,
    e.name AS actor_name,
    a.request_path,
    a.created_at
   FROM (public.audit_logs a
     LEFT JOIN public.employees e ON ((e.id = a.actor_id)))
  WHERE (lower((a.action)::text) = ANY (ARRAY['login'::text, 'logout'::text, 'token_refresh'::text, 'change_password'::text, 'failed_login'::text, 'get'::text, 'post'::text, 'put'::text, 'delete'::text]))
  ORDER BY a.created_at DESC;


ALTER TABLE public.v_security_audit OWNER TO admin;

--
-- Name: asset_changes id; Type: DEFAULT; Schema: cdc; Owner: admin
--

ALTER TABLE ONLY cdc.asset_changes ALTER COLUMN id SET DEFAULT nextval('cdc.asset_changes_id_seq'::regclass);


--
-- Name: alerts id; Type: DEFAULT; Schema: public; Owner: admin
--

ALTER TABLE ONLY public.alerts ALTER COLUMN id SET DEFAULT nextval('public.alerts_id_seq'::regclass);


--
-- Name: asset_assignments id; Type: DEFAULT; Schema: public; Owner: admin
--

ALTER TABLE ONLY public.asset_assignments ALTER COLUMN id SET DEFAULT nextval('public.asset_assignments_id_seq'::regclass);


--
-- Name: asset_history id; Type: DEFAULT; Schema: public; Owner: admin
--

ALTER TABLE ONLY public.asset_history ALTER COLUMN id SET DEFAULT nextval('public.asset_history_id_seq'::regclass);


--
-- Name: asset_maintenance_logs id; Type: DEFAULT; Schema: public; Owner: admin
--

ALTER TABLE ONLY public.asset_maintenance_logs ALTER COLUMN id SET DEFAULT nextval('public.asset_maintenance_logs_id_seq'::regclass);


--
-- Name: asset_types id; Type: DEFAULT; Schema: public; Owner: admin
--

ALTER TABLE ONLY public.asset_types ALTER COLUMN id SET DEFAULT nextval('public.asset_types_id_seq'::regclass);


--
-- Name: assets id; Type: DEFAULT; Schema: public; Owner: admin
--

ALTER TABLE ONLY public.assets ALTER COLUMN id SET DEFAULT nextval('public.assets_id_seq'::regclass);


--
-- Name: audit_logs id; Type: DEFAULT; Schema: public; Owner: admin
--

ALTER TABLE ONLY public.audit_logs ALTER COLUMN id SET DEFAULT nextval('public.audit_logs_id_seq'::regclass);


--
-- Name: audit_sessions id; Type: DEFAULT; Schema: public; Owner: admin
--

ALTER TABLE ONLY public.audit_sessions ALTER COLUMN id SET DEFAULT nextval('public.audit_sessions_id_seq'::regclass);


--
-- Name: audited_assets id; Type: DEFAULT; Schema: public; Owner: admin
--

ALTER TABLE ONLY public.audited_assets ALTER COLUMN id SET DEFAULT nextval('public.audited_assets_id_seq'::regclass);


--
-- Name: budget_alerts id; Type: DEFAULT; Schema: public; Owner: admin
--

ALTER TABLE ONLY public.budget_alerts ALTER COLUMN id SET DEFAULT nextval('public.budget_alerts_id_seq'::regclass);


--
-- Name: budget_transactions id; Type: DEFAULT; Schema: public; Owner: admin
--

ALTER TABLE ONLY public.budget_transactions ALTER COLUMN id SET DEFAULT nextval('public.budget_transactions_id_seq'::regclass);


--
-- Name: budgets id; Type: DEFAULT; Schema: public; Owner: admin
--

ALTER TABLE ONLY public.budgets ALTER COLUMN id SET DEFAULT nextval('public.budgets_id_seq'::regclass);


--
-- Name: compliance_alerts id; Type: DEFAULT; Schema: public; Owner: admin
--

ALTER TABLE ONLY public.compliance_alerts ALTER COLUMN id SET DEFAULT nextval('public.compliance_alerts_id_seq'::regclass);


--
-- Name: compliance_trend id; Type: DEFAULT; Schema: public; Owner: admin
--

ALTER TABLE ONLY public.compliance_trend ALTER COLUMN id SET DEFAULT nextval('public.compliance_trend_id_seq'::regclass);


--
-- Name: cost_centers id; Type: DEFAULT; Schema: public; Owner: admin
--

ALTER TABLE ONLY public.cost_centers ALTER COLUMN id SET DEFAULT nextval('public.cost_centers_id_seq'::regclass);


--
-- Name: departments id; Type: DEFAULT; Schema: public; Owner: admin
--

ALTER TABLE ONLY public.departments ALTER COLUMN id SET DEFAULT nextval('public.departments_id_seq'::regclass);


--
-- Name: email_logs id; Type: DEFAULT; Schema: public; Owner: admin
--

ALTER TABLE ONLY public.email_logs ALTER COLUMN id SET DEFAULT nextval('public.email_logs_id_seq'::regclass);


--
-- Name: employee_trainings id; Type: DEFAULT; Schema: public; Owner: admin
--

ALTER TABLE ONLY public.employee_trainings ALTER COLUMN id SET DEFAULT nextval('public.employee_trainings_id_seq'::regclass);


--
-- Name: employees id; Type: DEFAULT; Schema: public; Owner: admin
--

ALTER TABLE ONLY public.employees ALTER COLUMN id SET DEFAULT nextval('public.employees_id_seq'::regclass);


--
-- Name: governance_review_feedback id; Type: DEFAULT; Schema: public; Owner: admin
--

ALTER TABLE ONLY public.governance_review_feedback ALTER COLUMN id SET DEFAULT nextval('public.governance_review_feedback_id_seq'::regclass);


--
-- Name: kg_edges id; Type: DEFAULT; Schema: public; Owner: admin
--

ALTER TABLE ONLY public.kg_edges ALTER COLUMN id SET DEFAULT nextval('public.kg_edges_id_seq'::regclass);


--
-- Name: kg_nodes id; Type: DEFAULT; Schema: public; Owner: admin
--

ALTER TABLE ONLY public.kg_nodes ALTER COLUMN id SET DEFAULT nextval('public.kg_nodes_id_seq'::regclass);


--
-- Name: licenses id; Type: DEFAULT; Schema: public; Owner: admin
--

ALTER TABLE ONLY public.licenses ALTER COLUMN id SET DEFAULT nextval('public.software_licenses_id_seq'::regclass);


--
-- Name: locations id; Type: DEFAULT; Schema: public; Owner: admin
--

ALTER TABLE ONLY public.locations ALTER COLUMN id SET DEFAULT nextval('public.locations_id_seq'::regclass);


--
-- Name: ml_calibration_models id; Type: DEFAULT; Schema: public; Owner: admin
--

ALTER TABLE ONLY public.ml_calibration_models ALTER COLUMN id SET DEFAULT nextval('public.ml_calibration_models_id_seq'::regclass);


--
-- Name: role_delegations id; Type: DEFAULT; Schema: public; Owner: admin
--

ALTER TABLE ONLY public.role_delegations ALTER COLUMN id SET DEFAULT nextval('public.role_delegations_id_seq'::regclass);


--
-- Name: services id; Type: DEFAULT; Schema: public; Owner: admin
--

ALTER TABLE ONLY public.services ALTER COLUMN id SET DEFAULT nextval('public.services_id_seq'::regclass);


--
-- Name: sla_policies id; Type: DEFAULT; Schema: public; Owner: admin
--

ALTER TABLE ONLY public.sla_policies ALTER COLUMN id SET DEFAULT nextval('public.sla_policies_id_seq'::regclass);


--
-- Name: software_installations id; Type: DEFAULT; Schema: public; Owner: admin
--

ALTER TABLE ONLY public.software_installations ALTER COLUMN id SET DEFAULT nextval('public.software_installations_id_seq'::regclass);


--
-- Name: ticket_attachments id; Type: DEFAULT; Schema: public; Owner: admin
--

ALTER TABLE ONLY public.ticket_attachments ALTER COLUMN id SET DEFAULT nextval('public.ticket_attachments_id_seq'::regclass);


--
-- Name: ticket_categories id; Type: DEFAULT; Schema: public; Owner: admin
--

ALTER TABLE ONLY public.ticket_categories ALTER COLUMN id SET DEFAULT nextval('public.ticket_categories_id_seq'::regclass);


--
-- Name: ticket_comments id; Type: DEFAULT; Schema: public; Owner: admin
--

ALTER TABLE ONLY public.ticket_comments ALTER COLUMN id SET DEFAULT nextval('public.ticket_comments_id_seq'::regclass);


--
-- Name: tickets id; Type: DEFAULT; Schema: public; Owner: admin
--

ALTER TABLE ONLY public.tickets ALTER COLUMN id SET DEFAULT nextval('public.tickets_id_seq'::regclass);


--
-- Name: asset_changes asset_changes_pkey; Type: CONSTRAINT; Schema: cdc; Owner: admin
--

ALTER TABLE ONLY cdc.asset_changes
    ADD CONSTRAINT asset_changes_pkey PRIMARY KEY (id);


--
-- Name: alerts alerts_pkey; Type: CONSTRAINT; Schema: public; Owner: admin
--

ALTER TABLE ONLY public.alerts
    ADD CONSTRAINT alerts_pkey PRIMARY KEY (id);


--
-- Name: asset_assignments asset_assignments_pkey; Type: CONSTRAINT; Schema: public; Owner: admin
--

ALTER TABLE ONLY public.asset_assignments
    ADD CONSTRAINT asset_assignments_pkey PRIMARY KEY (id);


--
-- Name: asset_history asset_history_pkey; Type: CONSTRAINT; Schema: public; Owner: admin
--

ALTER TABLE ONLY public.asset_history
    ADD CONSTRAINT asset_history_pkey PRIMARY KEY (id);


--
-- Name: asset_maintenance_logs asset_maintenance_logs_pkey; Type: CONSTRAINT; Schema: public; Owner: admin
--

ALTER TABLE ONLY public.asset_maintenance_logs
    ADD CONSTRAINT asset_maintenance_logs_pkey PRIMARY KEY (id);


--
-- Name: asset_types asset_types_name_key; Type: CONSTRAINT; Schema: public; Owner: admin
--

ALTER TABLE ONLY public.asset_types
    ADD CONSTRAINT asset_types_name_key UNIQUE (name);


--
-- Name: asset_types asset_types_pkey; Type: CONSTRAINT; Schema: public; Owner: admin
--

ALTER TABLE ONLY public.asset_types
    ADD CONSTRAINT asset_types_pkey PRIMARY KEY (id);


--
-- Name: assets assets_asset_tag_key; Type: CONSTRAINT; Schema: public; Owner: admin
--

ALTER TABLE ONLY public.assets
    ADD CONSTRAINT assets_asset_tag_key UNIQUE (asset_tag);


--
-- Name: assets assets_pkey; Type: CONSTRAINT; Schema: public; Owner: admin
--

ALTER TABLE ONLY public.assets
    ADD CONSTRAINT assets_pkey PRIMARY KEY (id);


--
-- Name: audit_logs audit_logs_pkey; Type: CONSTRAINT; Schema: public; Owner: admin
--

ALTER TABLE ONLY public.audit_logs
    ADD CONSTRAINT audit_logs_pkey PRIMARY KEY (id);


--
-- Name: audit_sessions audit_sessions_pkey; Type: CONSTRAINT; Schema: public; Owner: admin
--

ALTER TABLE ONLY public.audit_sessions
    ADD CONSTRAINT audit_sessions_pkey PRIMARY KEY (id);


--
-- Name: audited_assets audited_assets_pkey; Type: CONSTRAINT; Schema: public; Owner: admin
--

ALTER TABLE ONLY public.audited_assets
    ADD CONSTRAINT audited_assets_pkey PRIMARY KEY (id);


--
-- Name: audited_assets audited_assets_session_id_asset_id_key; Type: CONSTRAINT; Schema: public; Owner: admin
--

ALTER TABLE ONLY public.audited_assets
    ADD CONSTRAINT audited_assets_session_id_asset_id_key UNIQUE (session_id, asset_id);


--
-- Name: budget_alerts budget_alerts_pkey; Type: CONSTRAINT; Schema: public; Owner: admin
--

ALTER TABLE ONLY public.budget_alerts
    ADD CONSTRAINT budget_alerts_pkey PRIMARY KEY (id);


--
-- Name: budget_transactions budget_transactions_pkey; Type: CONSTRAINT; Schema: public; Owner: admin
--

ALTER TABLE ONLY public.budget_transactions
    ADD CONSTRAINT budget_transactions_pkey PRIMARY KEY (id);


--
-- Name: budgets budgets_pkey; Type: CONSTRAINT; Schema: public; Owner: admin
--

ALTER TABLE ONLY public.budgets
    ADD CONSTRAINT budgets_pkey PRIMARY KEY (id);


--
-- Name: compliance_alerts compliance_alerts_pkey; Type: CONSTRAINT; Schema: public; Owner: admin
--

ALTER TABLE ONLY public.compliance_alerts
    ADD CONSTRAINT compliance_alerts_pkey PRIMARY KEY (id);


--
-- Name: compliance_trend compliance_trend_pkey; Type: CONSTRAINT; Schema: public; Owner: admin
--

ALTER TABLE ONLY public.compliance_trend
    ADD CONSTRAINT compliance_trend_pkey PRIMARY KEY (id);


--
-- Name: contracts contracts_pkey; Type: CONSTRAINT; Schema: public; Owner: admin
--

ALTER TABLE ONLY public.contracts
    ADD CONSTRAINT contracts_pkey PRIMARY KEY (id);


--
-- Name: cost_centers cost_centers_code_key; Type: CONSTRAINT; Schema: public; Owner: admin
--

ALTER TABLE ONLY public.cost_centers
    ADD CONSTRAINT cost_centers_code_key UNIQUE (code);


--
-- Name: cost_centers cost_centers_pkey; Type: CONSTRAINT; Schema: public; Owner: admin
--

ALTER TABLE ONLY public.cost_centers
    ADD CONSTRAINT cost_centers_pkey PRIMARY KEY (id);


--
-- Name: data_governance data_governance_pkey; Type: CONSTRAINT; Schema: public; Owner: admin
--

ALTER TABLE ONLY public.data_governance
    ADD CONSTRAINT data_governance_pkey PRIMARY KEY (entity_name);


--
-- Name: departments departments_name_key; Type: CONSTRAINT; Schema: public; Owner: admin
--

ALTER TABLE ONLY public.departments
    ADD CONSTRAINT departments_name_key UNIQUE (name);


--
-- Name: departments departments_pkey; Type: CONSTRAINT; Schema: public; Owner: admin
--

ALTER TABLE ONLY public.departments
    ADD CONSTRAINT departments_pkey PRIMARY KEY (id);


--
-- Name: email_logs email_logs_pkey; Type: CONSTRAINT; Schema: public; Owner: admin
--

ALTER TABLE ONLY public.email_logs
    ADD CONSTRAINT email_logs_pkey PRIMARY KEY (id);


--
-- Name: employee_trainings employee_trainings_pkey; Type: CONSTRAINT; Schema: public; Owner: admin
--

ALTER TABLE ONLY public.employee_trainings
    ADD CONSTRAINT employee_trainings_pkey PRIMARY KEY (id);


--
-- Name: employees employees_email_key; Type: CONSTRAINT; Schema: public; Owner: admin
--

ALTER TABLE ONLY public.employees
    ADD CONSTRAINT employees_email_key UNIQUE (email);


--
-- Name: employees employees_employee_nik_key; Type: CONSTRAINT; Schema: public; Owner: admin
--

ALTER TABLE ONLY public.employees
    ADD CONSTRAINT employees_employee_nik_key UNIQUE (employee_nik);


--
-- Name: employees employees_pkey; Type: CONSTRAINT; Schema: public; Owner: admin
--

ALTER TABLE ONLY public.employees
    ADD CONSTRAINT employees_pkey PRIMARY KEY (id);


--
-- Name: governance_review_feedback governance_review_feedback_pkey; Type: CONSTRAINT; Schema: public; Owner: admin
--

ALTER TABLE ONLY public.governance_review_feedback
    ADD CONSTRAINT governance_review_feedback_pkey PRIMARY KEY (id);


--
-- Name: kg_edges kg_edges_pkey; Type: CONSTRAINT; Schema: public; Owner: admin
--

ALTER TABLE ONLY public.kg_edges
    ADD CONSTRAINT kg_edges_pkey PRIMARY KEY (id);


--
-- Name: kg_nodes kg_nodes_entity_type_entity_id_key; Type: CONSTRAINT; Schema: public; Owner: admin
--

ALTER TABLE ONLY public.kg_nodes
    ADD CONSTRAINT kg_nodes_entity_type_entity_id_key UNIQUE (entity_type, entity_id);


--
-- Name: kg_nodes kg_nodes_pkey; Type: CONSTRAINT; Schema: public; Owner: admin
--

ALTER TABLE ONLY public.kg_nodes
    ADD CONSTRAINT kg_nodes_pkey PRIMARY KEY (id);


--
-- Name: locations locations_pkey; Type: CONSTRAINT; Schema: public; Owner: admin
--

ALTER TABLE ONLY public.locations
    ADD CONSTRAINT locations_pkey PRIMARY KEY (id);


--
-- Name: ml_calibration_models ml_calibration_models_pkey; Type: CONSTRAINT; Schema: public; Owner: admin
--

ALTER TABLE ONLY public.ml_calibration_models
    ADD CONSTRAINT ml_calibration_models_pkey PRIMARY KEY (id);


--
-- Name: problems problems_pkey; Type: CONSTRAINT; Schema: public; Owner: admin
--

ALTER TABLE ONLY public.problems
    ADD CONSTRAINT problems_pkey PRIMARY KEY (id);


--
-- Name: role_delegations role_delegations_pkey; Type: CONSTRAINT; Schema: public; Owner: admin
--

ALTER TABLE ONLY public.role_delegations
    ADD CONSTRAINT role_delegations_pkey PRIMARY KEY (id);


--
-- Name: services services_code_key; Type: CONSTRAINT; Schema: public; Owner: admin
--

ALTER TABLE ONLY public.services
    ADD CONSTRAINT services_code_key UNIQUE (code);


--
-- Name: services services_pkey; Type: CONSTRAINT; Schema: public; Owner: admin
--

ALTER TABLE ONLY public.services
    ADD CONSTRAINT services_pkey PRIMARY KEY (id);


--
-- Name: sla_policies sla_policies_pkey; Type: CONSTRAINT; Schema: public; Owner: admin
--

ALTER TABLE ONLY public.sla_policies
    ADD CONSTRAINT sla_policies_pkey PRIMARY KEY (id);


--
-- Name: software_installations software_installations_pkey; Type: CONSTRAINT; Schema: public; Owner: admin
--

ALTER TABLE ONLY public.software_installations
    ADD CONSTRAINT software_installations_pkey PRIMARY KEY (id);


--
-- Name: licenses software_licenses_pkey; Type: CONSTRAINT; Schema: public; Owner: admin
--

ALTER TABLE ONLY public.licenses
    ADD CONSTRAINT software_licenses_pkey PRIMARY KEY (id);


--
-- Name: ticket_attachments ticket_attachments_pkey; Type: CONSTRAINT; Schema: public; Owner: admin
--

ALTER TABLE ONLY public.ticket_attachments
    ADD CONSTRAINT ticket_attachments_pkey PRIMARY KEY (id);


--
-- Name: ticket_categories ticket_categories_code_key; Type: CONSTRAINT; Schema: public; Owner: admin
--

ALTER TABLE ONLY public.ticket_categories
    ADD CONSTRAINT ticket_categories_code_key UNIQUE (code);


--
-- Name: ticket_categories ticket_categories_pkey; Type: CONSTRAINT; Schema: public; Owner: admin
--

ALTER TABLE ONLY public.ticket_categories
    ADD CONSTRAINT ticket_categories_pkey PRIMARY KEY (id);


--
-- Name: ticket_comments ticket_comments_pkey; Type: CONSTRAINT; Schema: public; Owner: admin
--

ALTER TABLE ONLY public.ticket_comments
    ADD CONSTRAINT ticket_comments_pkey PRIMARY KEY (id);


--
-- Name: tickets tickets_pkey; Type: CONSTRAINT; Schema: public; Owner: admin
--

ALTER TABLE ONLY public.tickets
    ADD CONSTRAINT tickets_pkey PRIMARY KEY (id);


--
-- Name: assets unique_asset_tag; Type: CONSTRAINT; Schema: public; Owner: admin
--

ALTER TABLE ONLY public.assets
    ADD CONSTRAINT unique_asset_tag UNIQUE (asset_tag);


--
-- Name: employees unique_email; Type: CONSTRAINT; Schema: public; Owner: admin
--

ALTER TABLE ONLY public.employees
    ADD CONSTRAINT unique_email UNIQUE (email);


--
-- Name: licenses unique_license_key; Type: CONSTRAINT; Schema: public; Owner: admin
--

ALTER TABLE ONLY public.licenses
    ADD CONSTRAINT unique_license_key UNIQUE (license_key);


--
-- Name: sla_policies unique_sla_combo; Type: CONSTRAINT; Schema: public; Owner: admin
--

ALTER TABLE ONLY public.sla_policies
    ADD CONSTRAINT unique_sla_combo UNIQUE (impact, urgency, category_code, service_code);


--
-- Name: idx_alerts_acknowledged; Type: INDEX; Schema: public; Owner: admin
--

CREATE INDEX idx_alerts_acknowledged ON public.alerts USING btree (acknowledged);


--
-- Name: idx_asset_assignments_returned_by_employee_id; Type: INDEX; Schema: public; Owner: admin
--

CREATE INDEX idx_asset_assignments_returned_by_employee_id ON public.asset_assignments USING btree (returned_by_employee_id);


--
-- Name: idx_assets_asset_criticality; Type: INDEX; Schema: public; Owner: admin
--

CREATE INDEX idx_assets_asset_criticality ON public.assets USING btree (asset_criticality);


--
-- Name: idx_assets_compliance_flag; Type: INDEX; Schema: public; Owner: admin
--

CREATE INDEX idx_assets_compliance_flag ON public.assets USING btree (compliance_flag);


--
-- Name: idx_assets_contract_id; Type: INDEX; Schema: public; Owner: admin
--

CREATE INDEX idx_assets_contract_id ON public.assets USING btree (contract_id);


--
-- Name: idx_assets_deleted_at; Type: INDEX; Schema: public; Owner: admin
--

CREATE INDEX idx_assets_deleted_at ON public.assets USING btree (deleted_at);


--
-- Name: idx_assets_department_id; Type: INDEX; Schema: public; Owner: admin
--

CREATE INDEX idx_assets_department_id ON public.assets USING btree (department_id);


--
-- Name: idx_assets_governance_score; Type: INDEX; Schema: public; Owner: admin
--

CREATE INDEX idx_assets_governance_score ON public.assets USING btree (governance_score);


--
-- Name: idx_assets_lifecycle_stage; Type: INDEX; Schema: public; Owner: admin
--

CREATE INDEX idx_assets_lifecycle_stage ON public.assets USING btree (lifecycle_stage);


--
-- Name: idx_assets_location_id; Type: INDEX; Schema: public; Owner: admin
--

CREATE INDEX idx_assets_location_id ON public.assets USING btree (location_id);


--
-- Name: idx_assets_updated_month; Type: INDEX; Schema: public; Owner: admin
--

CREATE INDEX idx_assets_updated_month ON public.assets USING btree (updated_month);


--
-- Name: idx_audit_logs_category; Type: INDEX; Schema: public; Owner: admin
--

CREATE INDEX idx_audit_logs_category ON public.audit_logs USING btree (category);


--
-- Name: idx_audit_logs_created_at; Type: INDEX; Schema: public; Owner: admin
--

CREATE INDEX idx_audit_logs_created_at ON public.audit_logs USING btree (created_at DESC);


--
-- Name: idx_audit_logs_severity; Type: INDEX; Schema: public; Owner: admin
--

CREATE INDEX idx_audit_logs_severity ON public.audit_logs USING btree (severity);


--
-- Name: idx_bt_asset; Type: INDEX; Schema: public; Owner: admin
--

CREATE INDEX idx_bt_asset ON public.budget_transactions USING btree (asset_id);


--
-- Name: idx_bt_budget; Type: INDEX; Schema: public; Owner: admin
--

CREATE INDEX idx_bt_budget ON public.budget_transactions USING btree (budget_id);


--
-- Name: idx_bt_contract; Type: INDEX; Schema: public; Owner: admin
--

CREATE INDEX idx_bt_contract ON public.budget_transactions USING btree (contract_id);


--
-- Name: idx_bt_cost_center; Type: INDEX; Schema: public; Owner: admin
--

CREATE INDEX idx_bt_cost_center ON public.budget_transactions USING btree (cost_center_id);


--
-- Name: idx_bt_license; Type: INDEX; Schema: public; Owner: admin
--

CREATE INDEX idx_bt_license ON public.budget_transactions USING btree (license_id);


--
-- Name: idx_budget_alerts_budget_at; Type: INDEX; Schema: public; Owner: admin
--

CREATE INDEX idx_budget_alerts_budget_at ON public.budget_alerts USING btree (budget_id, alerted_at DESC);


--
-- Name: idx_budget_tx_budget_id; Type: INDEX; Schema: public; Owner: admin
--

CREATE INDEX idx_budget_tx_budget_id ON public.budget_transactions USING btree (budget_id);


--
-- Name: idx_compliance_alerts_created_at; Type: INDEX; Schema: public; Owner: admin
--

CREATE INDEX idx_compliance_alerts_created_at ON public.compliance_alerts USING btree (created_at DESC);


--
-- Name: idx_compliance_trend_created_at; Type: INDEX; Schema: public; Owner: admin
--

CREATE INDEX idx_compliance_trend_created_at ON public.compliance_trend USING btree (created_at DESC);


--
-- Name: idx_contracts_number; Type: INDEX; Schema: public; Owner: admin
--

CREATE INDEX idx_contracts_number ON public.contracts USING btree (lower(contract_number));


--
-- Name: idx_contracts_status; Type: INDEX; Schema: public; Owner: admin
--

CREATE INDEX idx_contracts_status ON public.contracts USING btree (status);


--
-- Name: idx_contracts_vendor; Type: INDEX; Schema: public; Owner: admin
--

CREATE INDEX idx_contracts_vendor ON public.contracts USING btree (lower(vendor));


--
-- Name: idx_departments_manager_id; Type: INDEX; Schema: public; Owner: admin
--

CREATE INDEX idx_departments_manager_id ON public.departments USING btree (manager_id);


--
-- Name: idx_email_logs_created_at; Type: INDEX; Schema: public; Owner: admin
--

CREATE INDEX idx_email_logs_created_at ON public.email_logs USING btree (created_at DESC);


--
-- Name: idx_employee_trainings_employee_id; Type: INDEX; Schema: public; Owner: admin
--

CREATE INDEX idx_employee_trainings_employee_id ON public.employee_trainings USING btree (employee_id);


--
-- Name: idx_grf_asset; Type: INDEX; Schema: public; Owner: admin
--

CREATE INDEX idx_grf_asset ON public.governance_review_feedback USING btree (asset_id);


--
-- Name: idx_grf_reviewer; Type: INDEX; Schema: public; Owner: admin
--

CREATE INDEX idx_grf_reviewer ON public.governance_review_feedback USING btree (reviewer_id);


--
-- Name: idx_kg_edges_dst; Type: INDEX; Schema: public; Owner: admin
--

CREATE INDEX idx_kg_edges_dst ON public.kg_edges USING btree (dst_node_id);


--
-- Name: idx_kg_edges_rel; Type: INDEX; Schema: public; Owner: admin
--

CREATE INDEX idx_kg_edges_rel ON public.kg_edges USING btree (rel_type);


--
-- Name: idx_kg_edges_src; Type: INDEX; Schema: public; Owner: admin
--

CREATE INDEX idx_kg_edges_src ON public.kg_edges USING btree (src_node_id);


--
-- Name: idx_kg_nodes_type; Type: INDEX; Schema: public; Owner: admin
--

CREATE INDEX idx_kg_nodes_type ON public.kg_nodes USING btree (entity_type);


--
-- Name: idx_licenses_compliance_status; Type: INDEX; Schema: public; Owner: admin
--

CREATE INDEX idx_licenses_compliance_status ON public.licenses USING btree (compliance_status);


--
-- Name: idx_role_delegations_active; Type: INDEX; Schema: public; Owner: admin
--

CREATE INDEX idx_role_delegations_active ON public.role_delegations USING btree (start_date, end_date);


--
-- Name: idx_role_delegations_delegatee_id; Type: INDEX; Schema: public; Owner: admin
--

CREATE INDEX idx_role_delegations_delegatee_id ON public.role_delegations USING btree (delegatee_id);


--
-- Name: idx_sla_violation_response; Type: INDEX; Schema: public; Owner: admin
--

CREATE INDEX idx_sla_violation_response ON public.sla_violation_report USING btree (overdue_response_hours DESC);


--
-- Name: idx_software_installations_active; Type: INDEX; Schema: public; Owner: admin
--

CREATE INDEX idx_software_installations_active ON public.software_installations USING btree (license_id) WHERE (removed_at IS NULL);


--
-- Name: idx_software_installations_asset_id; Type: INDEX; Schema: public; Owner: admin
--

CREATE INDEX idx_software_installations_asset_id ON public.software_installations USING btree (asset_id);


--
-- Name: idx_software_installations_license_id; Type: INDEX; Schema: public; Owner: admin
--

CREATE INDEX idx_software_installations_license_id ON public.software_installations USING btree (license_id);


--
-- Name: idx_ticket_attachments_comment_id; Type: INDEX; Schema: public; Owner: admin
--

CREATE INDEX idx_ticket_attachments_comment_id ON public.ticket_attachments USING btree (comment_id);


--
-- Name: idx_ticket_comments_ticket_id; Type: INDEX; Schema: public; Owner: admin
--

CREATE INDEX idx_ticket_comments_ticket_id ON public.ticket_comments USING btree (ticket_id);


--
-- Name: idx_tickets_response_due_at; Type: INDEX; Schema: public; Owner: admin
--

CREATE INDEX idx_tickets_response_due_at ON public.tickets USING btree (response_due_at);


--
-- Name: idx_tickets_sla_due_at; Type: INDEX; Schema: public; Owner: admin
--

CREATE INDEX idx_tickets_sla_due_at ON public.tickets USING btree (sla_due_at);


--
-- Name: idx_tickets_sla_policy_id; Type: INDEX; Schema: public; Owner: admin
--

CREATE INDEX idx_tickets_sla_policy_id ON public.tickets USING btree (sla_policy_id);


--
-- Name: uniq_asset_history_hash; Type: INDEX; Schema: public; Owner: admin
--

CREATE UNIQUE INDEX uniq_asset_history_hash ON public.asset_history USING btree (hash) WHERE (hash IS NOT NULL);


--
-- Name: v_budget_overview _RETURN; Type: RULE; Schema: public; Owner: admin
--

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


--
-- Name: assets set_timestamp_assets; Type: TRIGGER; Schema: public; Owner: admin
--

CREATE TRIGGER set_timestamp_assets BEFORE UPDATE ON public.assets FOR EACH ROW EXECUTE FUNCTION public.set_timestamp();


--
-- Name: budgets set_timestamp_budgets; Type: TRIGGER; Schema: public; Owner: admin
--

CREATE TRIGGER set_timestamp_budgets BEFORE UPDATE ON public.budgets FOR EACH ROW EXECUTE FUNCTION public.set_timestamp();


--
-- Name: contracts set_timestamp_contracts; Type: TRIGGER; Schema: public; Owner: admin
--

CREATE TRIGGER set_timestamp_contracts BEFORE UPDATE ON public.contracts FOR EACH ROW EXECUTE FUNCTION public.set_timestamp();


--
-- Name: departments set_timestamp_departments; Type: TRIGGER; Schema: public; Owner: admin
--

CREATE TRIGGER set_timestamp_departments BEFORE UPDATE ON public.departments FOR EACH ROW EXECUTE FUNCTION public.set_timestamp();


--
-- Name: employees set_timestamp_employees; Type: TRIGGER; Schema: public; Owner: admin
--

CREATE TRIGGER set_timestamp_employees BEFORE UPDATE ON public.employees FOR EACH ROW EXECUTE FUNCTION public.set_timestamp();


--
-- Name: assets trg_assets_cdc; Type: TRIGGER; Schema: public; Owner: admin
--

CREATE TRIGGER trg_assets_cdc AFTER INSERT OR DELETE OR UPDATE ON public.assets FOR EACH ROW EXECUTE FUNCTION public.trg_assets_cdc();


--
-- Name: audit_logs trg_audit_hash_chain; Type: TRIGGER; Schema: public; Owner: admin
--

CREATE TRIGGER trg_audit_hash_chain BEFORE INSERT ON public.audit_logs FOR EACH ROW EXECUTE FUNCTION public.trg_audit_chain();


--
-- Name: budget_transactions trg_budget_tx_sync; Type: TRIGGER; Schema: public; Owner: admin
--

CREATE TRIGGER trg_budget_tx_sync AFTER INSERT OR DELETE OR UPDATE ON public.budget_transactions FOR EACH ROW EXECUTE FUNCTION public.trg_budget_tx_sync();


--
-- Name: assets trg_set_updated_month; Type: TRIGGER; Schema: public; Owner: admin
--

CREATE TRIGGER trg_set_updated_month BEFORE INSERT OR UPDATE ON public.assets FOR EACH ROW EXECUTE FUNCTION public.set_updated_month();


--
-- Name: alerts alerts_acknowledged_by_fkey; Type: FK CONSTRAINT; Schema: public; Owner: admin
--

ALTER TABLE ONLY public.alerts
    ADD CONSTRAINT alerts_acknowledged_by_fkey FOREIGN KEY (acknowledged_by) REFERENCES public.employees(id);


--
-- Name: alerts alerts_asset_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: admin
--

ALTER TABLE ONLY public.alerts
    ADD CONSTRAINT alerts_asset_id_fkey FOREIGN KEY (asset_id) REFERENCES public.assets(id) ON DELETE SET NULL;


--
-- Name: asset_assignments asset_assignments_asset_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: admin
--

ALTER TABLE ONLY public.asset_assignments
    ADD CONSTRAINT asset_assignments_asset_id_fkey FOREIGN KEY (asset_id) REFERENCES public.assets(id);


--
-- Name: asset_assignments asset_assignments_assigned_by_employee_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: admin
--

ALTER TABLE ONLY public.asset_assignments
    ADD CONSTRAINT asset_assignments_assigned_by_employee_id_fkey FOREIGN KEY (assigned_by_employee_id) REFERENCES public.employees(id);


--
-- Name: asset_assignments asset_assignments_employee_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: admin
--

ALTER TABLE ONLY public.asset_assignments
    ADD CONSTRAINT asset_assignments_employee_id_fkey FOREIGN KEY (employee_id) REFERENCES public.employees(id);


--
-- Name: asset_assignments asset_assignments_returned_by_employee_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: admin
--

ALTER TABLE ONLY public.asset_assignments
    ADD CONSTRAINT asset_assignments_returned_by_employee_id_fkey FOREIGN KEY (returned_by_employee_id) REFERENCES public.employees(id);


--
-- Name: asset_history asset_history_actor_employee_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: admin
--

ALTER TABLE ONLY public.asset_history
    ADD CONSTRAINT asset_history_actor_employee_id_fkey FOREIGN KEY (actor_employee_id) REFERENCES public.employees(id);


--
-- Name: asset_history asset_history_asset_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: admin
--

ALTER TABLE ONLY public.asset_history
    ADD CONSTRAINT asset_history_asset_id_fkey FOREIGN KEY (asset_id) REFERENCES public.assets(id) ON DELETE CASCADE;


--
-- Name: asset_maintenance_logs asset_maintenance_logs_asset_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: admin
--

ALTER TABLE ONLY public.asset_maintenance_logs
    ADD CONSTRAINT asset_maintenance_logs_asset_id_fkey FOREIGN KEY (asset_id) REFERENCES public.assets(id);


--
-- Name: asset_maintenance_logs asset_maintenance_logs_performed_by_employee_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: admin
--

ALTER TABLE ONLY public.asset_maintenance_logs
    ADD CONSTRAINT asset_maintenance_logs_performed_by_employee_id_fkey FOREIGN KEY (performed_by_employee_id) REFERENCES public.employees(id) ON UPDATE CASCADE ON DELETE SET NULL;


--
-- Name: asset_maintenance_logs asset_maintenance_logs_ticket_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: admin
--

ALTER TABLE ONLY public.asset_maintenance_logs
    ADD CONSTRAINT asset_maintenance_logs_ticket_id_fkey FOREIGN KEY (ticket_id) REFERENCES public.tickets(id);


--
-- Name: assets assets_asset_type_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: admin
--

ALTER TABLE ONLY public.assets
    ADD CONSTRAINT assets_asset_type_id_fkey FOREIGN KEY (asset_type_id) REFERENCES public.asset_types(id);


--
-- Name: assets assets_budget_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: admin
--

ALTER TABLE ONLY public.assets
    ADD CONSTRAINT assets_budget_id_fkey FOREIGN KEY (budget_id) REFERENCES public.budgets(id) ON DELETE SET NULL;


--
-- Name: assets assets_contract_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: admin
--

ALTER TABLE ONLY public.assets
    ADD CONSTRAINT assets_contract_id_fkey FOREIGN KEY (contract_id) REFERENCES public.contracts(id) ON DELETE SET NULL;


--
-- Name: assets assets_cost_center_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: admin
--

ALTER TABLE ONLY public.assets
    ADD CONSTRAINT assets_cost_center_id_fkey FOREIGN KEY (cost_center_id) REFERENCES public.cost_centers(id) ON DELETE SET NULL;


--
-- Name: assets assets_department_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: admin
--

ALTER TABLE ONLY public.assets
    ADD CONSTRAINT assets_department_id_fkey FOREIGN KEY (department_id) REFERENCES public.departments(id) ON DELETE SET NULL;


--
-- Name: assets assets_disposed_approved_by_fkey; Type: FK CONSTRAINT; Schema: public; Owner: admin
--

ALTER TABLE ONLY public.assets
    ADD CONSTRAINT assets_disposed_approved_by_fkey FOREIGN KEY (disposed_approved_by) REFERENCES public.employees(id) ON DELETE SET NULL;


--
-- Name: assets assets_location_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: admin
--

ALTER TABLE ONLY public.assets
    ADD CONSTRAINT assets_location_id_fkey FOREIGN KEY (location_id) REFERENCES public.locations(id) ON DELETE SET NULL;


--
-- Name: audit_logs audit_logs_actor_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: admin
--

ALTER TABLE ONLY public.audit_logs
    ADD CONSTRAINT audit_logs_actor_id_fkey FOREIGN KEY (actor_id) REFERENCES public.employees(id) ON DELETE SET NULL;


--
-- Name: audit_sessions audit_sessions_created_by_employee_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: admin
--

ALTER TABLE ONLY public.audit_sessions
    ADD CONSTRAINT audit_sessions_created_by_employee_id_fkey FOREIGN KEY (created_by_employee_id) REFERENCES public.employees(id);


--
-- Name: audited_assets audited_assets_asset_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: admin
--

ALTER TABLE ONLY public.audited_assets
    ADD CONSTRAINT audited_assets_asset_id_fkey FOREIGN KEY (asset_id) REFERENCES public.assets(id);


--
-- Name: audited_assets audited_assets_session_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: admin
--

ALTER TABLE ONLY public.audited_assets
    ADD CONSTRAINT audited_assets_session_id_fkey FOREIGN KEY (session_id) REFERENCES public.audit_sessions(id) ON DELETE CASCADE;


--
-- Name: budget_alerts budget_alerts_budget_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: admin
--

ALTER TABLE ONLY public.budget_alerts
    ADD CONSTRAINT budget_alerts_budget_id_fkey FOREIGN KEY (budget_id) REFERENCES public.budgets(id);


--
-- Name: budget_transactions budget_transactions_asset_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: admin
--

ALTER TABLE ONLY public.budget_transactions
    ADD CONSTRAINT budget_transactions_asset_id_fkey FOREIGN KEY (asset_id) REFERENCES public.assets(id) ON DELETE SET NULL;


--
-- Name: budget_transactions budget_transactions_budget_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: admin
--

ALTER TABLE ONLY public.budget_transactions
    ADD CONSTRAINT budget_transactions_budget_id_fkey FOREIGN KEY (budget_id) REFERENCES public.budgets(id) ON DELETE CASCADE;


--
-- Name: budget_transactions budget_transactions_contract_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: admin
--

ALTER TABLE ONLY public.budget_transactions
    ADD CONSTRAINT budget_transactions_contract_id_fkey FOREIGN KEY (contract_id) REFERENCES public.contracts(id) ON DELETE SET NULL;


--
-- Name: budget_transactions budget_transactions_created_by_fkey; Type: FK CONSTRAINT; Schema: public; Owner: admin
--

ALTER TABLE ONLY public.budget_transactions
    ADD CONSTRAINT budget_transactions_created_by_fkey FOREIGN KEY (created_by) REFERENCES public.employees(id) ON DELETE SET NULL;


--
-- Name: budget_transactions budget_transactions_license_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: admin
--

ALTER TABLE ONLY public.budget_transactions
    ADD CONSTRAINT budget_transactions_license_id_fkey FOREIGN KEY (license_id) REFERENCES public.licenses(id) ON DELETE SET NULL;


--
-- Name: budgets budgets_approved_by_fkey; Type: FK CONSTRAINT; Schema: public; Owner: admin
--

ALTER TABLE ONLY public.budgets
    ADD CONSTRAINT budgets_approved_by_fkey FOREIGN KEY (approved_by) REFERENCES public.employees(id);


--
-- Name: budgets budgets_cost_center_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: admin
--

ALTER TABLE ONLY public.budgets
    ADD CONSTRAINT budgets_cost_center_id_fkey FOREIGN KEY (cost_center_id) REFERENCES public.cost_centers(id);


--
-- Name: budgets budgets_department_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: admin
--

ALTER TABLE ONLY public.budgets
    ADD CONSTRAINT budgets_department_id_fkey FOREIGN KEY (department_id) REFERENCES public.departments(id);


--
-- Name: contracts contracts_budget_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: admin
--

ALTER TABLE ONLY public.contracts
    ADD CONSTRAINT contracts_budget_id_fkey FOREIGN KEY (budget_id) REFERENCES public.budgets(id);


--
-- Name: data_governance data_governance_owner_employee_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: admin
--

ALTER TABLE ONLY public.data_governance
    ADD CONSTRAINT data_governance_owner_employee_id_fkey FOREIGN KEY (owner_employee_id) REFERENCES public.employees(id) ON DELETE SET NULL;


--
-- Name: departments departments_cost_center_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: admin
--

ALTER TABLE ONLY public.departments
    ADD CONSTRAINT departments_cost_center_id_fkey FOREIGN KEY (cost_center_id) REFERENCES public.cost_centers(id);


--
-- Name: departments departments_manager_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: admin
--

ALTER TABLE ONLY public.departments
    ADD CONSTRAINT departments_manager_id_fkey FOREIGN KEY (manager_id) REFERENCES public.employees(id);


--
-- Name: employee_trainings employee_trainings_employee_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: admin
--

ALTER TABLE ONLY public.employee_trainings
    ADD CONSTRAINT employee_trainings_employee_id_fkey FOREIGN KEY (employee_id) REFERENCES public.employees(id);


--
-- Name: employees employees_department_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: admin
--

ALTER TABLE ONLY public.employees
    ADD CONSTRAINT employees_department_id_fkey FOREIGN KEY (department_id) REFERENCES public.departments(id);


--
-- Name: assets fk_assets_created_by; Type: FK CONSTRAINT; Schema: public; Owner: admin
--

ALTER TABLE ONLY public.assets
    ADD CONSTRAINT fk_assets_created_by FOREIGN KEY (created_by) REFERENCES public.employees(id) ON DELETE SET NULL;


--
-- Name: assets fk_assets_updated_by; Type: FK CONSTRAINT; Schema: public; Owner: admin
--

ALTER TABLE ONLY public.assets
    ADD CONSTRAINT fk_assets_updated_by FOREIGN KEY (updated_by) REFERENCES public.employees(id) ON DELETE SET NULL;


--
-- Name: budget_transactions fk_bt_cost_center; Type: FK CONSTRAINT; Schema: public; Owner: admin
--

ALTER TABLE ONLY public.budget_transactions
    ADD CONSTRAINT fk_bt_cost_center FOREIGN KEY (cost_center_id) REFERENCES public.cost_centers(id) ON UPDATE CASCADE ON DELETE SET NULL;


--
-- Name: contracts fk_contract_cost_center; Type: FK CONSTRAINT; Schema: public; Owner: admin
--

ALTER TABLE ONLY public.contracts
    ADD CONSTRAINT fk_contract_cost_center FOREIGN KEY (cost_center_id) REFERENCES public.cost_centers(id) ON DELETE SET NULL;


--
-- Name: contracts fk_contracts_created_by; Type: FK CONSTRAINT; Schema: public; Owner: admin
--

ALTER TABLE ONLY public.contracts
    ADD CONSTRAINT fk_contracts_created_by FOREIGN KEY (created_by) REFERENCES public.employees(id) ON DELETE SET NULL;


--
-- Name: departments fk_departments_created_by; Type: FK CONSTRAINT; Schema: public; Owner: admin
--

ALTER TABLE ONLY public.departments
    ADD CONSTRAINT fk_departments_created_by FOREIGN KEY (created_by) REFERENCES public.employees(id) ON DELETE SET NULL;


--
-- Name: departments fk_departments_updated_by; Type: FK CONSTRAINT; Schema: public; Owner: admin
--

ALTER TABLE ONLY public.departments
    ADD CONSTRAINT fk_departments_updated_by FOREIGN KEY (updated_by) REFERENCES public.employees(id) ON DELETE SET NULL;


--
-- Name: licenses fk_licenses_contract; Type: FK CONSTRAINT; Schema: public; Owner: admin
--

ALTER TABLE ONLY public.licenses
    ADD CONSTRAINT fk_licenses_contract FOREIGN KEY (contract_id) REFERENCES public.contracts(id) ON UPDATE CASCADE ON DELETE SET NULL;


--
-- Name: licenses fk_licenses_created_by; Type: FK CONSTRAINT; Schema: public; Owner: admin
--

ALTER TABLE ONLY public.licenses
    ADD CONSTRAINT fk_licenses_created_by FOREIGN KEY (created_by) REFERENCES public.employees(id) ON DELETE SET NULL;


--
-- Name: locations fk_location_created_by; Type: FK CONSTRAINT; Schema: public; Owner: admin
--

ALTER TABLE ONLY public.locations
    ADD CONSTRAINT fk_location_created_by FOREIGN KEY (created_by) REFERENCES public.employees(id) ON DELETE SET NULL;


--
-- Name: locations fk_location_updated_by; Type: FK CONSTRAINT; Schema: public; Owner: admin
--

ALTER TABLE ONLY public.locations
    ADD CONSTRAINT fk_location_updated_by FOREIGN KEY (updated_by) REFERENCES public.employees(id) ON DELETE SET NULL;


--
-- Name: sla_policies fk_sla_created_by; Type: FK CONSTRAINT; Schema: public; Owner: admin
--

ALTER TABLE ONLY public.sla_policies
    ADD CONSTRAINT fk_sla_created_by FOREIGN KEY (created_by) REFERENCES public.employees(id) ON DELETE SET NULL;


--
-- Name: sla_policies fk_sla_updated_by; Type: FK CONSTRAINT; Schema: public; Owner: admin
--

ALTER TABLE ONLY public.sla_policies
    ADD CONSTRAINT fk_sla_updated_by FOREIGN KEY (updated_by) REFERENCES public.employees(id) ON DELETE SET NULL;


--
-- Name: tickets fk_ticket_linked_problem; Type: FK CONSTRAINT; Schema: public; Owner: admin
--

ALTER TABLE ONLY public.tickets
    ADD CONSTRAINT fk_ticket_linked_problem FOREIGN KEY (linked_problem_id) REFERENCES public.problems(id) ON DELETE SET NULL;


--
-- Name: governance_review_feedback governance_review_feedback_asset_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: admin
--

ALTER TABLE ONLY public.governance_review_feedback
    ADD CONSTRAINT governance_review_feedback_asset_id_fkey FOREIGN KEY (asset_id) REFERENCES public.assets(id) ON DELETE CASCADE;


--
-- Name: governance_review_feedback governance_review_feedback_reviewer_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: admin
--

ALTER TABLE ONLY public.governance_review_feedback
    ADD CONSTRAINT governance_review_feedback_reviewer_id_fkey FOREIGN KEY (reviewer_id) REFERENCES public.employees(id);


--
-- Name: kg_edges kg_edges_dst_node_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: admin
--

ALTER TABLE ONLY public.kg_edges
    ADD CONSTRAINT kg_edges_dst_node_id_fkey FOREIGN KEY (dst_node_id) REFERENCES public.kg_nodes(id) ON DELETE CASCADE;


--
-- Name: kg_edges kg_edges_src_node_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: admin
--

ALTER TABLE ONLY public.kg_edges
    ADD CONSTRAINT kg_edges_src_node_id_fkey FOREIGN KEY (src_node_id) REFERENCES public.kg_nodes(id) ON DELETE CASCADE;


--
-- Name: licenses licenses_budget_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: admin
--

ALTER TABLE ONLY public.licenses
    ADD CONSTRAINT licenses_budget_id_fkey FOREIGN KEY (budget_id) REFERENCES public.budgets(id) ON DELETE SET NULL;


--
-- Name: role_delegations role_delegations_delegatee_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: admin
--

ALTER TABLE ONLY public.role_delegations
    ADD CONSTRAINT role_delegations_delegatee_id_fkey FOREIGN KEY (delegatee_id) REFERENCES public.employees(id);


--
-- Name: role_delegations role_delegations_delegator_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: admin
--

ALTER TABLE ONLY public.role_delegations
    ADD CONSTRAINT role_delegations_delegator_id_fkey FOREIGN KEY (delegator_id) REFERENCES public.employees(id);


--
-- Name: sla_policies sla_policies_category_code_fkey; Type: FK CONSTRAINT; Schema: public; Owner: admin
--

ALTER TABLE ONLY public.sla_policies
    ADD CONSTRAINT sla_policies_category_code_fkey FOREIGN KEY (category_code) REFERENCES public.ticket_categories(code) ON DELETE SET NULL;


--
-- Name: sla_policies sla_policies_service_code_fkey; Type: FK CONSTRAINT; Schema: public; Owner: admin
--

ALTER TABLE ONLY public.sla_policies
    ADD CONSTRAINT sla_policies_service_code_fkey FOREIGN KEY (service_code) REFERENCES public.services(code) ON DELETE SET NULL;


--
-- Name: software_installations software_installations_asset_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: admin
--

ALTER TABLE ONLY public.software_installations
    ADD CONSTRAINT software_installations_asset_id_fkey FOREIGN KEY (asset_id) REFERENCES public.assets(id);


--
-- Name: software_installations software_installations_license_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: admin
--

ALTER TABLE ONLY public.software_installations
    ADD CONSTRAINT software_installations_license_id_fkey FOREIGN KEY (license_id) REFERENCES public.licenses(id);


--
-- Name: ticket_attachments ticket_attachments_comment_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: admin
--

ALTER TABLE ONLY public.ticket_attachments
    ADD CONSTRAINT ticket_attachments_comment_id_fkey FOREIGN KEY (comment_id) REFERENCES public.ticket_comments(id) ON DELETE CASCADE;


--
-- Name: ticket_attachments ticket_attachments_ticket_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: admin
--

ALTER TABLE ONLY public.ticket_attachments
    ADD CONSTRAINT ticket_attachments_ticket_id_fkey FOREIGN KEY (ticket_id) REFERENCES public.tickets(id) ON DELETE CASCADE;


--
-- Name: ticket_comments ticket_comments_employee_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: admin
--

ALTER TABLE ONLY public.ticket_comments
    ADD CONSTRAINT ticket_comments_employee_id_fkey FOREIGN KEY (employee_id) REFERENCES public.employees(id);


--
-- Name: ticket_comments ticket_comments_ticket_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: admin
--

ALTER TABLE ONLY public.ticket_comments
    ADD CONSTRAINT ticket_comments_ticket_id_fkey FOREIGN KEY (ticket_id) REFERENCES public.tickets(id) ON DELETE CASCADE;


--
-- Name: tickets tickets_assigned_to_employee_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: admin
--

ALTER TABLE ONLY public.tickets
    ADD CONSTRAINT tickets_assigned_to_employee_id_fkey FOREIGN KEY (assigned_to_employee_id) REFERENCES public.employees(id);


--
-- Name: tickets tickets_category_code_fkey; Type: FK CONSTRAINT; Schema: public; Owner: admin
--

ALTER TABLE ONLY public.tickets
    ADD CONSTRAINT tickets_category_code_fkey FOREIGN KEY (category_code) REFERENCES public.ticket_categories(code) ON DELETE SET NULL;


--
-- Name: tickets tickets_created_by_employee_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: admin
--

ALTER TABLE ONLY public.tickets
    ADD CONSTRAINT tickets_created_by_employee_id_fkey FOREIGN KEY (created_by_employee_id) REFERENCES public.employees(id);


--
-- Name: tickets tickets_last_assigned_by_fkey; Type: FK CONSTRAINT; Schema: public; Owner: admin
--

ALTER TABLE ONLY public.tickets
    ADD CONSTRAINT tickets_last_assigned_by_fkey FOREIGN KEY (last_assigned_by) REFERENCES public.employees(id);


--
-- Name: tickets tickets_related_asset_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: admin
--

ALTER TABLE ONLY public.tickets
    ADD CONSTRAINT tickets_related_asset_id_fkey FOREIGN KEY (related_asset_id) REFERENCES public.assets(id);


--
-- Name: tickets tickets_service_code_fkey; Type: FK CONSTRAINT; Schema: public; Owner: admin
--

ALTER TABLE ONLY public.tickets
    ADD CONSTRAINT tickets_service_code_fkey FOREIGN KEY (service_code) REFERENCES public.services(code) ON DELETE SET NULL;


--
-- Name: tickets tickets_sla_policy_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: admin
--

ALTER TABLE ONLY public.tickets
    ADD CONSTRAINT tickets_sla_policy_id_fkey FOREIGN KEY (sla_policy_id) REFERENCES public.sla_policies(id) ON DELETE SET NULL;


--
-- Name: tickets tickets_updated_by_fkey; Type: FK CONSTRAINT; Schema: public; Owner: admin
--

ALTER TABLE ONLY public.tickets
    ADD CONSTRAINT tickets_updated_by_fkey FOREIGN KEY (updated_by) REFERENCES public.employees(id);


--
-- Name: budgets; Type: ROW SECURITY; Schema: public; Owner: admin
--

ALTER TABLE public.budgets ENABLE ROW LEVEL SECURITY;

--
-- Name: budgets department_budget_access; Type: POLICY; Schema: public; Owner: admin
--

CREATE POLICY department_budget_access ON public.budgets FOR SELECT USING (((current_setting('app.current_department'::text, true) IS NOT NULL) AND (department_id = (current_setting('app.current_department'::text, true))::bigint)));


--
-- PostgreSQL database dump complete
--

