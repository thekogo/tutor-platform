// Package testutil provides real-Postgres databases and fixtures for
// integration tests.
//
// Database source, in order:
//  1. TEST_DATABASE_URL (e.g. the CI Postgres service) — each test gets its
//     own freshly migrated database created from it;
//  2. otherwise a postgres:17-alpine container started once per test binary
//     via testcontainers-go;
//  3. if Docker is unavailable the test is skipped.
package testutil

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"net/url"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/testcontainers/testcontainers-go"
	tcpostgres "github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"

	"github.com/thekogo/tutor-platform/backend/internal/store"
)

var (
	baseOnce sync.Once
	baseURL  string
	errBase  error
)

func base() (string, error) {
	baseOnce.Do(func() {
		if u := os.Getenv("TEST_DATABASE_URL"); u != "" {
			baseURL = u
			return
		}
		ctx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
		defer cancel()
		c, err := tcpostgres.Run(ctx, "postgres:17-alpine",
			tcpostgres.WithDatabase("tutor"), tcpostgres.WithUsername("tutor"), tcpostgres.WithPassword("tutor"),
			testcontainers.WithWaitStrategy(wait.ForLog("database system is ready to accept connections").
				WithOccurrence(2).WithStartupTimeout(2*time.Minute)),
		)
		if err != nil {
			errBase = err
			return
		}
		baseURL, errBase = c.ConnectionString(ctx, "sslmode=disable")
	})
	return baseURL, errBase
}

// NewDB returns a pool to a new, migrated, empty database dropped at test end.
func NewDB(t testing.TB) *pgxpool.Pool {
	t.Helper()
	u, err := base()
	if err != nil {
		t.Skipf("no test database (set TEST_DATABASE_URL or run Docker): %v", err)
	}
	ctx := context.Background()
	admin, err := pgx.Connect(ctx, u)
	if err != nil {
		t.Skipf("test database unreachable: %v", err)
	}
	defer func() { _ = admin.Close(ctx) }()
	b := make([]byte, 6)
	_, _ = rand.Read(b)
	name := "t_" + hex.EncodeToString(b)
	if _, err := admin.Exec(ctx, "CREATE DATABASE "+name); err != nil {
		t.Fatalf("create database: %v", err)
	}
	parsed, err := url.Parse(u)
	if err != nil {
		t.Fatal(err)
	}
	parsed.Path = "/" + name
	pool, err := store.Open(ctx, parsed.String())
	if err != nil {
		t.Fatal(err)
	}
	if err := store.Migrate(ctx, pool); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		pool.Close()
		c, err := pgx.Connect(context.Background(), u)
		if err != nil {
			return
		}
		defer func() { _ = c.Close(context.Background()) }()
		_, _ = c.Exec(context.Background(), fmt.Sprintf("DROP DATABASE IF EXISTS %s WITH (FORCE)", name))
	})
	return pool
}
