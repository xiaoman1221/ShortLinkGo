<template>
  <div class="geo-map">
    <svg :viewBox="`0 0 ${size.w} ${size.h}`" class="map-svg" role="img" :aria-label="ariaLabel" @mousemove="onTipMove">
      <path
        v-for="f in paths" :key="f.key" :d="f.d" :fill="f.fill"
        class="feature" @mouseenter="onMapEnter(f)" @mouseleave="onMapLeave"
      />
      <!-- 南海诸岛小图框（中国模式） -->
      <g v-if="inset" :transform="`translate(${size.w - inset.w - 14}, ${size.h - inset.h - 12})`">
        <rect x="0" y="0" :width="inset.w" :height="inset.h" class="inset-frame" />
        <path v-for="(f, i) in inset.paths" :key="'i' + i" :d="f.d" :fill="f.fill" class="feature" />
        <text :x="inset.w / 2" :y="inset.h - 6" class="inset-text">南海诸岛</text>
      </g>
    </svg>

    <div class="legend">
      <span class="legend-label">少</span>
      <span v-for="(a, i) in legendSteps" :key="i" class="legend-cell" :style="{ background: stepColor(a) }"></span>
      <span class="legend-label">多</span>
    </div>

    <Teleport to="body">
      <div v-if="hover" class="geo-tip" :style="{ left: tip.x + 'px', top: tip.y + 'px' }">
        <span class="tip-name">{{ hover.label }}</span>
        <span v-if="hover.count" class="tip-val num">{{ hover.count }} 次</span>
        <span v-else class="tip-val tip-empty">暂无访问</span>
      </div>
    </Teleport>
  </div>
</template>

<script setup>
import { computed, reactive, ref } from 'vue'

const props = defineProps({
  // FeatureCollection（world.json / china.json）
  map: { type: Object, required: true },
  // [{ key, name, count }]：key 与 featureKey(feature) 匹配
  points: { type: Array, default: () => [] },
  // 从 feature 提取匹配键
  featureKey: { type: Function, required: true },
  // feature 上显示的名称（未匹配到数据时）
  featureLabel: { type: Function, default: null },
  // 需要移出主图、绘制为右下角小图框的 feature 名称（如「南海诸岛」）
  insetName: { type: String, default: '' },
  // 主图 bbox 忽略低于该纬度的坐标（中国模式裁掉海南 feature 附带的南海岛礁，避免地图被拉长）
  clipLatBelow: { type: Number, default: null },
  ariaLabel: { type: String, default: '访问来源地图' }
})

const BASE_W = 1000

// Web Mercator 投影到 [0,1] 区间
function mercX(lon) { return (lon + 180) / 360 }
function mercY(lat) {
  const r = (Math.max(-84, Math.min(84, lat)) * Math.PI) / 180
  return (1 - Math.log(Math.tan(Math.PI / 4 + r / 2)) / Math.PI) / 2
}

// 分离主图与 inset（南海诸岛）feature
const mainFeatures = computed(() =>
  props.insetName ? props.map.features.filter((f) => f.properties.name !== props.insetName) : props.map.features)
const insetFeatures = computed(() =>
  props.insetName ? props.map.features.filter((f) => f.properties.name === props.insetName) : [])

// 主图投影 bbox（可选裁掉低纬坐标）
const bbox = computed(() => {
  let minX = 1, minY = 1, maxX = 0, maxY = 0
  const floor = props.clipLatBelow
  const walk = (coords) => {
    if (typeof coords[0] === 'number') {
      if (floor !== null && coords[1] < floor) return
      const x = mercX(coords[0]), y = mercY(coords[1])
      if (x < minX) minX = x
      if (x > maxX) maxX = x
      if (y < minY) minY = y
      if (y > maxY) maxY = y
      return
    }
    for (const c of coords) walk(c)
  }
  for (const f of mainFeatures.value) walk(f.geometry.coordinates)
  return { minX, minY, maxX, maxY }
})

const size = computed(() => {
  const b = bbox.value
  const h = Math.max(0.05, b.maxY - b.minY)
  const w = Math.max(0.05, b.maxX - b.minX)
  return { w: BASE_W, h: Math.round((BASE_W * h) / w) }
})

// 投影坐标 → viewBox 像素（留 2% 边距）
function toX(x) {
  const b = bbox.value, s = size.value
  return ((x - b.minX) / (b.maxX - b.minX)) * s.w * 0.98 + s.w * 0.01
}
function toY(y) {
  const b = bbox.value, s = size.value
  return ((y - b.minY) / (b.maxY - b.minY)) * s.h * 0.96 + s.h * 0.02
}

function buildPath(geometry, sx, sy) {
  const polys = geometry.type === 'Polygon' ? [geometry.coordinates] : geometry.coordinates
  let d = ''
  for (const rings of polys) {
    for (const ring of rings) {
      if (ring.length < 3) continue
      d += 'M' + ring.map(([lon, lat]) => sx(mercX(lon)).toFixed(1) + ' ' + sy(mercY(lat)).toFixed(1)).join('L') + 'Z'
    }
  }
  return d
}

// 访问量 → 匹配键索引
const valueMap = computed(() => {
  const m = new Map()
  for (const p of props.points) {
    if (p.key) m.set(p.key, p)
  }
  return m
})
const maxCount = computed(() => Math.max(1, ...props.points.map((p) => p.count)))

function fillFor(key) {
  const item = valueMap.value.get(key)
  if (!item) return 'var(--surface-2)'
  const t = Math.sqrt(item.count / maxCount.value)
  return `color-mix(in srgb, var(--ink) ${Math.round((0.14 + t * 0.66) * 100)}%, #ffffff)`
}

function labelFor(f, matched) {
  if (matched) return matched.name
  if (props.featureLabel) return props.featureLabel(f) || ''
  return f.properties.NAME || f.properties.name || ''
}

const paths = computed(() => {
  const out = []
  for (const f of mainFeatures.value) {
    const key = props.featureKey(f)
    const matched = valueMap.value.get(key)
    out.push({
      key: f.properties.NAME || f.properties.name || out.length,
      d: buildPath(f.geometry, toX, toY),
      fill: fillFor(key),
      label: labelFor(f, matched),
      count: matched ? matched.count : 0
    })
  }
  return out
})

// 南海诸岛小图框
const inset = computed(() => {
  if (!insetFeatures.value.length) return null
  let minX = 1, minY = 1, maxX = 0, maxY = 0
  const walk = (coords) => {
    if (typeof coords[0] === 'number') {
      const x = mercX(coords[0]), y = mercY(coords[1])
      minX = Math.min(minX, x); maxX = Math.max(maxX, x)
      minY = Math.min(minY, y); maxY = Math.max(maxY, y)
      return
    }
    for (const c of coords) walk(c)
  }
  for (const f of insetFeatures.value) walk(f.geometry.coordinates)
  const w = 46, h = 58
  const sx = (x) => ((x - minX) / Math.max(1e-6, maxX - minX)) * (w - 8) + 4
  const sy = (y) => ((y - minY) / Math.max(1e-6, maxY - minY)) * (h - 20) + 4
  const paths = insetFeatures.value.map((f) => ({
    d: buildPath(f.geometry, sx, sy),
    fill: fillFor(props.featureKey(f))
  }))
  return { w, h, paths }
})

// hover tooltip（fixed 定位跟随鼠标）
const hover = ref(null)
const tip = reactive({ x: 0, y: 0 })
function onTipMove(e) {
  tip.x = e.clientX + 14
  tip.y = e.clientY + 14
}
function onMapEnter(f) { hover.value = f }
function onMapLeave() { hover.value = null }

const legendSteps = [0.18, 0.34, 0.5, 0.66, 0.82]
function stepColor(alpha) {
  return `color-mix(in srgb, var(--ink) ${Math.round(alpha * 100)}%, #ffffff)`
}
</script>

<style scoped>
.geo-map { position: relative; }
.map-svg { width: 100%; display: block; background: var(--surface); }
.feature { stroke: var(--line-strong); stroke-width: 0.6; transition: fill var(--t-fast) var(--ease); }
.feature:hover { stroke: var(--ink); stroke-width: 1.1; }
.inset-frame { fill: var(--surface); stroke: var(--line-strong); stroke-width: 0.8; }
.inset-text { font-size: 13px; fill: var(--ink-4); text-anchor: middle; }

.legend {
  position: absolute; left: 12px; bottom: 10px;
  display: flex; align-items: center; gap: 4px;
  padding: 5px 9px; border: 1px solid var(--line); border-radius: 999px;
  background: color-mix(in srgb, var(--surface) 88%, transparent);
  backdrop-filter: blur(4px);
}
.legend-cell { width: 12px; height: 12px; border-radius: 3px; border: 1px solid var(--line); }
.legend-label { font-size: 11px; color: var(--ink-4); margin: 0 3px; }

.geo-tip {
  position: fixed; z-index: 90; pointer-events: none;
  display: flex; flex-direction: column; gap: 1px;
  background: var(--dark); color: #fff;
  border-radius: var(--r-sm); padding: 7px 11px;
  font-size: 12.5px; line-height: 1.45;
  box-shadow: 0 6px 20px rgba(0, 0, 0, 0.18);
}
.tip-name { font-weight: 600; }
.tip-val { color: rgba(255, 255, 255, 0.72); }
.tip-empty { color: rgba(255, 255, 255, 0.45); }
</style>

<style>
/* mousemove 写在 svg 上（需要冒泡），独立于 hover 状态 */
.geo-map svg { cursor: default; }
</style>
