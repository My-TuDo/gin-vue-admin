<template>
  <div class="purchase">
    <div class="panel">
      <div class="panel-head">
        <span class="title">新建采购单</span>
        <div class="spacer"></div>
        <el-select v-model="warehouseId" placeholder="入库仓库" style="width: 150px">
          <el-option v-for="w in warehouses" :key="w.ID" :label="w.name" :value="w.ID" />
        </el-select>
      </div>

      <div class="rows">
        <div v-for="(l, i) in lines" :key="i" class="row">
          <el-select v-model="l.skuId" filterable placeholder="选择规格（条码/颜色尺码）" style="flex: 1">
            <el-option v-for="s in skus" :key="s.ID" :label="`${s.skuCode}  ${s.color}/${s.size}`" :value="s.ID" />
          </el-select>
          <el-input-number v-model="l.qty" :min="1" size="small" controls-position="right" style="width: 100px" />
          <el-input-number v-model="l.unitCost" :min="0" :precision="2" size="small" controls-position="right"
            style="width: 120px" />
          <MoneyText :value="l.qty * l.unitCost" />
          <button class="rm" @click="lines.splice(i, 1)">×</button>
        </div>
        <el-button link type="primary" @click="addRow">+ 添加一行</el-button>
      </div>

      <div class="panel-foot">
        <span>合计 <MoneyText :value="total" /></span>
        <div class="spacer"></div>
        <el-button @click="lines = []">清空</el-button>
        <el-button type="primary" :loading="saving" @click="submit">提交（草稿）</el-button>
      </div>
    </div>

  </div>
</template>

<script setup>
import { computed, onMounted, ref } from 'vue'
import { ElMessage } from 'element-plus'
import MoneyText from '@/components/jxc/MoneyText.vue'
import { createPurchase, listSkus, listWarehouses } from '@/api/jxc'

const warehouses = ref([])
const warehouseId = ref(null)
const skus = ref([])
const lines = ref([])
const saving = ref(false)

const total = computed(() => lines.value.reduce((s, l) => s + l.qty * l.unitCost, 0))

const addRow = () => lines.value.push({ skuId: null, qty: 1, unitCost: 0 })


const submit = async () => {
  if (!warehouseId.value) return ElMessage.warning('请选择入库仓库')
  const items = lines.value.filter((l) => l.skuId && l.qty > 0)
  if (items.length === 0) return ElMessage.warning('请至少添加一行有效明细')
  saving.value = true
  try {
    const res = await createPurchase({
      warehouseId: warehouseId.value,
      items: items.map((l) => ({ skuId: l.skuId, qty: l.qty, unitCost: l.unitCost }))
    })
    if (res.code !== 0) return ElMessage.error(res.msg || '创建失败')
    ElMessage.success(`已创建 ${res.data.orderNo}，请到「单据」里审核并入库`)
    lines.value = []
  } finally {
    saving.value = false
  }
}




onMounted(async () => {
  const [w, s] = await Promise.all([listWarehouses(), listSkus({})])
  if (w.code === 0) {
    warehouses.value = w.data || []
    if (warehouses.value.length) warehouseId.value = warehouses.value[0].ID
  }
  if (s.code === 0) skus.value = s.data || []
})
</script>

<style scoped>
.purchase {
  display: flex;
  flex-direction: column;
  gap: 16px;
}
.panel {
  background: var(--el-bg-color);
  border: 1px solid var(--el-border-color-lighter);
  border-radius: 8px;
  padding: 12px;
}
.panel-head {
  display: flex;
  align-items: center;
  gap: 8px;
  margin-bottom: 10px;
}
.title {
  font-weight: 600;
}
.spacer {
  flex: 1;
}
.rows {
  display: flex;
  flex-direction: column;
  gap: 8px;
}
.row {
  display: flex;
  align-items: center;
  gap: 8px;
}
.rm {
  border: 0;
  background: none;
  color: var(--el-text-color-placeholder);
  cursor: pointer;
  font-size: 16px;
}
.panel-foot {
  display: flex;
  align-items: center;
  gap: 8px;
  margin-top: 10px;
  padding-top: 10px;
  border-top: 1px solid var(--el-border-color-lighter);
}
.muted {
  color: var(--el-text-color-secondary);
  font-size: 12px;
}
</style>
