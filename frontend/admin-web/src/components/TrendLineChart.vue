<script setup lang="ts">
import { onMounted, ref } from 'vue'
import * as echarts from 'echarts'

interface TrendData {
  date: string
  count: number
}

interface Props {
  data?: TrendData[]
  title?: string
}

const props = withDefaults(defineProps<Props>(), {
  title: '7 日赛事趋势',
  data: () => Array.from({ length: 7 }, (_, i) => ({
    date: new Date(Date.now() - (6 - i) * 86400000).toLocaleDateString('zh-CN', { month: '2-digit', day: '2-digit' }),
    count: Math.floor(Math.random() * 5),
  })),
})

const chartRef = ref<HTMLDivElement>()
const chartInstance = ref<echarts.ECharts>()

onMounted(() => {
  if (!chartRef.value) return

  chartInstance.value = echarts.init(chartRef.value, null, { renderer: 'canvas' })

  const option: echarts.EChartsOption = {
    title: {
      text: props.title,
      left: 'center',
      top: 10,
      textStyle: { fontSize: 14, fontWeight: 500 },
    },
    tooltip: { trigger: 'axis', axisPointer: { type: 'cross' } },
    grid: { left: 60, right: 20, top: 60, bottom: 40 },
    xAxis: {
      type: 'category',
      data: props.data.map(d => d.date),
      boundaryGap: false,
    },
    yAxis: {
      type: 'value',
      minInterval: 1,
      splitLine: { lineStyle: { color: '#e5e7eb' } },
    },
    series: [
      {
        data: props.data.map(d => d.count),
        type: 'line',
        smooth: true,
        itemStyle: { color: '#0b6ef2' },
        areaStyle: { color: 'rgba(11, 110, 242, 0.1)' },
        symbolSize: 6,
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
  <div ref="chartRef" class="trend-chart" />
</template>

<style scoped>
.trend-chart {
  width: 100%;
  height: 300px;
  border-radius: 8px;
  border: 1px solid var(--border-color, #e5e7eb);
  background: white;
}
</style>
