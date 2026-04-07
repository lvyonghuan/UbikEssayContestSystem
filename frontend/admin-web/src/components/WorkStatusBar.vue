<script setup lang="ts">
import { onMounted, ref } from 'vue'
import * as echarts from 'echarts'

interface WorkStatus {
  name: string
  value: number
}

interface Props {
  data?: WorkStatus[]
  title?: string
}

const props = withDefaults(defineProps<Props>(), {
  title: '作品状态分析',
  data: () => [
    { name: '草稿', value: 24 },
    { name: '待审核', value: 78 },
    { name: '已通过', value: 156 },
    { name: '已驳回', value: 12 },
  ],
})

const chartRef = ref<HTMLDivElement>()
const chartInstance = ref<echarts.ECharts>()

onMounted(() => {
  if (!chartRef.value) return

  chartInstance.value = echarts.init(chartRef.value, null, { renderer: 'canvas' })

  const colors = ['#f0ad4e', '#5bc0de', '#5cb85c', '#d9534f']

  const option: echarts.EChartsOption = {
    title: {
      text: props.title,
      left: 'center',
      top: 10,
      textStyle: { fontSize: 14, fontWeight: 500 },
    },
    tooltip: { trigger: 'axis', axisPointer: { type: 'shadow' } },
    grid: { left: 60, right: 20, top: 60, bottom: 40 },
    xAxis: {
      type: 'category',
      data: props.data.map(d => d.name),
    },
    yAxis: {
      type: 'value',
      minInterval: 1,
      splitLine: { lineStyle: { color: '#e5e7eb' } },
    },
    series: [
      {
        data: props.data.map((d, i) => ({ value: d.value, itemStyle: { color: colors[i] } })),
        type: 'bar',
        itemStyle: { borderRadius: [8, 8, 0, 0] },
      },
    ],
  }

  chartInstance.value.setOption(option)

  const handleResize = () => chartInstance.value?.resize()
  window.addEventListener('resize', handleResize)

  return () => window.removeEventListener('resize', handleResize)
})
</script>

<template>
  <div ref="chartRef" class="bar-chart" />
</template>

<style scoped>
.bar-chart {
  width: 100%;
  height: 300px;
  border-radius: 8px;
  border: 1px solid var(--border-color, #e5e7eb);
  background: white;
}
</style>
