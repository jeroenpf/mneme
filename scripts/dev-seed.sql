-- Dev seed: sample projects + documents so the registry UI renders
-- real data. Idempotent — safe to re-run. Load with `make seed`.
-- Timestamps are spread out so updated_at ordering is visible.

INSERT INTO projects (name, slug, description) VALUES
  ('Mneme',    'mneme',    'Local AI dev knowledge service'),
  ('Hyperion', 'hyperion', 'Home automation platform'),
  ('Dotfiles', 'dotfiles', 'Machine setup and shell environment')
ON CONFLICT (slug) DO NOTHING;

INSERT INTO documents
  (id, title, project, type, status, ticket, repo, tags,
   phase_current, phase_total, meta, body, created_at, updated_at)
VALUES
  ('repo-vs-mneme-delineation', 'Repo vs Mneme delineation', 'mneme', 'spec', 'complete',
   NULL, 'jeroenpf/mneme', '{architecture,docs}', NULL, NULL,
   '{"description": "Decision record for what lives in git versus Mneme: the repo owns durable present-tense docs about the artifact, Mneme owns evolving work docs. Pointers across the line, never copies."}',
   '{"sections": []}', now() - interval '20 days', now() - interval '1 day'),

  ('hyperion-q2-energy-report', 'Q2 energy consumption report', 'hyperion', 'report', 'complete',
   NULL, NULL, '{energy}', NULL, NULL,
   '{"description": "Quarterly breakdown of household consumption from the P1 meter: baseload crept up 8 percent, dryer dominates peaks, solar self-consumption at 61 percent."}',
   '{"sections": []}', now() - interval '10 days', now() - interval '6 days'),

  ('hyperion-presence-detection-adr', 'Room presence detection approach', 'hyperion', 'adr', 'blocked',
   'HYP-198', 'jeroenpf/hyperion', '{presence,architecture}', NULL, NULL,
   '{"description": "Choosing between mmWave sensors, BLE beacons, and camera-based detection for room-level presence. Blocked on mmWave sensor availability before a bench comparison can run."}',
   '{"sections": []}', now() - interval '30 days', now() - interval '12 days'),

  ('dotfiles-nix-darwin-brainstorm', 'nix-darwin migration ideas', 'dotfiles', 'brainstorm', 'todo',
   NULL, NULL, '{nix,macos}', NULL, NULL,
   '{"description": "Wild ideas for declaring the whole machine in nix-darwin: homebrew casks as nix packages, dock layout as config, one-command restore onto a blank Mac."}',
   '{"sections": []}', now() - interval '5 days', now() - interval '5 days'),

  ('mneme-session-journal-june', 'Session journal — June', 'mneme', 'journal', 'archived',
   NULL, NULL, '{journal}', NULL, NULL,
   '{"description": "Working log from the June sessions: schema decisions, pgx wrangling, MCP SDK evaluation notes. Superseded by the committed specs."}',
   '{"sections": []}', now() - interval '40 days', now() - interval '25 days'),

  ('hyperion-deconz-notes', 'deCONZ pairing notes', 'hyperion', 'journal', 'archived',
   NULL, NULL, '{zigbee,deconz}', NULL, NULL,
   '{"description": "Accumulated pairing quirks and firmware workarounds from the deCONZ era. Kept for reference; superseded by the zigbee2mqtt migration plan."}',
   '{"sections": []}', now() - interval '60 days', now() - interval '45 days')
ON CONFLICT (id) DO NOTHING;

-- Rich documents — the phase 1.7 viewer's reference material. Upserted
-- (not DO NOTHING) so re-running the seed refreshes meta/body on an
-- already-seeded database. The updated_at trigger stamps them on re-run.
INSERT INTO documents
  (id, title, project, type, status, ticket, repo, tags,
   phase_current, phase_total, meta, body, created_at, updated_at)
VALUES
  ('mneme-implementation', 'Mneme implementation', 'mneme', 'plan', 'in-progress',
   NULL, 'jeroenpf/mneme', '{go,vue,postgres}', 6, 9,
   $$
   {
     "description": "Phase plan for the core service: Go REST plus MCP server, Vue registry and viewer, trusted local TLS. Sub-phases 1.1 through 1.6 are done; the document viewer is in flight.",
     "phases": [
       { "title": "Scaffolding",      "status": "done" },
       { "title": "Documents API",    "status": "done" },
       { "title": "Search",           "status": "done" },
       { "title": "MCP server",       "status": "done" },
       { "title": "Vue scaffold",     "status": "done" },
       { "title": "Registry UI",      "status": "wip"  },
       { "title": "Document viewer",  "status": "todo" },
       { "title": "Local TLS",        "status": "todo" },
       { "title": "Phase 2 prep",     "status": "todo" }
     ],
     "custom_fields": {
       "Complexity": "Medium",
       "Sessions": "6 of 9",
       "Scope": "weekend-sized slices"
     }
   }
   $$::jsonb,
   $$
   {
     "sections": [
       {
         "type": "section", "id": "overview", "title": "Overview",
         "children": [
           { "type": "text", "id": "ov-p1",
             "content": "Mneme is a **local-first** knowledge service for AI-assisted development. Documents are born over MCP, live in Postgres, and render in a *read-mostly* Vue viewer — see the [delineation spec](/doc/repo-vs-mneme-delineation) for what belongs on which side of the git line. The stack is `Go 1.24` + `pgx/v5` + Vue 3." },
           { "type": "key-value", "id": "ov-kv", "title": "At a glance",
             "data": {
               "Stack": "Go + PostgreSQL + Vue 3",
               "Transport": "REST for the UI, MCP for mutations",
               "Host": "macOS laptop, Docker Compose",
               "Domain": "`https://mneme.local` via a `127.0.0.2` loopback alias"
             } },
           { "type": "callout", "id": "ov-c1", "variant": "info",
             "content": "The UI is a **read-mostly** viewer. Every mutation goes through MCP tools — `push_document`, `update_task`, `advance_phase`." }
         ]
       },
       {
         "type": "section", "id": "architecture", "title": "Architecture",
         "children": [
           { "type": "text", "id": "ar-p1",
             "content": "Requests fan out from one chi router: REST handlers feed the SPA, the `/mcp` endpoint feeds Claude Code. Both share the same `Store` interface, so the transport layer stays thin." },
           { "type": "diagram", "id": "ar-d1", "title": "Request flow",
             "content": "flowchart LR\n  CC[Claude Code] -->|MCP| S[Go server]\n  UI[Vue SPA] -->|REST| S\n  S --> PG[(PostgreSQL)]" },
           {
             "type": "section", "id": "api-surface", "title": "API surface",
             "children": [
               { "type": "table", "id": "as-t1", "title": "REST endpoints",
                 "cols": ["Method", "Path", "Purpose"],
                 "rows": [
                   ["GET", "`/api/v1/documents`", "list + filter, **cursor** paged"],
                   ["GET", "`/api/v1/documents/:id`", "single document"],
                   ["GET", "`/api/v1/projects`", "projects with per-status counts"]
                 ] },
               { "type": "code", "id": "as-c1", "lang": "go", "filename": "internal/store/store.go",
                 "content": "// Store abstracts persistence so handlers stay testable.\ntype Store interface {\n\tGetDocument(ctx context.Context, id string) (*models.Document, error)\n\tListDocuments(ctx context.Context, f models.DocumentFilter) ([]models.Document, string, error)\n\tUpsertDocument(ctx context.Context, doc *models.Document) error\n}" },
               { "type": "code", "id": "as-c2", "lang": "sql", "filename": "internal/migrations/sql/002_documents.up.sql",
                 "content": "-- weighted search vector, generated from title/ticket/tags/body\nsearch_vector TSVECTOR GENERATED ALWAYS AS (\n  documents_search_vector(title, ticket, tags, body)\n) STORED;\n\nCREATE INDEX documents_search_vector_idx\n  ON documents USING GIN (search_vector);" }
             ]
           },
           { "type": "callout", "id": "ar-c1", "variant": "warn",
             "content": "Postgres is tuned for a personal dataset — `shared_buffers=256MB`, `work_mem=8MB`, `max_connections=20`. Do **not** point a fleet at it." }
         ]
       },
       {
         "type": "subphase", "id": "sp-1-7", "num": "1.7", "title": "Document viewer", "session": 7,
         "description": "Meta chrome plus a recursive block renderer over `body.sections` — no runtime template compilation.",
         "tasks": [
           { "id": "t-171", "title": "`DocumentView.vue` shell — sidebar + content column", "done": true, "tags": ["vue"] },
           { "id": "t-172", "title": "Meta header from `doc.meta`", "done": true },
           { "id": "t-173", "title": "Phase tracker sidebar", "done": false,
             "content": "Built from `meta.phases[]`; scroll-aware active highlight rides the section observer." },
           { "id": "t-174", "title": "Prism + Mermaid lazy rendering", "done": false, "tags": ["prism", "mermaid"] }
         ],
         "children": [
           { "type": "callout", "id": "sp17-c1", "variant": "note",
             "content": "Body is structured JSON dispatched via `<component :is>` — the standard Vue runtime is enough. No `@vue/compiler-dom` at runtime." }
         ]
       },
       {
         "type": "section", "id": "decisions", "title": "Decisions & risks",
         "children": [
           { "type": "callout", "id": "de-c1", "variant": "success", "title": "Locked",
             "content": "The official `modelcontextprotocol/go-sdk` v1 powers the MCP endpoint — stable since phase 1.4." },
           { "type": "callout", "id": "de-c2", "variant": "danger", "title": "Watch",
             "content": "The loopback alias `127.0.0.2` must survive reboots or `mneme.local` silently dies — the LaunchDaemon in 1.8 owns this." },
           { "type": "task-list", "id": "de-t1", "title": "Open follow-ups",
             "tasks": [
               { "id": "t-d1", "title": "Graduation export — render a doc to repo markdown", "done": false },
               { "id": "t-d2", "title": "Snippet hits in `search_documents`", "done": false, "tags": ["search"] }
             ] },
           { "type": "text", "id": "de-p1",
             "content": "Everything else rides the [1.9 backlog](#decisions) until real friction shows up — we do not speculate." }
         ]
       }
     ]
   }
   $$::jsonb,
   now() - interval '50 days', now() - interval '2 hours'),

  ('hyperion-zigbee-migration', 'Zigbee2MQTT migration', 'hyperion', 'plan', 'in-progress',
   'HYP-231', 'jeroenpf/hyperion', '{zigbee,mqtt}', 2, 6,
   $$
   {
     "description": "Move all 40-plus devices off the aging deCONZ stack onto zigbee2mqtt: inventory, coordinator swap, room-by-room re-pairing, automation rewrites, deCONZ decommission.",
     "phases": [
       { "title": "Inventory",           "status": "done" },
       { "title": "Coordinator swap",    "status": "wip"  },
       { "title": "Re-pair rooms",       "status": "todo" },
       { "title": "Automation rewrites", "status": "todo" },
       { "title": "Cutover",             "status": "todo" },
       { "title": "Decommission",        "status": "todo" }
     ],
     "custom_fields": { "Devices": "43", "Coordinator": "SLZB-06M" }
   }
   $$::jsonb,
   $$
   {
     "sections": [
       {
         "type": "section", "id": "z-overview", "title": "Overview",
         "children": [
           { "type": "text", "id": "z-p1",
             "content": "deCONZ has been **end-of-life** in this house since the ConBee II started dropping its mesh weekly. Target stack: `zigbee2mqtt` on the SLZB-06M PoE coordinator, `ember` driver." },
           { "type": "table", "id": "z-t1", "title": "Device inventory",
             "cols": ["Room", "Devices", "Quirks"],
             "rows": [
               ["Living room", "12", "2 bulbs stuck on ancient firmware"],
               ["Kitchen", "8", "power plugs report watts as `state_l1`"],
               ["Bedrooms", "14", "Aqara sensors — re-pair *slowly*"],
               ["Utility", "9", "routers first, then end devices"]
             ] },
           { "type": "callout", "id": "z-c1", "variant": "warn",
             "content": "Pair **mains-powered routers before battery end devices** or the mesh forms star-shaped around the coordinator and dies at the first wall." }
         ]
       },
       {
         "type": "subphase", "id": "z-sp-2", "num": "2", "title": "Coordinator swap", "session": 2,
         "description": "Bring up zigbee2mqtt against the SLZB-06M without touching the deCONZ network yet.",
         "tasks": [
           { "id": "z-t21", "title": "Flash SLZB-06M to latest `ember` firmware", "done": true },
           { "id": "z-t22", "title": "zigbee2mqtt container + config", "done": true, "tags": ["docker"] },
           { "id": "z-t23", "title": "Form new network on a **different channel** than deCONZ", "done": false,
             "content": "deCONZ sits on channel 15 — form on 20 so both meshes coexist during migration." },
           { "id": "z-t24", "title": "Smoke-pair one sacrificial plug", "done": false }
         ],
         "children": [
           { "type": "code", "id": "z-c2", "lang": "yaml", "filename": "zigbee2mqtt/configuration.yaml",
             "content": "mqtt:\n  server: mqtt://mosquitto:1883\nserial:\n  adapter: ember\n  port: tcp://slzb-06m.lan:6638\nadvanced:\n  channel: 20\n  transmit_power: 20" },
           { "type": "diagram", "id": "z-d1", "title": "Swap sequence",
             "content": "flowchart TD\n  A[deCONZ ch15] -->|keep running| A\n  B[SLZB-06M] --> C[z2m forms ch20]\n  C --> D{smoke pair OK?}\n  D -->|yes| E[room-by-room re-pair]\n  D -->|no| F[check ember firmware]" }
         ]
       },
       {
         "type": "section", "id": "z-rollback", "title": "Rollback plan",
         "children": [
           { "type": "key-value", "id": "z-kv1",
             "data": {
               "Trigger": "pairing failure rate above 20 percent in any room",
               "Fallback": "deCONZ untouched on channel 15 — devices re-join it on reset",
               "Data loss": "none; automations keep dual entity ids until cutover"
             } }
         ]
       }
     ]
   }
   $$::jsonb,
   now() - interval '15 days', now() - interval '3 days')
ON CONFLICT (id) DO UPDATE SET
  title         = EXCLUDED.title,
  status        = EXCLUDED.status,
  ticket        = EXCLUDED.ticket,
  repo          = EXCLUDED.repo,
  tags          = EXCLUDED.tags,
  phase_current = EXCLUDED.phase_current,
  phase_total   = EXCLUDED.phase_total,
  meta          = EXCLUDED.meta,
  body          = EXCLUDED.body;

-- Sample decisions so the decision-log page renders real data. Fixed
-- UUIDs keep this idempotent (ON CONFLICT (id) DO NOTHING).
INSERT INTO decisions (id, title, project, decision, rationale, alternatives, consequences, status, created_at)
VALUES
  ('00000000-0000-0000-0000-0000000000d1', 'Raw SQL over an ORM', 'mneme',
   'Use jackc/pgx v5 with hand-written SQL; no ORM.',
   'Keeps queries explicit and debuggable; avoids ORM "magic" and N+1 surprises on a personal-scale dataset.',
   'GORM, sqlc, ent.', 'More boilerplate for scans; no compile-time query checking.',
   'accepted', now() - interval '35 days'),
  ('00000000-0000-0000-0000-0000000000d2', 'Go server terminates TLS directly', 'mneme',
   'Serve HTTPS from the Go process with an mkcert leaf cert; no reverse proxy.',
   'One fewer moving part for a local single-user tool; mkcert is already trusted in the Keychain.',
   'nginx/Caddy in front; plain HTTP behind Docker.', 'The app owns cert paths and renewal.',
   'accepted', now() - interval '18 days'),
  ('00000000-0000-0000-0000-0000000000d3', 'mmWave for room presence', 'hyperion',
   'Prefer mmWave sensors for room-level presence detection.',
   'Detects stationary presence that PIR misses; no cameras in living spaces.',
   'BLE beacons; camera CV.', 'Sensor availability is the current blocker.',
   'proposed', now() - interval '9 days')
ON CONFLICT (id) DO NOTHING;

-- Parent task 2.2 #5 (data half): global guidance that get_memory(global)
-- surfaces, telling Claude Code when to log decisions. DO UPDATE so a
-- re-seed refreshes the wording.
INSERT INTO memories (scope, key, value)
VALUES ('global', 'decision-logging',
  'Log architecture decisions with log_decision as you make them: tech/library choices, pattern selections, and trade-off resolutions. Capture title + decision + rationale (add alternatives/consequences when they matter). This mutable log is searchable via query_decisions; promote a decision to a repo ADR only once it hardens.')
ON CONFLICT (scope, project, area, key) DO UPDATE SET value = EXCLUDED.value;

-- Sample snippets so the snippet-library page renders real data. Fixed
-- UUIDs keep this idempotent (ON CONFLICT (id) DO NOTHING). Languages stay
-- within the grammars web/src/lib/highlight.ts loads (go, sql, typescript)
-- so previews highlight without touching that file. NULL project = global.
INSERT INTO snippets (id, title, project, language, content, tags, description, created_at)
VALUES
  ('00000000-0000-0000-0000-0000000000f1', 'Typed store error translation', 'mneme', 'go',
   E'var pgErr *pgconn.PgError\nif errors.As(err, &pgErr) && pgErr.Code == pgForeignKeyViolation {\n\treturn ErrInvalidProject\n}',
   ARRAY['errors', 'pgx'],
   'Map a Postgres FK violation to a typed domain error at the store boundary.',
   now() - interval '20 days'),
  ('00000000-0000-0000-0000-0000000000f2', 'Weighted FTS search_vector', 'mneme', 'sql',
   E'CREATE INDEX snippets_fts_idx ON snippets\n  USING GIN (search_vector);\nSELECT * FROM snippets\nWHERE search_vector @@ websearch_to_tsquery($1);',
   ARRAY['postgres', 'fts'],
   'Query the generated tsvector column with a GIN index for ranked search.',
   now() - interval '12 days'),
  ('00000000-0000-0000-0000-0000000000f3', 'errgroup bounded fan-out', NULL, 'go',
   E'g, ctx := errgroup.WithContext(ctx)\ng.SetLimit(8)\nfor _, item := range items {\n\tg.Go(func() error { return handle(ctx, item) })\n}\nreturn g.Wait()',
   ARRAY['concurrency'],
   'Parallelize independent calls with a concurrency cap; first error cancels the rest.',
   now() - interval '6 days'),
  ('00000000-0000-0000-0000-0000000000f4', 'Debounced ref composable', 'hyperion', 'typescript',
   E'export function useDebounced<T>(src: Ref<T>, ms = 200): Ref<T> {\n  const out = ref(src.value) as Ref<T>\n  watch(src, (v) => setTimeout(() => (out.value = v), ms))\n  return out\n}',
   ARRAY['vue', 'composable'],
   'Debounce a reactive source ref for search-as-you-type inputs.',
   now() - interval '3 days')
ON CONFLICT (id) DO NOTHING;

-- Sample journal entries so the dev-journal timeline renders real data.
-- Fixed UUIDs keep this idempotent (ON CONFLICT (id) DO NOTHING). NULL
-- project = a global/cross-project session entry.
INSERT INTO journal_entries (id, project, session_ref, summary, accomplished, deferred, created_at)
VALUES
  ('00000000-0000-0000-0000-0000000000a1', 'mneme', 'sp-2-2',
   'Built the decision log end to end.',
   ARRAY['decisions migration + store', 'log_decision / query_decisions', 'Vue decision page'],
   ARRAY['ranked search tuning'],
   now() - interval '10 days'),
  ('00000000-0000-0000-0000-0000000000a2', 'mneme', 'sp-2-3',
   'Shipped the snippet library.',
   ARRAY['snippets migration + FTS', 'save_snippet / search_snippets', 'copyable syntax-highlighted browser'],
   ARRAY['broaden Prism grammar set'],
   now() - interval '3 days'),
  ('00000000-0000-0000-0000-0000000000a3', 'hyperion', 'sp-1-4',
   'Migrated the coordinator onto zigbee2mqtt.',
   ARRAY['device inventory', 'coordinator swap'],
   ARRAY['room-by-room re-pairing'],
   now() - interval '6 days'),
  ('00000000-0000-0000-0000-0000000000a4', NULL, '',
   'Sorted out local TLS + hostname plumbing for the dev stack.',
   ARRAY['mkcert CA + leaf cert', 'mneme.dev hosts entry'],
   ARRAY['document the setup-host flow'],
   now() - interval '1 day')
ON CONFLICT (id) DO NOTHING;

-- Sample solutions so the error/solution browser renders real data.
-- Fixed UUIDs keep this idempotent (ON CONFLICT (id) DO NOTHING). NULL
-- project = a global/cross-project gotcha.
INSERT INTO solutions (id, project, error_description, solution, tags, source_url, created_at)
VALUES
  ('00000000-0000-0000-0000-0000000000b1', 'mneme',
   'mneme.dev resolves slowly (~5s stall) on macOS before hitting /etc/hosts',
   'macOS routes *.local through mDNS; use a non-.local host (mneme.dev) mapped in /etc/hosts instead of mneme.local',
   ARRAY['macos', 'dns', 'tls'], '',
   now() - interval '9 days'),
  ('00000000-0000-0000-0000-0000000000b2', NULL,
   'https://mneme.dev:8443 refuses connections while the container is healthy',
   'Docker Desktop port forwarder wedged; restart Docker Desktop entirely (a container recreate does not fix it)',
   ARRAY['docker', 'macos'], '',
   now() - interval '5 days'),
  ('00000000-0000-0000-0000-0000000000b3', 'mneme',
   'pgx scan fails: cannot scan NULL into *string for a nullable column',
   'Model nullable FK columns as *string, not string; only non-null text columns use a plain string field',
   ARRAY['go', 'pgx', 'postgres'], '',
   now() - interval '2 days'),
  ('00000000-0000-0000-0000-0000000000b4', 'hyperion',
   'zigbee2mqtt cannot open the coordinator after a host reboot',
   'The USB device path changed; pin it via /dev/serial/by-id in the compose device mapping',
   ARRAY['zigbee', 'docker'], 'https://www.zigbee2mqtt.io/guide/configuration/adapter-settings.html',
   now() - interval '4 days')
ON CONFLICT (id) DO NOTHING;
