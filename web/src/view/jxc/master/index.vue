<template>
  <div class="master">
    <el-tabs v-model="tab">
      <el-tab-pane label="商品与规格" name="goods">
        <div class="toolbar">
          <el-input v-model="kw" placeholder="搜索商品" clearable style="width: 200px" />
          <div class="spacer"></div>
          <el-button type="primary" @click="openGoods">新建商品</el-button>
        </div>

        <el-table v-loading="loadingGoods" :data="filteredGoods" size="small">
          <el-table-column prop="name" label="商品" min-width="160" />
          <el-table-column prop="code" label="商品编码" width="140" />
          <el-table-column label="规格（颜色/尺码）" min-width="280">
            <template #default="{ row }">
              <span v-for="s in row.skus" :key="s.ID" class="chip">
                {{ s.color }}/{{ s.size }} · {{ s.skuCode }} · ¥{{ Number(s.salePrice).toFixed(2) }}
              </span>
              <span v-if="!row.skus?.length" class="muted">无规格</span>
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
          <el-button type="primary" @click="openPartner">新建{{ partnerKind === 'supplier' ? '供应商' : '客户' }}</el-button>
        </div>
        <el-table v-loading="loadingPartner" :data="partners" size="small">
          <el-table-column prop="code" label="编码" width="140" />
          <el-table-column prop="name" label="名称" min-width="160" />
          <el-table-column prop="contact" label="联系人" width="110" />
          <el-table-column prop="phone" label="电话" width="140" />
          <el-table-column prop="address" label="地址" min-width="180" />
        </el-table>
      </el-tab-pane>

      <el-tab-pane label="仓库" name="warehouse">
        <div class="toolbar">
          <div class="spacer"></div>
          <el-button type="primary" @click="openWarehouse">新建仓库</el-button>
        </div>
        <el-table v-loading="loadingWh" :data="warehouses" size="small">
          <el-table-column prop="code" label="编码" width="140" />
          <el-table-column prop="name" label="名称" min-width="180" />
          <el-table-column label="状态" width="90">
            <template #default="{ row }">{{ row.status === 1 ? '启用' : '停用' }}</template>
          </el-table-column>
        </el-table>
      </el-tab-pane>
    </el-tabs>

    <!-- 新建商品：规格一次填完，不弹二次窗 -->
    <el-dialog v-model="goodsDialog" title="新建商品" width="720px">
      <el-form label-width="90px">
        <el-form-item label="商品编码"><el-input v-model="form.code" placeholder="如 WY-001" /></el-form-item>
        <el-form-item label="商品名称"><el-input v-model="form.name" placeholder="如 卫衣" /></el-form-item>
        <el-form-item label="规格">
          <div class="sku-editor">
            <div v-for="(s, i) in form.skus" :key="i" class="sku-row">
              <el-input v-model="s.color" placeholder="颜色" style="width: 110px" />
              <el-input v-model="s.size" placeholder="尺码" style="width: 100px" />
              <el-input v-model="s.skuCode" placeholder="条码（唯一）" style="width: 160px" />
              <el-input-number v-model="s.salePrice" :min="0" :precision="2" size="small" controls-position="right"
                style="width: 120px" />
              <button class="rm" @click="form.skus.splice(i, 1)">×</button>
            </div>
            <el-button link type="primary" @click="form.skus.push({ color: '', size: '', skuCode: '', salePrice: 0 })">
              + 添加规格
            </el-button>
          </div>
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="goodsDialog = false">取消</el-button>
        <el-button type="primary" :loading="saving" @click="submitGoods">保存</el-button>
      </template>
    </el-dialog>

    <el-dialog v-model="partnerDialog" :title="`新建${partnerKind === 'supplier' ? '供应商' : '客户'}`" width="460px">
      <el-form label-width="80px">
        <el-form-item label="编码"><el-input v-model="partner.code" placeholder="如 GYS-001" /></el-form-item>
        <el-form-item label="名称"><el-input v-model="partner.name" /></el-form-item>
        <el-form-item label="联系人"><el-input v-model="partner.contact" /></el-form-item>
        <el-form-item label="电话"><el-input v-model="partner.phone" /></el-form-item>
        <el-form-item label="地址"><el-input v-model="partner.address" /></el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="partnerDialog = false">取消</el-button>
        <el-button type="primary" :loading="saving" @click="submitPartner">保存</el-button>
      </template>
    </el-dialog>

    <el-dialog v-model="whDialog" title="新建仓库" width="420px">
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
import { ElMessage } from 'element-plus'
import { createGoods, createPartner, createWarehouse, listGoods, listPartners, listWarehouses } from '@/api/jxc'

const tab = ref('goods')
const kw = ref('')
const goods = ref([])
const warehouses = ref([])
const loadingGoods = ref(false)
const loadingWh = ref(false)
const saving = ref(false)

const goodsDialog = ref(false)
const whDialog = ref(false)

const form = reactive({ code: '', name: '', skus: [] })
const partners = ref([])
const partnerKind = ref('supplier')
const partnerDialog = ref(false)
const loadingPartner = ref(false)
const partner = reactive({ code: '', name: '', contact: '', phone: '', address: '' })
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

const openGoods = () => {
  form.code = ''
  form.name = ''
  form.skus = [{ color: '', size: '', skuCode: '', salePrice: 0 }]
  goodsDialog.value = true
}

const submitGoods = async () => {
  if (!form.code || !form.name) return ElMessage.warning('商品编码与名称必填')
  const skus = form.skus.filter((s) => s.skuCode)
  if (skus.length === 0) return ElMessage.warning('至少填写一个规格条码')
  saving.value = true
  try {
    const res = await createGoods({ code: form.code, name: form.name, skus })
    if (res.code !== 0) return ElMessage.error(res.msg || '保存失败')
    ElMessage.success('已创建')
    goodsDialog.value = false
    await loadGoods()
  } finally {
    saving.value = false
  }
}

const openWarehouse = () => {
  wh.code = ''
  wh.name = ''
  whDialog.value = true
}

const submitWh = async () => {
  if (!wh.code || !wh.name) return ElMessage.warning('编码与名称必填')
  saving.value = true
  try {
    const res = await createWarehouse({ code: wh.code, name: wh.name })
    if (res.code !== 0) return ElMessage.error(res.msg || '保存失败')
    ElMessage.success('已创建')
    whDialog.value = false
    await loadWarehouses()
  } finally {
    saving.value = false
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

const openPartner = () => {
  Object.assign(partner, { code: '', name: '', contact: '', phone: '', address: '' })
  partnerDialog.value = true
}

const submitPartner = async () => {
  if (!partner.code || !partner.name) return ElMessage.warning('编码与名称必填')
  saving.value = true
  try {
    const res = await createPartner({ ...partner, kind: partnerKind.value })
    if (res.code !== 0) return ElMessage.error(res.msg || '保存失败')
    ElMessage.success('已创建')
    partnerDialog.value = false
    await loadPartners()
  } finally {
    saving.value = false
  }
}

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
.muted {
  color: var(--el-text-color-placeholder);
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
