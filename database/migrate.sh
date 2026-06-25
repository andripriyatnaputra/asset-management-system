#!/usr/bin/env bash
# Usage:
#   ./database/migrate.sh up              # apply semua migration yang belum dijalankan
#   ./database/migrate.sh up V002         # apply sampai versi tertentu
#   ./database/migrate.sh down V002       # rollback versi tertentu
#   ./database/migrate.sh status          # lihat status migration
#
# Env yang harus diset:
#   DATABASE_URL=postgres://user:pass@host:5432/dbname?sslmode=disable

set -euo pipefail

DB="${DATABASE_URL:?DATABASE_URL belum diset}"
MIGRATIONS_DIR="$(cd "$(dirname "$0")/migrations" && pwd)"

# Buat tabel tracking jika belum ada
psql "$DB" -c "
CREATE TABLE IF NOT EXISTS schema_migrations (
    version     VARCHAR(50) PRIMARY KEY,
    applied_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);
" -q

case "${1:-up}" in

  status)
    echo "=== Migration Status ==="
    applied=$(psql "$DB" -t -c "SELECT version FROM schema_migrations ORDER BY version;" | tr -d ' ')
    for f in "$MIGRATIONS_DIR"/V*.up.sql; do
      ver=$(basename "$f" .up.sql)
      if echo "$applied" | grep -qx "$ver"; then
        echo "  ✅ $ver (applied)"
      else
        echo "  ⬜ $ver (pending)"
      fi
    done
    ;;

  up)
    target="${2:-}"
    applied=$(psql "$DB" -t -c "SELECT version FROM schema_migrations ORDER BY version;" | tr -d ' ')
    for f in $(ls "$MIGRATIONS_DIR"/V*.up.sql | sort); do
      ver=$(basename "$f" .up.sql)
      [[ -n "$target" && "$ver" > "$target" ]] && break
      if echo "$applied" | grep -qx "$ver"; then
        echo "  ✅ $ver sudah diapply, skip."
        continue
      fi
      echo "  ▶ Menjalankan $ver..."
      psql "$DB" -f "$f" -q
      psql "$DB" -c "INSERT INTO schema_migrations (version) VALUES ('$ver');" -q
      echo "  ✅ $ver selesai."
    done
    echo "Migration selesai."
    ;;

  down)
    ver="${2:?Tentukan versi: ./migrate.sh down V002}"
    applied=$(psql "$DB" -t -c "SELECT version FROM schema_migrations;" | tr -d ' ')
    if ! echo "$applied" | grep -qx "$ver"; then
      echo "  ⚠ $ver belum pernah diapply, skip rollback."
      exit 0
    fi
    down_file="$MIGRATIONS_DIR/${ver}.down.sql"
    [[ ! -f "$down_file" ]] && { echo "Down script tidak ditemukan: $down_file"; exit 1; }
    echo "  ▼ Rollback $ver..."
    psql "$DB" -f "$down_file" -q
    psql "$DB" -c "DELETE FROM schema_migrations WHERE version = '$ver';" -q
    echo "  ✅ Rollback $ver selesai."
    ;;

  *)
    echo "Usage: $0 [up|down|status] [version]"
    exit 1
    ;;
esac
