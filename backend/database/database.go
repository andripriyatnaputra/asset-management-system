package database

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/jackc/pgx/v5/pgxpool"
)

var Pool *pgxpool.Pool

// Connect membuka koneksi pool ke PostgreSQL.
func Connect() {
	databaseUrl := os.Getenv("DATABASE_URL")
	if databaseUrl == "" {
		databaseUrl = "postgres://admin:secret@db:5432/asset_db?sslmode=disable"
	}

	log.Println("Connecting to database...")

	var err error
	Pool, err = pgxpool.New(context.Background(), databaseUrl)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Unable to create connection pool: %v\n", err)
		os.Exit(1)
	}

	if err = Pool.Ping(context.Background()); err != nil {
		fmt.Fprintf(os.Stderr, "Ping to database failed: %v\n", err)
		os.Exit(1)
	}

	log.Println("Successfully connected to the database!")
}

// SetSessionVars menyetel variabel sesi PostgreSQL untuk audit/RLS.
func SetSessionVars(ctx context.Context, userID, deptID *int64) context.Context {
	if Pool == nil {
		return ctx
	}
	conn, err := Pool.Acquire(ctx)
	if err != nil {
		return ctx
	}
	defer conn.Release()

	// Gunakan SET, bukan SET LOCAL
	if userID != nil {
		sql := fmt.Sprintf("SET app.current_user_id = '%d'", *userID)
		_, _ = conn.Exec(ctx, sql)
	} else {
		_, _ = conn.Exec(ctx, "SET app.current_user_id = 'NULL'")
	}

	if deptID != nil {
		sql := fmt.Sprintf("SET app.current_department = '%d'", *deptID)
		_, _ = conn.Exec(ctx, sql)
	} else {
		_, _ = conn.Exec(ctx, "SET app.current_department = 'NULL'")
	}

	return ctx
}
