import { beforeEach, describe, expect, it, vi } from 'vitest'
import { flushPromises } from '@vue/test-utils'
import type { InstallInfo } from '@/api/install'
import { getInstall } from '@/api/install'
import { useInstallInfo } from './useInstallInfo'

vi.mock('@/api/install', () => ({ getInstall: vi.fn() }))

const sample: InstallInfo = {
  version: '0.1.1',
  mode: 'localhost',
  url: 'http://localhost:8901',
  mcp_endpoint: 'http://localhost:8901/mcp',
  db: { driver: 'sqlite', path: '/home/u/.mneme/mneme.db' },
  embeddings: { enabled: false, model: '' },
}

beforeEach(() => {
  vi.mocked(getInstall).mockReset().mockResolvedValue(sample)
})

describe('useInstallInfo', () => {
  it('loads install info immediately', async () => {
    const { info, loading } = useInstallInfo()
    await flushPromises()
    expect(vi.mocked(getInstall)).toHaveBeenCalled()
    expect(info.value?.mcp_endpoint).toBe('http://localhost:8901/mcp')
    expect(loading.value).toBe(false)
  })

  it('captures a load error and leaves info null', async () => {
    vi.mocked(getInstall).mockReset().mockRejectedValueOnce(new Error('boom'))
    const { info, error } = useInstallInfo()
    await flushPromises()
    expect(info.value).toBeNull()
    expect(error.value?.message).toBe('boom')
  })
})
