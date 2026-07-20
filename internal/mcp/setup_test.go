package mcp_test

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	sdk "github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/testcontainers/testcontainers-go"
	tcpostgres "github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"

	"github.com/jeroenpf/mneme/internal/embed"
	"github.com/jeroenpf/mneme/internal/live"
	mcpsrv "github.com/jeroenpf/mneme/internal/mcp"
	"github.com/jeroenpf/mneme/internal/migrations"
	"github.com/jeroenpf/mneme/internal/store"
)

// testPool is shared across the whole package; each test calls
// resetDB() to start from a clean slate.
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

// newClient brings up a fresh DB + MCP server + connected in-memory
// MCP client. Each test gets its own session, so they don't share
// state. Cleanup runs on t.Cleanup.
func newClient(t *testing.T) *sdk.ClientSession {
	return newClientWith(t, nil) // nil client ⇒ FTS-only (no embedding in tests)
}

// fakeEmbedClient is a deterministic embed.Client for MCP tests that need a
// non-nil client (e.g. to exercise the empty-result hint on the search tool).
type fakeEmbedClient struct{}

func (fakeEmbedClient) Model() string { return "fake" }
func (fakeEmbedClient) Embed(_ context.Context, texts []string, _ string) ([][]float32, error) {
	out := make([][]float32, len(texts))
	for i := range texts {
		out[i] = make([]float32, 1024)
	}
	return out, nil
}

// newClientWith is newClient with an injectable embed client.
func newClientWith(t *testing.T, client embed.Client) *sdk.ClientSession {
	t.Helper()
	resetDB(t)
	st := store.NewWithPool(testPool)
	return connect(t, mcpsrv.New(st, nil, nil, client))
}

// newClientWithBroadcaster is newClient with an injectable live broadcaster,
// so tests can assert which events a write emits. FTS-only (nil embed client).
func newClientWithBroadcaster(t *testing.T, bc live.Broadcaster) *sdk.ClientSession {
	t.Helper()
	resetDB(t)
	st := store.NewWithPool(testPool)
	return connect(t, mcpsrv.New(st, nil, bc, nil))
}

// connect wires an in-memory MCP client session to srv and registers
// teardown. Shared by the newClient* helpers.
func connect(t *testing.T, srv *mcpsrv.Server) *sdk.ClientSession {
	t.Helper()
	t1, t2 := sdk.NewInMemoryTransports()
	ctx := context.Background()

	server := sdkServer(srv)
	serverSess, err := server.Connect(ctx, t1, nil)
	if err != nil {
		t.Fatalf("server connect: %v", err)
	}

	sdkClient := sdk.NewClient(&sdk.Implementation{Name: "test", Version: "0"}, nil)
	clientSess, err := sdkClient.Connect(ctx, t2, nil)
	if err != nil {
		t.Fatalf("client connect: %v", err)
	}

	t.Cleanup(func() {
		_ = clientSess.Close()
		_ = serverSess.Wait()
	})
	return clientSess
}

// sdkServer reaches inside *mcpsrv.Server to grab the underlying
// *sdk.Server. The package exposes only an http.Handler in production
// since the only consumer is main.go; tests need the raw server to
// drive it through the in-memory transport.
func sdkServer(s *mcpsrv.Server) *sdk.Server {
	return s.SDKServer()
}

func resetDB(t *testing.T) {
	t.Helper()
	if _, err := testPool.Exec(context.Background(),
		`TRUNCATE documents, document_revisions, projects, embeddings, embed_failures RESTART IDENTITY CASCADE`); err != nil {
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

// --- tool-call helpers ----------------------------------------------

// call invokes a tool by name and decodes its structured output into
// out. Fails the test with a descriptive message on any error or if
// the result was marked IsError.
func call(t *testing.T, cs *sdk.ClientSession, name string, args any, out any) {
	t.Helper()
	res, err := cs.CallTool(context.Background(), &sdk.CallToolParams{
		Name:      name,
		Arguments: args,
	})
	if err != nil {
		t.Fatalf("call %s: %v", name, err)
	}
	if res.IsError {
		t.Fatalf("call %s returned tool error: %s", name, contentText(res))
	}
	if out == nil {
		return
	}
	// The SDK auto-fills text content with the JSON encoding of the
	// typed output, regardless of whether the handler also populated
	// StructuredContent. Decoding from Content is the most reliable
	// path for typed payloads.
	if err := json.Unmarshal([]byte(contentText(res)), out); err != nil {
		t.Fatalf("decode %s output: %v\nbody=%s", name, err, contentText(res))
	}
}

// callExpectError invokes a tool expecting either a transport error or
// an IsError tool result. Returns the message presented to the LLM.
func callExpectError(t *testing.T, cs *sdk.ClientSession, name string, args any) string {
	t.Helper()
	res, err := cs.CallTool(context.Background(), &sdk.CallToolParams{
		Name:      name,
		Arguments: args,
	})
	if err != nil {
		return err.Error()
	}
	if !res.IsError {
		t.Fatalf("call %s expected error, got success: %s", name, contentText(res))
	}
	return contentText(res)
}

func contentText(res *sdk.CallToolResult) string {
	for _, c := range res.Content {
		if tc, ok := c.(*sdk.TextContent); ok {
			return tc.Text
		}
	}
	return ""
}
