<template>
  <div class="master">
    <el-tabs v-model="tab">
      <el-tab-pane label="商品与规格" name="goods">
        <div class="toolbar">
          <el-input v-model="kw" placeholder="搜索商品" clearable style="width: 200px" />
          <div class="spacer"></div>
          <el-button type="primary" @click="openGoods()">新建商品</el-button>
        </div>

        <el-table v-loading="loadingGoods" :data="filteredGoods" size="small">
          <el-table-column prop="name" label="商品" min-width="160" />
          <el-table-column prop="code" label="商品编码" width="140" />
          <el-table-column label="规格（颜色/尺码）" min-width="300">
            <template #default="{ row }">
              <span v-for="s in row.skus" :key="s.ID" class="chip" :class="{ off: s.status !== 1 }">
                {{ s.color }}/{{ s.size }} · {{ s.skuCode }} · ¥{{ Number(s.salePrice).toFixed(2) }}
                <span v-if="s.status !== 1">（停用）</span>
              </span>
              <span v-if="!row.skus?.length" class="muted">无规格</span>
            </template>
          </el-table-column>
          <el-table-column label="状态" width="90">
            <template #default="{ row }">
              <el-tag :type="row.status === 1 ? 'success' : 'info'" size="small">
                {{ row.status === 1 ? '启用' : '停用' }}
              </el-tag>
            </template>
          </el-table-column>
          <el-table-column label="操作" width="150">
            <template #default="{ row }">
              <div class="flex items-center gap-1">
                <el-button link type="primary" @click="openGoods(row)">编辑</el-button>
                <el-button link :type="row.status === 1 ? 'danger' : 'success'" @click="toggleGoods(row)">
                  {{ row.status === 1 ? '停用' : '启用' }}
                </el-button>
              </div>
            </template>
          </el-table-column>
        </el-table>
      </el-tab-pane>

      <el-tab-pane label="往来单位" name="partner">
        <div class="toolbar">
          <el-radio-group v-model="partnerKind" @change="loadPartners">
            <el-radio-button label="supplier">供应商</el-radio-button>
            <el-radio-button label="customer">客户</el-radio-button>
          </el-radio-group>
          <div class="spacer"></div>
          <el-button type="primary" @click="openPartner()">新建{{ partnerKind === 'supplier' ? '供应商' : '客户' }}</el-button>
        </div>
        <el-table v-loading="loadingPartner" :data="partners" size="small">
          <el-table-column prop="code" label="编码" width="140" />
          <el-table-column prop="name" label="名称" min-width="160" />
          <el-table-column prop="contact" label="联系人" width="110" />
          <el-table-column prop="phone" label="电话" width="140" />
          <el-table-column prop="address" label="地址" min-width="180" />
          <el-table-column label="状态" width="90">
            <template #default="{ row }">
              <el-tag :type="row.status === 1 ? 'success' : 'info'" size="small">
                {{ row.status === 1 ? '启用' : '停用' }}
              </el-tag>
            </template>
          </el-table-column>
          <el-table-column label="操作" width="150">
            <template #default="{ row }">
              <div class="flex items-center gap-1">
                <el-button link type="primary" @click="openPartner(row)">编辑</el-button>
                <el-button link :type="row.status === 1 ? 'danger' : 'success'" @click="togglePartner(row)">
                  {{ row.status === 1 ? '停用' : '启用' }}
                </el-button>
              </div>
            </template>
          </el-table-column>
        </el-table>
      </el-tab-pane>

      <el-tab-pane label="仓库" name="warehouse">
        <div class="toolbar">
          <div class="spacer"></div>
          <el-button type="primary" @click="openWarehouse()">新建仓库</el-button>
        </div>
        <el-table v-loading="loadingWh" :data="warehouses" size="small">
          <el-table-column prop="code" label="编码" width="140" />
          <el-table-column prop="name" label="名称" min-width="180" />
          <el-table-column label="状态" width="90">
            <template #default="{ row }">
              <el-tag :type="row.status === 1 ? 'success' : 'info'" size="small">
                {{ row.status === 1 ? '启用' : '停用' }}
              </el-tag>
            </template>
          </el-table-column>
          <el-table-column label="操作" width="150">
            <template #default="{ row }">
              <div class="flex items-center gap-1">
                <el-button link type="primary" @click="openWarehouse(row)">编辑</el-button>
                <el-button link :type="row.status === 1 ? 'danger' : 'success'" @click="toggleWarehouse(row)">
                  {{ row.status === 1 ? '停用' : '启用' }}
                </el-button>
              </div>
            </template>
          </el-table-column>
        </el-table>
      </el-tab-pane>
    </el-tabs>

    <!-- 商品：规格一次填完，不弹二次窗；编辑时已有规格不可删除（历史账本还引用它） -->
    <el-dialog v-model="goodsDialog" :title="editingGoodsId ? '编辑商品' : '新建商品'" width="880px">
      <el-form label-width="90px">
        <el-form-item label="商品编码"><el-input v-model="form.code" placeholder="如 WY-001" /></el-form-item>
        <el-form-item label="商品名称"><el-input v-model="form.name" placeholder="如 卫衣" /></el-form-item>
        <el-form-item label="状态">
          <el-switch v-model="form.status" :active-value="1" :inactive-value="0" active-text="启用" inactive-text="停用" />
        </el-form-item>
        <el-form-item label="规格">
          <div class="sku-editor">
            <div v-for="(s, i) in form.skus" :key="s.ID || `new-${i}`" class="sku-row">
              <el-input v-model="s.color" placeholder="颜色" style="width: 100px" />
              <el-input v-model="s.size" placeholder="尺码" style="width: 90px" />
              <el-input v-model="s.skuCode" placeholder="条码（唯一）" style="width: 150px" />
              <el-input-number v-model="s.salePrice" :min="0" :precision="2" size="small" controls-position="right"
                style="width: 115px" title="销售价" />
              <el-input-number v-model="s.refCost" :min="0" :precision="2" size="small" controls-position="right"
                style="width: 115px" title="参考进价" />
              <el-switch v-model="s.status" :active-value="1" :inactive-value="0" size="small" title="规格启用状态" />
              <button v-if="!s.ID" class="rm" type="button" @click="form.skus.splice(i, 1)">×</button>
              <span v-else class="muted" title="已有规格不能删除，只能停用">历史</span>
            </div>
            <el-button link type="primary" @click="addSku">+ 添加规格</el-button>
          </div>
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="goodsDialog = false">取消</el-button>
        <el-button type="primary" :loading="saving" @click="submitGoods">保存</el-button>
      </template>
    </el-dialog>

    <el-dialog v-model="partnerDialog" :title="editingPartnerId ? '编辑' : '新建' + (partnerKind === 'supplier' ? '供应商' : '客户')" width="460px">
      <el-form label-width="80px">
        <el-form-item label="编码"><el-input v-model="partner.code" placeholder="如 GYS-001" /></el-form-item>
        <el-form-item label="名称"><el-input v-model="partner.name" /></el-form-item>
        <el-form-item label="联系人"><el-input v-model="partner.contact" /></el-form-item>
        <el-form-item label="电话"><el-input v-model="partner.phone" /></el-form-item>
        <el-form-item label="地址"><el-input v-model="partner.address" /></el-form-item>
        <el-form-item label="备注"><el-input v-model="partner.remark" type="textarea" :rows="2" /></el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="partnerDialog = false">取消</el-button>
        <el-button type="primary" :loading="saving" @click="submitPartner">保存</el-button>
      </template>
    </el-dialog>

    <el-dialog v-model="whDialog" :title="editingWhId ? '编辑仓库' : '新建仓库'" width="420px">
      <el-form label-width="80px">
        <el-form-item label="编码"><el-input v-model="wh.code" placeholder="如 WH1" /></el-form-item>
        <el-form-item label="名称"><el-input v-model="wh.name" placeholder="如 主仓" /></el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="whDialog = false">取消</el-button>
        <el-button type="primary" :loading="saving" @click="submitWh">保存</el-button>
      </template>
    </el-dialog>
  </div>
</template>

<script setup>
import { computed, onMounted, reactive, ref } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import {
  createGoods,
  createPartner,
  createWarehouse,
  listGoods,
  listPartners,
  listWarehouses,
  setGoodsStatus,
  setPartnerStatus,
  setWarehouseStatus,
  updateGoods,
  updatePartner,
  updateWarehouse
} from '@/api/jxc'

const tab = ref('goods')
const kw = ref('')
const goods = ref([])
const warehouses = ref([])
const loadingGoods = ref(false)
const loadingWh = ref(false)
const saving = ref(false)

const goodsDialog = ref(false)
const whDialog = ref(false)
const editingGoodsId = ref(0)
const editingWhId = ref(0)

const form = reactive({ code: '', name: '', status: 1, skus: [] })
const partners = ref([])
const partnerKind = ref('supplier')
const partnerDialog = ref(false)
const editingPartnerId = ref(0)
const loadingPartner = ref(false)
const partner = reactive({ code: '', name: '', contact: '', phone: '', address: '', remark: '' })
const wh = reactive({ code: '', name: '' })

const filteredGoods = computed(() => {
  const k = kw.value.trim().toLowerCase()
  if (!k) return goods.value
  return goods.value.filter((g) => `${g.name} ${g.code}`.toLowerCase().includes(k))
})

const loadGoods = async () => {
  loadingGoods.value = true
  try {
    const res = await listGoods()
    goods.value = res.code === 0 ? res.data || [] : []
  } finally {
    loadingGoods.value = false
  }
}

const loadWarehouses = async () => {
  loadingWh.value = true
  try {
    const res = await listWarehouses()
    warehouses.value = res.code === 0 ? res.data || [] : []
  } finally {
    loadingWh.value = false
  }
}

const loadPartners = async () => {
  loadingPartner.value = true
  try {
    const res = await listPartners(partnerKind.value)
    partners.value = res.code === 0 ? res.data || [] : []
  } finally {
    loadingPartner.value = false
  }
}

const addSku = () => form.skus.push({ color: '', size: '', skuCode: '', salePrice: 0, refCost: 0, status: 1 })
const emptySku = () => ({ color: '', size: '', skuCode: '', salePrice: 0, refCost: 0, status: 1 })

const openGoods = (row) => {
  editingGoodsId.value = row?.ID ?? 0
  if (row) {
    form.code = row.code
    form.name = row.name
    form.status = row.status
    form.skus = (row.skus || []).map((s) => ({ ...s }))
  } else {
    form.code = ''
    form.name = ''
    form.status = 1
    form.skus = [emptySku()]
  }
  goodsDialog.value = true
}

const submitGoods = async () => {
  if (!form.code || !form.name) return ElMessage.warning('商品编码与名称必填')
  const skus = form.skus.filter((s) => s.skuCode || s.color || s.size)
  if (skus.length === 0) return ElMessage.warning('至少填写一个规格条码')
  saving.value = true
  try {
    const payload = {
      code: form.code,
      name: form.name,
      status: form.status,
      skus: skus.map((s) => ({
        ID: s.ID || 0,
        color: s.color,
        size: s.size,
        skuCode: s.skuCode,
        salePrice: Number(s.salePrice) || 0,
        refCost: Number(s.refCost) || 0,
        status: s.status
      }))
    }
    const res = editingGoodsId.value
      ? await updateGoods(editingGoodsId.value, payload)
      : await createGoods(payload)
    if (res.code !== 0) return ElMessage.error(res.msg || '保存失败')
    ElMessage.success(editingGoodsId.value ? '已保存' : '已创建')
    goodsDialog.value = false
    await loadGoods()
  } finally {
    saving.value = false
  }
}

const openPartner = (row) => {
  editingPartnerId.value = row?.ID ?? 0
  Object.assign(partner, {
    code: row?.code ?? '',
    name: row?.name ?? '',
    contact: row?.contact ?? '',
    phone: row?.phone ?? '',
    address: row?.address ?? '',
    remark: row?.remark ?? ''
  })
  partnerDialog.value = true
}

const submitPartner = async () => {
  if (!partner.code || !partner.name) return ElMessage.warning('编码与名称必填')
  saving.value = true
  try {
    const payload = { ...partner, kind: partnerKind.value }
    const res = editingPartnerId.value
      ? await updatePartner(editingPartnerId.value, payload)
      : await createPartner(payload)
    if (res.code !== 0) return ElMessage.error(res.msg || '保存失败')
    ElMessage.success(editingPartnerId.value ? '已保存' : '已创建')
    partnerDialog.value = false
    await loadPartners()
  } finally {
    saving.value = false
  }
}

const openWarehouse = (row) => {
  editingWhId.value = row?.ID ?? 0
  wh.code = row?.code ?? ''
  wh.name = row?.name ?? ''
  whDialog.value = true
}

const submitWh = async () => {
  if (!wh.code || !wh.name) return ElMessage.warning('编码与名称必填')
  saving.value = true
  try {
    const res = editingWhId.value
      ? await updateWarehouse(editingWhId.value, { code: wh.code, name: wh.name })
      : await createWarehouse({ code: wh.code, name: wh.name })
    if (res.code !== 0) return ElMessage.error(res.msg || '保存失败')
    ElMessage.success(editingWhId.value ? '已保存' : '已创建')
    whDialog.value = false
    await loadWarehouses()
  } finally {
    saving.value = false
  }
}

// 停用/启用统一走确认 + 失败 reload，避免界面上的开关和服务端状态不一致。
const changeStatus = async (api, id, current, label, reload) => {
  const next = current === 1 ? 0 : 1
  const action = next === 0 ? '停用' : '启用'
  try {
    await ElMessageBox.confirm(`确定${action}${label}吗？`, '提示', { type: 'warning' })
  } catch {
    return
  }
  const res = await api(id, next)
  if (res.code !== 0) return ElMessage.error(res.msg || '操作失败')
  ElMessage.success(`已${action}`)
  await reload()
}

const toggleGoods = (row) => changeStatus(setGoodsStatus, row.ID, row.status, `商品「${row.name}」`, loadGoods)
const togglePartner = (row) => changeStatus(setPartnerStatus, row.ID, row.status, `「${row.name}」`, loadPartners)
const toggleWarehouse = (row) => changeStatus(setWarehouseStatus, row.ID, row.status, `仓库「${row.name}」`, loadWarehouses)

onMounted(() => {
  loadGoods()
  loadWarehouses()
  loadPartners()
})
</script>

<style scoped>
.toolbar {
  display: flex;
  gap: 8px;
  align-items: center;
  margin-bottom: 10px;
}
.spacer {
  flex: 1;
}
.chip {
  display: inline-block;
  border: 1px solid var(--el-border-color-lighter);
  border-radius: 4px;
  padding: 1px 6px;
  margin: 2px 4px 2px 0;
  font-size: 12px;
  color: var(--el-text-color-regular);
}
.chip.off {
  opacity: 0.55;
  text-decoration: line-through;
}
.muted {
  color: var(--el-text-color-placeholder);
  font-size: 12px;
}
.sku-editor {
  display: flex;
  flex-direction: column;
  gap: 8px;
  width: 100%;
}
.sku-row {
  display: flex;
  gap: 8px;
  align-items: center;
}
.rm {
  border: 0;
  background: none;
  color: var(--el-text-color-placeholder);
  cursor: pointer;
  font-size: 16px;
}
</style>
