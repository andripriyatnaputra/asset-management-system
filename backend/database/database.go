package database

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/jackc/pgx/v5/pgxpool"
)

var Pool *pgxpool.Pool

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
