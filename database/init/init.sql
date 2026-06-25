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

CREATE SCHEMA IF NOT EXISTS cdc;


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
    returned_by_employee_id bigint,
    status character varying(20) DEFAULT 'active' NOT NULL,
    CONSTRAINT asset_assignments_status_check CHECK (((status)::text = ANY (ARRAY[('active'::character varying)::text, ('returned'::character varying)::text, ('lost'::character varying)::text, ('damaged'::character varying)::text])))
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
    notes text,
    verified_by bigint,
    verified_at timestamp with time zone
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
    priority character varying(20) DEFAULT 'Medium'::character varying NOT NULL,
    assigned_to bigint,
    created_by bigint,
    updated_by bigint,
    root_cause text,
    workaround text,
    known_error boolean DEFAULT false NOT NULL,
    permanent_solution text,
    related_asset_id bigint,
    created_at timestamp with time zone DEFAULT now(),
    updated_at timestamp with time zone DEFAULT now() NOT NULL,
    resolved_at timestamp with time zone,
    deleted_at timestamp with time zone,
    CONSTRAINT problems_status_check CHECK ((status = ANY (ARRAY['Open'::text, 'Under Investigation'::text, 'Known Error'::text, 'Resolved'::text, 'Closed'::text]))),
    CONSTRAINT problems_priority_check CHECK (((priority)::text = ANY (ARRAY[('Low'::character varying)::text, ('Medium'::character varying)::text, ('High'::character varying)::text, ('Critical'::character varying)::text])))
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
    created_at timestamp with time zone DEFAULT now(),
    is_active boolean DEFAULT true NOT NULL,
    reason text,
    revoked_at timestamp with time zone,
    revoked_by bigint
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
-- Phase 4 Migrations: Service Catalog & Service Request Management (ISO 20000-1 Cl. 8.6)
--

CREATE SEQUENCE IF NOT EXISTS public.sr_number_seq START WITH 1;

CREATE TABLE IF NOT EXISTS public.service_catalog (
    id                     BIGSERIAL PRIMARY KEY,
    code                   VARCHAR(50) NOT NULL UNIQUE,
    name                   VARCHAR(255) NOT NULL,
    category               VARCHAR(100),
    description            TEXT,
    sla_policy_id          BIGINT REFERENCES public.sla_policies(id) ON DELETE SET NULL,
    approval_required      BOOLEAN NOT NULL DEFAULT false,
    fulfillment_sla_minutes INT,
    is_active              BOOLEAN NOT NULL DEFAULT true,
    created_by             BIGINT REFERENCES public.employees(id),
    created_at             TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at             TIMESTAMPTZ NOT NULL DEFAULT now(),
    deleted_at             TIMESTAMPTZ
);
CREATE INDEX IF NOT EXISTS idx_sc_category   ON public.service_catalog (category);
CREATE INDEX IF NOT EXISTS idx_sc_is_active  ON public.service_catalog (is_active) WHERE is_active = true;
CREATE INDEX IF NOT EXISTS idx_sc_deleted_at ON public.service_catalog (deleted_at);

CREATE TABLE IF NOT EXISTS public.service_requests (
    id                  BIGSERIAL PRIMARY KEY,
    sr_number           VARCHAR(50) NOT NULL UNIQUE,
    service_catalog_id  BIGINT NOT NULL REFERENCES public.service_catalog(id),
    subject             VARCHAR(255) NOT NULL,
    description         TEXT,
    status              VARCHAR(30) NOT NULL DEFAULT 'submitted'
        CONSTRAINT sr_status_check CHECK (status IN (
            'submitted','pending_approval','approved',
            'in_fulfillment','completed','cancelled','rejected'
        )),
    priority            VARCHAR(20) NOT NULL DEFAULT 'Medium'
        CONSTRAINT sr_priority_check CHECK (priority IN ('Low','Medium','High','Critical')),
    requested_by        BIGINT NOT NULL REFERENCES public.employees(id),
    assigned_to         BIGINT REFERENCES public.employees(id),
    department_id       BIGINT REFERENCES public.departments(id),
    related_asset_id    BIGINT REFERENCES public.assets(id) ON DELETE SET NULL,
    notes               TEXT,
    fulfilled_at        TIMESTAMPTZ,
    closed_at           TIMESTAMPTZ,
    created_at          TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at          TIMESTAMPTZ NOT NULL DEFAULT now(),
    deleted_at          TIMESTAMPTZ
);
CREATE INDEX IF NOT EXISTS idx_sr_status        ON public.service_requests (status);
CREATE INDEX IF NOT EXISTS idx_sr_requested_by  ON public.service_requests (requested_by);
CREATE INDEX IF NOT EXISTS idx_sr_assigned_to   ON public.service_requests (assigned_to);
CREATE INDEX IF NOT EXISTS idx_sr_catalog_id    ON public.service_requests (service_catalog_id);
CREATE INDEX IF NOT EXISTS idx_sr_department_id ON public.service_requests (department_id);
CREATE INDEX IF NOT EXISTS idx_sr_deleted_at    ON public.service_requests (deleted_at);

CREATE TABLE IF NOT EXISTS public.approval_workflows (
    id          BIGSERIAL PRIMARY KEY,
    entity_type VARCHAR(50) NOT NULL
        CONSTRAINT aw_entity_check CHECK (entity_type IN ('service_request','change_request')),
    entity_id   BIGINT NOT NULL,
    level       INT NOT NULL DEFAULT 1,
    approver_id BIGINT NOT NULL REFERENCES public.employees(id),
    status      VARCHAR(20) NOT NULL DEFAULT 'pending'
        CONSTRAINT aw_status_check CHECK (status IN ('pending','approved','rejected','skipped')),
    comment     TEXT,
    decided_at  TIMESTAMPTZ,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT uq_approval_workflow UNIQUE (entity_type, entity_id, level, approver_id)
);
CREATE INDEX IF NOT EXISTS idx_aw_entity   ON public.approval_workflows (entity_type, entity_id);
CREATE INDEX IF NOT EXISTS idx_aw_approver ON public.approval_workflows (approver_id);
CREATE INDEX IF NOT EXISTS idx_aw_status   ON public.approval_workflows (status);
CREATE INDEX IF NOT EXISTS idx_aw_level    ON public.approval_workflows (entity_type, entity_id, level);

ALTER TABLE public.tickets
    ADD COLUMN IF NOT EXISTS ticket_type VARCHAR(20) NOT NULL DEFAULT 'incident'
        CONSTRAINT tickets_type_check CHECK (ticket_type IN ('incident','request','problem','change'));
CREATE INDEX IF NOT EXISTS idx_tickets_ticket_type ON public.tickets (ticket_type);

--
-- Phase 3 Migrations: Change Management (ISO 20000-1 Cl. 9.2 & ITIL)
--

CREATE SEQUENCE IF NOT EXISTS public.cr_number_seq START WITH 1;

CREATE TABLE IF NOT EXISTS public.change_requests (
    id                    BIGSERIAL PRIMARY KEY,
    cr_number             VARCHAR(50) NOT NULL UNIQUE,
    title                 VARCHAR(255) NOT NULL,
    description           TEXT,
    type                  VARCHAR(20) NOT NULL DEFAULT 'normal'
        CONSTRAINT cr_type_check CHECK (type IN ('standard','normal','emergency')),
    status                VARCHAR(30) NOT NULL DEFAULT 'draft'
        CONSTRAINT cr_status_check CHECK (status IN (
            'draft','submitted','under_review','approved',
            'scheduled','implementing','implemented','verified','closed','rejected'
        )),
    risk_level            VARCHAR(20) NOT NULL DEFAULT 'medium'
        CONSTRAINT cr_risk_check CHECK (risk_level IN ('low','medium','high','critical')),
    impact_assessment     TEXT,
    rollback_plan         TEXT,
    change_window_start   TIMESTAMPTZ,
    change_window_end     TIMESTAMPTZ,
    cab_required          BOOLEAN NOT NULL DEFAULT false,
    related_asset_id      BIGINT REFERENCES public.assets(id) ON DELETE SET NULL,
    related_ticket_id     BIGINT REFERENCES public.tickets(id) ON DELETE SET NULL,
    created_by            BIGINT REFERENCES public.employees(id),
    approved_by           BIGINT REFERENCES public.employees(id),
    implemented_by        BIGINT REFERENCES public.employees(id),
    submitted_at          TIMESTAMPTZ,
    approved_at           TIMESTAMPTZ,
    implemented_at        TIMESTAMPTZ,
    verified_at           TIMESTAMPTZ,
    closed_at             TIMESTAMPTZ,
    created_at            TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at            TIMESTAMPTZ NOT NULL DEFAULT now(),
    deleted_at            TIMESTAMPTZ
);
CREATE INDEX IF NOT EXISTS idx_cr_status       ON public.change_requests (status);
CREATE INDEX IF NOT EXISTS idx_cr_type         ON public.change_requests (type);
CREATE INDEX IF NOT EXISTS idx_cr_risk_level   ON public.change_requests (risk_level);
CREATE INDEX IF NOT EXISTS idx_cr_created_by   ON public.change_requests (created_by);
CREATE INDEX IF NOT EXISTS idx_cr_window_start ON public.change_requests (change_window_start);
CREATE INDEX IF NOT EXISTS idx_cr_deleted_at   ON public.change_requests (deleted_at);

CREATE TABLE IF NOT EXISTS public.change_approvals (
    id          BIGSERIAL PRIMARY KEY,
    change_id   BIGINT NOT NULL REFERENCES public.change_requests(id) ON DELETE CASCADE,
    approver_id BIGINT NOT NULL REFERENCES public.employees(id),
    decision    VARCHAR(20) NOT NULL
        CONSTRAINT ca_decision_check CHECK (decision IN ('approved','rejected','abstain','pending')),
    comment     TEXT,
    decided_at  TIMESTAMPTZ,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT uq_change_approval UNIQUE (change_id, approver_id)
);
CREATE INDEX IF NOT EXISTS idx_ca_change_id   ON public.change_approvals (change_id);
CREATE INDEX IF NOT EXISTS idx_ca_approver_id ON public.change_approvals (approver_id);
CREATE INDEX IF NOT EXISTS idx_ca_decision    ON public.change_approvals (decision);

CREATE TABLE IF NOT EXISTS public.change_tasks (
    id           BIGSERIAL PRIMARY KEY,
    change_id    BIGINT NOT NULL REFERENCES public.change_requests(id) ON DELETE CASCADE,
    title        VARCHAR(255) NOT NULL,
    description  TEXT,
    status       VARCHAR(20) NOT NULL DEFAULT 'pending'
        CONSTRAINT ct_status_check CHECK (status IN ('pending','in_progress','done','skipped')),
    assigned_to  BIGINT REFERENCES public.employees(id),
    seq_order    INT NOT NULL DEFAULT 1,
    completed_at TIMESTAMPTZ,
    created_at   TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at   TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS idx_ct_change_id   ON public.change_tasks (change_id);
CREATE INDEX IF NOT EXISTS idx_ct_assigned_to ON public.change_tasks (assigned_to);
CREATE INDEX IF NOT EXISTS idx_ct_status      ON public.change_tasks (status);

ALTER TABLE public.tickets
    ADD COLUMN IF NOT EXISTS change_request_id BIGINT
        REFERENCES public.change_requests(id) ON DELETE SET NULL;
CREATE INDEX IF NOT EXISTS idx_tickets_change_request_id ON public.tickets (change_request_id);

--
-- Phase 2 Migrations: Problem Management, Post-Mortem, Escalation Rules
--

CREATE TABLE IF NOT EXISTS public.problem_incidents (
    id          BIGSERIAL PRIMARY KEY,
    problem_id  BIGINT NOT NULL REFERENCES public.problems(id) ON DELETE CASCADE,
    ticket_id   BIGINT NOT NULL REFERENCES public.tickets(id)  ON DELETE CASCADE,
    linked_by   BIGINT REFERENCES public.employees(id),
    linked_at   TIMESTAMPTZ NOT NULL DEFAULT now(),
    notes       TEXT,
    CONSTRAINT uq_problem_incident UNIQUE (problem_id, ticket_id)
);
CREATE INDEX IF NOT EXISTS idx_pi_problem_id ON public.problem_incidents (problem_id);
CREATE INDEX IF NOT EXISTS idx_pi_ticket_id  ON public.problem_incidents (ticket_id);

CREATE TABLE IF NOT EXISTS public.incident_postmortems (
    id                   BIGSERIAL PRIMARY KEY,
    ticket_id            BIGINT NOT NULL REFERENCES public.tickets(id) ON DELETE CASCADE,
    problem_id           BIGINT REFERENCES public.problems(id) ON DELETE SET NULL,
    timeline             JSONB  NOT NULL DEFAULT '[]',
    root_cause           TEXT,
    contributing_factors TEXT,
    lessons_learned      TEXT,
    action_items         JSONB  NOT NULL DEFAULT '[]',
    reviewed_by          BIGINT REFERENCES public.employees(id),
    reviewed_at          TIMESTAMPTZ,
    created_by           BIGINT REFERENCES public.employees(id),
    created_at           TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at           TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT uq_postmortem_ticket UNIQUE (ticket_id)
);
CREATE INDEX IF NOT EXISTS idx_postmortem_ticket_id   ON public.incident_postmortems (ticket_id);
CREATE INDEX IF NOT EXISTS idx_postmortem_problem_id  ON public.incident_postmortems (problem_id);
CREATE INDEX IF NOT EXISTS idx_postmortem_reviewed_by ON public.incident_postmortems (reviewed_by);

CREATE TABLE IF NOT EXISTS public.escalation_rules (
    id                   BIGSERIAL PRIMARY KEY,
    name                 VARCHAR(255) NOT NULL,
    category_code        TEXT REFERENCES public.ticket_categories(code) ON DELETE SET NULL,
    service_code         TEXT REFERENCES public.services(code)          ON DELETE SET NULL,
    priority             VARCHAR(20)  NOT NULL
        CONSTRAINT escalation_rules_priority_check CHECK (priority IN ('Low','Medium','High','Critical')),
    trigger_after_minutes INT NOT NULL,
    action               VARCHAR(50)  NOT NULL DEFAULT 'reassign'
        CONSTRAINT escalation_rules_action_check CHECK (action IN ('reassign','notify','raise_priority','raise_escalation_level')),
    escalate_to_role     VARCHAR(50),
    escalate_to_employee BIGINT REFERENCES public.employees(id) ON DELETE SET NULL,
    notify_emails        TEXT,
    is_active            BOOLEAN      NOT NULL DEFAULT true,
    created_by           BIGINT REFERENCES public.employees(id),
    created_at           TIMESTAMPTZ  NOT NULL DEFAULT now(),
    updated_at           TIMESTAMPTZ  NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS idx_escr_priority  ON public.escalation_rules (priority);
CREATE INDEX IF NOT EXISTS idx_escr_is_active ON public.escalation_rules (is_active) WHERE is_active = true;
CREATE INDEX IF NOT EXISTS idx_escr_category  ON public.escalation_rules (category_code);

--
-- Phase 1 Migrations: Schema fixes for ITAM/ITSM compliance
--

-- FK: audited_assets.verified_by → employees
ALTER TABLE ONLY public.audited_assets
    ADD CONSTRAINT audited_assets_verified_by_fkey FOREIGN KEY (verified_by) REFERENCES public.employees(id);

-- FK: problems.assigned_to, created_by, updated_by, related_asset_id → employees/assets
ALTER TABLE ONLY public.problems
    ADD CONSTRAINT problems_assigned_to_fkey FOREIGN KEY (assigned_to) REFERENCES public.employees(id);
ALTER TABLE ONLY public.problems
    ADD CONSTRAINT problems_created_by_fkey FOREIGN KEY (created_by) REFERENCES public.employees(id);
ALTER TABLE ONLY public.problems
    ADD CONSTRAINT problems_updated_by_fkey FOREIGN KEY (updated_by) REFERENCES public.employees(id);
ALTER TABLE ONLY public.problems
    ADD CONSTRAINT problems_related_asset_id_fkey FOREIGN KEY (related_asset_id) REFERENCES public.assets(id);

-- FK: role_delegations.revoked_by → employees
ALTER TABLE ONLY public.role_delegations
    ADD CONSTRAINT role_delegations_revoked_by_fkey FOREIGN KEY (revoked_by) REFERENCES public.employees(id);

-- Indexes: problems
CREATE INDEX idx_problems_status      ON public.problems USING btree (status);
CREATE INDEX idx_problems_assigned_to ON public.problems USING btree (assigned_to);
CREATE INDEX idx_problems_known_error ON public.problems USING btree (known_error) WHERE known_error = true;

-- Index: role_delegations aktif
CREATE INDEX idx_role_delegations_is_active ON public.role_delegations USING btree (is_active) WHERE is_active = true;

-- ============================================================
-- FASE 7: Integrations — Webhooks, QR/Barcode, LDAP Sync, DR/BCP
-- ============================================================

-- 7.1 Webhook Subscriptions
CREATE TABLE IF NOT EXISTS public.webhook_subscriptions (
    id         BIGSERIAL PRIMARY KEY,
    name       VARCHAR(255) NOT NULL,
    url        TEXT NOT NULL,
    events     TEXT[] NOT NULL DEFAULT '{}',
    secret     VARCHAR(255),
    is_active  BOOLEAN NOT NULL DEFAULT true,
    created_by BIGINT REFERENCES public.employees(id) ON DELETE SET NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS idx_ws_is_active ON public.webhook_subscriptions (is_active);

-- 7.2 Webhook Delivery Logs
CREATE TABLE IF NOT EXISTS public.webhook_delivery_logs (
    id              BIGSERIAL PRIMARY KEY,
    subscription_id BIGINT NOT NULL REFERENCES public.webhook_subscriptions(id) ON DELETE CASCADE,
    event_type      VARCHAR(100) NOT NULL,
    payload         TEXT NOT NULL DEFAULT '{}',
    status          VARCHAR(20) NOT NULL DEFAULT 'pending'
        CONSTRAINT wdl_status_check CHECK (status IN ('pending','delivered','failed')),
    response_code   INT,
    response_body   TEXT,
    attempt_count   INT NOT NULL DEFAULT 0,
    last_attempt_at TIMESTAMPTZ,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS idx_wdl_subscription_id ON public.webhook_delivery_logs (subscription_id);
CREATE INDEX IF NOT EXISTS idx_wdl_status          ON public.webhook_delivery_logs (status);
CREATE INDEX IF NOT EXISTS idx_wdl_created_at      ON public.webhook_delivery_logs (created_at);

-- 7.3 Asset QR Codes
CREATE TABLE IF NOT EXISTS public.asset_qr_codes (
    id         BIGSERIAL PRIMARY KEY,
    asset_id   BIGINT NOT NULL REFERENCES public.assets(id) ON DELETE CASCADE,
    qr_data    TEXT NOT NULL,
    format     VARCHAR(20) NOT NULL DEFAULT 'qr'
        CONSTRAINT aqc_format_check CHECK (format IN ('qr','barcode','datamatrix')),
    label_data TEXT,
    printed_at TIMESTAMPTZ,
    printed_by BIGINT REFERENCES public.employees(id) ON DELETE SET NULL,
    created_by BIGINT REFERENCES public.employees(id) ON DELETE SET NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS idx_aqc_asset_id ON public.asset_qr_codes (asset_id);

-- 7.4 LDAP Sync Config
CREATE TABLE IF NOT EXISTS public.ldap_sync_configs (
    id            BIGSERIAL PRIMARY KEY,
    name          VARCHAR(100) NOT NULL DEFAULT 'default',
    host          VARCHAR(255) NOT NULL,
    port          INT NOT NULL DEFAULT 389,
    use_tls       BOOLEAN NOT NULL DEFAULT false,
    base_dn       TEXT NOT NULL,
    bind_dn       TEXT NOT NULL,
    bind_password TEXT,
    user_filter   TEXT NOT NULL DEFAULT '(objectClass=person)',
    field_map     TEXT NOT NULL DEFAULT '{"sAMAccountName":"username","cn":"name","mail":"email"}',
    is_active     BOOLEAN NOT NULL DEFAULT false,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at    TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- 7.5 LDAP Sync Logs
CREATE TABLE IF NOT EXISTS public.ldap_sync_logs (
    id            BIGSERIAL PRIMARY KEY,
    config_id     BIGINT NOT NULL REFERENCES public.ldap_sync_configs(id) ON DELETE CASCADE,
    status        VARCHAR(20) NOT NULL
        CONSTRAINT lsl_status_check CHECK (status IN ('running','success','partial','failed')),
    users_found   INT NOT NULL DEFAULT 0,
    users_synced  INT NOT NULL DEFAULT 0,
    users_skipped INT NOT NULL DEFAULT 0,
    errors        TEXT NOT NULL DEFAULT '[]',
    started_at    TIMESTAMPTZ NOT NULL DEFAULT now(),
    finished_at   TIMESTAMPTZ,
    triggered_by  BIGINT REFERENCES public.employees(id) ON DELETE SET NULL
);
CREATE INDEX IF NOT EXISTS idx_lsl_config_id  ON public.ldap_sync_logs (config_id);
CREATE INDEX IF NOT EXISTS idx_lsl_started_at ON public.ldap_sync_logs (started_at);

-- 7.6 DR Plans
CREATE TABLE IF NOT EXISTS public.dr_plans (
    id             BIGSERIAL PRIMARY KEY,
    name           VARCHAR(255) NOT NULL,
    description    TEXT,
    plan_type      VARCHAR(30) NOT NULL DEFAULT 'dr'
        CONSTRAINT drp_type_check CHECK (plan_type IN ('dr','bcp','contingency')),
    rto_hours      NUMERIC(6,2),
    rpo_hours      NUMERIC(6,2),
    status         VARCHAR(30) NOT NULL DEFAULT 'draft'
        CONSTRAINT drp_status_check CHECK (status IN ('draft','active','archived','under_review')),
    owner_id       BIGINT REFERENCES public.employees(id) ON DELETE SET NULL,
    last_tested_at TIMESTAMPTZ,
    next_test_due  DATE,
    created_by     BIGINT REFERENCES public.employees(id) ON DELETE SET NULL,
    created_at     TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at     TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS idx_drp_status    ON public.dr_plans (status);
CREATE INDEX IF NOT EXISTS idx_drp_plan_type ON public.dr_plans (plan_type);

-- 7.7 DR Plan Steps
CREATE TABLE IF NOT EXISTS public.dr_plan_steps (
    id               BIGSERIAL PRIMARY KEY,
    plan_id          BIGINT NOT NULL REFERENCES public.dr_plans(id) ON DELETE CASCADE,
    step_order       INT NOT NULL,
    title            VARCHAR(255) NOT NULL,
    description      TEXT,
    responsible      BIGINT REFERENCES public.employees(id) ON DELETE SET NULL,
    duration_minutes INT,
    is_critical      BOOLEAN NOT NULL DEFAULT false,
    created_at       TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT dr_plan_steps_uq UNIQUE (plan_id, step_order)
);
CREATE INDEX IF NOT EXISTS idx_drps_plan_id ON public.dr_plan_steps (plan_id);

-- 7.8 DR Tests
CREATE TABLE IF NOT EXISTS public.dr_tests (
    id                 BIGSERIAL PRIMARY KEY,
    plan_id            BIGINT NOT NULL REFERENCES public.dr_plans(id) ON DELETE CASCADE,
    test_type          VARCHAR(30) NOT NULL DEFAULT 'tabletop'
        CONSTRAINT drt_type_check CHECK (test_type IN ('tabletop','walkthrough','simulation','full_test')),
    scheduled_at       TIMESTAMPTZ NOT NULL,
    started_at         TIMESTAMPTZ,
    completed_at       TIMESTAMPTZ,
    status             VARCHAR(20) NOT NULL DEFAULT 'scheduled'
        CONSTRAINT drt_status_check CHECK (status IN ('scheduled','in_progress','completed','cancelled')),
    rto_achieved_hours NUMERIC(6,2),
    rpo_achieved_hours NUMERIC(6,2),
    outcome            VARCHAR(20)
        CONSTRAINT drt_outcome_check CHECK (outcome IN ('passed','partial','failed')),
    notes              TEXT,
    conducted_by       BIGINT REFERENCES public.employees(id) ON DELETE SET NULL,
    created_by         BIGINT REFERENCES public.employees(id) ON DELETE SET NULL,
    created_at         TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at         TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS idx_drt_plan_id      ON public.dr_tests (plan_id);
CREATE INDEX IF NOT EXISTS idx_drt_status       ON public.dr_tests (status);
CREATE INDEX IF NOT EXISTS idx_drt_scheduled_at ON public.dr_tests (scheduled_at);

-- 7.9 DR Test Results
CREATE TABLE IF NOT EXISTS public.dr_test_results (
    id                      BIGSERIAL PRIMARY KEY,
    test_id                 BIGINT NOT NULL REFERENCES public.dr_tests(id) ON DELETE CASCADE,
    step_id                 BIGINT REFERENCES public.dr_plan_steps(id) ON DELETE SET NULL,
    status                  VARCHAR(20) NOT NULL
        CONSTRAINT drtr_status_check CHECK (status IN ('passed','failed','skipped','not_tested')),
    actual_duration_minutes INT,
    notes                   TEXT,
    created_at              TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS idx_drtr_test_id ON public.dr_test_results (test_id);

-- ============================================================
-- FASE 6: Compliance Reporting, Vendor Performance, Service Availability
-- ISO 19770-1, ISO 20000-1, ITIL 4
-- ============================================================

-- 6.1 Compliance Frameworks
CREATE TABLE IF NOT EXISTS public.compliance_frameworks (
    id          BIGSERIAL PRIMARY KEY,
    code        VARCHAR(50)  NOT NULL,
    name        VARCHAR(255) NOT NULL,
    version     VARCHAR(50),
    description TEXT,
    is_active   BOOLEAN NOT NULL DEFAULT true,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT compliance_frameworks_code_key UNIQUE (code)
);
CREATE INDEX IF NOT EXISTS idx_cf_is_active ON public.compliance_frameworks (is_active);

-- 6.2 Compliance Controls
CREATE TABLE IF NOT EXISTS public.compliance_controls (
    id           BIGSERIAL PRIMARY KEY,
    framework_id BIGINT NOT NULL REFERENCES public.compliance_frameworks(id) ON DELETE CASCADE,
    control_code VARCHAR(100) NOT NULL,
    name         VARCHAR(255) NOT NULL,
    description  TEXT,
    category     VARCHAR(100),
    severity     VARCHAR(20)
        CONSTRAINT cc_severity_check CHECK (severity IN ('low','medium','high','critical')),
    is_active    BOOLEAN NOT NULL DEFAULT true,
    created_at   TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT compliance_controls_uq UNIQUE (framework_id, control_code)
);
CREATE INDEX IF NOT EXISTS idx_cc_framework_id ON public.compliance_controls (framework_id);
CREATE INDEX IF NOT EXISTS idx_cc_severity     ON public.compliance_controls (severity);

-- 6.3 Compliance Evidence
CREATE TABLE IF NOT EXISTS public.compliance_evidence (
    id            BIGSERIAL PRIMARY KEY,
    control_id    BIGINT NOT NULL REFERENCES public.compliance_controls(id) ON DELETE CASCADE,
    entity_type   VARCHAR(50) NOT NULL
        CONSTRAINT ce_entity_type_check CHECK (entity_type IN (
            'asset','ticket','change_request','service_request','audit_session','license'
        )),
    entity_id     BIGINT NOT NULL,
    evidence_type VARCHAR(50) NOT NULL
        CONSTRAINT ce_evidence_type_check CHECK (evidence_type IN (
            'document','screenshot','log','report','config','test_result'
        )),
    title         VARCHAR(255) NOT NULL,
    description   TEXT,
    file_url      TEXT,
    status        VARCHAR(30) NOT NULL DEFAULT 'pending'
        CONSTRAINT ce_status_check CHECK (status IN ('pending','accepted','rejected','expired')),
    reviewed_by   BIGINT REFERENCES public.employees(id) ON DELETE SET NULL,
    reviewed_at   TIMESTAMPTZ,
    expires_at    TIMESTAMPTZ,
    submitted_by  BIGINT REFERENCES public.employees(id) ON DELETE SET NULL,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at    TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS idx_ce_control_id ON public.compliance_evidence (control_id);
CREATE INDEX IF NOT EXISTS idx_ce_status     ON public.compliance_evidence (status);
CREATE INDEX IF NOT EXISTS idx_ce_entity     ON public.compliance_evidence (entity_type, entity_id);


-- 6.5 Vendor Performance
CREATE TABLE IF NOT EXISTS public.vendor_performance (
    id                  BIGSERIAL PRIMARY KEY,
    vendor_name         VARCHAR(255) NOT NULL,
    contract_id         BIGINT REFERENCES public.contracts(id) ON DELETE SET NULL,
    period_start        DATE NOT NULL,
    period_end          DATE NOT NULL,
    sla_compliance_pct  NUMERIC(5,2),
    avg_response_hours  NUMERIC(7,2),
    total_tickets       INT NOT NULL DEFAULT 0,
    open_tickets        INT NOT NULL DEFAULT 0,
    critical_incidents  INT NOT NULL DEFAULT 0,
    nps_score           INT CHECK (nps_score BETWEEN -100 AND 100),
    notes               TEXT,
    recorded_by         BIGINT REFERENCES public.employees(id) ON DELETE SET NULL,
    created_at          TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at          TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS idx_vp_vendor_name  ON public.vendor_performance (vendor_name);
CREATE INDEX IF NOT EXISTS idx_vp_period_start ON public.vendor_performance (period_start);
CREATE INDEX IF NOT EXISTS idx_vp_contract_id  ON public.vendor_performance (contract_id);

-- 6.6 Service Availability (availability_pct = generated stored column)
CREATE TABLE IF NOT EXISTS public.service_availability (
    id                       BIGSERIAL PRIMARY KEY,
    service_code             TEXT NOT NULL REFERENCES public.services(code) ON DELETE CASCADE,
    period_start             TIMESTAMPTZ NOT NULL,
    period_end               TIMESTAMPTZ NOT NULL,
    downtime_minutes         INT NOT NULL DEFAULT 0,
    planned_downtime_minutes INT NOT NULL DEFAULT 0,
    incident_count           INT NOT NULL DEFAULT 0,
    availability_pct         NUMERIC(7,4) GENERATED ALWAYS AS (
        ROUND(
            (1.0 - downtime_minutes::NUMERIC /
             NULLIF(EXTRACT(EPOCH FROM (period_end - period_start)) / 60.0, 0)
            ) * 100, 4
        )
    ) STORED,
    notes                    TEXT,
    recorded_by              BIGINT REFERENCES public.employees(id) ON DELETE SET NULL,
    created_at               TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at               TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS idx_sa_service_code ON public.service_availability (service_code);
CREATE INDEX IF NOT EXISTS idx_sa_period_start ON public.service_availability (period_start);

-- Seed framework ISO/ITIL
INSERT INTO public.compliance_frameworks (code, name, version, description)
VALUES
    ('ISO19770-1',  'ISO/IEC 19770-1 Software Asset Management', '2017', 'Kerangka SAM untuk pengelolaan lisensi software'),
    ('ISO19770-2',  'ISO/IEC 19770-2 Software Identification Tags (SWID)', '2015', 'Standar tagging identifikasi software'),
    ('ISO19770-10', 'ISO/IEC 19770-10 Overview and Vocabulary', '2015', 'Lifecycle aset IT termasuk disposal'),
    ('ISO20000-1',  'ISO/IEC 20000-1 IT Service Management', '2018', 'ITSM standar internasional'),
    ('ITIL4',       'ITIL 4 Framework', '2019', 'Best practice ITSM: Incident, Problem, Change, Service Request Management')
ON CONFLICT (code) DO NOTHING;

-- ============================================================
-- FASE 5: ITAM Enhancement — Asset Specifications, SAM, Disposal
-- ISO 19770-1 (SAM), ISO 19770-2 (SWID), ISO 19770-10 (Lifecycle)
-- ============================================================

-- 5.1 Extend lifecycle_stage enum on assets table
ALTER TABLE public.assets DROP CONSTRAINT IF EXISTS assets_lifecycle_stage_check;
ALTER TABLE public.assets
    ADD CONSTRAINT assets_lifecycle_stage_check
    CHECK (lifecycle_stage IS NULL OR lifecycle_stage IN (
        'planning','procurement','receiving','in_use','maintenance',
        'retirement_pending','retired',
        'disposal_pending','disposal_approved','disposed',
        'under_remediation'
    ));

-- 5.2 Asset Specifications (1-to-1 with assets, ISO 19770-2 SWID)
CREATE TABLE IF NOT EXISTS public.asset_specifications (
    id               BIGSERIAL PRIMARY KEY,
    asset_id         BIGINT NOT NULL REFERENCES public.assets(id) ON DELETE CASCADE,
    -- CPU
    cpu_model        VARCHAR(255),
    cpu_cores        INT,
    cpu_speed_ghz    NUMERIC(5,2),
    -- Memory
    ram_gb           NUMERIC(7,2),
    ram_type         VARCHAR(50),
    -- Storage
    storage_gb       NUMERIC(10,2),
    storage_type     VARCHAR(50),
    -- Display
    screen_size_inch NUMERIC(5,2),
    resolution       VARCHAR(30),
    -- Network
    mac_address      VARCHAR(20),
    ip_address       INET,
    -- Firmware / OS
    bios_version     VARCHAR(100),
    firmware_version VARCHAR(100),
    os_name          VARCHAR(100),
    os_version       VARCHAR(100),
    os_license_key   TEXT,
    -- Physical
    form_factor      VARCHAR(50),
    color            VARCHAR(50),
    weight_kg        NUMERIC(6,2),
    -- Audit metadata
    last_scanned_at  TIMESTAMPTZ,
    created_by       BIGINT REFERENCES public.employees(id) ON DELETE SET NULL,
    created_at       TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at       TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT asset_specifications_asset_id_key UNIQUE (asset_id)
);
CREATE INDEX IF NOT EXISTS idx_asset_spec_asset_id ON public.asset_specifications (asset_id);

-- 5.3 Software Usage Logs — SAM metering per license/device/user
CREATE TABLE IF NOT EXISTS public.software_usage_logs (
    id            BIGSERIAL PRIMARY KEY,
    license_id    BIGINT NOT NULL REFERENCES public.licenses(id) ON DELETE CASCADE,
    asset_id      BIGINT REFERENCES public.assets(id) ON DELETE SET NULL,
    employee_id   BIGINT REFERENCES public.employees(id) ON DELETE SET NULL,
    session_start TIMESTAMPTZ NOT NULL,
    session_end   TIMESTAMPTZ,
    usage_minutes INT,
    source        VARCHAR(20) NOT NULL DEFAULT 'manual'
        CONSTRAINT sul_source_check CHECK (source IN ('manual','agent','import','sccm','jamf')),
    created_at    TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS idx_sul_license_id   ON public.software_usage_logs (license_id);
CREATE INDEX IF NOT EXISTS idx_sul_asset_id     ON public.software_usage_logs (asset_id);
CREATE INDEX IF NOT EXISTS idx_sul_employee_id  ON public.software_usage_logs (employee_id);
CREATE INDEX IF NOT EXISTS idx_sul_session_start ON public.software_usage_logs (session_start);

-- 5.4 License Reconciliation View (SAM — ISO 19770-1)
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

-- 5.5 Asset Disposal Records — regulatory compliance (RoHS/WEEE, ISO 19770-10)
CREATE TABLE IF NOT EXISTS public.asset_disposal_records (
    id                      BIGSERIAL PRIMARY KEY,
    asset_id                BIGINT NOT NULL REFERENCES public.assets(id) ON DELETE CASCADE,
    disposal_method         VARCHAR(50) NOT NULL
        CONSTRAINT adr_method_check CHECK (disposal_method IN (
            'resell','recycle','destroy','donate','return_to_vendor','write_off'
        )),
    data_wipe_method        VARCHAR(100),
    data_wipe_completed     BOOLEAN NOT NULL DEFAULT false,
    certificate_number      VARCHAR(100),
    certificate_url         TEXT,
    environmental_compliant BOOLEAN NOT NULL DEFAULT false,
    regulatory_notes        TEXT,
    vendor                  VARCHAR(255),
    disposal_value          NUMERIC(15,2),
    authorization_by        BIGINT NOT NULL REFERENCES public.employees(id),
    executed_by             BIGINT REFERENCES public.employees(id),
    date_disposed           DATE NOT NULL,
    created_by              BIGINT REFERENCES public.employees(id) ON DELETE SET NULL,
    created_at              TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at              TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS idx_adr_asset_id      ON public.asset_disposal_records (asset_id);
CREATE INDEX IF NOT EXISTS idx_adr_date_disposed ON public.asset_disposal_records (date_disposed);
CREATE INDEX IF NOT EXISTS idx_adr_env_compliant ON public.asset_disposal_records (environmental_compliant);

-- 6.4 v_asset_disposal_compliance (dipindah ke sini agar asset_disposal_records sudah ada)
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

-- ============================================================
-- Notifications
-- ============================================================
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
CREATE INDEX IF NOT EXISTS idx_notif_user_id    ON public.notifications (user_id);
CREATE INDEX IF NOT EXISTS idx_notif_is_read    ON public.notifications (user_id, is_read) WHERE is_read = false;
CREATE INDEX IF NOT EXISTS idx_notif_created_at ON public.notifications (created_at);

--
-- PostgreSQL database dump complete
--

