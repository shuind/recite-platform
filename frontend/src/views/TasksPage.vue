<!-- src/views/TasksPage.vue -->
<template>
  <el-button @click="openScore">积分趋势</el-button>

  <el-dialog
    v-model="showScore"
    title="积分趋势"
    width="800px"
    @opened="renderChart"
  >
    <div class="flex items-center gap-3 mb-2">
      <el-segmented
        v-model="period"
        :options="[
          { label: '每日', value: 'day' },
          { label: '每周', value: 'week' },
          { label: '每月', value: 'month' }
        ]"
        @change="loadScore"
      />
    </div>
    <div ref="chartRef" style="width:100%;height:380px;"></div>
  </el-dialog>

  <div class="p-4">
    <div class="flex items-center gap-3 mb-4">
      <el-segmented v-model="view" :options="['auto','manual']" @change="reload" />
      <el-select
        v-model="status"
        placeholder="状态(可选)"
        clearable
        style="width: 160px"
        @change="reload"
        :disabled="scope==='archived'"
      >
        <el-option label="未完成" value="" />
        <el-option label="已完成" value="done" />
      </el-select>
      <el-segmented
        v-model="scope"
        :options="[
          { label:'活动', value:'active' },
          { label:'归档', value:'archived' },
          { label:'全部', value:'all' }
        ]"
        @change="reload"
      />
      <el-button type="primary" @click="openCreate">新建任务</el-button>
      <el-text type="info">视图：{{ view==='auto' ? '自动排序' : '手动排序(可拖拽)' }}</el-text>
    </div>

    <!-- 自动排序视图 -->
    <el-table
      v-if="view==='auto'"
      :data="items"
      v-loading="loading"
      border
      size="small"
    >
      <el-table-column label="标题" min-width="280">
        <template #default="{ row }">
          <el-checkbox
            :model-value="row.status==='done'"
            :disabled="isArchived(row)"
            @change="() => onComplete(row)"
          />
          <span :style="{ textDecoration: row.status==='done' ? 'line-through' : 'none', marginLeft:'8px' }">
            {{ row.title }}
          </span>
        </template>
      </el-table-column>

      <el-table-column label="优先级" width="90">
        <template #default="{ row }">
          <el-tag :type="priorityType(row.priority)" size="small">{{ priorityLabel(row.priority) }}</el-tag>
        </template>
      </el-table-column>

      <el-table-column label="积分" width="80">
        <template #default="{ row }">
          <el-tag size="small">{{ row.score ?? 0 }}</el-tag>
        </template>
      </el-table-column>

      <el-table-column label="开始时间" width="170">
        <template #default="{ row }">{{ fmt(row.start_at) }}</template>
      </el-table-column>

      <el-table-column label="截止时间" width="170">
        <template #default="{ row }">
          <span :style="{ color: isDueSoon(row) ? '#F56C6C' : '' }">{{ fmt(row.due_at) }}</span>
        </template>
      </el-table-column>

      <el-table-column label="操作" width="260" fixed="right">
        <template #default="{ row }">
          <el-button size="small" @click="() => onSnooze(row)">延后30分</el-button>
          <el-button size="small" @click="() => openEdit(row)">编辑</el-button>
          <el-button size="small" type="danger" @click="() => onDelete(row)">删除</el-button>
        </template>
      </el-table-column>
    </el-table>

    <!-- 手动排序视图 -->
    <div v-else>
      <draggable
        v-model="manualList"
        item-key="id"
        handle=".drag-handle"
        animation="180"
        @end="onDragEnd"
      >
        <template #item="{ element }">
          <el-card class="mb-2">
            <div class="flex items-center justify-between">
              <div class="flex items-center gap-2">
                <el-icon class="drag-handle" style="cursor:move"><rank /></el-icon>
                <el-tag :type="priorityType(element.priority)" size="small">{{ priorityLabel(element.priority) }}</el-tag>
                <el-tag size="small">+{{ element.score ?? 0 }}</el-tag>
                <span :style="{ textDecoration: element.status==='done' ? 'line-through' : 'none' }">{{ element.title }}</span>
              </div>
              <div class="flex items-center gap-2">
                <el-text type="info">{{ fmt(element.start_at) }} → {{ fmt(element.due_at) }}</el-text>
                <el-button size="small" @click="() => onSnooze(element)">延后</el-button>
                <el-button size="small" @click="() => openEdit(element)">编辑</el-button>
                <el-button size="small" type="success" @click="() => onComplete(element)">完成</el-button>
                <el-button size="small" type="danger" @click="() => onDelete(element)">删除</el-button>
              </div>
            </div>
          </el-card>
        </template>
      </draggable>
      <el-empty v-if="!manualList.length && !loading" description="暂无任务" />
      <el-skeleton v-if="loading" animated :rows="4" />
    </div>

    <!-- 创建/编辑弹窗 -->
    <el-dialog v-model="showDialog" :title="editing ? '编辑任务' : '新建任务'" width="520px">
      <el-form :model="form" label-width="88px">
        <el-form-item label="标题">
          <el-input v-model="form.title" maxlength="200" show-word-limit />
        </el-form-item>
        <el-form-item label="描述">
          <el-input v-model="form.description" type="textarea" :rows="3" />
        </el-form-item>
        <el-form-item label="优先级">
          <el-select v-model="form.priority" style="width: 160px">
            <el-option v-for="p in [0,1,2,3]" :key="p" :label="priorityLabel(p)" :value="p" />
          </el-select>
        </el-form-item>
        <el-form-item label="积分">
          <el-input-number v-model="form.score" :min="0" :step="1" />
        </el-form-item>
        <el-form-item label="开始时间">
          <el-date-picker v-model="form.start_at" type="datetime" placeholder="可选" style="width: 100%" />
        </el-form-item>
        <el-form-item label="截止时间">
          <el-date-picker v-model="form.due_at" type="datetime" placeholder="可选" style="width: 100%" />
        </el-form-item>
        <el-form-item label="预计时长">
          <el-input-number v-model="form.estimate_min" :min="0" :step="15" /> 分钟
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="showDialog=false">取消</el-button>
        <el-button type="primary" @click="onSubmit">{{ editing ? '保存' : '创建' }}</el-button>
      </template>
    </el-dialog>
  </div>
</template>

<script setup>
import { ref, computed, watch, onBeforeUnmount } from 'vue'
import { useTaskStore } from '@/stores/tasks'
import dayjs from 'dayjs'
import draggable from 'vuedraggable'
import { Rank as rank } from '@element-plus/icons-vue'
import * as echarts from 'echarts'
import api from '@/api'
import { ElMessageBox } from 'element-plus'

const showScore = ref(false)
const period = ref('day')            // day | week | month
const chartRef = ref(null)
let chart = null
let removeResize = null

async function openScore () {
  showScore.value = true
  await loadScore()
}

async function loadScore () {
  const { data } = await api.get('/tasks/score-trend', { params: { period: period.value } })
  trendData.value = data.items || []
  renderChart()
}

const trendData = ref([])

function renderChart () {
  if (!chartRef.value) return
  if (!chart) {
    chart = echarts.init(chartRef.value)
    const onResize = () => chart && chart.resize()
    window.addEventListener('resize', onResize)
    removeResize = () => window.removeEventListener('resize', onResize)
  }
  const x = trendData.value.map(i => i.date)
  const y = trendData.value.map(i => i.score)
  chart.setOption({
    tooltip: { trigger: 'axis' },
    grid: { left: 40, right: 24, top: 30, bottom: 40 },
    xAxis: { type: 'category', data: x, boundaryGap: false, axisTick: { show: false } },
    yAxis: { type: 'value' },
    series: [
      { type: 'line', data: y, smooth: true, symbol: 'circle', areaStyle: {} }
    ]
  })
}

onBeforeUnmount(() => {
  if (removeResize) removeResize()
  if (chart) {
    chart.dispose()
    chart = null
  }
})

const store = useTaskStore()
const view = ref('auto')
const status = ref('')
const scope = ref('active')
const showDialog = ref(false)
const editing = ref(false)
const editingId = ref(null)

const form = ref({
  title: '',
  description: '',
  priority: 2,
  start_at: null,
  due_at: null,
  estimate_min: null,
  score: 0
})

const items = computed(() => store.items)
const loading = computed(() => store.loading)
const manualList = ref([])

watch(() => store.items, (val) => {
  if (view.value === 'manual') manualList.value = [...val]
}, { immediate: true })

watch(view, () => reload())

function reload () {
  store.fetch({ view: view.value, status: status.value, scope: scope.value })
  if (view.value === 'manual') manualList.value = [...store.items]
}

function priorityLabel (p) { return ['P0', 'P1', 'P2', 'P3'][p ?? 2] }
function priorityType (p) { return ['danger', 'warning', 'info', ''][p ?? 2] }
function fmt (v) { return v ? dayjs(v).format('MM-DD HH:mm') : '-' }
function isDueSoon (row) {
  if (!row.due_at) return false
  return dayjs(row.due_at).diff(dayjs(), 'hour') <= 24
}
function isArchived (row) {
  return scope.value === 'archived' || !!row.archived_at
}

function openCreate () {
  editing.value = false
  editingId.value = null
  form.value = { title: '', description: '', priority: 2, start_at: null, due_at: null, estimate_min: null, score: 0 }
  showDialog.value = true
}

function openEdit (row) {
  editing.value = true
  editingId.value = row.id
  form.value = {
    title: row.title,
    description: row.description,
    priority: row.priority,
    start_at: row.start_at ? new Date(row.start_at) : null,
    due_at: row.due_at ? new Date(row.due_at) : null,
    estimate_min: row.estimate_min ?? null,
    score: row.score ?? 0
  }
  showDialog.value = true
}

async function onSubmit () {
  const payload = { ...form.value }
  if (editing.value) {
    await store.update(editingId.value, payload)
  } else {
    await store.create(payload)
  }
  showDialog.value = false
}

async function onComplete (row) {
  if (isArchived(row)) return
  if (row.status === 'done') {
    await store.undo(row.id)
  } else {
    await store.complete(row.id)
  }
}

async function onSnooze (row) { await store.snooze(row.id, 30) }

async function onDelete (row) {
  await ElMessageBox.confirm(`确定删除任务「${row.title}」吗？`, '删除确认', { type: 'warning' })
  await store.remove(row.id)
}

async function onDragEnd () {
  const ids = manualList.value.map(i => i.id)
  await store.reorder(ids)
}

reload()
</script>

<style scoped>
.p-4 { padding: 16px; }
.mb-2 { margin-bottom: 8px; }
.mb-4 { margin-bottom: 16px; }
.flex { display: flex; }
.items-center { align-items: center; }
.justify-between { justify-content: space-between; }
.gap-2 { gap: 8px; }
.gap-3 { gap: 12px; }
</style>
