<script setup lang="ts">
import { computed } from 'vue'
import { useInstallInfo } from '@/composables/useInstallInfo'
import { buildPortPrompt } from '@/content/portPrompt'
import MCode from '@/blocks/MCode.vue'

const { info, loading, error, refresh } = useInstallInfo()

const url = computed(() => info.value?.url ?? '')
const mcp = computed(() => info.value?.mcp_endpoint ?? '')
// Node's npx often resolves `localhost` to IPv6 ::1 first, but a loopback
// server binds IPv4 — so the stdio bridge must target 127.0.0.1.
const mcpBridge = computed(() => mcp.value.replace('//localhost', '//127.0.0.1'))
const isHTTPS = computed(() => info.value?.mode === 'mnemedev')

const searchMode = computed(() =>
  info.value?.embeddings.enabled
    ? `semantic (Voyage ${info.value.embeddings.model})`
    : 'lexical only (FTS, fully local)',
)

const claudeCodeCmd = computed(
  () => `claude mcp add --transport http --scope user mneme ${mcp.value}`,
)
const codexToml = computed(() => `[mcp_servers.mneme]\nurl = "${mcp.value}"`)
const claudeDesktopJson = computed(
  () => `{
  "mcpServers": {
    "mneme": {
      "command": "npx",
      "args": ["-y", "mcp-remote", "${mcpBridge.value}", "--transport", "http-only"]
    }
  }
}`,
)

const settingsToml = `[embeddings]
voyage_api_key = "pa-..."`

const portPrompt = computed(() => buildPortPrompt({ url: url.value, mcpEndpoint: mcp.value }))

const agentsSnippet = `## Mneme (project knowledge)
At session start, call get_context_bundle("<project>"). Before assuming
something is missing, search() first. Record durable decisions with
log_decision, append a dev-journal entry with append_journal, and push
plans/specs with push_document. Prefer the surgical tools (tick_task,
update_task, advance_phase) over rewriting whole documents.`
</script>

<template>
  <div>
    <main class="content" data-test="help">
      <p class="mn-body-sm intro">
        How to reach this Mneme, wire it into your coding agents, turn on semantic search, and
        teach an agent to use it. Everything below reflects <em>this</em> running install.
      </p>

      <p v-if="loading && !info" class="mn-mono-sm py-8 text-center" data-test="loading">loading…</p>

      <div v-else-if="error" class="error mn-body-sm" data-test="error">
        <p>could not load install info: {{ error.message }}</p>
        <button class="retry mn-mono-sm" @click="refresh()">retry</button>
      </div>

      <template v-else-if="info">
        <!-- 1. Access -->
        <section class="sec" data-test="access">
          <h2 class="mn-h2">Access</h2>
          <dl class="facts">
            <div class="fact">
              <dt class="mn-label">open</dt>
              <dd class="mn-mono"><a :href="url" target="_blank" rel="noreferrer">{{ url }}</a></dd>
            </div>
            <div class="fact">
              <dt class="mn-label">MCP endpoint</dt>
              <dd class="mn-mono">{{ mcp }}</dd>
            </div>
            <div class="fact">
              <dt class="mn-label">storage</dt>
              <dd class="mn-mono">{{ info.db.path || info.db.driver }} <span class="muted">({{ info.db.driver }})</span></dd>
            </div>
            <div class="fact">
              <dt class="mn-label">search</dt>
              <dd class="mn-mono">{{ searchMode }}</dd>
            </div>
            <div class="fact">
              <dt class="mn-label">version</dt>
              <dd class="mn-mono">{{ info.version }}</dd>
            </div>
          </dl>
        </section>

        <!-- 2. Connect an agent -->
        <section class="sec" data-test="connect">
          <h2 class="mn-h2">Connect an agent</h2>
          <p class="mn-body-sm">
            Register mneme once per client and it's available in every session. The commands
            below are pre-filled with <em>your</em> endpoint.
          </p>

          <h3 class="mn-h3">Claude Code</h3>
          <p class="mn-body-sm">Global across all projects (drop <span class="mn-code-inline">--scope user</span> for just this one):</p>
          <MCode lang="bash" :content="claudeCodeCmd" />

          <h3 class="mn-h3">Codex</h3>
          <p class="mn-body-sm">
            Add to <span class="mn-code-inline">~/.codex/config.toml</span>. The same global config
            also powers the Codex / ChatGPT <strong>desktop app</strong> (it runs locally, so it
            reaches this server):
          </p>
          <MCode lang="toml" filename="~/.codex/config.toml" :content="codexToml" />

          <h3 class="mn-h3">Claude Desktop</h3>
          <p class="mn-body-sm">
            Claude Desktop speaks stdio only, so bridge it with
            <span class="mn-code-inline">mcp-remote</span> (needs Node). Edit
            <span class="mn-code-inline">claude_desktop_config.json</span>
            (Settings → Developer → Edit Config), then fully quit and reopen the app:
          </p>
          <MCode lang="json" filename="claude_desktop_config.json" :content="claudeDesktopJson" />

          <div v-if="isHTTPS" class="note mn-body-sm" data-test="https-note">
            You're in HTTPS (mneme.dev) mode. For the Claude Desktop bridge only, point
            <span class="mn-code-inline">NODE_EXTRA_CA_CERTS</span> at your mkcert root CA so Node
            trusts the certificate.
          </div>
          <div v-else class="note mn-body-sm" data-test="advanced-note">
            <strong>Advanced (HTTPS):</strong> running in mneme.dev mode instead swaps these URLs for
            <span class="mn-code-inline">https://mneme.dev:PORT/mcp</span>; the Claude Desktop bridge
            then also needs <span class="mn-code-inline">NODE_EXTRA_CA_CERTS</span> pointed at your
            mkcert root CA.
          </div>

          <div class="note note-muted mn-body-sm" data-test="chatgpt-note">
            <strong>ChatGPT on the web</strong> connects from OpenAI's cloud, so it can't reach a
            local server — that would need a public HTTPS tunnel, contrary to mneme's local-only
            design. The Codex/ChatGPT <em>desktop</em> app works via the Codex config above.
          </div>
        </section>

        <!-- 3. Enable semantic search -->
        <section class="sec" data-test="voyage">
          <h2 class="mn-h2">Enable semantic search (Voyage)</h2>
          <p v-if="info.embeddings.enabled" class="mn-body-sm" data-test="voyage-on">
            Semantic search is <strong>on</strong> (Voyage {{ info.embeddings.model }}). Watch the
            index on the <RouterLink to="/embeddings" class="mn-anchor">Search index</RouterLink> page.
          </p>
          <template v-else>
            <p class="mn-body-sm">
              Search is lexical-only right now — everything stays on your machine. To add semantic
              (hybrid) search, provide a <a class="mn-anchor" href="https://voyageai.com" target="_blank" rel="noreferrer">Voyage</a>
              API key one of two ways, then restart <span class="mn-code-inline">mneme server</span>:
            </p>
            <MCode lang="toml" filename="~/.mneme/settings.toml" :content="settingsToml" />
            <p class="mn-body-sm">…or via the environment (overrides the file):</p>
            <MCode lang="bash" content="export MNEME_VOYAGE_API_KEY=pa-..." />
            <p class="mn-body-sm">
              After restart, the <RouterLink to="/embeddings" class="mn-anchor">Search index</RouterLink>
              page shows the backfill progress. The key is stored in plaintext (same trust model as a
              <span class="mn-code-inline">.env</span>) and document text is sent to Voyage to compute
              vectors — the one external flow, and it's opt-in.
            </p>
          </template>
        </section>

        <!-- 4. Teach your agent -->
        <section class="sec" data-test="teach">
          <h2 class="mn-h2">Teach your agent to use mneme</h2>
          <p class="mn-body-sm">
            Drop this into your project's <span class="mn-code-inline">AGENTS.md</span> or
            <span class="mn-code-inline">CLAUDE.md</span> so the agent reaches for mneme by habit:
          </p>
          <MCode lang="markdown" filename="AGENTS.md / CLAUDE.md" :content="agentsSnippet" />
          <p class="mn-body-sm">
            Or just tell it in the moment — for example:
          </p>
          <ul class="prompts mn-body-sm">
            <li>“Start by calling <span class="mn-code-inline">get_context_bundle</span> for this project, then summarize where we left off.”</li>
            <li>“Log that decision in mneme with <span class="mn-code-inline">log_decision</span>, and append a journal entry when we're done.”</li>
            <li>“Update my <span class="mn-code-inline">AGENTS.md</span> so future sessions always check mneme first and push plans there.”</li>
          </ul>

          <div data-test="port">
            <h3 class="mn-h3">Port an existing workflow</h3>
            <p class="mn-body-sm">
              Already planning through superpowers, a <span class="mn-code-inline">docs/</span>
              folder, Notion, or another system? This one-time prompt briefs an agent on what
              mneme is, has it inventory the repo's current workflow, rewrite the instruction
              files, and propose a migration it may only perform after you confirm. Run it once,
              inside the target repo — it is pre-filled with <em>your</em> endpoint:
            </p>
            <MCode lang="markdown" filename="port-to-mneme.md" :content="portPrompt" />
          </div>
        </section>
      </template>
    </main>
  </div>
</template>

<style scoped>
.content {
  max-width: var(--content-max);
  width: 100%;
  margin: 0 auto;
  padding: var(--space-8) var(--space-6);
  display: flex;
  flex-direction: column;
  gap: var(--space-6);
  min-width: 0;
}
.intro {
  color: var(--text-muted);
  margin-top: calc(-1 * var(--space-2));
}
.sec {
  display: flex;
  flex-direction: column;
  gap: var(--space-3);
}
.sec .mn-h3 {
  margin-top: var(--space-2);
}
.facts {
  display: flex;
  flex-direction: column;
  gap: var(--space-2);
  margin: 0;
}
.fact {
  display: grid;
  grid-template-columns: minmax(120px, 160px) minmax(0, 1fr);
  gap: var(--space-3);
  align-items: baseline;
}
.fact dt {
  color: var(--text-faint);
}
.fact dd {
  margin: 0;
  color: var(--text-primary);
  overflow-wrap: anywhere;
}
.fact .muted {
  color: var(--text-faint);
}
.note {
  border: 1px solid var(--border);
  background: var(--bg-surface);
  border-radius: var(--radius-md);
  padding: var(--space-3) var(--space-4);
  color: var(--text-secondary);
}
.note-muted {
  color: var(--text-muted);
}
.prompts {
  margin: 0;
  padding-left: var(--space-5);
  display: flex;
  flex-direction: column;
  gap: var(--space-2);
  color: var(--text-secondary);
}
.prompts li {
  list-style: disc;
}
.error {
  border: 1px solid var(--red-border);
  background: var(--red-dim);
  border-radius: var(--radius-md);
  padding: var(--space-4);
  color: var(--text-secondary);
}
.retry {
  margin-top: var(--space-2);
  padding: 4px 10px;
  border: 1px solid var(--border-strong);
  border-radius: var(--radius-sm);
  background: var(--bg-elevated);
  color: var(--text-primary);
  cursor: pointer;
}
</style>
