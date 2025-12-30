// ================== Core ==================
export interface Department {
  id: number
  name: string
  code?: string | null
  created_at?: string
}

export interface CostCenter {
  id: number
  code: string
  name: string
}

export interface Employee {
  id: number
  name: string
  email: string
  department_id?: number | null
  department_name?: string | null
  // disamakan dgn komponen lama (non-optional string)
  employee_nik: string
  role?: 'super_admin' | 'employee' | string
}

// ================== Asset ==================
export type AssetStatus = 'in_stock' | 'assigned' | 'maintenance' | 'retired' | 'disposed'

export interface AssetType {
  id: number
  name: string
  code?: string | null
  description?: string | null
  created_at?: string
  updated_at?: string
}

export interface DepreciationInfo {
  method: 'straight_line' | 'declining' | string
  useful_life_months: number
  salvage_value: number
  straight_line_monthly?: number | null
  accumulated?: number | null
  book_value?: number | null
  as_of_date?: string
}

export interface Asset {
  id: number
  name: string
  asset_tag: string
  status: string

  // relasi & id
  asset_type_id?: number | null
  department_id?: number | null
  cost_center_id?: number | null
  location_id?: number | null

  // data finansial & lifecycle
  purchase_date?: string | null
  purchase_cost?: number | null
  initial_price?: number | null
  vendor?: string | null
  warranty_expiry?: string | null
  useful_life_months?: number | null
  depreciation_method?: string | null
  salvage_value?: number | null
  serial_number?: string | null
  asset_condition?: string | null
  acquisition_type?: string | null
  ownership_type?: string | null
  disposal_date?: string | null
  disposed?: boolean
  notes?: string | null

  // metadata
  created_at?: string
  updated_at?: string
  deleted_at?: string | null

  // kolom “nama”/teks yang dipakai komponen
  asset_type_name?: string | null
  department_name?: string | null
  owner_department_name?: string | null
  current_location_text?: string | null
  assigned_to_employee_name?: string | null

  // lokasi (opsional)
  location_site?: string | null
  location_building?: string | null
  location_room?: string | null

  // untuk tab Depreciation
  depreciation?: {
    method?: string
    useful_life_months?: number
    salvage_value?: number
    book_value?: number
    accumulated?: number
  }

  compliance_status?: string | null
  asset_health_score?: number | null // ✅ tambahkan ini

  // ✅ Tambahkan field governance & compliance
  contract_id?: number | null
  license_id?: number | null
  budget_id?: number | null
  lifecycle_stage?: string | null
  asset_criticality?: string | null
  compliance_flag?: boolean | null
  compliance_note?: string | null
}


export interface AssetHistoryRecord {
  id: number
  asset_id: number
  action: 'created' | 'updated' | 'assigned' | 'returned' | 'maintenance' | 'retired' | 'disposed' | string
  detail?: string | null
  actor_name?: string | null
  from_status?: AssetStatus | null
  to_status?: AssetStatus | null
  created_at: string
}

export interface AssetHistoryResponse {
  asset: { id: number; asset_tag: string; name: string }
  history: AssetHistoryRecord[]
}

export interface AssetMaintenanceLog {
  id: number
  asset_id: number
  log_type: 'Maintenance' | 'Repair' | 'Upgrade' | string
  description: string
  cost: number
  log_date: string
  ticket_id?: number | null
  created_at?: string
}

export interface AssetItem {
  id: number
  asset_tag: string
  name: string
  type?: string
  status?: AssetStatus
  category_code?: string
  service_code?: string

  assigned_to_employee_id?: number | null
  assigned_to_employee_name?: string | null
}

// ================== Ticketing (ringkas) ==================
export type Priority = 'Low' | 'Medium' | 'High' | 'Critical'
export type Status = 'Open' | 'In Progress' | 'Resolved' | 'Closed'
export type Level = 'Low' | 'Medium' | 'High'

export interface TicketAttachment {
  id: number
  ticket_id?: number
  comment_id?: number | null
  filename: string
  path?: string
  url?: string
  mime_type?: string | null
  size?: number | null
  created_at: string
}

export interface TicketCommentInfo {
  id: number
  employee_id: number
  employee_name: string
  comment: string
  created_at: string
  attachments?: TicketAttachment[]
}

export interface TicketInfo {
  id: number
  subject: string
  status: Status
  priority: Priority

  category_code?: string | null
  service_code?: string | null
  impact?: Level | null
  urgency?: Level | null

  created_by_employee_id: number
  created_by_employee_name: string
  assigned_to_employee_id?: number | null
  assigned_to_employee_name?: string | null

  /** 🆕 Tambahan field assignment tracking */
  last_assigned_by?: number | null
  last_assigned_by_name?: string | null
  last_assigned_at?: string | null

  related_asset_id?: number | null

  sla_due_at?: string | null
  sla_breached_at?: string | null

  created_at: string
  updated_at: string
}


export interface TicketDetail extends TicketInfo {
  description: string
  comments: TicketCommentInfo[]
  maintenance_logs?: any[]
  escalation_level?: number 
}

// ================== Budgets ==================
export interface Budget {
  id: number
  name: string
  department_id?: number | null
  department_name?: string | null
  start_date: string
  end_date: string
  amount: number
  spent?: number
  // alias yang dipakai komponen lama:
  total_amount?: number
  spent_amount?: number
  created_at?: string
}

// ================== Licenses ==================
// ================== Licenses ==================
export interface SoftwareLicense {
  id: number

  // Nama & identifikasi software
  name: string
  license_key?: string | null
  vendor?: string | null
  publisher?: string | null
  version?: string | null

  // Jenis & model lisensi (ISO/IEC 19770-10)
  license_type?: string | null       // contoh: Perpetual, Subscription, OEM
  license_model?: string | null      // contoh: Per User, Per Device
  metric?: string | null
  category?: string | null
  cost_center?: string | null
  status?: string | null             // active, expired, retired, pending-renewal

  // Hubungan ke kontrak
  contract_id?: number | null
  contract_number?: string | null    // opsional jika API di-expand

  // Kepatuhan & lifecycle
  compliance_status?: string | null  // compliant, non-compliant, unknown
  verification_date?: string | null
  maintenance_expiry?: string | null

  // Kapasitas & pemakaian
  total_seats: number
  used_seats?: number | null

  // Keuangan & tanggal
  cost?: number | null
  purchase_date?: string | null
  expiration_date?: string | null

  // Metadata tambahan
  entitlement_doc?: string | null
  procurement_reference?: string | null
  notes?: string | null
  created_at?: string
  updated_at?: string

  // ✅ alias untuk backward compatibility (komponen lama)
  software_name?: string
  seats_total?: number
  purchase_cost?: number | null
  expiry_date?: string | null
}


// ================== Pagination & AuditSession ==================
export interface PaginationData {
  current_page: number
  total_pages: number
  limit?: number
}

export interface AuditSession {
  id: number
  name: string
  started_at: string
  ended_at?: string | null
  total_assets?: number
  checked_assets?: number

  // 🔥 tambahan agar cocok dengan komponen
  status: 'Ongoing' | 'Completed' | string
  created_at: string
}

// =============================================================
// ✅ ASSET & COMPLIANCE TYPES
// =============================================================
export interface AssetCompliance {
  id: number
  name: string
  asset_tag: string
  department_name?: string | null
  owner_department_name?: string | null
  status?: string
  lifecycle_stage?: string | null
  compliance_flag?: boolean | null
  compliance_note?: string | null
  contract_id?: number | null
  license_id?: number | null
  budget_id?: number | null
  disposed_approved_by?: number | null
  updated_at: string
  governance_score?: number
}

// =============================================================
// ✅ BUDGET TYPES
// =============================================================
export interface BudgetOverview {
  budget_id: number
  budget_name: string
  category?: string | null
  currency?: string | null
  total_amount: number
  realized_amount: number
  remaining_amount: number
  realization_percent: number
  status: string
}

// =============================================================
// ✅ SLA DASHBOARD TYPES
// =============================================================
export interface SLADashboard {
  open_tickets: number
  breached_tickets: number
  resolved_tickets: number
  sla_compliance_rate: number
  avg_mttr_minutes?: number | null
  avg_mtta_minutes?: number | null
}

// =============================================================
// ✅ AUDIT LOG TYPES
// =============================================================
export interface AuditLog {
  entity_name: string
  action: string
  actor_name: string
  created_at: string
}

// =============================================================
// ✅ HEALTH & CORRELATION TYPES
// =============================================================
export interface HealthHeatmapRow {
  department: string
  avg_health: number
  alert_count: number
}

export interface PredictiveForecastRow {
  department: string
  avg_health: number
  alert_count: number
  forecast_next_7: number
  forecast_next_30: number
}


export interface CorrelationRow {
  department: string
  assets: number
  tickets: number
  alerts: number
  avg_health: number
  alert_ratio: number
  ticket_ratio: number
}
