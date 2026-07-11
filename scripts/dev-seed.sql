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
  ('mneme-implementation', 'Mneme implementation', 'mneme', 'plan', 'in-progress',
   NULL, 'jeroenpfeil/mneme', '{go,vue,postgres}', 6, 9,
   '{"description": "Phase plan for the core service: Go REST plus MCP server, Vue registry and viewer, trusted local TLS. Sub-phases 1.1 through 1.5 are done; the registry UI is in flight."}',
   '{"sections": []}', now() - interval '50 days', now() - interval '2 hours'),

  ('repo-vs-mneme-delineation', 'Repo vs Mneme delineation', 'mneme', 'spec', 'complete',
   NULL, 'jeroenpfeil/mneme', '{architecture,docs}', NULL, NULL,
   '{"description": "Decision record for what lives in git versus Mneme: the repo owns durable present-tense docs about the artifact, Mneme owns evolving work docs. Pointers across the line, never copies."}',
   '{"sections": []}', now() - interval '20 days', now() - interval '1 day'),

  ('hyperion-zigbee-migration', 'Zigbee2MQTT migration', 'hyperion', 'plan', 'in-progress',
   'HYP-231', 'jeroenpfeil/hyperion', '{zigbee,mqtt}', 2, 6,
   '{"description": "Move all 40-plus devices off the aging deCONZ stack onto zigbee2mqtt: inventory, coordinator swap, room-by-room re-pairing, automation rewrites, deCONZ decommission."}',
   '{"sections": []}', now() - interval '15 days', now() - interval '3 days'),

  ('hyperion-q2-energy-report', 'Q2 energy consumption report', 'hyperion', 'report', 'complete',
   NULL, NULL, '{energy}', NULL, NULL,
   '{"description": "Quarterly breakdown of household consumption from the P1 meter: baseload crept up 8 percent, dryer dominates peaks, solar self-consumption at 61 percent."}',
   '{"sections": []}', now() - interval '10 days', now() - interval '6 days'),

  ('hyperion-presence-detection-adr', 'Room presence detection approach', 'hyperion', 'adr', 'blocked',
   'HYP-198', 'jeroenpfeil/hyperion', '{presence,architecture}', NULL, NULL,
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
