<template>
  <div class="jxc-page">
    <div class="gva-table-box">
      <div class="gva-btn-list">
        <el-button type="primary" icon="plus" @click="openDialog()">新增分类</el-button>
      </div>
      <el-table
        row-key="ID"
        :data="tableData"
        :tree-props="{ children: 'children', hasChildren: 'hasChildren' }"
        border
        default-expand-all
        v-loading="loading"
      >
        <el-table-column label="编码" prop="code" width="140" />
        <el-table-column label="分类名称" prop="name" min-width="160" />
        <el-table-column label="排序" prop="sort" width="80" align="center" />
        <el-table-column label="状态" width="80" align="center">
          <template #default="{ row }">
            <el-switch :model-value="row.status === 1" @change="toggleStatus(row)" />
          </template>
        </el-table-column>
        <el-table-column label="备注" prop="remark" min-width="140" :show-overflow-tooltip="true" />
        <el-table-column label="操作" width="200" fixed="right">
          <template #default="{ row }">
            <el-button type="primary" link @click="openDialog(row.ID)">编辑</el-button>
            <el-button type="primary" link @click="openDialog(row.ID, 'child')">添加子分类</el-button>
            <el-popconfirm title="确认删除该分类?" @confirm="handleDelete(row.ID)">
              <template #reference>
                <el-button type="danger" link>删除</el-button>
              </template>
            </el-popconfirm>
          </template>
        </el-table-column>
      </el-table>
    </div>