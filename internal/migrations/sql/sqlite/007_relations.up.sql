-- Relations: the polymorphic edges table (spec-relations). Endpoints are
-- public ids; to_ref keeps the reference as written so dangling wikilink
-- refs resolve at query time; origin separates scanner-owned mention rows
-- from explicit typed links.
CREATE TABLE relations (
    id         INTEGER PRIMARY KEY AUTOINCREMENT,
    from_id    TEXT NOT NULL,
    to_ref     TEXT NOT NULL,
    to_id      TEXT,
    rel_type   TEXT NOT NULL,
    origin     TEXT NOT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT (strftime('%Y-%m-%d %H:%M:%f', 'now')),
    UNIQUE (from_id, to_ref, rel_type)
);

CREATE INDEX relations_from_idx ON relations (from_id);
CREATE INDEX relations_to_id_idx ON relations (to_id);
CREATE INDEX relations_to_ref_idx ON relations (to_ref);
