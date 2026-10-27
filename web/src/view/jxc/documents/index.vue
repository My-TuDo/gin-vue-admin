<template>
  <div class="flex h-full min-h-0 flex-col gap-4">
    <PageHeader
      eyebrow="DOCUMENTS"
      title="单据"
      description="采购、销售、退货与红冲都在这里推进到下一步；点单号查看完整明细与账本影响。"
    >
      <template #actions>
        <el-button @click="load">
          <svg-icon icon="lucide:refresh-cw" class="mr-1.5 h-3.5 w-3.5" />
          刷新
        </el-button>
      </template>
    </PageHeader>

    <SectionCard :padded="false" class="min-h-0 flex-1">
      <template #header>
        <div class="flex flex-wrap items-center gap-1">
          <button
            v-for="item in TABS"
            :key="item.key"
            type="button"
            class="flex h-8 items-center gap-1.5 rounded-md px-3 text-[12.5px] transition-colors"
            :class="tab === item.key ? 'bg-primary-50 font-medium text-primary-700' : 'text-muted-foreground hover:bg-muted hover:text-base-text'"
            @click="switchTab(item.key)"
          >
            <svg-icon :icon="item.icon" class="h-3.5 w-3.5" />
            {{ item.label }}
            <span class="rounded bg-muted px-1.5 py-0.5 text-[10px] tabular-nums" :class="tab === item.key ? 'bg-primary-100 text-primary-700' : ''">
              {{ tabCount(item.key) }}
            </span>
          </button>
        </div>
        <div class="ml-auto flex items-center gap-2">
          <span class="hidden items-center gap-1.5 text-[11px] text-muted-foreground lg:flex">
            <svg-icon icon="lucide:mouse-pointer-click" class="h-3.5 w-3.5" />
            点行查看明细
          </span>
        </div>
      </template>

      <div class="flex h-full min-h-0 flex-col">
        <div class="flex flex-wrap items-center gap-3 border-b border-border px-4 py-2.5">
          <div class="w-[260px] shrink-0">
            <el-input v-model="keyword" class="w-full" placeholder="搜索单号、往来单位、原因" clearable>
              <template #prefix><svg-icon icon="lucide:search" class="h-3.5 w-3.5" /></template>
            </el-input>
          </div>

          <div v-if="statusOptions.length > 1" class="flex flex-wrap items-center gap-1">
            <button
              v-for="option in statusOptions"
              :key="option.value"
              type="button"
              class="flex h-7 items-center gap-1 rounded-md px-2.5 text-[11.5px] transition-colors"
              :class="status === option.value ? 'bg-muted font-medium text-base-text' : 'text-muted-foreground hover:bg-muted hover:text-base-text'"
              @click="status = option.value"
            >
              {{ option.label }}
              <span class="tabular-nums text-[10px] text-muted-foreground">{{ option.count }}</span>
            </button>
          </div>

          <div class="ml-auto text-[11px] text-muted-foreground">
            {{ tabMeta.description }}
          </div>
        </div>

        <div v-loading="loading" class="min-h-0 flex-1 overflow-hidden">
          <el-table
            :data="pageRows"
            height="100%"
            :class="tab !== 'reversal' ? 'el-table--clickable' : ''"
            @row-click="onRowClick"
          >
            <el-table-column v-if="tab === 'sale'" label="单号" width="168">
              <template #default="{ row }">
                <el-button link type="primary" class="font-mono text-[12px]" @click.stop="openDetail('sale', row)">
                  {{ row.orderNo }}
                </el-button>
              </template>
            </el-table-column>
            <el-table-column v-if="tab === 'purchase'" label="单号" width="168">
              <template #default="{ row }">
                <el-button link type="primary" class="font-mono text-[12px]" @click.stop="openDetail('purchase', row)">
                  {{ row.orderNo }}
                </el-button>
              </template>
            </el-table-column>
            <el-table-column v-if="tab === 'return'" prop="returnNo" label="退货单号" width="168">
              <template #default="{ row }">
                <span class="font-mono text-[12px]">{{ row.returnNo }}</span>
              </template>
            </el-table-column>
            <el-table-column v-if="tab === 'purchaseReturn'" label="退货单号" width="160">
              <template #default="{ row }">
                <el-button link type="primary" class="font-mono text-[12px]" @click.stop="openDetail('purchaseReturn', row)">
                  {{ row.returnNo }}
                </el-button>
              </template>
            </el-table-column>
            <el-table-column v-if="tab === 'reversal'" prop="reversalNo" label="红冲单号" width="168">
              <template #default="{ row }">
                <span class="font-mono text-[12px]">{{ row.reversalNo }}</span>
              </template>
            </el-table-column>

            <el-table-column label="状态" width="100">
              <template #default="{ row }">
                <StatusTag
                  v-if="row.reversedAt"
                  kind="reversal"
                  :status="1"
                  :title="`原状态：${statusMeta(tab, row.status).label}`"
                />
                <StatusTag v-else :kind="tab" :status="row.status" />
              </template>
            </el-table-column>

            <el-table-column v-if="tab === 'sale'" label="客户" min-width="100">
              <template #default="{ row }">{{ customerName(row.customerId) }}</template>
            </el-table-column>
            <el-table-column v-if="tab === 'purchase'" label="供应商" min-width="100">
              <template #default="{ row }">{{ partnerName(row.supplierId) }}</template>
            </el-table-column>
            <el-table-column v-if="tab === 'purchase'" prop="remark" label="备注" min-width="160" show-overflow-tooltip>
              <template #default="{ row }"><span class="text-[11.5px] text-muted-foreground">{{ row.remark || '—' }}</span></template>
            </el-table-column>
            <el-table-column v-if="tab === 'purchaseReturn'" label="原采购单" width="160">
              <template #default="{ row }"><span class="font-mono text-[12px]">{{ row.purchaseNo }}</span></template>
            </el-table-column>
            <el-table-column v-if="tab === 'purchaseReturn'" label="供应商" min-width="90">
              <template #default="{ row }">{{ partnerName(row.supplierId) }}</template>
            </el-table-column>
            <el-table-column v-if="tab === 'return'" prop="saleNo" label="原销售单" width="168">
              <template #default="{ row }"><span class="font-mono text-[12px]">{{ row.saleNo }}</span></template>
            </el-table-column>
            <el-table-column v-if="tab === 'reversal'" label="来源" width="100">
              <template #default="{ row }">
                <span class="text-[12px]">{{ row.sourceType === 'sale' ? '销售单' : '采购单' }}</span>
              </template>
            </el-table-column>
            <el-table-column v-if="tab === 'reversal'" prop="sourceNo" label="原单号" width="168">
              <template #default="{ row }"><span class="font-mono text-[12px]">{{ row.sourceNo }}</span></template>
            </el-table-column>

            <el-table-column v-if="tab === 'sale'" label="销售额" width="96" align="right">
              <template #default="{ row }"><MoneyText :value="row.totalAmount" size="sm" /></template>
            </el-table-column>
            <el-table-column v-if="tab === 'sale'" label="成本" width="88" align="right">
              <template #default="{ row }"><MoneyText :value="row.costAmount" size="sm" /></template>
            </el-table-column>
            <el-table-column v-if="tab === 'sale'" label="毛利" width="88" align="right">
              <template #default="{ row }"><MoneyText :value="round2(row.totalAmount - row.costAmount)" size="sm" strong /></template>
            </el-table-column>
            <el-table-column v-if="tab === 'purchase'" label="金额" width="96" align="right">
              <template #default="{ row }"><MoneyText :value="row.totalAmount" size="sm" /></template>
            </el-table-column>
            <el-table-column v-if="tab === 'purchase'" label="明细" width="76">
              <template #default="{ row }">{{ row.items?.length || 0 }} 项</template>
            </el-table-column>
            <el-table-column v-if="tab === 'purchaseReturn'" label="退货金额" width="90" align="right">
              <template #default="{ row }"><MoneyText :value="row.amount" size="sm" /></template>
            </el-table-column>
            <el-table-column v-if="tab === 'purchaseReturn'" prop="reason" label="原因" min-width="130" show-overflow-tooltip>
              <template #default="{ row }"><span class="text-[11.5px] text-muted-foreground">{{ row.reason || '—' }}</span></template>
            </el-table-column>
            <el-table-column v-if="tab === 'return'" label="退款" width="96" align="right">
              <template #default="{ row }"><MoneyText :value="row.amount" size="sm" /></template>
            </el-table-column>
            <el-table-column v-if="tab === 'return'" label="退回成本" width="96" align="right">
              <template #default="{ row }"><MoneyText :value="row.costAmount" size="sm" /></template>
            </el-table-column>
            <el-table-column v-if="tab === 'return'" prop="reason" label="原因" min-width="110" show-overflow-tooltip />
            <el-table-column v-if="tab === 'reversal'" label="冲销金额" width="96" align="right">
              <template #default="{ row }"><MoneyText :value="row.amount" size="sm" /></template>
            </el-table-column>
            <el-table-column v-if="tab === 'reversal'" label="件数" width="76" align="right">
              <template #default="{ row }"><span class="tabular-nums">{{ row.qty }}</span></template>
            </el-table-column>
            <el-table-column v-if="tab === 'reversal'" prop="reason" label="原因" min-width="160" show-overflow-tooltip />

            <el-table-column v-if="tab !== 'reversal'" label="过账时间" width="132">
              <template #default="{ row }">
                <span class="text-[12px] tabular-nums text-muted-foreground">{{ formatDateTime(row.postedAt) || '—' }}</span>
              </template>
            </el-table-column>
            <el-table-column v-if="tab === 'reversal'" label="时间" width="132">
              <template #default="{ row }">
                <span class="text-[12px] tabular-nums text-muted-foreground">{{ formatDateTime(row.postedAt) }}</span>
              </template>
            </el-table-column>

            <el-table-column v-if="tab !== 'reversal'" label="操作" min-width="150" fixed="right">
              <template #default="{ row }">
                <div class="flex items-center gap-2">
                  <template v-if="tab === 'sale'">
                    <el-button v-if="row.status === 1" link type="primary" @click.stop="ship(row)">出库</el-button>
                    <el-button v-if="row.status === 1" link type="danger" @click.stop="cancelSaleOrder(row)">取消</el-button>
                    <el-button v-if="row.status === 2 && !row.reversedAt" link @click.stop="openReturn(row)">退货</el-button>
                    <el-button v-if="row.status === 2 && !row.reversedAt" link type="danger" @click.stop="openReversal('sale', row)">红冲</el-button>
                    <span v-if="row.reversedAt" class="text-[11px] text-muted-foreground">已冲销</span>
                    <span v-else-if="row.status === 3" class="text-[11px] text-muted-foreground">—</span>
                  </template>
                  <template v-else-if="tab === 'purchase'">
                    <el-button v-if="row.status === 1" link type="primary" @click.stop="approvePurchaseRow(row)">审核</el-button>
                    <el-button v-if="row.status === 2" link type="success" @click.stop="receivePurchaseRow(row)">入库</el-button>
                    <el-button v-if="row.status === 1 || row.status === 2" link type="danger" @click.stop="cancelPurchaseRow(row)">取消</el-button>
                    <el-button v-if="row.status === 3 && !row.reversedAt" link type="primary" @click.stop="openPurchaseReturn(row)">退货</el-button>
                    <el-button v-if="row.status === 3 && !row.reversedAt" link type="danger" @click.stop="openReversal('purchase', row)">红冲</el-button>
                    <span v-if="row.reversedAt" class="text-[11px] text-muted-foreground">已冲销</span>
                    <span v-else-if="row.status === 4" class="text-[11px] text-muted-foreground">—</span>
                  </template>
                  <template v-else-if="tab === 'purchaseReturn'">
                    <el-button v-if="row.status === 1" link type="success" @click.stop="postPurchaseReturnRow(row)">出库</el-button>
                    <el-button v-if="row.status === 1" link type="danger" @click.stop="cancelPurchaseReturnRow(row)">取消</el-button>
                    <span v-if="row.status !== 1" class="text-[11px] text-muted-foreground">已结束</span>
                  </template>
                  <template v-else-if="tab === 'return'">
                    <el-button v-if="row.status === 1" link type="success" @click.stop="doPostReturn(row)">入库</el-button>
                    <el-button v-if="row.status === 1" link type="danger" @click.stop="doCancelReturn(row)">取消</el-button>
                    <span v-if="row.status !== 1" class="text-[11px] text-muted-foreground">已结束</span>
                  </template>
                </div>
              </template>
            </el-table-column>

            <template #empty>
              <EmptyState
                icon="lucide:files"
                title="没有符合条件单据"
                :description="tab === 'purchaseReturn'
                  ? '从「采购单」页签的已入库行点「退货」发起；出库后才扣减库存。'
                  : '调整状态筛选或搜索词；新开的单据会出现在这里。'"
              >
                <el-button v-if="tab === 'purchaseReturn'" @click="switchTab('purchase')">去采购单发起退货</el-button>
              </EmptyState>
            </template>
          </el-table>
        </div>

        <div class="flex shrink-0 items-center justify-between border-t border-border px-4 py-2.5">
          <span class="text-[11px] text-muted-foreground">共 {{ activeRows.length }} 条 · 按创建时间倒序</span>
          <el-pagination
            v-model:current-page="page"
            :page-size="pageSize"
            :total="activeRows.length"
            layout="prev, pager, next"
            background
          />
        </div>
      </div>
    </SectionCard>

    <!-- 单据详情 -->
    <el-drawer v-model="detailOpen" :title="detailTitle" size="780px" destroy-on-close>
      <div v-loading="detailLoading" class="min-h-[200px]">
        <template v-if="detail">
          <div class="flex flex-wrap items-center gap-2">
            <StatusTag :kind="detailKind" :status="detail.status" />
            <StatusTag v-if="detail.reversedAt" kind="reversal" :status="1" />
            <span class="ml-auto text-[11px] text-muted-foreground">制单 {{ formatDateTime(detail.CreatedAt) }}</span>
          </div>

          <div class="mt-4 grid grid-cols-2 gap-x-8 gap-y-3 rounded-lg border border-border bg-muted px-4 py-3 text-[12px]">
            <div class="flex gap-3"><span class="w-16 shrink-0 text-muted-foreground">仓库</span><span>{{ warehouseName(detail.warehouseId) }}</span></div>
            <div v-if="detailKind === 'sale'" class="flex gap-3"><span class="w-16 shrink-0 text-muted-foreground">客户</span><span>{{ customerName(detail.customerId) }}</span></div>
            <div v-if="detailKind === 'purchase'" class="flex gap-3"><span class="w-16 shrink-0 text-muted-foreground">供应商</span><span>{{ partnerName(detail.supplierId) }}</span></div>
            <div v-if="detailKind === 'purchaseReturn'" class="flex gap-3"><span class="w-16 shrink-0 text-muted-foreground">原采购单</span><span class="font-mono">{{ detail.purchaseNo }}</span></div>
            <div v-if="detailKind === 'purchaseReturn'" class="flex gap-3"><span class="w-16 shrink-0 text-muted-foreground">供应商</span><span>{{ partnerName(detail.supplierId) }}</span></div>
            <div v-if="detailKind === 'return'" class="flex gap-3"><span class="w-16 shrink-0 text-muted-foreground">原销售单</span><span class="font-mono">{{ detail.saleNo }}</span></div>
            <div class="flex gap-3"><span class="w-16 shrink-0 text-muted-foreground">过账时间</span><span class="tabular-nums">{{ formatDateTime(detail.postedAt) || '—' }}</span></div>
            <div v-if="detailKind !== 'purchaseReturn'" class="flex gap-3"><span class="w-16 shrink-0 text-muted-foreground">备注</span><span class="min-w-0 break-words">{{ detail.remark || '—' }}</span></div>
            <div v-if="detailKind === 'purchaseReturn'" class="flex gap-3"><span class="w-16 shrink-0 text-muted-foreground">原因</span><span class="min-w-0 break-words">{{ detail.reason || '—' }}</span></div>
          </div>

          <div class="mt-4 grid gap-3" :class="detailKind === 'sale' ? 'grid-cols-3' : 'grid-cols-2'">
            <div v-if="detailKind === 'sale'" class="rounded-lg border border-border px-4 py-3">
              <div class="text-[11px] text-muted-foreground">销售额</div>
              <MoneyText :value="detail.totalAmount" size="lg" strong class="mt-1" />
            </div>
            <div v-if="detailKind === 'sale'" class="rounded-lg border border-border px-4 py-3">
              <div class="text-[11px] text-muted-foreground">出库成本</div>
              <MoneyText :value="detail.costAmount" size="lg" class="mt-1" />
            </div>
            <div v-if="detailKind === 'sale'" class="rounded-lg border border-border px-4 py-3">
              <div class="text-[11px] text-muted-foreground">毛利</div>
              <MoneyText :value="round2(detail.totalAmount - detail.costAmount)" size="lg" strong class="mt-1" />
            </div>
            <div v-if="detailKind === 'purchase'" class="rounded-lg border border-border px-4 py-3">
              <div class="text-[11px] text-muted-foreground">采购金额</div>
              <MoneyText :value="detail.totalAmount" size="lg" strong class="mt-1" />
            </div>
            <div v-if="detailKind === 'purchase'" class="rounded-lg border border-border px-4 py-3">
              <div class="text-[11px] text-muted-foreground">明细行</div>
              <div class="mt-1 text-[16px] font-semibold tabular-nums">{{ detail.items?.length || 0 }}</div>
            </div>
            <div v-if="detailKind === 'purchaseReturn'" class="rounded-lg border border-border px-4 py-3">
              <div class="text-[11px] text-muted-foreground">退货金额（原采购价）</div>
              <MoneyText :value="detail.amount" size="lg" strong class="mt-1" />
            </div>
            <div v-if="detailKind === 'purchaseReturn'" class="rounded-lg border border-border px-4 py-3">
              <div class="text-[11px] text-muted-foreground">明细行</div>
              <div class="mt-1 text-[16px] font-semibold tabular-nums">{{ detail.items?.length || 0 }}</div>
            </div>
            <div v-if="detailKind === 'return'" class="rounded-lg border border-border px-4 py-3">
              <div class="text-[11px] text-muted-foreground">退款金额</div>
              <MoneyText :value="detail.amount" size="lg" strong class="mt-1" />
            </div>
            <div v-if="detailKind === 'return'" class="rounded-lg border border-border px-4 py-3">
              <div class="text-[11px] text-muted-foreground">退回成本</div>
              <MoneyText :value="detail.costAmount" size="lg" class="mt-1" />
            </div>
          </div>

          <div class="mt-5">
            <h3 class="mb-2 text-[13px] font-semibold">明细</h3>
            <el-table :data="detail.items || []" size="small" max-height="360">
              <el-table-column label="商品 / 规格" min-width="200">
                <template #default="{ row }">
                  <span>{{ skuInfo(row.skuId).goodsName }}</span>
                  <span class="ml-1 text-[11px] text-muted-foreground">{{ skuInfo(row.skuId).color }}/{{ skuInfo(row.skuId).size }}</span>
                </template>
              </el-table-column>
              <el-table-column label="条码" width="140">
                <template #default="{ row }"><span class="font-mono text-[11px] text-muted-foreground">{{ skuInfo(row.skuId).skuCode }}</span></template>
              </el-table-column>
              <el-table-column prop="qty" label="数量" width="72" align="right" />
              <el-table-column v-if="detailKind !== 'return'" :label="detailKind === 'purchaseReturn' ? '原采购单价' : '单价'" width="104" align="right">
                <template #default="{ row }"><MoneyText :value="detailKind === 'sale' ? row.price : row.unitCost" size="sm" /></template>
              </el-table-column>
              <el-table-column v-if="detailKind !== 'return'" label="金额" width="104" align="right">
                <template #default="{ row }"><MoneyText :value="row.amount" size="sm" /></template>
              </el-table-column>
              <el-table-column v-if="detailKind === 'sale'" label="该行成本" width="104" align="right">
                <template #default="{ row }"><MoneyText :value="row.costAmt" size="sm" /></template>
              </el-table-column>
              <el-table-column v-if="detailKind === 'return'" label="退回单价" width="94" align="right">
                <template #default="{ row }"><MoneyText :value="row.price" size="sm" /></template>
              </el-table-column>
              <el-table-column v-if="detailKind === 'return'" label="退款" width="104" align="right">
                <template #default="{ row }"><MoneyText :value="row.amount" size="sm" /></template>
              </el-table-column>
              <el-table-column v-if="detailKind === 'return'" label="回补成本" width="104" align="right">
                <template #default="{ row }"><MoneyText :value="row.costAmount" size="sm" /></template>
              </el-table-column>
            </el-table>
          </div>
        </template>
      </div>

      <template #footer>
        <div class="flex items-center justify-end gap-2">
          <el-button @click="detailOpen = false">关闭</el-button>
          <el-button
            v-for="action in detailActions"
            :key="action.key"
            :type="action.type"
            @click="onDetailAction(action.key)"
          >{{ action.label }}</el-button>
        </div>
      </template>
    </el-drawer>

    <!-- 红冲 -->
    <el-dialog v-model="revDialog" title="整单红冲" width="460px">
      <div class="space-y-3">
        <div class="rounded-lg border border-warning-200 bg-warning-50 px-3 py-2.5 text-[12px] leading-5 text-warning-800">
          红冲会生成反向账本记录。原单作废但历史保留，操作不可撤销。若只是部分退货，请用「退货」。
        </div>
        <div class="flex items-center justify-between rounded-lg border border-border px-3 py-2.5 text-[12px]">
          <span class="text-muted-foreground">单据</span>
          <span class="font-mono">{{ revTarget?.orderNo || revTarget?.returnNo }}</span>
        </div>
        <label class="block">
          <span class="mb-1.5 block text-[11px] text-muted-foreground">红冲原因（必填）</span>
          <el-input v-model="revReason" type="textarea" :rows="3" placeholder="例如：供应商发错货 / 单据录错" />
        </label>
      </div>
      <template #footer>
        <el-button @click="revDialog = false">取消</el-button>
        <el-button type="danger" :loading="saving" @click="submitReversal">确认红冲</el-button>
      </template>
    </el-dialog>

    <!-- 退货 -->
    <el-dialog v-model="retDialog" title="销售退货" width="760px">
      <div class="space-y-3">
        <div class="flex flex-wrap items-center justify-between gap-2 rounded-lg border border-border bg-muted px-3 py-2.5 text-[12px]">
          <span>原销售单 <span class="font-mono">{{ retTarget?.orderNo }}</span></span>
          <span class="text-muted-foreground">成本按原单出库时冻结的金额回补，不用当前均价</span>
        </div>
        <el-table :data="returnable" size="small" max-height="320">
          <el-table-column label="商品 / 规格" min-width="180">
            <template #default="{ row }">
              <span>{{ skuInfo(row.skuId).goodsName }}</span>
              <span class="ml-1 text-[11px] text-muted-foreground">{{ skuInfo(row.skuId).color }}/{{ skuInfo(row.skuId).size }}</span>
            </template>
          </el-table-column>
          <el-table-column prop="soldQty" label="原售" width="64" align="right" />
          <el-table-column prop="returnedQty" label="已退" width="64" align="right" />
          <el-table-column prop="canReturn" label="可退" width="64" align="right" />
          <el-table-column label="单价" width="90" align="right">
            <template #default="{ row }"><MoneyText :value="row.price" size="sm" /></template>
          </el-table-column>
          <el-table-column label="退货数量" width="132">
            <template #default="{ row }">
              <el-input-number v-model="row.qty" :min="0" :max="row.canReturn" :disabled="row.canReturn === 0" class="w-full" />
            </template>
          </el-table-column>
        </el-table>
        <label class="block">
          <span class="mb-1.5 block text-[11px] text-muted-foreground">退货原因</span>
          <el-input v-model="retReason" placeholder="例如：尺码不合适" />
        </label>
      </div>
      <template #footer>
        <el-button @click="retDialog = false">取消</el-button>
        <el-button type="primary" :loading="saving" @click="submitReturn">创建退货单</el-button>
      </template>
    </el-dialog>

    <!-- 采购退货：按原采购价部分退货 -->
    <el-dialog v-model="prDialog" :title="`采购退货 · ${prTarget?.orderNo || ''}`" width="820px">
      <div class="space-y-3">
        <div class="flex flex-wrap items-center justify-between gap-2 rounded-lg border border-border bg-muted px-3 py-2.5 text-[12px]">
          <span>原采购单 <span class="font-mono">{{ prTarget?.orderNo }}</span></span>
          <span class="text-muted-foreground">按原采购单价出库；创建后为草稿，出库才扣库存</span>
        </div>
        <div class="grid grid-cols-2 gap-x-6 gap-y-1 text-[12px]">
          <div class="flex gap-3"><span class="w-14 text-muted-foreground">供应商</span><span>{{ partnerName(prTarget?.supplierId) }}</span></div>
          <div class="flex gap-3"><span class="w-14 text-muted-foreground">仓库</span><span>{{ warehouseName(prTarget?.warehouseId) }}</span></div>
        </div>

        <el-table :data="prReturnable" size="small" max-height="320">
          <el-table-column label="商品 / 规格" min-width="180">
            <template #default="{ row }">
              <span>{{ skuInfo(row.skuId).goodsName }}</span>
              <span class="ml-1 text-[11px] text-muted-foreground">{{ skuInfo(row.skuId).color }}/{{ skuInfo(row.skuId).size }}</span>
            </template>
          </el-table-column>
          <el-table-column prop="purchasedQty" label="原采购" width="68" align="right" />
          <el-table-column prop="returnedQty" label="已退" width="60" align="right" />
          <el-table-column prop="canReturn" label="可退" width="60" align="right" />
          <el-table-column label="原采购单价" width="104" align="right">
            <template #default="{ row }"><MoneyText :value="row.unitCost" size="sm" /></template>
          </el-table-column>
          <el-table-column label="退货数量" width="122">
            <template #default="{ row }">
              <el-input-number v-model="row.qty" :min="0" :max="row.canReturn" :disabled="row.canReturn === 0" class="w-full" />
            </template>
          </el-table-column>
          <el-table-column label="金额" width="96" align="right">
            <template #default="{ row }"><MoneyText :value="round2(Number(row.qty) * Number(row.unitCost))" size="sm" /></template>
          </el-table-column>
          <template #empty>
            <div class="py-6 text-center text-[12px] text-muted-foreground">该采购单没有可退明细</div>
          </template>
        </el-table>

        <div class="flex items-center justify-between border-t border-border pt-3">
          <span class="text-[12px] text-muted-foreground">预计退货金额</span>
          <MoneyText :value="prTotal" size="lg" strong />
        </div>

        <label class="block">
          <span class="mb-1.5 block text-[11px] text-muted-foreground">退货原因</span>
          <el-input v-model="prReason" placeholder="例如：质量问题 / 尺码不合适 / 多订" />
        </label>
      </div>
      <template #footer>
        <el-button @click="prDialog = false">取消</el-button>
        <el-button type="primary" :loading="saving" :disabled="prTotal <= 0" @click="submitPurchaseReturn">创建退货单</el-button>
      </template>
    </el-dialog>
  </div>
</template>

<script setup>
import { computed, onMounted, ref, watch } from 'vue'
import { useRoute } from 'vue-router'
import { ElMessage, ElMessageBox } from 'element-plus'
import PageHeader from '@/components/jxc/PageHeader.vue'
import SectionCard from '@/components/jxc/SectionCard.vue'
import EmptyState from '@/components/jxc/EmptyState.vue'
import MoneyText from '@/components/jxc/MoneyText.vue'
import StatusTag from '@/components/jxc/StatusTag.vue'
import {
  approvePurchase, cancelPurchase, cancelPurchaseReturn, cancelReturn, cancelSale,
  createPurchaseReturn, createReturn, createReversal, getPurchase, getPurchaseReturn,
  getPurchaseReturnable, getReturn, getReturnable, getSale, listGoods, listPartners, listWarehouses,
  pagePurchase, pagePurchaseReturn, pageReturn, pageReversal, pageSale,
  postPurchaseReturn, postReturn, receivePurchase, shipSale
} from '@/api/jxc'
import { formatDateTime, round2 } from '@/view/jxc/shared/format'
import { statusMeta } from '@/view/jxc/shared/status'

const route = useRoute()
const loading = ref(false)
const saving = ref(false)
const tab = ref('sale')
const status = ref('')
const keyword = ref('')
const page = ref(1)
const pageSize = 20

const sales = ref([])
const purchases = ref([])
const purchaseReturns = ref([])
const returns = ref([])
const reversals = ref([])
const warehouseMap = ref({})
const partnerMap = ref({})
const skuMap = ref({})

const TABS = [
  { key: 'sale', label: '销售单', icon: 'lucide:shopping-cart', description: '开单即预留，出库后计入销售与成本' },
  { key: 'purchase', label: '采购单', icon: 'lucide:package-plus', description: '审核不动库存，入库才产生账本行' },
  { key: 'purchaseReturn', label: '采购退货', icon: 'lucide:package-minus', description: '按原采购价部分退货，出库才扣减库存' },
  { key: 'return', label: '销售退货', icon: 'lucide:rotate-ccw', description: '按原单冻结成本回补，退款线下完成' },
  { key: 'reversal', label: '红冲记录', icon: 'lucide:history', description: '整单反向记账，原单保留可追溯' }
]
const TAB_MAP = Object.fromEntries(TABS.map((item) => [item.key, item]))
const tabMeta = computed(() => TAB_MAP[tab.value] || TABS[0])

const sourceRows = computed(() => {
  if (tab.value === 'sale') return sales.value
  if (tab.value === 'purchase') return purchases.value
  if (tab.value === 'purchaseReturn') return purchaseReturns.value
  if (tab.value === 'return') return returns.value
  return reversals.value
})

const haystack = (row) => [
  row.orderNo, row.returnNo, row.reversalNo, row.saleNo, row.purchaseNo, row.sourceNo,
  row.reason, customerName(row.customerId), partnerName(row.supplierId),
  row.remark, row.sourceType === 'sale' ? '销售单' : row.sourceType === 'purchase' ? '采购单' : ''
].filter(Boolean).join(' ').toLowerCase()

const tabRows = computed(() => {
  const k = keyword.value.trim().toLowerCase()
  if (!k) return sourceRows.value
  return sourceRows.value.filter((row) => haystack(row).includes(k))
})

const activeRows = computed(() => {
  if (!status.value) return tabRows.value
  return tabRows.value.filter((row) => String(row.status) === status.value)
})

const pageRows = computed(() => {
  const start = (page.value - 1) * pageSize
  return activeRows.value.slice(start, start + pageSize)
})

const tabCount = (key) => {
  if (key === 'sale') return sales.value.length
  if (key === 'purchase') return purchases.value.length
  if (key === 'purchaseReturn') return purchaseReturns.value.length
  if (key === 'return') return returns.value.length
  return reversals.value.length
}

const STATUS_OPTIONS = {
  sale: [
    { value: '', label: '全部' },
    { value: '1', label: '待出库', status: 1 },
    { value: '2', label: '已出库', status: 2 },
    { value: '3', label: '已取消', status: 3 }
  ],
  purchase: [
    { value: '', label: '全部' },
    { value: '1', label: '草稿', status: 1 },
    { value: '2', label: '已审核', status: 2 },
    { value: '3', label: '已入库', status: 3 },
    { value: '4', label: '已取消', status: 4 }
  ],
  purchaseReturn: [
    { value: '', label: '全部' },
    { value: '1', label: '草稿', status: 1 },
    { value: '2', label: '已退货', status: 2 },
    { value: '3', label: '已取消', status: 3 }
  ],
  return: [
    { value: '', label: '全部' },
    { value: '1', label: '待入库', status: 1 },
    { value: '2', label: '已入库', status: 2 },
    { value: '3', label: '已取消', status: 3 }
  ],
  reversal: [{ value: '', label: '全部' }]
}

const statusOptions = computed(() => {
  const defs = STATUS_OPTIONS[tab.value] || []
  const counts = {}
  for (const row of tabRows.value) {
    const key = String(row.status ?? '')
    counts[key] = (counts[key] || 0) + 1
  }
  return defs.map((item) => ({
    ...item,
    count: item.value === '' ? tabRows.value.length : (counts[item.value] || 0)
  }))
})

const switchTab = (key) => {
  tab.value = key
  status.value = ''
  page.value = 1
}

const partnerName = (id) => (id ? partnerMap.value[String(id)]?.name || '—' : '—')
const customerName = (id) => (id ? partnerName(id) : '散客')
const warehouseName = (id) => warehouseMap.value[String(id)]?.name || '—'
const skuInfo = (id) => skuMap.value[String(id)] || { goodsName: '未知规格', color: '', size: '', skuCode: '' }

const applyQuery = () => {
  const q = route.query
  tab.value = typeof q.tab === 'string' && TAB_MAP[q.tab] ? q.tab : 'sale'
  status.value = typeof q.status === 'string' ? q.status : ''
  keyword.value = typeof q.keyword === 'string' ? q.keyword : ''
  page.value = 1
}

const load = async () => {
  loading.value = true
  try {
    const [saleRes, purchaseRes, purchaseReturnRes, returnRes, reversalRes, warehouseRes, supplierRes, customerRes, goodsRes] = await Promise.all([
      pageSale({ page: 1, pageSize: 100 }),
      pagePurchase({ page: 1, pageSize: 100 }),
      pagePurchaseReturn({ page: 1, pageSize: 100 }),
      pageReturn({ page: 1, pageSize: 100 }),
      pageReversal({ page: 1, pageSize: 100 }),
      listWarehouses(),
      listPartners('supplier'),
      listPartners('customer'),
      listGoods()
    ])

    // 采购退货只对老板/店长开放；店员打开单据页时接口会返回权限错误（code 7），
    // 不要因此让整页弹「部分单据加载失败」。选项卡仍展示为空，操作时由后端再次拦截。
    const purchaseReturnBlocked = purchaseReturnRes.code === 7
    if (saleRes.code !== 0 || purchaseRes.code !== 0 || returnRes.code !== 0 || reversalRes.code !== 0 ||
      (purchaseReturnRes.code !== 0 && !purchaseReturnBlocked)) {
      ElMessage.error('部分单据加载失败，请刷新重试')
    }
    sales.value = saleRes.code === 0 ? saleRes.data.list || [] : []
    purchases.value = purchaseRes.code === 0 ? purchaseRes.data.list || [] : []
    purchaseReturns.value = purchaseReturnRes.code === 0 ? purchaseReturnRes.data.list || [] : []
    returns.value = returnRes.code === 0 ? returnRes.data.list || [] : []
    reversals.value = reversalRes.code === 0 ? reversalRes.data.list || [] : []

    if (warehouseRes.code === 0) {
      warehouseMap.value = Object.fromEntries((warehouseRes.data || []).map((item) => [String(item.ID), item]))
    }
    const partners = [
      ...(supplierRes.code === 0 ? supplierRes.data || [] : []),
      ...(customerRes.code === 0 ? customerRes.data || [] : [])
    ]
    partnerMap.value = Object.fromEntries(partners.map((item) => [String(item.ID), item]))

    if (goodsRes.code === 0) {
      const map = {}
      for (const product of goodsRes.data || []) {
        for (const sku of product.skus || []) {
          map[String(sku.ID)] = { ...sku, goodsName: product.name, goodsCode: product.code }
        }
      }
      skuMap.value = map
    }
  } finally {
    loading.value = false
  }
}

// ---------- 详情 ----------
const detailOpen = ref(false)
const detailLoading = ref(false)
const detailKind = ref('sale')
const detail = ref(null)
const detailTitle = computed(() => {
  if (!detail.value) return '单据详情'
  if (detailKind.value === 'return') return detail.value.returnNo || '销售退货单'
  if (detailKind.value === 'purchaseReturn') return detail.value.returnNo || '采购退货单'
  return detail.value.orderNo || '单据详情'
})

const openDetail = async (kind, row) => {
  detailKind.value = kind
  detailOpen.value = true
  detailLoading.value = true
  detail.value = null
  try {
    const res = kind === 'sale'
      ? await getSale(row.ID)
      : kind === 'purchase'
        ? await getPurchase(row.ID)
        : kind === 'purchaseReturn'
          ? await getPurchaseReturn(row.ID)
          : await getReturn(row.ID)
    if (res.code !== 0) return ElMessage.error(res.msg || '加载单据失败')
    detail.value = res.data
  } finally {
    detailLoading.value = false
  }
}

const onRowClick = (row) => {
  if (tab.value === 'reversal') return
  openDetail(tab.value, row)
}

const detailActions = computed(() => {
  if (!detail.value) return []
  const statusValue = detail.value.status
  const reversed = Boolean(detail.value.reversedAt)
  if (detailKind.value === 'sale') {
    if (statusValue === 1) return [
      { key: 'ship', label: '出库', type: 'primary' },
      { key: 'cancel-sale', label: '取消草稿', type: 'danger' }
    ]
    if (statusValue === 2 && !reversed) return [
      { key: 'return', label: '退货', type: 'primary' },
      { key: 'reversal', label: '整单红冲', type: 'danger' }
    ]
  }
  if (detailKind.value === 'purchase') {
    if (statusValue === 1) return [
      { key: 'approve', label: '审核', type: 'primary' },
      { key: 'cancel-purchase', label: '取消', type: 'danger' }
    ]
    if (statusValue === 2) return [
      { key: 'receive', label: '入库', type: 'primary' },
      { key: 'cancel-purchase', label: '取消', type: 'danger' }
    ]
    if (statusValue === 3 && !reversed) return [
      { key: 'purchase-return', label: '退货', type: 'primary' },
      { key: 'reversal', label: '整单红冲', type: 'danger' }
    ]
  }
  if (detailKind.value === 'purchaseReturn' && statusValue === 1) return [
    { key: 'post-purchase-return', label: '出库', type: 'primary' },
    { key: 'cancel-purchase-return', label: '取消', type: 'danger' }
  ]
  if (detailKind.value === 'return' && statusValue === 1) return [
    { key: 'post-return', label: '入库', type: 'primary' },
    { key: 'cancel-return', label: '取消', type: 'danger' }
  ]
  return []
})

const refreshDetail = async () => {
  if (!detailOpen.value || !detail.value) return
  await openDetail(detailKind.value, { ID: detail.value.ID })
}

const onDetailAction = async (key) => {
  const row = detail.value
  if (!row) return
  if (key === 'ship') return ship(row)
  if (key === 'cancel-sale') return cancelSaleOrder(row)
  if (key === 'return') return openReturn(row)
  if (key === 'reversal') return openReversal(detailKind.value, row)
  if (key === 'purchase-return') return openPurchaseReturn(row)
  if (key === 'post-purchase-return') return postPurchaseReturnRow(row)
  if (key === 'cancel-purchase-return') return cancelPurchaseReturnRow(row)
  if (key === 'approve') return approvePurchaseRow(row)
  if (key === 'receive') return receivePurchaseRow(row)
  if (key === 'cancel-purchase') return cancelPurchaseRow(row)
  if (key === 'post-return') return doPostReturn(row)
  if (key === 'cancel-return') return doCancelReturn(row)
}

// ---------- 列表操作 ----------
const afterWrite = async (message) => {
  if (message) ElMessage.success(message)
  await load()
  await refreshDetail()
}

const ship = async (row) => {
  const res = await shipSale(row.ID)
  if (res.code !== 0) return ElMessage.error(res.msg || '出库失败')
  await afterWrite('已出库')
}
const cancelSaleOrder = async (row) => {
  try {
    await ElMessageBox.confirm('取消后释放预留库存，草稿单不能恢复。确认取消？', '取消销售单', { type: 'warning' })
  } catch { return }
  const res = await cancelSale(row.ID)
  if (res.code !== 0) return ElMessage.error(res.msg || '取消失败')
  await afterWrite('已取消')
}
const approvePurchaseRow = async (row) => {
  const res = await approvePurchase(row.ID)
  if (res.code !== 0) return ElMessage.error(res.msg || '审核失败')
  await afterWrite('已审核')
}
const receivePurchaseRow = async (row) => {
  const res = await receivePurchase(row.ID)
  if (res.code !== 0) return ElMessage.error(res.msg || '入库失败')
  await afterWrite('已入库')
}
const cancelPurchaseRow = async (row) => {
  try {
    await ElMessageBox.confirm('取消后该采购单不再推进。确认取消？', '取消采购单', { type: 'warning' })
  } catch { return }
  const res = await cancelPurchase(row.ID)
  if (res.code !== 0) return ElMessage.error(res.msg || '取消失败')
  await afterWrite('已取消')
}
const doPostReturn = async (row) => {
  const res = await postReturn(row.ID)
  if (res.code !== 0) return ElMessage.error(res.msg || '退货入库失败')
  await afterWrite('退货已入库')
}
const doCancelReturn = async (row) => {
  try {
    await ElMessageBox.confirm('取消后该退货单不再推进。确认取消？', '取消退货单', { type: 'warning' })
  } catch { return }
  const res = await cancelReturn(row.ID)
  if (res.code !== 0) return ElMessage.error(res.msg || '取消失败')
  await afterWrite('已取消')
}

// ---------- 红冲 ----------
const revDialog = ref(false)
const revTarget = ref(null)
const revSource = ref('sale')
const revReason = ref('')

const openReversal = (source, row) => {
  detailOpen.value = false
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
      sourceType: revSource.value,
      sourceId: revTarget.value.ID,
      reason: revReason.value.trim()
    })
    if (res.code !== 0) return ElMessage.error(res.msg || '红冲失败')
    ElMessage.success(`已红冲 ${res.data.reversalNo}`)
    revDialog.value = false
    detailOpen.value = false
    await load()
  } finally {
    saving.value = false
  }
}

// ---------- 退货 ----------
const retDialog = ref(false)
const retTarget = ref(null)
const retReason = ref('')
const returnable = ref([])

const openReturn = async (row) => {
  detailOpen.value = false
  retTarget.value = row
  retReason.value = ''
  const res = await getReturnable(row.ID)
  if (res.code !== 0) return ElMessage.error(res.msg || '加载可退明细失败')
  returnable.value = (res.data || []).map((line) => ({ ...line, qty: 0 }))
  retDialog.value = true
}

const submitReturn = async () => {
  const items = returnable.value.filter((line) => Number(line.qty) > 0).map((line) => ({ skuId: line.skuId, qty: Number(line.qty) }))
  if (!items.length) return ElMessage.warning('请填写退货数量')
  saving.value = true
  try {
    const res = await createReturn({ saleId: retTarget.value.ID, items, reason: retReason.value })
    if (res.code !== 0) return ElMessage.error(res.msg || '创建退货单失败')
    ElMessage.success(`已创建退货单 ${res.data.returnNo}`)
    retDialog.value = false
    detailOpen.value = false
    tab.value = 'return'
    status.value = ''
    keyword.value = ''
    await load()
  } finally {
    saving.value = false
  }
}

// ---------- 采购退货 ----------
const prDialog = ref(false)
const prTarget = ref(null)
const prReturnable = ref([])
const prReason = ref('')
const prTotal = computed(() => round2(
  prReturnable.value.reduce((sum, line) => sum + Number(line.qty || 0) * Number(line.unitCost || 0), 0)
))

const openPurchaseReturn = async (row) => {
  detailOpen.value = false
  prTarget.value = row
  prReason.value = ''
  const res = await getPurchaseReturnable(row.ID)
  if (res.code !== 0) return ElMessage.error(res.msg || '加载可退明细失败')
  prReturnable.value = (res.data || []).map((line) => ({ ...line, qty: 0 }))
  prDialog.value = true
}

const submitPurchaseReturn = async () => {
  const items = prReturnable.value
    .filter((line) => Number(line.qty) > 0)
    .map((line) => ({ skuId: line.skuId, qty: Number(line.qty) }))
  if (!items.length) return ElMessage.warning('请填写退货数量')
  saving.value = true
  try {
    const res = await createPurchaseReturn({
      purchaseId: prTarget.value.ID,
      items,
      reason: prReason.value
    })
    if (res.code !== 0) return ElMessage.error(res.msg || '创建采购退货单失败')
    ElMessage.success(`已创建采购退货单 ${res.data.returnNo}`)
    prDialog.value = false
    tab.value = 'purchaseReturn'
    status.value = ''
    keyword.value = ''
    await load()
    await openDetail('purchaseReturn', { ID: res.data.ID })
  } finally {
    saving.value = false
  }
}

const postPurchaseReturnRow = async (row) => {
  try {
    await ElMessageBox.confirm(
      '出库会按原采购单价扣减库存并写入账本，草稿单随后不可再改。确认出库？',
      '采购退货出库',
      { type: 'warning', confirmButtonText: '确认出库' }
    )
  } catch { return }
  const res = await postPurchaseReturn(row.ID)
  if (res.code !== 0) return ElMessage.error(res.msg || '采购退货出库失败')
  await afterWrite('采购退货已出库')
}

const cancelPurchaseReturnRow = async (row) => {
  try {
    await ElMessageBox.confirm('取消后该采购退货单不再推进。确认取消？', '取消采购退货单', { type: 'warning' })
  } catch { return }
  const res = await cancelPurchaseReturn(row.ID)
  if (res.code !== 0) return ElMessage.error(res.msg || '取消失败')
  await afterWrite('已取消')
}

watch(activeRows, () => {
  const maxPage = Math.max(1, Math.ceil(activeRows.value.length / pageSize))
  if (page.value > maxPage) page.value = maxPage
})

watch([tab, status, keyword], () => {
  page.value = 1
})

applyQuery()
// 组件被复用时（从工作台待办跳进来、或点导航回「单据」），query 变化必须重新应用筛选。
watch(() => route.query, applyQuery, { deep: true })
onMounted(load)

</script>
