import { createRouter, createWebHistory } from 'vue-router'

// 动态导入视图组件（代码分包，推荐）
const Home = () => import('../views/index.vue')
const Category = () => import('../views/Category.vue')
const router = createRouter({
  history: createWebHistory(import.meta.env.BASE_URL),
  routes: [
    {
      path: '/',
      name: 'home',
      component: Home,
      meta: { title: '首页' },
    },
    {
      path: '/gallery/:folderKey',
      name: 'category',
      component: Category,
      meta: { title: '分类' },
    },
    {
      path: '/c/:folderKey',
      redirect: (to) => `/gallery/${String(to.params.folderKey ?? '')}`,
    },
    // 可添加 404 重定向等
  ],
})

export default router
