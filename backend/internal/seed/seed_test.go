package seed_test

import (
	"context"
	"testing"

	"github.com/thekogo/tutor-platform/backend/internal/app"
	"github.com/thekogo/tutor-platform/backend/internal/seed"
	"github.com/thekogo/tutor-platform/backend/internal/testutil"
)

// courses are the real graphs under content/, by Course slug.
var courses = []string{"cu-best-math", "cu-best-business", "cu-tep"}

func graphFile(slug string) string { return "../../../content/" + slug + "/graph.yaml" }

func TestParseRealGraphs(t *testing.T) {
	for _, slug := range courses {
		doc, err := seed.ParseFile(graphFile(slug))
		if err != nil {
			t.Fatalf("%s: %v", slug, err)
		}
		if doc.Course.ID != slug || len(doc.Topics) == 0 || len(doc.Concepts) == 0 || len(doc.QuestionFormats) == 0 {
			t.Fatalf("%s: unexpected doc: %d topics %d concepts", slug, len(doc.Topics), len(doc.Concepts))
		}
	}
}

func TestImportRealGraphsIdempotentAndSeparate(t *testing.T) {
	pool := testutil.NewDB(t)
	svc := app.New(pool, nil)
	ids := map[string]string{} // Course id → slug
	for _, slug := range courses {
		doc, err := seed.ParseFile(graphFile(slug))
		if err != nil {
			t.Fatal(err)
		}
		first, err := svc.ImportCourseGraph(context.Background(), doc, false)
		if err != nil {
			t.Fatalf("%s: %v", slug, err)
		}
		second, err := svc.ImportCourseGraph(context.Background(), doc, false)
		if err != nil {
			t.Fatalf("%s: %v", slug, err)
		}
		if first.CourseID != second.CourseID || first.Concepts != len(doc.Concepts) || second.Prerequisites != first.Prerequisites {
			t.Fatalf("%s not idempotent: %+v vs %+v", slug, first, second)
		}
		if other, dup := ids[first.CourseID]; dup {
			t.Fatalf("%s overwrote Course %s", slug, other)
		}
		ids[first.CourseID] = slug
	}
	// Importing the second Course must leave the first one intact.
	cs, err := svc.ListCourses(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(cs) != len(courses) {
		t.Fatalf("want %d Courses, got %d", len(courses), len(cs))
	}
	for _, c := range cs {
		g, err := svc.Graph(context.Background(), c.ID)
		if err != nil {
			t.Fatal(err)
		}
		doc, err := seed.ParseFile(graphFile(c.Slug))
		if err != nil {
			t.Fatal(err)
		}
		if len(g.LiveConcepts()) != len(doc.Concepts) {
			t.Fatalf("%s: %d live Concepts after importing both, want %d", c.Slug, len(g.LiveConcepts()), len(doc.Concepts))
		}
	}
}
