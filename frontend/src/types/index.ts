// File: src/types/index.ts

// Tipe untuk data pagination dari API
export interface PaginationData {
  total_records: number;
  total_pages: number;
  current_page: number;
  page_size: number;
}

// Tipe untuk data Aset
export interface Asset {
  id: number;
  name: string;
  asset_tag: string;
  status: string;
  asset_type_id?: number | null;
  asset_type_name?: string;
  purchase_date: string;
  initial_price: number;
}

// Tipe untuk data Karyawan
export interface Employee {
  id: number;
  employee_nik: string;
  name: string;
  email: string;
  department_id?: number | null;
  department_name?: string;
  role: string;
}

// Tipe untuk data Departemen
export interface Department {
  id: number;
  name: string;
}

// Tipe untuk data Tipe Aset
export interface AssetType {
  id: number;
  name: string;
}

export interface AssetMaintenanceLog {
  id: number;
  asset_id: number;
  ticket_id?: number | null;
  log_type: string;
  description: string;
  cost: number;
  log_date: string; // Tanggal akan diterima sebagai string format ISO
  created_at: string;
}

export interface SoftwareLicense {
  id: number;
  name: string;
  license_key?: string;
  total_seats: number;
  purchase_date?: string;
  expiration_date?: string;
  cost?: number;
}

export interface InstalledSoftwareInfo {
  installation_id: number;
  license_id: number;
  license_name: string;
  license_key?: string;
  installation_date: string;
}

export interface TicketInfo {
  id: number;
  subject: string;
  status: string;
  priority: string;
  created_by_employee_id: number;
  created_by_employee_name: string;
  related_asset_id?: number | null;
  created_at: string;
  updated_at: string;
}

export interface AssetHistoryResponse  {
  assignment_id: number;
  employee_nik: string;
  employee_name: string;
  assigned_at: string;
  returned_at: string | null;
  notes: string;
}

export interface DepreciationInfo {
  asset_name: string;
  asset_tag: string;
  initial_price: number;
  purchase_date: string;
  useful_life_years: number;
  age_in_years: number;
  depreciation_per_year: number;
  total_depreciation: number;
  current_book_value: number;
}

export interface TicketCommentInfo {
  id: number;
  employee_id: number;
  employee_name: string;
  comment: string;
  created_at: string;
}

export interface TicketDetail extends TicketInfo {
  description: string;
  comments: TicketCommentInfo[];
  maintenance_logs: AssetMaintenanceLog[];
}

export interface Budget {
  id: number;
  name: string;
  department_id?: number | null;
  start_date: string;
  end_date: string;
  total_amount: number;
  spent_amount: number; // Add this field
}

export interface AuditSession {
  id: number;
  name: string;
  status: string;
  created_at: string;
  completed_at?: string | null;
}

// ... tambahkan interface lain di sini jika dibutuhkan ...