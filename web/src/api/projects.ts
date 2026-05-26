import { apiGet } from './client'
import type { ProjectStats } from '@/types'

interface ProjectsResponse {
  items: ProjectStats[]
}

export function listProjects(): Promise<ProjectStats[]> {
  return apiGet<ProjectsResponse>('/api/v1/projects').then((r) => r.items)
}
