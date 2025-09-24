# File: Dockerfile (di folder root)

# --- Tahap 1: Build Frontend ---
# Menggunakan image Node.js versi 18 untuk membangun aplikasi React
FROM node:18-alpine AS frontend-builder
WORKDIR /app/frontend
# Salin file package.json dan package-lock.json terlebih dahulu untuk caching
COPY frontend/package*.json ./
RUN npm install
# Gunakan 'npm ci' untuk instalasi yang lebih cepat dan konsisten di CI/CD
RUN npm ci
# Salin sisa kode frontend
COPY frontend/ ./
# Jalankan perintah build untuk menghasilkan file statis
RUN npm run build

# --- Tahap 2: Build Backend ---
# Menggunakan image Go versi 1.24 yang ringan
FROM golang:1.24-alpine AS backend-builder
WORKDIR /app
# Salin file modul Go untuk caching dependensi
COPY go.mod go.sum ./
RUN go mod download
# Salin seluruh kode proyek
COPY . .
# Build binary Go yang statis dan siap produksi
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-w -s" -o /app/main ./backend

# --- Tahap 3: Image Final ---
# Mulai dari image Alpine Linux yang sangat kecil
FROM alpine:latest
WORKDIR /app
# Ambil binary Go yang sudah di-build dari tahap backend-builder
COPY --from=backend-builder /app/main .
# Ambil folder build frontend dari tahap frontend-builder
COPY --from=frontend-builder /app/frontend/dist ./frontend/dist

# Expose port yang digunakan oleh server Go
EXPOSE 8080

# Perintah untuk menjalankan aplikasi Go saat kontainer dimulai
CMD ["/app/main"]