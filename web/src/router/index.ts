import { createRouter, createWebHistory, type RouteRecordRaw } from 'vue-router'

const routes: RouteRecordRaw[] = [
  {
    path: '/',
    name: 'registry',
    component: () => import('@/pages/RegistryView.vue'),
  },
  {
    path: '/project/:slug',
    name: 'project',
    component: () => import('@/pages/ProjectHome.vue'),
    props: true,
  },
  {
    path: '/doc/:id',
    name: 'document',
    component: () => import('@/pages/DocumentView.vue'),
    props: true,
  },
  {
    path: '/memory',
    name: 'memory',
    component: () => import('@/pages/MemoryView.vue'),
  },
  {
    path: '/env',
    name: 'env',
    component: () => import('@/pages/EnvView.vue'),
  },
  {
    path: '/decisions',
    name: 'decisions',
    component: () => import('@/pages/DecisionsView.vue'),
  },
  {
    path: '/snippets',
    name: 'snippets',
    component: () => import('@/pages/SnippetsView.vue'),
  },
  {
    path: '/journal',
    name: 'journal',
    component: () => import('@/pages/JournalView.vue'),
  },
  {
    path: '/solutions',
    name: 'solutions',
    component: () => import('@/pages/SolutionsView.vue'),
  },
  {
    path: '/bundle',
    name: 'bundle',
    component: () => import('@/pages/BundleView.vue'),
  },
  {
    path: '/search',
    name: 'search',
    component: () => import('@/pages/SearchView.vue'),
  },
  {
    path: '/embeddings',
    name: 'embeddings',
    component: () => import('@/pages/EmbeddingsView.vue'),
  },
  {
    path: '/help',
    name: 'help',
    component: () => import('@/pages/HelpView.vue'),
  },
]

export const router = createRouter({
  history: createWebHistory(),
  routes,
})
