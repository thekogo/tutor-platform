// Command seed imports a course graph into the database at DATABASE_URL:
//
//	DATABASE_URL=postgres://... go run ./cmd/seed -file ../content/cu-best-math/graph.yaml
//
// It runs migrations first and upserts by slug, so it is safe to re-run.
// The same import is available in the server image as `/server seed -file ...`.
package main

import (
	"context"
	"flag"
	"fmt"
	"log/slog"
	"os"
	"os/signal"

	"github.com/thekogo/tutor-platform/backend/internal/seed"
)

func main() {
	file := flag.String("file", "", "path to content/<course>/graph.yaml")
	archive := flag.Bool("archive-missing", false, "archive live Concepts absent from the file")
	flag.Parse()
	dsn := os.Getenv("DATABASE_URL")
	if *file == "" || dsn == "" {
		fmt.Fprintln(os.Stderr, "usage: DATABASE_URL=... seed -file content/<course>/graph.yaml")
		os.Exit(2)
	}
	log := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()
	res, err := seed.Run(ctx, dsn, *file, *archive, log)
	if err != nil {
		log.Error("seed failed", "error", err)
		stop()
		os.Exit(1) //nolint:gocritic // stop called explicitly before exit
	}
	fmt.Printf("seeded: %d topics, %d concepts, %d prerequisites\n", res.Topics, res.Concepts, res.Prerequisites)
}
