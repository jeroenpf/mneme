package api_test

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/testcontainers/testcontainers-go"
	tcpostgres "github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"

	"github.com/jeroenpfeil/mneme/internal/api"
	"github.com/jeroenpfeil/mneme/internal/config"
	"github.com/jeroenpfeil/mneme/internal/migrations"
	"github.com/jeroenpfeil/mneme/internal/store"
)

// testPool is set up once in TestMain and shared by all tests. Each
// test calls resetDB to TRUNCATE state, so they don't share rows.
var testPool *pgxpool.Pool

func TestMain(m *testing.M) {
	ctx := context.Background()

	container, err := tcpostgres.Run(ctx,
		"pgvector/pgvector:pg16",
		tcpostgres.WithDatabase("mneme"),
		tcpostgres.WithUsername("mneme"),
		tcpostgres.WithPassword("mneme"),
		testcontainers.WithWaitStrategy(
			wait.ForLog("database system is ready to accept connections").
				WithOccurrence(2).
				WithStartupTimeout(60*time.Second),
		),
	)
	if err != nil {
		fmt.Fprintf(os.Stderr, "start postgres container: %v\n", err)
		os.Exit(1)
	}
	defer func() { _ = testcontainers.TerminateContainer(container) }()

	dsn, err := container.ConnectionString(ctx, "sslmode=disable")
	if err != nil {
		fmt.Fprintf(os.Stderr, "connection string: %v\n", err)
		os.Exit(1)
	}
	if err := migrations.Up(dsn); err != nil {
		fmt.Fprintf(os.Stderr, "migrate up: %v\n", err)
		os.Exit(1)
	}
	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		fmt.Fprintf(os.Stderr, "open pool: %v\n", err)
		os.Exit(1)
	}
	testPool = pool
	defer testPool.Close()

	os.Exit(m.Run())
}

// newServer wires the production Router against a fresh database and
// returns a running httptest.Server. The server is torn down on test
// completion.
func newServer(t *testing.T) (*httptest.Server, *store.PostgresStore) {
	t.Helper()
	resetDB(t)
	st := store.NewWithPool(testPool)
	// NewWithPool calls Close() on the shared pool when the store's
	// Close runs — so we don't defer Close here; each test relies on
	// the TestMain teardown for the pool.
	cfg := &config.Config{CORSOrigins: []string{"http://localhost:5173"}}
	srv := httptest.NewServer(api.Router(cfg, st, nil, nil, nil, nil)) // nil client ⇒ FTS-only, enabled=false; nil hub ⇒ no SSE
	t.Cleanup(srv.Close)
	return srv, st
}

func resetDB(t *testing.T) {
	t.Helper()
	if _, err := testPool.Exec(context.Background(),
		`TRUNCATE documents, projects, embeddings RESTART IDENTITY CASCADE`); err != nil {
		t.Fatalf("truncate: %v", err)
	}
}

func seedProject(t *testing.T, slug string) {
	t.Helper()
	if _, err := testPool.Exec(context.Background(),
		`INSERT INTO projects (name, slug) VALUES ($1, $1)`, slug); err != nil {
		t.Fatalf("seed project %q: %v", slug, err)
	}
}

// --- HTTP helpers ----------------------------------------------------

func doJSON(t *testing.T, method, url string, body any) *http.Response {
	t.Helper()
	var reader io.Reader
	if body != nil {
		raw, err := json.Marshal(body)
		if err != nil {
			t.Fatalf("marshal body: %v", err)
		}
		reader = strings.NewReader(string(raw))
	}
	req, err := http.NewRequest(method, url, reader)
	if err != nil {
		t.Fatalf("build request: %v", err)
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("%s %s: %v", method, url, err)
	}
	return resp
}

func decodeBody(t *testing.T, resp *http.Response, v any) {
	t.Helper()
	defer resp.Body.Close()
	if err := json.NewDecoder(resp.Body).Decode(v); err != nil {
		t.Fatalf("decode response: %v", err)
	}
}

func requireStatus(t *testing.T, resp *http.Response, want int) {
	t.Helper()
	if resp.StatusCode != want {
		body, _ := io.ReadAll(resp.Body)
		resp.Body.Close()
		t.Fatalf("status: got %d, want %d. body=%s", resp.StatusCode, want, body)
	}
}
