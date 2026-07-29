package relations

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"strings"

	"github.com/jeroenpf/mneme/internal/ids"
	"github.com/jeroenpf/mneme/internal/models"
	"github.com/jeroenpf/mneme/internal/store"
)

// RelTypes is the closed set of explicit link kinds. "mentions" is reserved
// for the scanner and never accepted from callers.
var RelTypes = []string{"relates-to", "implements", "supersedes", "depends-on", "blocks"}

// Service owns explicit typed links and the enriched related view. The
// scanner half of the graph (auto mentions) lives in extract.go and the
// command write path.
type Service struct {
	Store store.Store
}

// Entry is one related entity, enriched for display: the endpoint's public
// id (or raw slug when dangling), its kind, a title, the edge's rel type and
// direction relative to the queried entity.
type Entry struct {
	ID        string `json:"id"`
	Kind      string `json:"kind"`
	Title     string `json:"title"`
	RelType   string `json:"rel_type"`
	Direction string `json:"direction"` // out | in
	DocStatus string `json:"doc_status,omitempty"`
	Dangling  bool   `json:"dangling,omitempty"`
}

// Bundle groups an entity's edges: typed links (both directions), outgoing
// mentions, and incoming mentions — the backlinks.
type Bundle struct {
	Links       []Entry `json:"links"`
	Mentions    []Entry `json:"mentions"`
	MentionedBy []Entry `json:"mentioned_by"`
}

// entity is one resolved endpoint.
type entity struct {
	publicID  string
	altRef    string // document slug; "" for other kinds
	kind      string
	title     string
	docStatus string
}

// Link records an explicit typed edge between two existing entities. Both
// refs may be public ids; a bare document slug also resolves.
func (s *Service) Link(ctx context.Context, from, to, relType string) (*models.Relation, error) {
	if !slices.Contains(RelTypes, relType) {
		return nil, fmt.Errorf("rel_type %q is not one of: %s", relType, strings.Join(RelTypes, ", "))
	}
	src, err := s.resolve(ctx, from)
	if err != nil {
		return nil, err
	}
	dst, err := s.resolve(ctx, to)
	if err != nil {
		return nil, err
	}
	if src.publicID == dst.publicID {
		return nil, errors.New("cannot link an entity to itself")
	}
	id := dst.publicID
	rel := &models.Relation{FromID: src.publicID, ToRef: id, ToID: &id, RelType: relType, Origin: "explicit"}
	if err := s.Store.CreateRelation(ctx, rel); err != nil {
		return nil, err
	}
	return rel, nil
}

// Unlink removes explicit edges between the pair — all rel types when
// relType is nil — and reports how many were removed. Mention edges are
// scanner-owned and untouched.
func (s *Service) Unlink(ctx context.Context, from, to string, relType *string) (int64, error) {
	src, err := s.resolve(ctx, from)
	if err != nil {
		return 0, err
	}
	dst, err := s.resolve(ctx, to)
	if err != nil {
		return 0, err
	}
	return s.Store.DeleteExplicitRelations(ctx, src.publicID, dst.publicID, relType)
}

// Related returns the entity's enriched edge bundle: typed links in both
// directions, outgoing mentions, and the mentioned-by backlinks.
func (s *Service) Related(ctx context.Context, ref string) (*Bundle, error) {
	e, err := s.resolve(ctx, ref)
	if err != nil {
		return nil, err
	}
	out, in, err := s.Store.ListRelations(ctx, e.publicID, e.altRef)
	if err != nil {
		return nil, err
	}

	b := &Bundle{Links: []Entry{}, Mentions: []Entry{}, MentionedBy: []Entry{}}
	cache := map[string]*entity{e.publicID: e}
	for _, r := range out {
		entry := s.targetEntry(ctx, cache, r)
		entry.RelType = r.RelType
		entry.Direction = "out"
		if r.RelType == "mentions" {
			b.Mentions = append(b.Mentions, entry)
		} else {
			b.Links = append(b.Links, entry)
		}
	}
	for _, r := range in {
		entry := s.endpointEntry(ctx, cache, r.FromID)
		entry.RelType = r.RelType
		entry.Direction = "in"
		if r.RelType == "mentions" {
			b.MentionedBy = append(b.MentionedBy, entry)
		} else {
			b.Links = append(b.Links, entry)
		}
	}
	return b, nil
}

// targetEntry describes an edge's far end. A dangling to_ref (a wikilink
// slug whose document does not exist yet) renders as a dimmed document
// placeholder rather than an error.
func (s *Service) targetEntry(ctx context.Context, cache map[string]*entity, r *models.Relation) Entry {
	ref := r.ToRef
	if r.ToID != nil {
		ref = *r.ToID
	}
	return s.entryFor(ctx, cache, ref)
}

func (s *Service) endpointEntry(ctx context.Context, cache map[string]*entity, publicID string) Entry {
	return s.entryFor(ctx, cache, publicID)
}

func (s *Service) entryFor(ctx context.Context, cache map[string]*entity, ref string) Entry {
	if e, ok := cache[ref]; ok {
		return Entry{ID: e.publicID, Kind: e.kind, Title: e.title, DocStatus: e.docStatus}
	}
	e, err := s.resolve(ctx, ref)
	if err != nil {
		// Dangling: the reference names nothing (yet). Wikilink slugs are
		// document refs by convention.
		return Entry{ID: ref, Kind: "document", Title: ref, Dangling: true}
	}
	cache[ref] = e
	return Entry{ID: e.publicID, Kind: e.kind, Title: e.title, DocStatus: e.docStatus}
}

// resolve turns a user-supplied ref — a public id of any linkable kind, or a
// bare document slug — into a resolved endpoint.
func (s *Service) resolve(ctx context.Context, ref string) (*entity, error) {
	ref = strings.TrimSpace(ref)
	if ref == "" {
		return nil, errors.New("ref is required")
	}
	kind, _, err := ids.Parse(ref)
	if err != nil {
		doc, derr := s.Store.GetDocument(ctx, ref)
		if derr != nil {
			return nil, fmt.Errorf("%q is neither a public id nor a document id", ref)
		}
		return &entity{publicID: doc.PublicID, altRef: doc.ID, kind: "document", title: doc.Title, docStatus: doc.Status}, nil
	}
	switch kind {
	case ids.KindDocument:
		doc, err := s.Store.GetDocumentByPublicID(ctx, ref)
		if err != nil {
			return nil, notFound(err, kind, ref)
		}
		return &entity{publicID: doc.PublicID, altRef: doc.ID, kind: "document", title: doc.Title, docStatus: doc.Status}, nil
	case ids.KindDecision:
		d, err := s.Store.GetDecisionByPublicID(ctx, ref)
		if err != nil {
			return nil, notFound(err, kind, ref)
		}
		return &entity{publicID: d.PublicID, kind: "decision", title: d.Title}, nil
	case ids.KindSnippet:
		sn, err := s.Store.GetSnippetByPublicID(ctx, ref)
		if err != nil {
			return nil, notFound(err, kind, ref)
		}
		return &entity{publicID: sn.PublicID, kind: "snippet", title: sn.Title}, nil
	case ids.KindJournal:
		je, err := s.Store.GetJournalEntryByPublicID(ctx, ref)
		if err != nil {
			return nil, notFound(err, kind, ref)
		}
		return &entity{publicID: je.PublicID, kind: "journal", title: je.Summary}, nil
	case ids.KindSolution:
		sol, err := s.Store.GetSolutionByPublicID(ctx, ref)
		if err != nil {
			return nil, notFound(err, kind, ref)
		}
		return &entity{publicID: sol.PublicID, kind: "solution", title: sol.ErrorDescription}, nil
	default:
		return nil, fmt.Errorf("%s entities cannot be relation endpoints", kind)
	}
}

func notFound(err error, kind ids.Kind, ref string) error {
	if errors.Is(err, store.ErrNotFound) {
		return fmt.Errorf("no %s found for %q", kind, ref)
	}
	return err
}
