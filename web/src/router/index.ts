import { createRouter, createWebHistory } from 'vue-router'
import type { RouteRecordRaw } from 'vue-router'

const routes: RouteRecordRaw[] = [
  {
    path: '/',
    component: () => import('@/components/layout/MainLayout.vue'),
    children: [
      {
        path: '',
        name: 'dashboard',
        component: () => import('@/views/Dashboard.vue'),
        meta: { title: 'Dashboard' },
      },
      {
        path: 'workflows',
        name: 'workflows',
        component: () => import('@/views/WorkflowList.vue'),
        meta: { title: 'Workflows' },
      },
      {
        path: 'workflows/new',
        name: 'workflow-new',
        component: () => import('@/views/WorkflowEditor.vue'),
        meta: { title: 'New Workflow' },
      },
      {
        path: 'workflows/:id',
        name: 'workflow-editor',
        component: () => import('@/views/WorkflowEditor.vue'),
        meta: { title: 'Edit Workflow' },
        props: true,
      },
      {
        path: 'executions',
        name: 'executions',
        component: () => import('@/views/ExecutionHistory.vue'),
        meta: { title: 'Executions' },
      },
      {
        path: 'executions/:id',
        name: 'execution-detail',
        component: () => import('@/views/ExecutionDetail.vue'),
        meta: { title: 'Execution Detail' },
        props: true,
      },
      {
        path: 'credentials',
        name: 'credentials',
        component: () => import('@/views/Credentials.vue'),
        meta: { title: 'Credentials' },
      },
      {
        path: 'settings',
        name: 'settings',
        component: () => import('@/views/Settings.vue'),
        meta: { title: 'Settings' },
      },
    ],
  },
  {
    path: '/login',
    name: 'login',
    component: () => import('@/views/Login.vue'),
    meta: { title: 'Login', public: true },
  },
  {
    path: '/:pathMatch(.*)*',
    name: 'not-found',
    component: () => import('@/views/NotFound.vue'),
    meta: { title: 'Not Found' },
  },
]

const router = createRouter({
  history: createWebHistory(),
  routes,
})

// Update document title on navigation
router.beforeEach((to, _from, next) => {
  const title = to.meta.title as string | undefined
  document.title = title ? `${title} - n8n-go` : 'n8n-go'
  next()
})

export default router
