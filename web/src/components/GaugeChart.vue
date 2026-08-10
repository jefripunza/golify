<script setup lang="ts">
/**
 * GaugeChart.vue — reusable Highcharts speedometer gauge (SVG-rendered).
 *
 * Props mirror the classic Highcharts gauge demo:
 *   value     — current needle value
 *   min/max   — axis bounds (default 0/100)
 *   unit      — suffix shown next to the value (e.g. '%', 'km/h')
 *   title     — small label under/above the gauge (optional)
 *   plotBands — colored zones [{from,to,color}]
 *
 * Updates: watch(value) → point.update() — smooth needle animation, no
 * full chart re-create. Highcharts renders SVG (not canvas), so it works
 * everywhere (incl. headless/embedded contexts where RAF is throttled).
 */
import { onBeforeUnmount, onMounted, ref, watch } from 'vue'
import Highcharts from 'highcharts/es-modules/masters/highcharts.src.js'
import 'highcharts/es-modules/masters/highcharts-more.src.js'

// Highcharts 12 formats numbers via Intl with its configured locale; an
// empty lang causes `RangeError: invalid language tag: ""` in some engines.
// Pin a safe default locale so the gauge always renders.
Highcharts.setOptions({
  lang: {
    locale: 'en-US',
    decimalPoint: '.',
    thousandsSep: ',',
    numericSymbols: ['k', 'M', 'G', 'T', 'P', 'E'],
  },
})

interface PlotBand {
  from: number
  to: number
  color: string
}

const props = withDefaults(
  defineProps<{
    value: number
    min?: number
    max?: number
    unit?: string
    title?: string
    color?: string
    plotBands?: PlotBand[]
  }>(),
  {
    min: 0,
    max: 100,
    unit: '%',
    title: '',
    color: '#58a6ff',
    plotBands: () => [
      { from: 0, to: 70, color: '#3fb950' },
      { from: 70, to: 90, color: '#d29922' },
      { from: 90, to: 100, color: '#f85149' },
    ],
  },
)

const container = ref<HTMLDivElement | null>(null)
let chart: Highcharts.Chart | null = null

function buildOptions(): Highcharts.Options {
  const { min, max, color, plotBands } = props
  return {
    chart: {
      type: 'gauge',
      height: '80%',
      backgroundColor: 'transparent',
    },
    title: { text: undefined },
    pane: {
      startAngle: -90,
      endAngle: 90,
      size: '100%',
      background: [
        {
          outerRadius: '100%',
          innerRadius: '60%',
          shape: 'arc',
          borderWidth: 0,
          backgroundColor: '#21262d',
        },
      ],
    },
    yAxis: {
      min,
      max,
      tickPixelInterval: 40,
      tickColor: 'transparent',
      gridLineColor: 'transparent',
      labels: {
        distance: 14,
        style: { fontSize: '10px', color: '#8b949e' },
      },
      lineWidth: 0,
      plotBands,
    },
    tooltip: { enabled: false },
    credits: { enabled: false },
    series: [
      {
        type: 'gauge',
        name: props.title || 'value',
        data: [props.value],
        dataLabels: { enabled: false },
        dial: {
          radius: '90%',
          baseWidth: 4,
          topWidth: 1,
          baseLength: '80%',
          rearLength: '-20%',
          backgroundColor: color,
        },
        pivot: {
          radius: 5,
          backgroundColor: color,
        },
      },
    ],
  }
}

onMounted(() => {
  if (!container.value) return
  chart = Highcharts.chart(container.value, buildOptions())
})

// smooth needle update on value change
watch(
  () => props.value,
  (v) => {
    if (!chart) return
    const point = chart.series[0]?.points?.[0]
    if (point) point.update(v, true)
  },
)

onBeforeUnmount(() => {
  if (chart) {
    chart.destroy()
    chart = null
  }
})

defineExpose({ update: (v: number) => props.value !== v && (chart?.series[0]?.points?.[0] as any)?.update(v, true) })
</script>

<template>
  <div ref="container" class="gauge-chart w-full" />
</template>

<style scoped>
.gauge-chart {
  min-height: 140px;
}
</style>
