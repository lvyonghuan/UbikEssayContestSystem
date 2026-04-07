<script setup lang="ts">
import { onMounted, ref } from 'vue'
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
  data: () => [
    { name: '美文类', value: 45 },
    { name: '诗歌类', value: 32 },
    { name: '散文类', value: 28 },
    { name: '小说类', value: 38 },
  ],
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

  chartInstance.value.setOption(option)

  const handleResize = () => chartInstance.value?.resize()
  window.addEventListener('resize', handleResize)

  return () => window.removeEventListener('resize', handleResize)
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
