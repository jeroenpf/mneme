package mcp

import (
	"context"

	sdk "github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/jeroenpf/mneme/internal/store"
)

type lintInput struct{}

// lintDocHit is a lintHit stamped with the owning document — doc_id plus
// block_id are exactly what update_section/update_task need to fix it.
type lintDocHit struct {
	lintHit
	DocID     string `json:"doc_id"`
	DocTitle  string `json:"doc_title"`
	DocStatus string `json:"doc_status"`
}

type lintOutput struct {
	Hits         []lintDocHit `json:"hits"`
	DocsScanned  int          `json:"docs_scanned"`
	DocsWithHits int          `json:"docs_with_hits"`
}

// lintDocuments sweeps every stored document — all projects and statuses,
// archived included — through lintBody. Read-only: it reports what the
// write path would reject today and changes nothing.
func (t *tools) lintDocuments(ctx context.Context, _ *sdk.CallToolRequest, _ lintInput) (*sdk.CallToolResult, *lintOutput, error) {
	docs, err := t.store.ListDocuments(ctx, store.Filter{})
	if err != nil {
		return nil, nil, translateStoreErr(err)
	}
	out := &lintOutput{Hits: []lintDocHit{}, DocsScanned: len(docs)}
	for _, d := range docs {
		hs := lintBody(d.Body)
		if len(hs) == 0 {
			continue
		}
		out.DocsWithHits++
		for _, h := range hs {
			out.Hits = append(out.Hits, lintDocHit{lintHit: h, DocID: d.ID, DocTitle: d.Title, DocStatus: d.Status})
		}
	}
	return nil, out, nil
}
