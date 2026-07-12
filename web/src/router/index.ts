import { createRouter, createWebHistory, type RouteRecordRaw } from 'vue-router'

const routes: RouteRecordRaw[] = [
  {
    path: '/',
    name: 'registry',
    component: () => import('@/pages/RegistryView.vue'),
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
]

export const router = createRouter({
  history: createWebHistory(),
  routes,
})
