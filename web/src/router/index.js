import { createRouter, createWebHashHistory } from 'vue-router'

const routes = [
  {
    path: '/',
    redirect: '/login'
  },
  {
    path: '/init',
    name: 'Init',
    component: () => import('@/view/init/index.vue')
  },
  {
    path: '/login',
    name: 'Login',
    component: () => import('@/view/login/index.vue')
  },
  {
    path: '/scanUpload',
    name: 'ScanUpload',
    meta: {
      title: '扫码上传',
      client: true
    },
    component: () => import('@/view/media/scanUpload.vue')
  },
  {
    path: '/forceChangePassword',
    name: 'ForceChangePassword',
    component: () => import('@/view/system/security/forceChangePassword.vue'),
    meta: { title: '修改密码' }
  },
  {
    // 进销存工作台：独立壳（自有导航），不走后台菜单表。
    // 路由是常量路由，接口仍由 JWT + Casbin 逐条校验（docs/sql/jxc-casbin.sql）。
    path: '/jxc',
    component: () => import('@/view/jxc/shell.vue'),
    redirect: '/jxc/workspace',
    children: [
      { path: 'workspace', component: () => import('@/view/jxc/workspace/index.vue') },
      { path: 'sale', component: () => import('@/view/jxc/sale/index.vue') },
      { path: 'purchase', component: () => import('@/view/jxc/purchase/index.vue') },
      { path: 'inventory', component: () => import('@/view/jxc/inventory/index.vue') },
      { path: 'master', component: () => import('@/view/jxc/master/index.vue') }
    ]
  },
  {
    path: '/:catchAll(.*)',
    meta: {
      closeTab: true
    },
    component: () => import('@/view/error/index.vue')
  },
]

const router = createRouter({
  history: createWebHashHistory(),
  routes
})

export default router
