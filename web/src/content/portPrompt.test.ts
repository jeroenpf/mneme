import { describe, expect, it } from 'vitest'
import { buildPortPrompt } from './portPrompt'

const prompt = buildPortPrompt({
  url: 'http://localhost:8901',
  mcpEndpoint: 'http://localhost:8901/mcp',
})
// The prompt is hard-wrapped for readability; assert against a flat form so
// tests don't break when a phrase happens to straddle a line break.
const flat = prompt.replace(/\s+/g, ' ')

describe('buildPortPrompt', () => {
  it('interpolates the live install endpoints', () => {
    expect(flat).toContain('http://localhost:8901/mcp')
    expect(flat).toContain('http://localhost:8901/help')
  })

  it('briefs the agent on the core verbs', () => {
    for (const tool of [
      'get_context_bundle',
      'search',
      'push_document',
      'log_decision',
      'append_journal',
      'create_project',
      'tick_task',
    ]) {
      expect(flat, `should mention ${tool}`).toContain(tool)
    }
  })

  it('states the delineation rule', () => {
    expect(flat).toContain('graduate')
    expect(flat).toContain('never copies')
    expect(flat).toContain('Never bulk-import repo files')
  })

  it('covers the instruction files to rewrite', () => {
    expect(flat).toContain('CLAUDE.md')
    expect(flat).toContain('AGENTS.md')
  })

  it('hard-gates the migration on user confirmation', () => {
    expect(flat).toContain('wait for my explicit confirmation')
    expect(flat).toContain('Do not migrate anything before I confirm')
    expect(flat).toContain('delete nothing without separate, explicit approval')
  })
})
