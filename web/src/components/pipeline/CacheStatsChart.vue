<template>
  <div class="cache-stats-chart">
    <a-card title="缓存使用趋势" :bordered="false" style="margin-bottom: 16px">
      <div ref="usageChartRef" style="width: 100%; height: 300px"></div>
    </a-card>
    
    <a-card title="缓存命中率趋势" :bordered="false">
      <div ref="hitRateChartRef" style="width: 100%; height: 300px"></div>
    </a-card>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted, onUnmounted, watch } from 'vue'
import * as echarts from 'echarts'

interface Props {
  usageData?: Array<{ timestamp: string; value: number }>
  hitRateData?: Array<{ timestamp: string; value: number }>
}

const props = withDefaults(defineProps<Props>(), {
  usageData: () => [],
  hitRateData: () => [],
})

const usageChartRef = ref<HTMLElement>()
const hitRateChartRef = ref<HTMLElement>()
let usageChart: echarts.ECharts | null = null
let hitRateChart: echarts.ECharts | null = null

// 初始化缓存使用趋势图
const initUsageChart = () => {
  if (!usageChartRef.value) return
  
  usageChart = echarts.init(usageChartRef.value)
  
  const option: echarts.EChartsOption = {
    tooltip: {
      trigger: 'axis',
      axisPointer: {
        type: 'cross',
        label: {
          backgroundColor: '#6a7985',
        },
      },
    },
    legend: {
      data: ['缓存大小'],
    },
    grid: {
      left: '3%',
      right: '4%',
      bottom: '3%',
      containLabel: true,
    },
    xAxis: {
      type: 'category',
      boundaryGap: false,
      data: props.usageData.map(item => item.timestamp),
    },
    yAxis: {
      type: 'value',
      name: '大小 (MB)',
      axisLabel: {
        formatter: '{value}',
      },
    },
    series: [
      {
        name: '缓存大小',
        type: 'line',
        smooth: true,
        areaStyle: {
          color: new echarts.graphic.LinearGradient(0, 0, 0, 1, [
            { offset: 0, color: 'rgba(24, 144, 255, 0.3)' },
            { offset: 1, color: 'rgba(24, 144, 255, 0.05)' },
          ]),
        },
        lineStyle: {
          color: '#1890ff',
          width: 2,
        },
        itemStyle: {
          color: '#1890ff',
        },
        data: props.usageData.map(item => item.value),
      },
    ],
  }
  
  usageChart.setOption(option)
}

// 初始化命中率趋势图
const initHitRateChart = () => {
  if (!hitRateChartRef.value) return
  
  hitRateChart = echarts.init(hitRateChartRef.value)
  
  const option: echarts.EChartsOption = {
    tooltip: {
      trigger: 'axis',
      axisPointer: {
        type: 'cross',
        label: {
          backgroundColor: '#6a7985',
        },
      },
      formatter: (params: any) => {
        const param = params[0]
        return `${param.name}<br/>${param.seriesName}: ${param.value}%`
      },
    },
    legend: {
      data: ['命中率'],
    },
    grid: {
      left: '3%',
      right: '4%',
      bottom: '3%',
      containLabel: true,
    },
    xAxis: {
      type: 'category',
      boundaryGap: false,
      data: props.hitRateData.map(item => item.timestamp),
    },
    yAxis: {
      type: 'value',
      name: '命中率 (%)',
      min: 0,
      max: 100,
      axisLabel: {
        formatter: '{value}%',
      },
    },
    series: [
      {
        name: '命中率',
        type: 'line',
        smooth: true,
        areaStyle: {
          color: new echarts.graphic.LinearGradient(0, 0, 0, 1, [
            { offset: 0, color: 'rgba(82, 196, 26, 0.3)' },
            { offset: 1, color: 'rgba(82, 196, 26, 0.05)' },
          ]),
        },
        lineStyle: {
          color: '#52c41a',
          width: 2,
        },
        itemStyle: {
          color: '#52c41a',
        },
        data: props.hitRateData.map(item => item.value),
      },
    ],
  }
  
  hitRateChart.setOption(option)
}

// 更新图表数据
const updateCharts = () => {
  if (usageChart && props.usageData.length > 0) {
    usageChart.setOption({
      xAxis: {
        data: props.usageData.map(item => item.timestamp),
      },
      series: [
        {
          data: props.usageData.map(item => item.value),
        },
      ],
    })
  }
  
  if (hitRateChart && props.hitRateData.length > 0) {
    hitRateChart.setOption({
      xAxis: {
        data: props.hitRateData.map(item => item.timestamp),
      },
      series: [
        {
          data: props.hitRateData.map(item => item.value),
        },
      ],
    })
  }
}

// 响应式调整
const handleResize = () => {
  usageChart?.resize()
  hitRateChart?.resize()
}

// 监听数据变化
watch(() => [props.usageData, props.hitRateData], () => {
  updateCharts()
}, { deep: true })

onMounted(() => {
  initUsageChart()
  initHitRateChart()
  window.addEventListener('resize', handleResize)
})

onUnmounted(() => {
  usageChart?.dispose()
  hitRateChart?.dispose()
  window.removeEventListener('resize', handleResize)
})
</script>

<style scoped>
.cache-stats-chart {
  width: 100%;
}
</style>
