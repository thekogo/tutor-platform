package seed_test

import (
	"context"
	"testing"

	"github.com/thekogo/tutor-platform/backend/internal/app"
	"github.com/thekogo/tutor-platform/backend/internal/seed"
	"github.com/thekogo/tutor-platform/backend/internal/testutil"
)

const graphFile = "../../../content/cu-best-math/graph.yaml"

func TestParseRealGraph(t *testing.T) {
	doc, err := seed.ParseFile(graphFile)
	if err != nil {
		t.Fatal(err)
	}
	if doc.Course.ID != "cu-best-math" || len(doc.Topics) == 0 || len(doc.Concepts) == 0 || len(doc.QuestionFormats) == 0 {
		t.Fatalf("unexpected doc: %d topics %d concepts", len(doc.Topics), len(doc.Concepts))
	}
}

func TestImportRealGraphIdempotent(t *testing.T) {
	pool := testutil.NewDB(t)
	doc, err := seed.ParseFile(graphFile)
	if err != nil {
		t.Fatal(err)
	}
	svc := app.New(pool, nil)
	first, err := svc.ImportCourseGraph(context.Background(), doc, false)
	if err != nil {
		t.Fatal(err)
	}
	second, err := svc.ImportCourseGraph(context.Background(), doc, false)
	if err != nil {
		t.Fatal(err)
	}
	if first.CourseID != second.CourseID || first.Concepts != len(doc.Concepts) || second.Prerequisites != first.Prerequisites {
		t.Fatalf("not idempotent: %+v vs %+v", first, second)
	}
}
