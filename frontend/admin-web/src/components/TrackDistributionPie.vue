<script setup lang="ts">
import { onBeforeUnmount, onMounted, ref, watch } from 'vue'
import * as echarts from 'echarts'

interface TrackDistribution {
  name: string
  value: number
}

interface Props {
  data?: TrackDistribution[]
  title?: string
}

const props = withDefaults(defineProps<Props>(), {
  title: '赛道分布',
  data: () => [],
})

const chartRef = ref<HTMLDivElement>()
const chartInstance = ref<echarts.ECharts>()
let handleResize: (() => void) | null = null

function renderChart() {
  if (!chartInstance.value) {
    return
  }

  const option: echarts.EChartsOption = {
    title: {
      text: props.title,
      left: 'center',
      top: 10,
      textStyle: { fontSize: 14, fontWeight: 500 },
    },
    tooltip: { trigger: 'item', formatter: '{b}: {c} ({d}%)' },
    legend: { orient: 'vertical', right: 20, top: 'center' },
    series: [
      {
        type: 'pie',
        radius: [40, 80],
        center: ['40%', '50%'],
        data: props.data,
        itemStyle: { borderRadius: 8, borderColor: '#fff', borderWidth: 2 },
      },
    ],
  }

  chartInstance.value.setOption(option, true)
}

onMounted(() => {
  if (!chartRef.value) return

  chartInstance.value = echarts.init(chartRef.value, null, { renderer: 'canvas' })
  renderChart()

  handleResize = () => chartInstance.value?.resize()
  window.addEventListener('resize', handleResize)
})

watch(
  () => ({ data: props.data, title: props.title }),
  () => {
    renderChart()
  },
  { deep: true },
)

onBeforeUnmount(() => {
  if (handleResize) {
    window.removeEventListener('resize', handleResize)
  }
  chartInstance.value?.dispose()
})
</script>

<template>
  <div ref="chartRef" class="pie-chart" />
</template>

<style scoped>
.pie-chart {
  width: 100%;
  height: 300px;
  border-radius: 8px;
  border: 1px solid var(--border-color, #e5e7eb);
  background: white;
}
</style>
