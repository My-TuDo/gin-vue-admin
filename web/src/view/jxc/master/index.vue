<template>
  <div class="flex h-full min-h-0 flex-col gap-4">
    <PageHeader
      eyebrow="MASTER DATA"
      title="基础资料"
      description="商品、规格、往来单位与仓库。历史单据引用主数据，所以只能停用，不能删除。"
    />

    <div class="flex flex-wrap items-center gap-1">
      <button
        v-for="item in TABS"
        :key="item.key"
        type="button"
        class="flex h-8 items-center gap-1.5 rounded-md px-3 text-[12.5px] transition-colors"
        :class="tab === item.key ? 'bg-primary-50 font-medium text-primary-700' : 'text-muted-foreground hover:bg-muted hover:text-base-text'"
        @click="tab = item.key"
      >
        <svg-icon :icon="item.icon" class="h-3.5 w-3.5" />
        {{ item.label }}
        <span class="rounded bg-muted px-1.5 py-0.5 text-[10px] tabular-nums" :class="tab === item.key ? 'bg-primary-100 text-primary-700' : ''">{{ item.count() }}</span>
      </button>

      <div class="ml-auto">
        <el-button v-if="tab === 'goods'" type="primary" @click="openGoods()">
          <svg-icon icon="lucide:plus" class="mr-1.5 h-3.5 w-3.5" />新建商品
        </el-button>
        <el-button v-else-if="tab === 'partner'" type="primary" @click="openPartner()">
          <svg-icon icon="lucide:plus" class="mr-1.5 h-3.5 w-3.5" />新建{{ partnerKind === 'supplier' ? '供应商' : '客户' }}
        </el-button>
        <el-button v-else type="primary" @click="openWarehouse()">
          <svg-icon icon="lucide:plus" class="mr-1.5 h-3.5 w-3.5" />新建仓库
        </el-button>
      </div>
    </div>

    <!-- 商品：左列表 + 右详情，避免一表一弹窗的来回跳 -->
    <div v-if="tab === 'goods'" class="grid min-h-0 flex-1 grid-cols-1 gap-4 xl:grid-cols-[360px_minmax(0,1fr)]">
      <SectionCard title="商品" :description="`${filteredGoods.length} 个`" :padded="false" class="min-h-0">
        <div class="flex h-full min-h-0 flex-col">
          <div class="border-b border-border px-3 py-2.5">
            <el-input v-model="goodsKeyword" placeholder="搜索商品名 / 编码" clearable>
              <template #prefix><svg-icon icon="lucide:search" class="h-3.5 w-3.5" /></template>
            </el-input>
          </div>
          <div v-loading="loadingGoods" class="min-h-0 flex-1 overflow-y-auto">
            <button
              v-for="item in filteredGoods"
              :key="item.ID"
              type="button"
              class="flex w-full items-center gap-3 border-b border-border px-4 py-3 text-left transition-colors last:border-b-0"
              :class="selectedGoodsId === item.ID ? 'bg-primary-50' : 'hover:bg-muted'"
              @click="selectedGoodsId = item.ID"
            >
              <span class="min-w-0 flex-1">
                <span class="flex items-center gap-2">
                  <span class="truncate text-[12.5px] font-medium" :class="selectedGoodsId === item.ID ? 'text-primary-700' : 'text-base-text'">{{ item.name }}</span>
                  <StatusTag kind="active" :status="item.status" size="xs" />
                </span>
                <span class="mt-0.5 flex items-center gap-2 text-[11px] text-muted-foreground">
                  <span class="font-mono">{{ item.code }}</span>
                  <span>{{ item.skus?.length || 0 }} 个规格</span>
                </span>
              </span>
              <svg-icon icon="lucide:chevron-right" class="h-3.5 w-3.5 shrink-0" :class="selectedGoodsId === item.ID ? 'text-primary-500' : 'text-muted-foreground'" />
            </button>
            <EmptyState
              v-if="!loadingGoods && filteredGoods.length === 0"
              icon="lucide:package-search"
              title="没有商品"
              :description="goodsKeyword ? '换个商品名或编码试试。' : '先新建一个商品，再添加规格。'"
            />
          </div>
        </div>
      </SectionCard>

      <SectionCard v-if="selectedGoods" :padded="true" class="min-h-0">
        <template #header>
          <div class="flex min-w-0 items-center gap-2">
            <h2 class="truncate text-[13px] font-semibold">{{ selectedGoods.name }}</h2>
            <span class="font-mono text-[11px] text-muted-foreground">{{ selectedGoods.code }}</span>
            <StatusTag kind="active" :status="selectedGoods.status" size="xs" />
          </div>
          <div class="ml-auto flex items-center gap-2">
            <el-button size="small" @click="openGoods(selectedGoods)">编辑</el-button>
            <el-button size="small" :type="selectedGoods.status === 1 ? 'danger' : 'success'" plain @click="toggleGoods(selectedGoods)">
              {{ selectedGoods.status === 1 ? '停用' : '启用' }}
            </el-button>
          </div>
        </template>

        <div class="grid grid-cols-4 gap-px overflow-hidden rounded-lg border border-border bg-border">
          <div class="bg-container px-3 py-2.5">
            <div class="text-[10.5px] text-muted-foreground">规格数</div>
            <div class="mt-0.5 text-[16px] font-semibold tabular-nums">{{ selectedGoods.skus?.length || 0 }}</div>
          </div>
          <div class="bg-container px-3 py-2.5">
            <div class="text-[10.5px] text-muted-foreground">启用规格</div>
            <div class="mt-0.5 text-[16px] font-semibold tabular-nums">{{ enabledSkuCount }}</div>
          </div>
          <div class="bg-container px-3 py-2.5">
            <div class="text-[10.5px] text-muted-foreground">售价区间</div>
            <div class="mt-1 text-[12px] font-medium tabular-nums">{{ priceRange(selectedGoods) }}</div>
          </div>
          <div class="bg-container px-3 py-2.5">
            <div class="text-[10.5px] text-muted-foreground">参考进价区间</div>
            <div class="mt-1 text-[12px] font-medium tabular-nums">{{ costRange(selectedGoods) }}</div>
          </div>
        </div>

        <div class="mt-4 flex items-center justify-between">
          <h3 class="text-[12.5px] font-semibold">规格与价格</h3>
          <span class="text-[11px] text-muted-foreground">停用规格后不能再开单，但历史单据仍可查</span>
        </div>
        <el-table :data="selectedGoods.skus || []" class="mt-2" size="small" max-height="420">
          <el-table-column label="规格" width="130">
            <template #default="{ row }">
              <span class="font-medium">{{ row.color || '—' }}/{{ row.size || '—' }}</span>
            </template>
          </el-table-column>
          <el-table-column label="条码" width="170">
            <template #default="{ row }"><span class="font-mono text-[11.5px]">{{ row.skuCode }}</span></template>
          </el-table-column>
          <el-table-column label="售价" width="110" align="right">
            <template #default="{ row }"><MoneyText :value="row.salePrice" size="sm" /></template>
          </el-table-column>
          <el-table-column label="参考进价" width="110" align="right">
            <template #default="{ row }"><MoneyText :value="row.refCost" size="sm" /></template>
          </el-table-column>
          <el-table-column label="状态" width="90">
            <template #default="{ row }"><StatusTag kind="active" :status="row.status" size="xs" /></template>
          </el-table-column>
          <template #empty>
            <EmptyState icon="lucide:boxes" title="还没有规格" description="编辑商品，添加颜色 / 尺码与条码后才能开单。" />
          </template>
        </el-table>
      </SectionCard>

      <SectionCard v-else :padded="true" class="min-h-0">
        <EmptyState
          icon="lucide:mouse-pointer-click"
          title="选择一个商品"
          description="左侧选中商品后，这里显示规格、售价与启用状态。"
        />
      </SectionCard>
    </div>

    <!-- 往来单位 -->
    <SectionCard v-else-if="tab === 'partner'" :padded="false" class="min-h-0 flex-1">
      <div class="flex h-full min-h-0 flex-col">
        <div class="flex flex-wrap items-center gap-2 border-b border-border px-4 py-2.5">
          <div class="flex items-center gap-1 rounded-md border border-border bg-muted p-0.5">
            <button
              v-for="kind in PARTNER_KINDS"
              :key="kind.key"
              type="button"
              class="h-7 rounded px-2.5 text-[12px] transition-colors"
              :class="partnerKind === kind.key ? 'bg-container font-medium text-base-text shadow-sm' : 'text-muted-foreground hover:text-base-text'"
              @click="switchPartnerKind(kind.key)"
            >{{ kind.label }}</button>
          </div>
          <div class="w-[260px] shrink-0">
            <el-input v-model="partnerKeyword" class="w-full" placeholder="搜索编码 / 名称 / 联系人" clearable>
              <template #prefix><svg-icon icon="lucide:search" class="h-3.5 w-3.5" /></template>
            </el-input>
          </div>
          <span class="ml-auto text-[11px] text-muted-foreground">共 {{ filteredPartners.length }} 个</span>
        </div>
        <div v-loading="loadingPartner" class="min-h-0 flex-1 overflow-hidden">
          <el-table :data="filteredPartners" height="100%">
            <el-table-column prop="code" label="编码" width="150">
              <template #default="{ row }"><span class="font-mono text-[12px]">{{ row.code }}</span></template>
            </el-table-column>
            <el-table-column prop="name" label="名称" min-width="170" />
            <el-table-column prop="contact" label="联系人" width="110" />
            <el-table-column prop="phone" label="电话" width="140" />
            <el-table-column prop="address" label="地址" min-width="200" show-overflow-tooltip />
            <el-table-column label="状态" width="90">
              <template #default="{ row }"><StatusTag kind="active" :status="row.status" size="sm" /></template>
            </el-table-column>
            <el-table-column label="操作" width="130" fixed="right">
              <template #default="{ row }">
                <el-button link type="primary" @click="openPartner(row)">编辑</el-button>
                <el-button link :type="row.status === 1 ? 'danger' : 'success'" @click="togglePartner(row)">
                  {{ row.status === 1 ? '停用' : '启用' }}
                </el-button>
              </template>
            </el-table-column>
            <template #empty>
              <EmptyState icon="lucide:users" title="还没有往来单位" description="新建供应商或客户，采购单可以关联供应商。" />
            </template>
          </el-table>
        </div>
      </div>
    </SectionCard>

    <!-- 仓库 -->
    <SectionCard v-else :padded="false" class="min-h-0 flex-1">
      <div class="flex h-full min-h-0 flex-col">
        <div class="flex flex-wrap items-center gap-2 border-b border-border px-4 py-2.5">
          <div class="w-[260px] shrink-0">
            <el-input v-model="warehouseKeyword" class="w-full" placeholder="搜索仓库编码 / 名称" clearable>
              <template #prefix><svg-icon icon="lucide:search" class="h-3.5 w-3.5" /></template>
            </el-input>
          </div>
          <span class="ml-auto text-[11px] text-muted-foreground">共 {{ filteredWarehouses.length }} 个</span>
        </div>
        <div v-loading="loadingWarehouse" class="min-h-0 flex-1 overflow-hidden">
          <el-table :data="filteredWarehouses" height="100%">
            <el-table-column prop="code" label="编码" width="180">
              <template #default="{ row }"><span class="font-mono text-[12px]">{{ row.code }}</span></template>
            </el-table-column>
            <el-table-column prop="name" label="名称" min-width="220" />
            <el-table-column label="状态" width="100">
              <template #default="{ row }"><StatusTag kind="active" :status="row.status" size="sm" /></template>
            </el-table-column>
            <el-table-column label="创建时间" width="160">
              <template #default="{ row }"><span class="text-[12px] tabular-nums text-muted-foreground">{{ formatDateTime(row.CreatedAt) }}</span></template>
            </el-table-column>
            <el-table-column label="操作" width="140" fixed="right">
              <template #default="{ row }">
                <el-button link type="primary" @click="openWarehouse(row)">编辑</el-button>
                <el-button link :type="row.status === 1 ? 'danger' : 'success'" @click="toggleWarehouse(row)">
                  {{ row.status === 1 ? '停用' : '启用' }}
                </el-button>
              </template>
            </el-table-column>
            <template #empty>
              <EmptyState icon="lucide:warehouse" title="还没有仓库" description="至少建一个仓库，才能开单与补货。" />
            </template>
          </el-table>
        </div>
      </div>
    </SectionCard>

    <!-- 商品编辑 -->
    <el-dialog v-model="goodsDialog" :title="editingGoodsId ? '编辑商品' : '新建商品'" width="900px">
      <div class="grid grid-cols-3 gap-x-4">
        <label class="block">
          <span class="mb-1.5 block text-[11px] text-muted-foreground">商品编码</span>
          <el-input v-model="goodsForm.code" placeholder="如 WY-001" />
        </label>
        <label class="block">
          <span class="mb-1.5 block text-[11px] text-muted-foreground">商品名称</span>
          <el-input v-model="goodsForm.name" placeholder="如 连帽卫衣" />
        </label>
        <label class="block">
          <span class="mb-1.5 block text-[11px] text-muted-foreground">商品状态</span>
          <div class="flex h-8 items-center">
            <el-switch v-model="goodsForm.status" :active-value="1" :inactive-value="0" active-text="启用" inactive-text="停用" />
          </div>
        </label>
      </div>

      <div class="mt-5 flex items-center justify-between">
        <div class="flex items-center gap-2">
          <h3 class="text-[12.5px] font-semibold">规格</h3>
          <span class="text-[11px] text-muted-foreground">已有规格不可删除，只能停用</span>
        </div>
        <el-button size="small" @click="addSku">
          <svg-icon icon="lucide:plus" class="mr-1 h-3 w-3" />添加规格
        </el-button>
      </div>

      <div class="mt-2 overflow-hidden rounded-lg border border-border">
        <div class="grid grid-cols-[92px_82px_minmax(0,1fr)_112px_112px_54px_30px] items-center gap-2 border-b border-border bg-muted px-3 py-2 text-[11px] text-muted-foreground">
          <span>颜色</span><span>尺码</span><span>条码</span><span>售价</span><span>参考进价</span><span>启用</span><span></span>
        </div>
        <div
          v-for="(sku, index) in goodsForm.skus"
          :key="sku.ID || `new-${index}`"
          class="grid grid-cols-[92px_82px_minmax(0,1fr)_112px_112px_54px_30px] items-center gap-2 border-b border-border px-3 py-2 last:border-b-0"
        >
          <el-input v-model="sku.color" placeholder="黑" />
          <el-input v-model="sku.size" placeholder="M" />
          <el-input v-model="sku.skuCode" placeholder="唯一条码" />
          <el-input-number v-model="sku.salePrice" :min="0" :precision="2" :controls="false" class="w-full" />
          <el-input-number v-model="sku.refCost" :min="0" :precision="2" :controls="false" class="w-full" />
          <el-switch v-model="sku.status" :active-value="1" :inactive-value="0" />
          <button
            v-if="!sku.ID"
            type="button"
            class="grid h-6 w-6 place-items-center rounded text-muted-foreground transition-colors hover:bg-error-50 hover:text-error"
            aria-label="删除该规格"
            @click="goodsForm.skus.splice(index, 1)"
          >
            <svg-icon icon="lucide:x" class="h-3.5 w-3.5" />
          </button>
          <el-tooltip v-else content="已有规格不能删除，只能停用" placement="top">
            <svg-icon icon="lucide:lock" class="h-3.5 w-3.5 text-muted-foreground" />
          </el-tooltip>
        </div>
        <div v-if="!goodsForm.skus.length" class="px-3 py-8 text-center text-[12px] text-muted-foreground">
          还没有规格，点右上角「添加规格」
        </div>
      </div>

      <template #footer>
        <el-button @click="goodsDialog = false">取消</el-button>
        <el-button type="primary" :loading="goodsSaving" @click="submitGoods">保存</el-button>
      </template>
    </el-dialog>

    <!-- 往来单位编辑 -->
    <el-dialog v-model="partnerDialog" :title="(editingPartnerId ? '编辑' : '新建') + (partnerKind === 'supplier' ? '供应商' : '客户')" width="520px">
      <div class="grid grid-cols-2 gap-x-4">
        <label class="block">
          <span class="mb-1.5 block text-[11px] text-muted-foreground">编码</span>
          <el-input v-model="partnerForm.code" placeholder="如 GYS-001" />
        </label>
        <label class="block">
          <span class="mb-1.5 block text-[11px] text-muted-foreground">名称</span>
          <el-input v-model="partnerForm.name" />
        </label>
        <label class="block">
          <span class="mb-1.5 block text-[11px] text-muted-foreground">联系人</span>
          <el-input v-model="partnerForm.contact" />
        </label>
        <label class="block">
          <span class="mb-1.5 block text-[11px] text-muted-foreground">电话</span>
          <el-input v-model="partnerForm.phone" />
        </label>
        <label class="col-span-2 block">
          <span class="mb-1.5 block text-[11px] text-muted-foreground">地址</span>
          <el-input v-model="partnerForm.address" />
        </label>
        <label class="col-span-2 block">
          <span class="mb-1.5 block text-[11px] text-muted-foreground">备注</span>
          <el-input v-model="partnerForm.remark" type="textarea" :rows="2" />
        </label>
      </div>
      <template #footer>
        <el-button @click="partnerDialog = false">取消</el-button>
        <el-button type="primary" :loading="partnerSaving" @click="submitPartner">保存</el-button>
      </template>
    </el-dialog>

    <!-- 仓库编辑 -->
    <el-dialog v-model="warehouseDialog" :title="editingWarehouseId ? '编辑仓库' : '新建仓库'" width="440px">
      <div class="space-y-3">
        <label class="block">
          <span class="mb-1.5 block text-[11px] text-muted-foreground">仓库编码</span>
          <el-input v-model="warehouseForm.code" placeholder="如 WH-MAIN" />
        </label>
        <label class="block">
          <span class="mb-1.5 block text-[11px] text-muted-foreground">仓库名称</span>
          <el-input v-model="warehouseForm.name" placeholder="如 主仓" />
        </label>
      </div>
      <template #footer>
        <el-button @click="warehouseDialog = false">取消</el-button>
        <el-button type="primary" :loading="warehouseSaving" @click="submitWarehouse">保存</el-button>
      </template>
    </el-dialog>
  </div>
</template>

<script setup>
import { computed, onMounted, reactive, ref } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import PageHeader from '@/components/jxc/PageHeader.vue'
import SectionCard from '@/components/jxc/SectionCard.vue'
import EmptyState from '@/components/jxc/EmptyState.vue'
import MoneyText from '@/components/jxc/MoneyText.vue'
import StatusTag from '@/components/jxc/StatusTag.vue'
import {
  createGoods, createPartner, createWarehouse, listGoods, listPartners, listWarehouses,
  setGoodsStatus, setPartnerStatus, setWarehouseStatus, updateGoods, updatePartner, updateWarehouse
} from '@/api/jxc'
import { formatDateTime } from '@/view/jxc/shared/format'

const tab = ref('goods')
const TABS = [
  { key: 'goods', label: '商品与规格', icon: 'lucide:shirt', count: () => goods.value.length },
  { key: 'partner', label: '往来单位', icon: 'lucide:users', count: () => partnerCache.supplier.length + partnerCache.customer.length },
  { key: 'warehouse', label: '仓库', icon: 'lucide:warehouse', count: () => warehouses.value.length }
]

// ---------- 商品 ----------
const goods = ref([])
const goodsKeyword = ref('')
const selectedGoodsId = ref(0)
const loadingGoods = ref(false)
const goodsDialog = ref(false)
const goodsSaving = ref(false)
const editingGoodsId = ref(0)
const goodsForm = reactive({ code: '', name: '', status: 1, skus: [] })

const filteredGoods = computed(() => {
  const k = goodsKeyword.value.trim().toLowerCase()
  if (!k) return goods.value
  return goods.value.filter((item) => `${item.name} ${item.code}`.toLowerCase().includes(k))
})
const selectedGoods = computed(() => goods.value.find((item) => item.ID === selectedGoodsId.value) || null)
const enabledSkuCount = computed(() => (selectedGoods.value?.skus || []).filter((sku) => sku.status === 1).length)

const priceRange = (item) => rangeText(item?.skus?.map((sku) => Number(sku.salePrice) || 0))
const costRange = (item) => rangeText(item?.skus?.map((sku) => Number(sku.refCost) || 0))

function rangeText(values) {
  const list = (values || []).filter((value) => Number.isFinite(value))
  if (!list.length) return '—'
  const min = Math.min(...list)
  const max = Math.max(...list)
  return min === max ? `¥${min.toFixed(2)}` : `¥${min.toFixed(2)} – ¥${max.toFixed(2)}`
}

const loadGoods = async () => {
  loadingGoods.value = true
  try {
    const res = await listGoods()
    if (res.code !== 0) return ElMessage.error(res.msg || '加载商品失败')
    goods.value = res.data || []
    if (!goods.value.some((item) => item.ID === selectedGoodsId.value)) {
      selectedGoodsId.value = goods.value[0]?.ID || 0
    }
  } finally {
    loadingGoods.value = false
  }
}

const emptySku = () => ({ ID: 0, color: '', size: '', skuCode: '', salePrice: 0, refCost: 0, status: 1 })

const openGoods = (item) => {
  editingGoodsId.value = item?.ID || 0
  if (item) {
    goodsForm.code = item.code
    goodsForm.name = item.name
    goodsForm.status = item.status
    goodsForm.skus = (item.skus || []).map((sku) => ({ ...sku }))
  } else {
    goodsForm.code = ''
    goodsForm.name = ''
    goodsForm.status = 1
    goodsForm.skus = [emptySku()]
  }
  goodsDialog.value = true
}

const addSku = () => goodsForm.skus.push(emptySku())

const submitGoods = async () => {
  if (!goodsForm.code.trim() || !goodsForm.name.trim()) return ElMessage.warning('商品编码与名称必填')
  const rows = goodsForm.skus.filter((sku) => sku.skuCode || sku.color || sku.size)
  if (!rows.length) return ElMessage.warning('至少填写一个规格')
  if (rows.some((sku) => !String(sku.skuCode || '').trim())) return ElMessage.warning('每个规格都必须填写条码')
  if (rows.length > 1 && rows.some((sku) => !String(sku.color || '').trim() && !String(sku.size || '').trim())) {
    return ElMessage.warning('多个规格时，颜色或尺码至少填一个')
  }

  const codes = new Set()
  const specs = new Set()
  for (const sku of rows) {
    const code = String(sku.skuCode).trim()
    const spec = `${String(sku.color || '').trim()}|${String(sku.size || '').trim()}`
    if (codes.has(code)) return ElMessage.warning(`条码重复：${code}`)
    if (specs.has(spec)) return ElMessage.warning(`颜色/尺码重复：${spec.replace('|', '/')}`)
    codes.add(code)
    specs.add(spec)
  }

  goodsSaving.value = true
  try {
    const payload = {
      code: goodsForm.code.trim(),
      name: goodsForm.name.trim(),
      status: goodsForm.status,
      skus: rows.map((sku) => ({
        ID: sku.ID || 0,
        color: String(sku.color || '').trim(),
        size: String(sku.size || '').trim(),
        skuCode: String(sku.skuCode).trim(),
        salePrice: Number(sku.salePrice) || 0,
        refCost: Number(sku.refCost) || 0,
        status: sku.status
      }))
    }
    const res = editingGoodsId.value
      ? await updateGoods(editingGoodsId.value, payload)
      : await createGoods(payload)
    if (res.code !== 0) return ElMessage.error(res.msg || '保存失败')
    ElMessage.success(editingGoodsId.value ? '已保存' : '已创建')
    goodsDialog.value = false
    await loadGoods()
    if (res.data?.ID) selectedGoodsId.value = res.data.ID
    else if (editingGoodsId.value) selectedGoodsId.value = editingGoodsId.value
  } finally {
    goodsSaving.value = false
  }
}

const toggleGoods = async (item) => {
  const next = item.status === 1 ? 0 : 1
  const action = next === 0 ? '停用' : '启用'
  try {
    await ElMessageBox.confirm(
      `确定${action}商品「${item.name}」吗？${next === 0 ? '停用后开单页不再显示该商品，已有规格不会被自动停用。' : '启用后开单页重新可见。'}`,
      '提示',
      { type: 'warning' }
    )
  } catch { return }
  const res = await setGoodsStatus(item.ID, next)
  if (res.code !== 0) return ElMessage.error(res.msg || '操作失败')
  ElMessage.success(`已${action}`)
  await loadGoods()
}

// ---------- 往来单位 ----------
const PARTNER_KINDS = [{ key: 'supplier', label: '供应商' }, { key: 'customer', label: '客户' }]
const partnerKind = ref('supplier')
const partnerCache = reactive({ supplier: [], customer: [] })
const partners = computed(() => partnerCache[partnerKind.value] || [])
const partnerKeyword = ref('')
const loadingPartner = ref(false)
const partnerDialog = ref(false)
const partnerSaving = ref(false)
const editingPartnerId = ref(0)
const partnerForm = reactive({ code: '', name: '', contact: '', phone: '', address: '', remark: '' })

const filteredPartners = computed(() => {
  const k = partnerKeyword.value.trim().toLowerCase()
  if (!k) return partners.value
  return partners.value.filter((item) => `${item.code} ${item.name} ${item.contact} ${item.phone}`.toLowerCase().includes(k))
})

// 一次把供应商和客户都取回来：切页签不需要再发请求，页签总数也不会只数当前类型。
const loadPartners = async () => {
  loadingPartner.value = true
  try {
    const [supplierRes, customerRes] = await Promise.all([
      listPartners('supplier'),
      listPartners('customer')
    ])
    partnerCache.supplier = supplierRes.code === 0 ? supplierRes.data || [] : []
    partnerCache.customer = customerRes.code === 0 ? customerRes.data || [] : []
  } finally {
    loadingPartner.value = false
  }
}

const switchPartnerKind = (kind) => {
  if (partnerKind.value === kind) return
  partnerKind.value = kind
  partnerKeyword.value = ''
}

const openPartner = (item) => {
  editingPartnerId.value = item?.ID || 0
  Object.assign(partnerForm, {
    code: item?.code ?? '',
    name: item?.name ?? '',
    contact: item?.contact ?? '',
    phone: item?.phone ?? '',
    address: item?.address ?? '',
    remark: item?.remark ?? ''
  })
  partnerDialog.value = true
}

const submitPartner = async () => {
  if (!partnerForm.code.trim() || !partnerForm.name.trim()) return ElMessage.warning('编码与名称必填')
  partnerSaving.value = true
  try {
    const payload = { ...partnerForm, kind: partnerKind.value }
    const res = editingPartnerId.value
      ? await updatePartner(editingPartnerId.value, payload)
      : await createPartner(payload)
    if (res.code !== 0) return ElMessage.error(res.msg || '保存失败')
    ElMessage.success(editingPartnerId.value ? '已保存' : '已创建')
    partnerDialog.value = false
    await loadPartners()
  } finally {
    partnerSaving.value = false
  }
}

const togglePartner = async (item) => {
  const next = item.status === 1 ? 0 : 1
  const action = next === 0 ? '停用' : '启用'
  try {
    await ElMessageBox.confirm(`确定${action}「${item.name}」吗？`, '提示', { type: 'warning' })
  } catch { return }
  const res = await setPartnerStatus(item.ID, next)
  if (res.code !== 0) return ElMessage.error(res.msg || '操作失败')
  ElMessage.success(`已${action}`)
  await loadPartners()
}

// ---------- 仓库 ----------
const warehouses = ref([])
const warehouseKeyword = ref('')
const loadingWarehouse = ref(false)
const warehouseDialog = ref(false)
const warehouseSaving = ref(false)
const editingWarehouseId = ref(0)
const warehouseForm = reactive({ code: '', name: '' })

const filteredWarehouses = computed(() => {
  const k = warehouseKeyword.value.trim().toLowerCase()
  if (!k) return warehouses.value
  return warehouses.value.filter((item) => `${item.code} ${item.name}`.toLowerCase().includes(k))
})

const loadWarehouses = async () => {
  loadingWarehouse.value = true
  try {
    const res = await listWarehouses()
    warehouses.value = res.code === 0 ? res.data || [] : []
  } finally {
    loadingWarehouse.value = false
  }
}

const openWarehouse = (item) => {
  editingWarehouseId.value = item?.ID || 0
  warehouseForm.code = item?.code ?? ''
  warehouseForm.name = item?.name ?? ''
  warehouseDialog.value = true
}

const submitWarehouse = async () => {
  if (!warehouseForm.code.trim() || !warehouseForm.name.trim()) return ElMessage.warning('编码与名称必填')
  warehouseSaving.value = true
  try {
    const res = editingWarehouseId.value
      ? await updateWarehouse(editingWarehouseId.value, { code: warehouseForm.code.trim(), name: warehouseForm.name.trim() })
      : await createWarehouse({ code: warehouseForm.code.trim(), name: warehouseForm.name.trim() })
    if (res.code !== 0) return ElMessage.error(res.msg || '保存失败')
    ElMessage.success(editingWarehouseId.value ? '已保存' : '已创建')
    warehouseDialog.value = false
    await loadWarehouses()
  } finally {
    warehouseSaving.value = false
  }
}

const toggleWarehouse = async (item) => {
  const next = item.status === 1 ? 0 : 1
  const action = next === 0 ? '停用' : '启用'
  try {
    await ElMessageBox.confirm(`确定${action}仓库「${item.name}」吗？停用后开单页将不再显示。`, '提示', { type: 'warning' })
  } catch { return }
  const res = await setWarehouseStatus(item.ID, next)
  if (res.code !== 0) return ElMessage.error(res.msg || '操作失败')
  ElMessage.success(`已${action}`)
  await loadWarehouses()
}

onMounted(() => {
  loadGoods()
  loadPartners()
  loadWarehouses()
})
</script>
