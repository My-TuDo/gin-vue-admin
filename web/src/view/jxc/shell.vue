<template>
  <div class="jxc-shell">
    <!-- 左侧导航：按「今天要干什么」组织，不按数据表组织 -->
    <aside class="rail">
      <div class="brand">
        <span class="logo">进</span>
        <span class="name">进销存</span>
      </div>
      <nav>
        <router-link v-for="item in nav" :key="item.path" :to="item.path" class="nav-item"
          :class="{ active: isActive(item.path) }">
          <span class="icon">{{ item.icon }}</span>
          <span>{{ item.label }}</span>
        </router-link>
      </nav>
      <div class="rail-foot">
        <button class="link" @click="backToAdmin">返回管理后台</button>
      </div>
    </aside>

    <div class="main">
      <header class="topbar">
        <h1>{{ currentTitle }}</h1>
        <div class="spacer"></div>
        <!-- 对账状态常驻：账不平是大事，不该藏在二级页面 -->
        <button class="recon" :class="reconState" @click="$router.push('/jxc/inventory')">
          {{ reconText }}
        </button>
        <span class="now">{{ clock }}</span>
      </header>
      <main class="content">
        <router-view />
      </main>
    </div>
  </div>
</template>

<script setup>
import { computed, onMounted, onUnmounted, ref } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { ledgerReconcile } from '@/api/jxc'

const route = useRoute()
const router = useRouter()

const nav = [
  { path: '/jxc/workspace', label: '工作台', icon: '◱' },
  { path: '/jxc/sale', label: '开单', icon: '◈' },
  { path: '/jxc/purchase', label: '补货', icon: '▤' },
  { path: '/jxc/inventory', label: '库存', icon: '▦' },
  { path: '/jxc/master', label: '基础资料', icon: '⌗' }
]

const isActive = (p) => route.path.startsWith(p)
const currentTitle = computed(() => nav.find((n) => isActive(n.path))?.label ?? '进销存')

const reconState = ref('unknown')
const reconText = computed(() => {
  if (reconState.value === 'ok') return '✅ 账实相符'
  if (reconState.value === 'bad') return '⚠️ 对账异常'
  if (reconState.value === 'error') return '对账不可用'
  return '对账检查中…'
})

const clock = ref('')
let timer = null
const tick = () => {
  const d = new Date()
  const p = (n) => String(n).padStart(2, '0')
  clock.value = `${d.getMonth() + 1}/${d.getDate()} ${p(d.getHours())}:${p(d.getMinutes())}`
}

const checkReconcile = async () => {
  try {
    const res = await ledgerReconcile({})
    if (res.code === 0) reconState.value = res.data?.ok ? 'ok' : 'bad'
    else reconState.value = 'error'
  } catch {
    reconState.value = 'error'
  }
}

const backToAdmin = () => router.push('/layout/dashboard')

onMounted(() => {
  tick()
  timer = setInterval(tick, 30000)
  checkReconcile()
})
onUnmounted(() => clearInterval(timer))
</script>

<style scoped>
.jxc-shell {
  display: flex;
  height: 100vh;
  background: var(--el-bg-color-page);
  font-size: 13px;
}
.rail {
  width: 168px;
  flex: 0 0 168px;
  background: var(--el-bg-color);
  border-right: 1px solid var(--el-border-color-lighter);
  display: flex;
  flex-direction: column;
}
.brand {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 16px 16px 20px;
}
.logo {
  width: 24px;
  height: 24px;
  border-radius: 6px;
  background: var(--el-color-primary);
  color: #fff;
  display: grid;
  place-items: center;
  font-size: 13px;
  font-weight: 600;
}
.name {
  font-weight: 600;
  font-size: 14px;
}
nav {
  display: flex;
  flex-direction: column;
  gap: 2px;
  padding: 0 8px;
}
.nav-item {
  display: flex;
  align-items: center;
  gap: 10px;
  padding: 9px 10px;
  border-radius: 6px;
  color: var(--el-text-color-regular);
  text-decoration: none;
}
.nav-item:hover {
  background: var(--el-fill-color-light);
}
.nav-item.active {
  background: var(--el-color-primary-light-9);
  color: var(--el-color-primary);
  font-weight: 500;
}
.icon {
  width: 16px;
  text-align: center;
  opacity: 0.8;
}
.rail-foot {
  margin-top: auto;
  padding: 12px 16px;
  border-top: 1px solid var(--el-border-color-lighter);
}
.link {
  background: none;
  border: 0;
  color: var(--el-text-color-secondary);
  cursor: pointer;
  padding: 0;
  font-size: 12px;
}
.link:hover {
  color: var(--el-color-primary);
}
.main {
  flex: 1;
  min-width: 0;
  display: flex;
  flex-direction: column;
}
.topbar {
  height: 48px;
  display: flex;
  align-items: center;
  gap: 12px;
  padding: 0 20px;
  background: var(--el-bg-color);
  border-bottom: 1px solid var(--el-border-color-lighter);
}
.topbar h1 {
  font-size: 14px;
  font-weight: 600;
  margin: 0;
}
.spacer {
  flex: 1;
}
.recon {
  border: 1px solid var(--el-border-color-lighter);
  background: var(--el-bg-color);
  border-radius: 14px;
  padding: 3px 10px;
  font-size: 12px;
  cursor: pointer;
}
.recon.ok {
  color: var(--el-color-success);
  border-color: var(--el-color-success-light-7);
}
.recon.bad {
  color: var(--el-color-danger);
  border-color: var(--el-color-danger-light-7);
}
.now {
  color: var(--el-text-color-secondary);
  font-variant-numeric: tabular-nums;
}
.content {
  flex: 1;
  min-height: 0;
  padding: 16px 20px;
  overflow: auto;
}
</style>
