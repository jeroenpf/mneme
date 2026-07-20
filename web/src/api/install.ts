import { apiGet } from './client'

// Mirrors internal/appinfo.Info — the effective install facts the Help page
// renders (URL, MCP endpoint, storage, embeddings). Never carries secrets.
export interface InstallInfo {
  version: string
  mode: string
  url: string
  mcp_endpoint: string
  db: { driver: string; path: string }
  embeddings: { enabled: boolean; model: string }
}

export function getInstall(): Promise<InstallInfo> {
  return apiGet<InstallInfo>('/api/v1/install')
}
