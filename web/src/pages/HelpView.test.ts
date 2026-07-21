import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { flushPromises, mount, RouterLinkStub } from '@vue/test-utils'
import { getInstall, type InstallInfo } from '@/api/install'
import MCode from '@/blocks/MCode.vue'
import HelpView from './HelpView.vue'

vi.mock('@/api/install', () => ({ getInstall: vi.fn() }))

const sample: InstallInfo = {
  version: '0.1.1',
  mode: 'localhost',
  url: 'http://localhost:8901',
  mcp_endpoint: 'http://localhost:8901/mcp',
  db: { driver: 'sqlite', path: '/home/u/.mneme/mneme.db' },
  embeddings: { enabled: false, model: '' },
}

async function mountView() {
  const w = mount(HelpView, {
    attachTo: document.body,
    global: { stubs: { RouterLink: RouterLinkStub } },
  })
  await flushPromises()
  return w
}

const codeContents = (w: ReturnType<typeof mount>) =>
  w.findAllComponents(MCode).map((c) => c.props('content') as string)

beforeEach(() => {
  vi.mocked(getInstall).mockReset().mockResolvedValue(sample)
})

afterEach(() => {
  document.body.innerHTML = ''
})

describe('HelpView', () => {
  it('renders the four sections', async () => {
    const w = await mountView()
    for (const s of ['access', 'connect', 'voyage', 'teach']) {
      expect(w.find(`[data-test="${s}"]`).exists()).toBe(true)
    }
  })

  it('shows live install values in Access', async () => {
    const w = await mountView()
    const access = w.get('[data-test="access"]').text()
    expect(access).toContain('http://localhost:8901/mcp')
    expect(access).toContain('/home/u/.mneme/mneme.db')
    expect(access).toContain('lexical')
  })

  it('pre-fills connect commands with the live MCP endpoint', async () => {
    const w = await mountView()
    const codes = codeContents(w)
    expect(codes.some((c) => c.includes('claude mcp add') && c.includes('http://localhost:8901/mcp'))).toBe(true)
    expect(codes.some((c) => c.includes('[mcp_servers.mneme]') && c.includes('http://localhost:8901/mcp'))).toBe(true)
    // Claude Desktop bridge must use the 127.0.0.1 variant, not localhost.
    expect(codes.some((c) => c.includes('mcp-remote') && c.includes('http://127.0.0.1:8901/mcp'))).toBe(true)
  })

  it('teaches the agent with a get_context_bundle snippet', async () => {
    const w = await mountView()
    expect(codeContents(w).some((c) => c.includes('get_context_bundle'))).toBe(true)
  })

  it('offers the workflow-port prompt, pre-filled with the live endpoints', async () => {
    const w = await mountView()
    expect(w.find('[data-test="port"]').exists()).toBe(true)
    const port = codeContents(w).find((c) => c.includes('inventory the current workflow'))
    expect(port).toBeDefined()
    expect(port).toContain('http://localhost:8901/mcp')
    expect(port).toContain('http://localhost:8901/help')
    // The migration step must stay gated on the user.
    expect(port).toContain('wait for my explicit confirmation')
  })

  it('shows how to enable Voyage when embeddings are off', async () => {
    const w = await mountView()
    expect(codeContents(w).some((c) => c.includes('MNEME_VOYAGE_API_KEY'))).toBe(true)
  })

  it('reflects embeddings-on state', async () => {
    vi.mocked(getInstall).mockResolvedValue({
      ...sample,
      embeddings: { enabled: true, model: 'voyage-4-large' },
    })
    const w = await mountView()
    expect(w.find('[data-test="voyage-on"]').exists()).toBe(true)
    expect(w.get('[data-test="voyage-on"]').text()).toContain('voyage-4-large')
  })
})
