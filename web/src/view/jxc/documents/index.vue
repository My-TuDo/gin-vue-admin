<template>
  <div class="documents">
    <el-tabs v-model="tab" @tab-change="onTab">
      <!-- 销售单 -->
      <el-tab-pane label="销售单" name="sale">
        <el-table v-loading="loading" :data="sales" size="small">
          <el-table-column label="单号" width="170">
            <template #default="{ row }">
              <el-button link type="primary" class="mono" @click="openDetail('sale', row)">{{ row.orderNo }}</el-button>
            </template>
          </el-table-column>
          <el-table-column label="状态" width="100">
            <template #default="{ row }">
              <StatusTag kind="sale" :status="row.status" />
              <el-tag v-if="row.reversedAt" size="small" type="info" effect="plain" class="ml4">已红冲</el-tag>
            </template>
          </el-table-column>
          <el-table-column label="销售额" width="110" align="right">
            <template #default="{ row }"><MoneyText :value="row.totalAmount" /></template>
          </el-table-column>
          <el-table-column label="成本" width="110" align="right">
            <template #default="{ row }"><MoneyText :value="row.costAmount" /></template>
          </el-table-column>
          <el-table-column label="毛利" width="110" align="right">
            <template #default="{ row }">
              <MoneyText :value="rounded(row.totalAmount - row.costAmount)" />
            </template>
          </el-table-column>
          <el-table-column label="出库时间" width="150">
            <template #default="{ row }">{{ fmt(row.postedAt) || '—' }}</template>
          </el-table-column>
          <el-table-column label="操作" min-width="230">
            <template #default="{ row }">
              <el-button v-if="row.status === 1" link type="primary" @click="ship(row)">出库</el-button>
              <el-button v-if="row.status === 1" link type="danger" @click="cancelSaleOrder(row)">取消</el-button>
              <el-button v-if="row.status === 2 && !row.reversedAt" link type="warning" @click="openReturn(row)">退货</el-button>
              <el-button v-if="row.status === 2 && !row.reversedAt" link type="danger" @click="openReversal('sale', row)">红冲</el-button>
              <span v-if="row.status === 2 && row.reversedAt" class="muted">已整单冲销</span>
            </template>
          </el-table-column>
        </el-table>
      </el-tab-pane>

      <!-- 采购单 -->
      <el-tab-pane label="采购单" name="purchase">
        <el-table v-loading="loading" :data="purchases" size="small">
          <el-table-column label="单号" width="170">
            <template #default="{ row }">
              <el-button link type="primary" class="mono" @click="openDetail('purchase', row)">{{ row.orderNo }}</el-button>
            </template>
          </el-table-column>
          <el-table-column label="状态" width="110">
            <template #default="{ row }">
              <StatusTag kind="purchase" :status="row.status" />
              <el-tag v-if="row.reversedAt" size="small" type="info" effect="plain" class="ml4">已红冲</el-tag>
            </template>
          </el-table-column>
          <el-table-column label="金额" width="110" align="right">
            <template #default="{ row }"><MoneyText :value="row.totalAmount" /></template>
          </el-table-column>
          <el-table-column label="明细" width="90">
            <template #default="{ row }">{{ row.items?.length || 0 }} 项</template>
          </el-table-column>
          <el-table-column label="入库时间" width="150">
            <template #default="{ row }">{{ fmt(row.postedAt) || '—' }}</template>
          </el-table-column>
          <el-table-column label="操作" min-width="200">
            <template #default="{ row }">
              <el-button v-if="row.status === 1" link type="primary" @click="approvePurchaseRow(row)">审核</el-button>
              <el-button v-if="row.status === 2" link type="success" @click="receivePurchaseRow(row)">入库</el-button>
              <el-button v-if="row.status === 1 || row.status === 2" link type="danger" @click="cancelPurchaseRow(row)">取消</el-button>
              <el-button v-if="row.status === 3 && !row.reversedAt" link type="danger" @click="openReversal('purchase', row)">红冲</el-button>
            </template>
          </el-table-column>
        </el-table>
      </el-tab-pane>

      <!-- 退货单 -->
      <el-tab-pane label="退货单" name="return">
        <el-table v-loading="loading" :data="returns" size="small">
          <el-table-column prop="returnNo" label="退货单号" width="170" />
          <el-table-column prop="saleNo" label="原销售单" width="170" />
          <el-table-column label="状态" width="100">
            <template #default="{ row }">
              <span class="tag" :class="retTone(row.status)">{{ retLabel(row.status) }}</span>
            </template>
          </el-table-column>
          <el-table-column label="退款" width="110" align="right">
            <template #default="{ row }"><MoneyText :value="row.amount" /></template>
          </el-table-column>
          <el-table-column label="退回成本" width="110" align="right">
            <template #default="{ row }"><MoneyText :value="row.costAmount" /></template>
          </el-table-column>
          <el-table-column prop="reason" label="原因" min-width="160" />
          <el-table-column label="操作" width="160">
            <template #default="{ row }">
              <el-button v-if="row.status === 1" link type="success" @click="doPostReturn(row)">入库</el-button>
              <el-button v-if="row.status === 1" link type="danger" @click="doCancelReturn(row)">取消</el-button>
            </template>
          </el-table-column>
        </el-table>
      </el-tab-pane>

      <!-- 红冲记录 -->
      <el-tab-pane label="红冲记录" name="reversal">
        <el-table v-loading="loading" :data="reversals" size="small">
          <el-table-column prop="reversalNo" label="红冲单号" width="170" />
          <el-table-column label="来源" width="100">
            <template #default="{ row }">{{ row.sourceType === 'sale' ? '销售单' : '采购单' }}</template>
          </el-table-column>
          <el-table-column prop="sourceNo" label="原单号" width="170" />
          <el-table-column label="冲销金额" width="120" align="right">
            <template #default="{ row }"><MoneyText :value="row.amount" /></template>
          </el-table-column>
          <el-table-column label="件数" width="80" align="right">
            <template #default="{ row }">{{ row.qty }}</template>
          </el-table-column>
          <el-table-column prop="reason" label="原因" min-width="180" />
          <el-table-column label="时间" width="150">
            <template #default="{ row }">{{ fmt(row.postedAt) }}</template>
          </el-table-column>
        </el-table>
      </el-tab-pane>
    </el-tabs>

    <!-- 单据详情 -->
    <el-drawer v-model="detailDialog" :title="detail?.orderNo || '单据详情'" size="620px">
      <div v-if="detail" class="detail">
        <div class="d-row">
          <span class="d-k">状态</span>
          <span>
            <StatusTag v-if="detailKind === 'sale'" kind="sale" :status="detail.status" />
            <StatusTag v-else kind="purchase" :status="detail.status" />
            <el-tag v-if="detail.reversedAt" size="small" type="info" effect="plain" class="ml4">已红冲</el-tag>
          </span>
        </div>
        <div class="d-row"><span class="d-k">仓库</span><span>{{ detail.warehouseId }}</span></div>
        <div class="d-row"><span class="d-k">金额</span><span><MoneyText :value="detail.totalAmount" /></span></div>
        <div v-if="detailKind === 'sale'" class="d-row">
          <span class="d-k">成本</span><span><MoneyText :value="detail.costAmount" /></span>
        </div>
        <div class="d-row"><span class="d-k">过账时间</span><span>{{ fmt(detail.postedAt) || '—' }}</span></div>
        <el-table :data="detail.items || []" size="small" style="margin-top: 12px">
          <el-table-column prop="skuId" label="SKU" width="70" />
          <el-table-column prop="qty" label="数量" width="80" align="right" />
          <el-table-column v-if="detailKind === 'sale'" label="售价" width="100" align="right">
            <template #default="{ row }"><MoneyText :value="row.price" /></template>
          </el-table-column>
          <el-table-column v-else label="进价" width="100" align="right">
            <template #default="{ row }"><MoneyText :value="row.unitCost" /></template>
          </el-table-column>
          <el-table-column label="金额" width="110" align="right">
            <template #default="{ row }"><MoneyText :value="row.amount" /></template>
          </el-table-column>
          <el-table-column v-if="detailKind === 'sale'" label="该行成本" min-width="100" align="right">
            <template #default="{ row }"><MoneyText :value="row.costAmt" /></template>
          </el-table-column>
        </el-table>
      </div>
    </el-drawer>

    <!-- 红冲：原因必填 -->
    <el-dialog v-model="revDialog" title="整单红冲" width="460px">
      <el-alert type="warning" :closable="false" show-icon
        title="红冲会生成反向账本记录，原单作废但历史保留"
        description="红冲不能撤销，也不能对同一张单重复操作。若只是部分退货，请用「退货」。" />
      <el-form label-width="80px" style="margin-top: 12px">
        <el-form-item label="单据">
          <span class="mono">{{ revTarget?.orderNo }}</span>
        </el-form-item>
        <el-form-item label="原因">
          <el-input v-model="revReason" type="textarea" :rows="3" placeholder="例如：供应商发错货 / 录错单据" />
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="revDialog = false">取消</el-button>
        <el-button type="danger" :loading="saving" @click="submitReversal">确认红冲</el-button>
      </template>
    </el-dialog>

    <!-- 退货：按可退数量录入 -->
    <el-dialog v-model="retDialog" title="销售退货" width="620px">
      <div class="ret-head">
        原销售单 <span class="mono">{{ retTarget?.orderNo }}</span>
        <span class="muted">（成本按原单出库时冻结的成本回补，不用当前均价）</span>
      </div>
      <el-table :data="returnable" size="small" style="margin: 10px 0">
        <el-table-column prop="skuId" label="SKU" width="70" />
        <el-table-column prop="soldQty" label="原售" width="70" align="right" />
        <el-table-column prop="returnedQty" label="已退" width="70" align="right" />
        <el-table-column prop="canReturn" label="可退" width="70" align="right" />
        <el-table-column label="单价" width="100" align="right">
          <template #default="{ row }"><MoneyText :value="row.price" /></template>
        </el-table-column>
        <el-table-column label="退货数量" width="140">
          <template #default="{ row }">
            <el-input-number v-model="row.qty" :min="0" :max="row.canReturn" size="small" controls-position="right"
              style="width: 110px" :disabled="row.canReturn === 0" />
          </template>
        </el-table-column>
      </el-table>
      <el-form label-width="80px">
        <el-form-item label="原因">
          <el-input v-model="retReason" placeholder="例如：尺码不合适" />
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="retDialog = false">取消</el-button>
        <el-button type="primary" :loading="saving" @click="submitReturn">创建退货单</el-button>
      </template>
    </el-dialog>
  </div>
</template>

<script setup>
import { onMounted, ref } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import MoneyText from '@/components/jxc/MoneyText.vue'
import StatusTag from '@/components/jxc/StatusTag.vue'
import {
  approvePurchase, cancelPurchase, cancelSale, createReturn, createReversal,
  getPurchase, getReturnable, getSale, pagePurchase, pageReturn, pageReversal, pageSale,
  postReturn, cancelReturn, receivePurchase, shipSale
} from '@/api/jxc'

const tab = ref('sale')
const loading = ref(false)
const saving = ref(false)

const sales = ref([])
const purchases = ref([])
const returns = ref([])
const reversals = ref([])

const detailDialog = ref(false)
const detail = ref(null)
const detailKind = ref('sale')
const revDialog = ref(false)
const revTarget = ref(null)
const revSource = ref('sale')
const revReason = ref('')

const retDialog = ref(false)
const retTarget = ref(null)
const returnable = ref([])
const retReason = ref('')

const rounded = (v) => Math.round((Number(v) || 0) * 100) / 100
const fmt = (s) => {
  if (!s) return ''
  const d = new Date(s)
  const p = (n) => String(n).padStart(2, '0')
  return `${d.getFullYear()}-${p(d.getMonth() + 1)}-${p(d.getDate())} ${p(d.getHours())}:${p(d.getMinutes())}`
}

const RET = { 1: ['草稿', 'muted'], 2: ['已入库', 'ok'], 3: ['已取消', 'muted'] }
const retLabel = (s) => RET[s]?.[0] ?? '未知'
const retTone = (s) => RET[s]?.[1] ?? 'muted'

const load = async () => {
  loading.value = true
  try {
    const [s, p, r, rv] = await Promise.all([
      pageSale({ page: 1, pageSize: 50 }),
      pagePurchase({ page: 1, pageSize: 50 }),
      pageReturn({ page: 1, pageSize: 50 }),
      pageReversal({ page: 1, pageSize: 50 })
    ])
    sales.value = s.code === 0 ? s.data.list || [] : []
    purchases.value = p.code === 0 ? p.data.list || [] : []
    returns.value = r.code === 0 ? r.data.list || [] : []
    reversals.value = rv.code === 0 ? rv.data.list || [] : []
  } finally {
    loading.value = false
  }
}

const onTab = () => load()

const openDetail = async (kind, row) => {
  detailKind.value = kind
  const res = kind === 'sale' ? await getSale(row.ID) : await getPurchase(row.ID)
  if (res.code !== 0) return ElMessage.error(res.msg)
  detail.value = res.data
  detailDialog.value = true
}

const ship = async (row) => {
  const res = await shipSale(row.ID)
  if (res.code !== 0) return ElMessage.error(res.msg)
  ElMessage.success('已出库')
  await load()
}
const cancelSaleOrder = async (row) => {
  const res = await cancelSale(row.ID)
  if (res.code !== 0) return ElMessage.error(res.msg)
  ElMessage.success('已取消')
  await load()
}
const approvePurchaseRow = async (row) => {
  const res = await approvePurchase(row.ID)
  if (res.code !== 0) return ElMessage.error(res.msg)
  ElMessage.success('已审核')
  await load()
}
const receivePurchaseRow = async (row) => {
  const res = await receivePurchase(row.ID)
  if (res.code !== 0) return ElMessage.error(res.msg)
  ElMessage.success('已入库')
  await load()
}
const cancelPurchaseRow = async (row) => {
  const res = await cancelPurchase(row.ID)
  if (res.code !== 0) return ElMessage.error(res.msg)
  ElMessage.success('已取消')
  await load()
}

const openReversal = (source, row) => {
  revSource.value = source
  revTarget.value = row
  revReason.value = ''
  revDialog.value = true
}

const submitReversal = async () => {
  if (!revReason.value.trim()) return ElMessage.warning('请填写红冲原因')
  saving.value = true
  try {
    const res = await createReversal({
      sourceType: revSource.value, sourceId: revTarget.value.ID, reason: revReason.value
    })
    if (res.code !== 0) return ElMessage.error(res.msg)
    ElMessage.success(`已红冲 ${res.data.reversalNo}`)
    revDialog.value = false
    await load()
  } finally {
    saving.value = false
  }
}

const openReturn = async (row) => {
  retTarget.value = row
  retReason.value = ''
  const res = await getReturnable(row.ID)
  if (res.code !== 0) return ElMessage.error(res.msg)
  returnable.value = (res.data || []).map((l) => ({ ...l, qty: 0 }))
  retDialog.value = true
}

const submitReturn = async () => {
  const items = returnable.value.filter((l) => l.qty > 0).map((l) => ({ skuId: l.skuId, qty: l.qty }))
  if (items.length === 0) return ElMessage.warning('请填写退货数量')
  saving.value = true
  try {
    const res = await createReturn({ saleId: retTarget.value.ID, items, reason: retReason.value })
    if (res.code !== 0) return ElMessage.error(res.msg)
    ElMessage.success(`已创建退货单 ${res.data.returnNo}`)
    retDialog.value = false
    tab.value = 'return'
    await load()
  } finally {
    saving.value = false
  }
}

const doPostReturn = async (row) => {
  const res = await postReturn(row.ID)
  if (res.code !== 0) return ElMessage.error(res.msg)
  ElMessage.success('退货已入库')
  await load()
}
const doCancelReturn = async (row) => {
  await ElMessageBox.confirm('确认取消该退货单？', '提示', { type: 'warning' })
  const res = await cancelReturn(row.ID)
  if (res.code !== 0) return ElMessage.error(res.msg)
  ElMessage.success('已取消')
  await load()
}

onMounted(load)
</script>

<style scoped>
.muted {
  color: var(--el-text-color-placeholder);
  font-size: 12px;
}
.mono {
  font-family: ui-monospace, monospace;
}
.ml4 {
  margin-left: 4px;
}
.tag {
  font-size: 12px;
  padding: 1px 8px;
  border-radius: 10px;
  background: var(--el-fill-color-light);
  color: var(--el-text-color-secondary);
}
.tag.ok {
  background: var(--el-color-success-light-9);
  color: var(--el-color-success);
}
.ret-head {
  font-size: 13px;
}
.detail .d-row {
  display: flex;
  gap: 12px;
  padding: 6px 0;
  border-bottom: 1px dashed var(--el-border-color-lighter);
  font-size: 13px;
}
.d-k {
  color: var(--el-text-color-secondary);
  width: 72px;
  flex: 0 0 72px;
}
</style>
