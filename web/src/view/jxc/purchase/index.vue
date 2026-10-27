<template>
  <div class="flex h-full min-h-0 flex-col gap-4">
    <PageHeader
      eyebrow="PURCHASE"
      title="补货"
      description="建采购单 → 审核 → 入库。审核只改状态，入库才写账本行与成本。"
    >
      <template #actions>
        <el-button @click="goAllPurchases">
          <svg-icon icon="lucide:files" class="mr-1.5 h-3.5 w-3.5" />
          查看采购单
        </el-button>
      </template>
    </PageHeader>

    <div
      v-if="lastOrder"
      class="flex flex-wrap items-center gap-3 rounded-lg border border-success-200 bg-success-50 px-4 py-2.5 text-[12px] text-success-800"
    >
      <svg-icon icon="lucide:circle-check" class="h-4 w-4 shrink-0 text-success" />
      <span>已创建采购单 <span class="font-mono font-medium">{{ lastOrder.orderNo }}</span>（草稿），审核并入库后才会增加库存。</span>
      <div class="ml-auto flex items-center gap-2">
        <el-button size="small" @click="goReview(lastOrder)">去审核</el-button>
        <button type="button" class="text-success-700/70 hover:text-success-800" aria-label="关闭提示" @click="lastOrder = null">
          <svg-icon icon="lucide:x" class="h-3.5 w-3.5" />
        </button>
      </div>
    </div>

    <div class="grid min-h-0 flex-1 grid-cols-1 gap-4 xl:grid-cols-[minmax(0,1fr)_340px]">
      <SectionCard title="采购明细" description="扫码或搜索商品后回车添加" :padded="false" class="min-h-0">
        <template #actions>
          <span class="text-[11px] text-muted-foreground">{{ lineCount }} 行 · {{ qtyTotal }} 件</span>
        </template>

        <div class="flex h-full min-h-0 flex-col">
          <div class="border-b border-border px-4 py-3">
            <el-select
              v-model="pickerSkuId"
              class="w-full"
              filterable
              clearable
              placeholder="扫码 / 搜索商品名、编码、条码，选中即添加"
              @change="addSkuLine"
            >
              <el-option
                v-for="sku in skuOptions"
                :key="sku.ID"
                :label="`${sku.goodsName} · ${sku.color}/${sku.size} · ${sku.skuCode}`"
                :value="sku.ID"
              >
                <div class="flex items-center gap-2">
                  <span class="min-w-0 flex-1 truncate text-[12.5px]">{{ sku.goodsName }}</span>
                  <span class="text-[11px] text-muted-foreground">{{ sku.color }}/{{ sku.size }}</span>
                  <span class="font-mono text-[10px] text-muted-foreground">{{ sku.skuCode }}</span>
                </div>
              </el-option>
            </el-select>
          </div>

          <div class="min-h-0 flex-1 overflow-y-auto">
            <template v-if="lines.length">
              <div class="sticky top-0 z-10 grid grid-cols-[minmax(0,1fr)_96px_112px_104px_24px] items-center gap-2 border-b border-border bg-container px-4 py-2 text-[11px] text-muted-foreground">
                <span>商品</span>
                <span class="text-center">数量</span>
                <span class="text-center">进价</span>
                <span class="text-right">小计</span>
                <span></span>
              </div>
              <div
                v-for="(line, index) in lines"
                :key="line.skuId"
                class="grid grid-cols-[minmax(0,1fr)_96px_112px_104px_24px] items-center gap-2 border-b border-border px-4 py-2.5 last:border-b-0"
              >
                <div class="min-w-0">
                  <div class="truncate text-[12.5px] font-medium text-base-text">{{ line.goodsName }}</div>
                  <div class="mt-0.5 truncate text-[11px] text-muted-foreground">
                    {{ line.color }}/{{ line.size }} · <span class="font-mono">{{ line.skuCode }}</span>
                  </div>
                </div>
                <el-input-number v-model="line.qty" :min="1" :precision="0" :controls="false" class="w-full" />
                <el-input-number v-model="line.unitCost" :min="0" :precision="2" :controls="false" class="w-full" />
                <MoneyText :value="round2(line.qty * line.unitCost)" size="sm" strong />
                <button
                  type="button"
                  class="grid h-6 w-6 place-items-center rounded text-muted-foreground transition-colors hover:bg-error-50 hover:text-error"
                  aria-label="移除该行"
                  @click="removeLine(index)"
                >
                  <svg-icon icon="lucide:x" class="h-3.5 w-3.5" />
                </button>
              </div>
            </template>

            <EmptyState
              v-else
              icon="lucide:package-plus"
              title="还没有采购明细"
              description="从上方搜索框选择商品。选中后会带出参考进价，仍可按本次实际进价修改。"
            />
          </div>

          <div class="flex shrink-0 items-center gap-2 border-t border-border bg-muted px-4 py-2 text-[11px] text-muted-foreground">
            <svg-icon icon="lucide:info" class="h-3.5 w-3.5" />
            <span>进价只影响本次入库成本；历史毛利不会被后续进价改写</span>
          </div>
        </div>
      </SectionCard>

      <div class="flex min-h-0 flex-col gap-4 overflow-y-auto pr-1">
        <SectionCard title="单据信息" :padded="true">
          <div class="space-y-3">
            <label class="block">
              <span class="mb-1.5 block text-[11px] text-muted-foreground">供应商（可空）</span>
              <el-select v-model="supplierId" class="w-full" clearable placeholder="选择供应商">
                <el-option v-for="supplier in suppliers" :key="supplier.ID" :label="supplier.name" :value="supplier.ID" />
              </el-select>
            </label>
            <label class="block">
              <span class="mb-1.5 block text-[11px] text-muted-foreground">入库仓库</span>
              <el-select v-model="warehouseId" class="w-full" placeholder="选择仓库">
                <el-option v-for="warehouse in warehouses" :key="warehouse.ID" :label="warehouse.name" :value="warehouse.ID" />
              </el-select>
            </label>
            <label class="block">
              <span class="mb-1.5 block text-[11px] text-muted-foreground">备注</span>
              <el-input v-model="remark" type="textarea" :rows="2" maxlength="200" show-word-limit placeholder="例如：秋季补货，供应商承诺 3 天到" />
            </label>
          </div>

          <div class="mt-4 space-y-2 border-t border-border pt-3">
            <div class="flex items-center justify-between text-[12px] text-muted-foreground">
              <span>明细</span>
              <span class="tabular-nums">{{ lineCount }} 行 · {{ qtyTotal }} 件</span>
            </div>
            <div class="flex items-baseline justify-between">
              <span class="text-[12px] text-muted-foreground">合计</span>
              <MoneyText :value="total" size="xl" strong />
            </div>
            <div class="flex items-center gap-2 pt-1">
              <el-button class="flex-1" @click="reset">清空</el-button>
              <el-button type="primary" class="flex-[2]" :loading="saving" :disabled="!lines.length" @click="submit">
                提交采购单
              </el-button>
            </div>
            <div class="mt-2 flex items-center gap-1.5 text-[10.5px] text-muted-foreground">
              <kbd class="jxc-kbd">Ctrl ↵</kbd><span>快捷提交</span>
            </div>
          </div>
        </SectionCard>

        <SectionCard title="提交之后" description="两步才影响库存">
          <ol class="space-y-3 text-[12px] leading-5 text-muted-foreground">
            <li class="flex gap-2.5">
              <span class="grid h-5 w-5 shrink-0 place-items-center rounded-full bg-muted text-[10px] font-semibold text-muted-foreground">1</span>
              <span><b class="font-medium text-base-text">审核</b>：确认单据无误，不产生库存流水。</span>
            </li>
            <li class="flex gap-2.5">
              <span class="grid h-5 w-5 shrink-0 place-items-center rounded-full bg-muted text-[10px] font-semibold text-muted-foreground">2</span>
              <span><b class="font-medium text-base-text">入库</b>：生成账本行，成本冻结在这一刻。</span>
            </li>
          </ol>
          <el-button link type="primary" class="mt-3" @click="goDocuments">
            打开待审核采购
            <svg-icon icon="lucide:arrow-up-right" class="ml-1 h-3.5 w-3.5" />
          </el-button>
        </SectionCard>
      </div>
    </div>
  </div>
</template>

<script setup>
import { computed, onBeforeUnmount, onMounted, ref } from 'vue'
import { ElMessage } from 'element-plus'
import PageHeader from '@/components/jxc/PageHeader.vue'
import SectionCard from '@/components/jxc/SectionCard.vue'
import EmptyState from '@/components/jxc/EmptyState.vue'
import MoneyText from '@/components/jxc/MoneyText.vue'
import { createPurchase, listGoods, listPartners, listWarehouses } from '@/api/jxc'
import { round2 } from '@/view/jxc/shared/format'
import { useRouter } from 'vue-router'

const router = useRouter()
const warehouses = ref([])
const suppliers = ref([])
const skuOptions = ref([])
const supplierId = ref(null)
const warehouseId = ref(null)
const pickerSkuId = ref(null)
const lines = ref([])
const remark = ref('')
const saving = ref(false)
const lastOrder = ref(null)

const total = computed(() => round2(lines.value.reduce((sum, line) => sum + Number(line.qty || 0) * Number(line.unitCost || 0), 0)))
const qtyTotal = computed(() => lines.value.reduce((sum, line) => sum + Number(line.qty || 0), 0))
const lineCount = computed(() => lines.value.length)

const addSkuLine = (skuId) => {
  if (!skuId) return
  const sku = skuOptions.value.find((item) => item.ID === skuId)
  if (!sku) return
  const exist = lines.value.find((line) => line.skuId === skuId)
  if (exist) {
    exist.qty += 1
  } else {
    lines.value.push({
      skuId: sku.ID,
      skuCode: sku.skuCode,
      goodsName: sku.goodsName,
      color: sku.color,
      size: sku.size,
      qty: 1,
      unitCost: Number(sku.refCost) || 0
    })
  }
  pickerSkuId.value = null
}

const removeLine = (index) => {
  lines.value.splice(index, 1)
}

const reset = () => {
  lines.value = []
  remark.value = ''
  lastOrder.value = null
}

const goAllPurchases = () => router.push('/jxc/documents?tab=purchase')
const goDocuments = () => router.push('/jxc/documents?tab=purchase&status=1')
const goReview = (order) => router.push({ path: '/jxc/documents', query: { tab: 'purchase', status: '1', keyword: order.orderNo } })

const submit = async () => {
  if (saving.value) return
  if (!warehouseId.value) return ElMessage.warning('请选择入库仓库')
  if (!lines.value.length) return ElMessage.warning('请至少添加一行采购明细')
  if (lines.value.some((line) => !line.qty || line.qty <= 0)) return ElMessage.warning('采购数量必须大于 0')
  if (lines.value.some((line) => Number(line.unitCost) < 0)) return ElMessage.warning('进价不能为负数')

  saving.value = true
  try {
    const res = await createPurchase({
      warehouseId: warehouseId.value,
      supplierId: supplierId.value || null,
      remark: remark.value,
      items: lines.value.map((line) => ({ skuId: line.skuId, qty: Number(line.qty), unitCost: Number(line.unitCost) }))
    })
    if (res.code !== 0) return ElMessage.error(res.msg || '创建采购单失败')
    lastOrder.value = { ID: res.data.ID, orderNo: res.data.orderNo }
    lines.value = []
    remark.value = ''
    ElMessage.success(`已创建采购单 ${res.data.orderNo}`)
  } finally {
    saving.value = false
  }
}

const onKeydown = (event) => {
  if (event.key === 'Enter' && (event.ctrlKey || event.metaKey)) {
    event.preventDefault()
    submit()
  }
}

onMounted(async () => {
  window.addEventListener('keydown', onKeydown)
  const [warehouseRes, supplierRes, goodsRes] = await Promise.all([
    listWarehouses({ active: 1 }),
    listPartners('supplier', { active: 1 }),
    listGoods({ active: 1 })
  ])
  if (warehouseRes.code === 0) {
    warehouses.value = warehouseRes.data || []
    if (warehouses.value.length) warehouseId.value = warehouses.value[0].ID
  }
  if (supplierRes.code === 0) suppliers.value = supplierRes.data || []
  if (goodsRes.code === 0) {
    skuOptions.value = (goodsRes.data || []).flatMap((product) =>
      (product.skus || [])
        .filter((sku) => sku.status === 1)
        .map((sku) => ({ ...sku, goodsName: product.name, goodsCode: product.code }))
    )
  }
})

onBeforeUnmount(() => {
  window.removeEventListener('keydown', onKeydown)
})
</script>
