<script setup lang="ts">
// Static product page: what mneme is, why it exists, what it is not, and the
// repo-vs-mneme delineation. Hand-written content that ships with the binary
// — deliberately not rendered from a mneme document, which would make the
// page describing the product depend on this install's data.

const DELINEATION = [
  ['README & contributor docs', 'repo', 'present-tense, describes the artifact as it is'],
  ['Accepted specs & ADRs', 'repo', 'hardened; versioned with the code they describe'],
  ['Active plans & in-flight specs', 'mneme', 'change daily; structured tasks and phases'],
  ['Brainstorms & scratch notes', 'mneme', 'exploratory — most never harden'],
  ['Dev journal & decision log', 'mneme', 'append-only history of the work'],
  ['Snippets, env facts, memory', 'mneme', 'operational knowledge, not documentation'],
] as const
</script>

<template>
  <main class="content" data-test="about">
    <header class="head">
      <p class="mn-label eyebrow">mneme</p>
      <h1 class="mn-h1">About</h1>
    </header>

    <section class="sec" data-test="what">
      <h2 class="mn-h2">What mneme is</h2>
      <p class="mn-body-sm">
        Mneme is a local, single-user knowledge service for AI-assisted development. It gives
        the knowledge <em>around</em> your projects — plans, decisions, journal entries,
        snippets — a durable, structured home on your own machine. Coding agents read and
        write it over MCP while they work; this web UI is the human window onto the same
        data. Mneme was the Greek muse of memory.
      </p>
    </section>

    <section class="sec" data-test="why">
      <h2 class="mn-h2">Why it exists</h2>
      <p class="mn-body-sm">
        Development produces two kinds of writing. One kind describes the artifact — READMEs,
        accepted specs, API docs — and belongs in git, versioned alongside the code it
        describes. The other kind is the thinking around the work: plans, brainstorms,
        running notes, the reasons behind choices. That second kind has never had a good
        home. Committed to the repo it goes stale and buries reviewers; in a note app it
        drifts away from the project; most often it simply evaporates.
      </p>
      <p class="mn-body-sm">
        Coding agents make the loss acute. The agent that planned a feature with you
        yesterday starts today's session remembering none of it — decisions get re-litigated
        and context gets rebuilt by hand, every session. Mneme addresses both at once: the
        working knowledge gets a durable home, and because that home speaks MCP, agents load
        it at session start and keep it current as they work.
      </p>
    </section>

    <section class="sec" data-test="idea">
      <h2 class="mn-h2">How it's meant to work</h2>
      <ul class="ideas mn-body-sm">
        <li>
          <strong>Knowledge lives beside the repo, not in it.</strong> Each mneme project
          shadows a repository; the two link by pointers, never copies.
        </li>
        <li>
          <strong>Agents write, humans read.</strong> Mutations flow through the MCP tools;
          this UI is a read-mostly viewer for the person steering the work.
        </li>
        <li>
          <strong>Sessions start with context.</strong> One call
          (<span class="mn-code-inline">get_context_bundle</span>) hands an agent the active
          plan, recent decisions, journal and memory — no re-explaining.
        </li>
        <li>
          <strong>Documents graduate.</strong> Plans and specs are born in mneme; when one
          hardens into durable documentation it moves to the repo as markdown, leaving a
          pointer behind.
        </li>
        <li>
          <strong>Local-only by design.</strong> Your data sits in a local database and the
          server answers only on this machine. The one optional external flow — Voyage
          embeddings for semantic search — is off until you provide a key.
        </li>
      </ul>
    </section>

    <section class="sec" data-test="not">
      <h2 class="mn-h2">What mneme is not</h2>
      <ul class="ideas mn-body-sm">
        <li>
          <strong>Not a second brain.</strong> Its scope is the development work around your
          repositories, not general-purpose note capture.
        </li>
        <li>
          <strong>Not a team wiki.</strong> Single-user and local by design — no accounts, no
          sharing, no cloud sync.
        </li>
        <li>
          <strong>Not an issue tracker.</strong> Plans carry task lists, but there is no
          backlog, no sprints, no assignees.
        </li>
        <li>
          <strong>Not search over your codebase.</strong> Mneme never ingests repo files;
          code and its documentation are already searchable where they live — in git.
        </li>
        <li>
          <strong>Not documentation hosting.</strong> Durable docs belong in the repo; mneme
          is where they gestate.
        </li>
      </ul>
    </section>

    <section class="sec" data-test="delineation">
      <h2 class="mn-h2">The repo owns the code, mneme owns the work</h2>
      <p class="mn-body-sm">
        The dividing line: git holds durable, present-tense documents about the artifact;
        mneme holds the evolving work around it. Documents are born in mneme and graduate to
        the repo when they harden. Across the line, pointers only — never copies.
      </p>
      <div class="table-wrap">
        <table class="mn-body-sm">
          <thead>
            <tr>
              <th class="mn-label">document</th>
              <th class="mn-label">home</th>
              <th class="mn-label">why</th>
            </tr>
          </thead>
          <tbody>
            <tr v-for="[doc, home, why] in DELINEATION" :key="doc">
              <td>{{ doc }}</td>
              <td class="mn-mono-sm">{{ home }}</td>
              <td>{{ why }}</td>
            </tr>
          </tbody>
        </table>
      </div>
    </section>

    <p class="mn-body-sm outro">
      Ready to wire it into an agent?
      <RouterLink to="/help" class="mn-anchor">Help</RouterLink> has connect commands
      pre-filled for this install, and a one-time prompt for porting an existing workflow.
    </p>
  </main>
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
.head {
  display: flex;
  flex-direction: column;
  gap: var(--space-1);
}
.eyebrow {
  color: var(--text-faint);
}
.sec {
  display: flex;
  flex-direction: column;
  gap: var(--space-3);
}
.sec p {
  color: var(--text-secondary);
}
.ideas {
  margin: 0;
  padding-left: var(--space-5);
  display: flex;
  flex-direction: column;
  gap: var(--space-2);
  color: var(--text-secondary);
}
.ideas li {
  list-style: disc;
}
.ideas strong {
  color: var(--text-primary);
}
/* Two-tone table, same recipe as MTable: shaded header row, lighter body. */
.table-wrap {
  border: 1px solid var(--border-strong);
  border-radius: var(--radius-md);
  background: var(--bg-elevated);
  overflow-x: auto;
}
table {
  width: 100%;
  border-collapse: collapse;
}
th {
  text-align: left;
  padding: var(--space-3) var(--space-4);
  background: var(--bg-surface);
  border-bottom: 1px solid var(--border-strong);
}
td {
  padding: var(--space-3) var(--space-4);
  vertical-align: top;
  color: var(--text-secondary);
}
td:first-child {
  color: var(--text-primary);
  font-weight: 500;
}
tbody tr + tr td {
  border-top: 1px solid var(--border);
}
.outro {
  color: var(--text-muted);
}
</style>
