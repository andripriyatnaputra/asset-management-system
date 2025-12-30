-- File: database/init/init.sql (Versi Final)

-- Tabel baru untuk Departemen
CREATE TABLE departments (
    id BIGSERIAL PRIMARY KEY,
    name VARCHAR(255) NOT NULL UNIQUE
);

CREATE TABLE asset_types (
    id BIGSERIAL PRIMARY KEY,
    name VARCHAR(255) NOT NULL UNIQUE
);

CREATE TABLE assets (
    id BIGSERIAL PRIMARY KEY,
    name VARCHAR(255) NOT NULL,
    asset_tag VARCHAR(100) NOT NULL UNIQUE,
    status VARCHAR(50) NOT NULL,
    asset_type_id BIGINT REFERENCES asset_types(id),
    purchase_date DATE,
    initial_price DECIMAL(15, 2),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMPTZ 
);

CREATE TABLE employees (
    id BIGSERIAL PRIMARY KEY,
    employee_nik VARCHAR(50) NOT NULL UNIQUE,
    name VARCHAR(255) NOT NULL,
    email VARCHAR(255) NOT NULL UNIQUE,
    -- Kolom 'department' diubah menjadi 'department_id'
    department_id BIGINT REFERENCES departments(id),
    password_hash VARCHAR(255) NOT NULL,
    role VARCHAR(50) NOT NULL DEFAULT 'employee',
    deleted_at TIMESTAMPTZ
);

CREATE TABLE tickets (
    id BIGSERIAL PRIMARY KEY,
    subject VARCHAR(255) NOT NULL,
    description TEXT,
    status VARCHAR(50) NOT NULL DEFAULT 'Open', -- Open, In Progress, Closed, On Hold
    priority VARCHAR(50) NOT NULL DEFAULT 'Medium', -- Low, Medium, High, Critical
    created_by_employee_id BIGINT NOT NULL REFERENCES employees(id),
    assigned_to_employee_id BIGINT REFERENCES employees(id), -- Siapa yang mengerjakan (bisa NULL)
    related_asset_id BIGINT REFERENCES assets(id), -- Aset yang terkait (bisa NULL)
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMPTZ
);

CREATE TABLE asset_assignments (
    id BIGSERIAL PRIMARY KEY,
    asset_id BIGINT NOT NULL REFERENCES assets(id),
    employee_id BIGINT NOT NULL REFERENCES employees(id),
    assigned_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    returned_at TIMESTAMPTZ, 
    notes TEXT
);

CREATE TABLE asset_maintenance_logs (
    id BIGSERIAL PRIMARY KEY,
    asset_id BIGINT NOT NULL REFERENCES assets(id),
    ticket_id BIGINT REFERENCES tickets(id),
    log_type VARCHAR(50) NOT NULL,
    description TEXT NOT NULL,
    cost DECIMAL(15, 2) DEFAULT 0,
    log_date DATE NOT NULL DEFAULT CURRENT_DATE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- TABEL BARU UNTUK LISENSI SOFTWARE
CREATE TABLE software_licenses (
    id BIGSERIAL PRIMARY KEY,
    name VARCHAR(255) NOT NULL,
    license_key VARCHAR(255),
    total_seats INT NOT NULL, -- Jumlah total lisensi yang dibeli
    purchase_date DATE,
    expiration_date DATE,
    cost DECIMAL(15, 2),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMPTZ
);

-- TABEL BARU PENGHUBUNG ASET DAN LISENSI
CREATE TABLE software_installations (
    id BIGSERIAL PRIMARY KEY,
    asset_id BIGINT NOT NULL REFERENCES assets(id),
    license_id BIGINT NOT NULL REFERENCES software_licenses(id),
    installation_date DATE NOT NULL DEFAULT CURRENT_DATE,
    notes TEXT,
    -- Pastikan satu lisensi tidak bisa di-install dua kali di aset yang sama
    UNIQUE(asset_id, license_id)
);


-- TABEL BARU UNTUK KOMENTAR PADA TIKET
CREATE TABLE ticket_comments (
    id BIGSERIAL PRIMARY KEY,
    ticket_id BIGINT NOT NULL REFERENCES tickets(id) ON DELETE CASCADE,
    employee_id BIGINT NOT NULL REFERENCES employees(id),
    comment TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);


-- TABEL BARU UNTUK ANGGARAN
CREATE TABLE budgets (
    id BIGSERIAL PRIMARY KEY,
    name VARCHAR(255) NOT NULL,
    department_id BIGINT REFERENCES departments(id), -- Anggaran bisa spesifik untuk 1 departemen, atau NULL untuk umum
    start_date DATE NOT NULL,
    end_date DATE NOT NULL,
    total_amount DECIMAL(15, 2) NOT NULL DEFAULT 0,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMPTZ
);

-- TABEL BARU UNTUK MENCATAT SETIAP TRANSAKSI TERHADAP ANGGARAN
CREATE TABLE budget_transactions (
    id BIGSERIAL PRIMARY KEY,
    budget_id BIGINT NOT NULL REFERENCES budgets(id),
    asset_id BIGINT NOT NULL REFERENCES assets(id), -- Setiap transaksi harus terkait dengan pembelian aset
    amount DECIMAL(15, 2) NOT NULL,
    transaction_date DATE NOT NULL,
    notes TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE audit_sessions (
    id BIGSERIAL PRIMARY KEY,
    name VARCHAR(255) NOT NULL,
    status VARCHAR(50) NOT NULL DEFAULT 'In Progress', -- In Progress, Completed
    created_by_employee_id BIGINT NOT NULL REFERENCES employees(id),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    completed_at TIMESTAMPTZ
);

-- TABEL BARU UNTUK ITEM ASET YANG DIAUDIT DALAM SEBUAH SESI
CREATE TABLE audited_assets (
    id BIGSERIAL PRIMARY KEY,
    session_id BIGINT NOT NULL REFERENCES audit_sessions(id) ON DELETE CASCADE,
    asset_id BIGINT NOT NULL REFERENCES assets(id),
    -- Status awal 'Missing', berubah menjadi 'Found' jika dipindai
    status VARCHAR(50) NOT NULL DEFAULT 'Missing', 
    found_at TIMESTAMPTZ, -- Waktu saat aset ditemukan/dipindai
    notes TEXT,
    UNIQUE(session_id, asset_id) -- Pastikan satu aset hanya bisa ada sekali dalam satu sesi
);

-- --- Data Awal (Seeder Mini) ---

-- Tambahkan beberapa departemen awal
INSERT INTO departments (name) VALUES ('Product Development'), ('Direksi'), ('HR & GA'), ('Finance & Acc'), ('Procurement & Logistic'), ('Sales & Marketing'), ('Operation & Maintenance');

-- TAMBAHKAN DATA AWAL UNTUK TIPE ASET
INSERT INTO asset_types (name) VALUES ('Laptop'), ('Monitor'), ('Server'), ('Keyboard'), ('Mouse'), ('Printer');

-- Tambahkan Super Admin dan hubungkan ke departemen 'IT Infrastructure' (ID: 1)
INSERT INTO employees (employee_nik, name, email, department_id, password_hash, role)
VALUES (
    'SUPERADMIN-001',
    'Super Admin',
    'admin.admin@example.com',
    1, -- Merujuk ke ID 'IT Infrastructure'
    '$2a$12$ULKxws0htrXha9KKlBDqpOaHGnVKjU3VYl9C87H7rraDYK0b2Iuey', -- Password: superadmin123
    'super_admin'
);