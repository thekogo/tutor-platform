// Package seed imports a content/<course>/graph.yaml file into the database
// through the same app use case as the import_course_graph MCP tool
// (idempotent upsert by slug).
package seed

import (
	"bytes"
	"context"
	"fmt"
	"log/slog"
	"os"

	"gopkg.in/yaml.v3"

	"github.com/thekogo/tutor-platform/backend/internal/app"
	"github.com/thekogo/tutor-platform/backend/internal/store"
)

// ParseFile reads a graph.yaml file.
func ParseFile(path string) (app.GraphDoc, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return app.GraphDoc{}, err
	}
	var doc app.GraphDoc
	dec := yaml.NewDecoder(bytes.NewReader(raw))
	dec.KnownFields(true)
	if err := dec.Decode(&doc); err != nil {
		return app.GraphDoc{}, fmt.Errorf("seed: parse %s: %w", path, err)
	}
	return doc, nil
}

// Run migrates the database and imports the graph file.
func Run(ctx context.Context, dsn, path string, archiveMissing bool, log *slog.Logger) (app.ImportResult, error) {
	doc, err := ParseFile(path)
	if err != nil {
		return app.ImportResult{}, err
	}
	pool, err := store.Open(ctx, dsn)
	if err != nil {
		return app.ImportResult{}, err
	}
	defer pool.Close()
	if err := store.Migrate(ctx, pool); err != nil {
		return app.ImportResult{}, err
	}
	res, err := app.New(pool, log).ImportCourseGraph(ctx, doc, archiveMissing)
	if err != nil {
		return app.ImportResult{}, err
	}
	log.InfoContext(ctx, "seeded course graph", "course", doc.Course.ID, "topics", res.Topics,
		"concepts", res.Concepts, "prerequisites", res.Prerequisites, "archived", len(res.Archived))
	return res, nil
}
