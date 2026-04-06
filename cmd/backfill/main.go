package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"github.com/AgnivaSengupta/analytics-engine/internal/rollups"
	"github.com/jackc/pgx/v5/pgxpool"
)

func main() {
	builderFlag := flag.String("builders", "all", "comma-separated builder names or 'all'")
	fromFlag := flag.String("from", "", "inclusive RFC3339 UTC start")
	toFlag := flag.String("to", "", "exclusive RFC3339 UTC end")
	flag.Parse()

	if strings.TrimSpace(*fromFlag) == "" || strings.TrimSpace(*toFlag) == "" {
		log.Fatal("both -from and -to are required")
	}

	from, err := time.Parse(time.RFC3339, *fromFlag)
	if err != nil {
		log.Fatalf("parse -from: %v", err)
	}
	to, err := time.Parse(time.RFC3339, *toFlag)
	if err != nil {
		log.Fatalf("parse -to: %v", err)
	}
	if !from.Before(to) {
		log.Fatal("-from must be before -to")
	}

	dbURL := os.Getenv("DB_DSN")
	if dbURL == "" {
		dbURL = "postgres://postgres:password@localhost:5432/analytics?sslmode=disable"
	}

	dbPool, err := pgxpool.New(context.Background(), dbURL)
	if err != nil {
		log.Fatalf("connect db: %v", err)
	}
	defer dbPool.Close()

	builders, err := selectedBuilders(*builderFlag)
	if err != nil {
		log.Fatal(err)
	}

	for _, builder := range builders {
		log.Printf("backfilling %s from %s to %s", builder.Name, from.UTC().Format(time.RFC3339), to.UTC().Format(time.RFC3339))
		if err := rollups.RunBuilder(context.Background(), dbPool, builder, from.UTC(), to.UTC()); err != nil {
			log.Fatalf("backfill %s failed: %v", builder.Name, err)
		}
	}

	log.Printf("backfill complete for builders: %s", rollups.Describe(builders))
}

func selectedBuilders(selection string) ([]rollups.Builder, error) {
	if strings.EqualFold(strings.TrimSpace(selection), "all") {
		return rollups.AllBuilders(), nil
	}

	parts := strings.Split(selection, ",")
	builders := make([]rollups.Builder, 0, len(parts))
	for _, part := range parts {
		name := strings.TrimSpace(part)
		if name == "" {
			continue
		}
		builder, err := rollups.RequireBuilder(name)
		if err != nil {
			return nil, err
		}
		builders = append(builders, builder)
	}
	if len(builders) == 0 {
		return nil, fmt.Errorf("no builders selected")
	}
	return builders, nil
}
