<template>
  <div>
    <section class="overview">
      <div class="overview-lead">
        <p class="eyebrow">统计 · Overview</p>
        <h1>
          <span class="num count">{{ summary.total_links }}</span>
          <span class="count-unit">个短链接</span>
        </h1>
        <p class="overview-sub">你的短链接入口，以及它们累计带来的访问。</p>
      </div>
      <dl class="metrics">
        <div v-for="m in metrics" :key="m.label" class="metric">
          <dt>{{ m.label }}</dt>
          <dd class="num">{{ m.value }}</dd>
        </div>
      </dl>
    </section>

    <div class="workspace">
      <div class="main-col">
        <!-- 访问地图 -->
        <section class="panel">
          <div class="section-head">
            <div>
              <p class="eyebrow">访问地图 · Geo</p>
              <h2>访客分布</h2>
            </div>
            <div class="map-tabs" role="tablist" aria-label="地图模式">
              <button
                v-for="m in MAP_MODES" :key="m.value" type="button" role="tab"
                :aria-selected="mapMode === m.value"
                :class="{ active: mapMode === m.value }"
                @click="switchMode(m.value)"
              >{{ m.label }}</button>
            </div>
          </div>
          <p v-if="loading" class="state-note">加载中…</p>
          <template v-else>
            <div v-if="mapData && mapPoints.length" class="map-wrap">
              <GeoMap
                :map="mapMode === 'world' ? worldMap : chinaMap"
                :points="mapPoints"
                :feature-key="mapMode === 'world' ? featureKeyWorld : featureKeyChina"
                :feature-label="mapMode === 'world' ? featureLabelWorld : featureLabelChina"
                :inset-name="mapMode === 'china' ? '南海诸岛' : ''"
                :clip-lat-below="mapMode === 'china' ? 16 : null"
                :aria-label="mapMode === 'world' ? '世界访客分布地图' : '中国访客分布地图'"
              />
            </div>
            <div v-else class="empty-mini">
              <p>{{ mapMode === 'world' ? '暂无可定位的访问来源。' : '暂无中国境内的访问来源。' }}</p>
            </div>
            <div v-if="chips.length" class="local-chips">
              <span v-for="r in chips" :key="r.name" class="chip">
                {{ r.name }} <b class="num">{{ r.count }}</b>
              </span>
            </div>
            <p class="map-note">{{ mapNote }}</p>
          </template>
        </section>

        <!-- 访问趋势 -->
        <section class="panel">
          <div class="section-head">
            <div>
              <p class="eyebrow">访问趋势 · 近 14 天</p>
              <h2>访问趋势</h2>
            </div>
            <span class="count-note">合计 {{ trendTotal }} 次</span>
          </div>
          <p v-if="loading" class="state-note">加载中…</p>
          <div v-else-if="trendTotal === 0" class="empty-mini"><p>近 14 天还没有访问记录。</p></div>
          <div v-else class="chart">
            <div v-for="d in trend" :key="d.date" class="bar-col" :title="d.date + '：' + d.count + ' 次'">
              <div class="bar-area">
                <span v-if="d.count > 0" class="bar-val num">{{ d.count }}</span>
                <div class="bar" :style="{ height: barHeight(d.count) }"></div>
              </div>
              <span class="bar-label">{{ d.date }}</span>
            </div>
          </div>
        </section>
      </div>

      <aside class="side-col">
        <section class="panel">
          <div class="section-head tight">
            <div>
              <p class="eyebrow">排行 · Top</p>
              <h3>访问最多的链接</h3>
            </div>
          </div>
          <p v-if="loading" class="state-note">加载中…</p>
          <ol v-else-if="top.length" class="top-list">
            <li v-for="(l, i) in top" :key="l.id" class="top-row">
              <span class="top-rank num">{{ String(i + 1).padStart(2, '0') }}</span>
              <div class="top-main">
                <span class="top-code">{{ l.code }}</span>
                <a class="top-url" :href="l.url" target="_blank" rel="noreferrer">{{ l.url }}</a>
              </div>
              <span class="top-visits num">{{ l.visit_count }} 次</span>
            </li>
          </ol>
          <div v-else class="empty-mini"><p>暂无访问数据。</p></div>
        </section>

        <section class="panel">
          <div class="section-head tight">
            <div>
              <p class="eyebrow">访客 · Top IP</p>
              <h3>访问最多的 IP</h3>
            </div>
          </div>
          <p v-if="loading" class="state-note">加载中…</p>
          <ol v-else-if="topIPs.length" class="top-list">
            <li v-for="(t, i) in topIPs" :key="t.ip" class="top-row">
              <span class="top-rank num">{{ String(i + 1).padStart(2, '0') }}</span>
              <div class="top-main">
                <span class="top-code">{{ t.ip }}</span>
                <span class="top-url">{{ t.country }}</span>
              </div>
              <span class="top-visits num">{{ t.count }} 次</span>
            </li>
          </ol>
          <div v-else class="empty-mini"><p>暂无访问数据。</p></div>
        </section>
      </aside>
    </div>
  </div>
</template>

<script setup>
import { computed, onMounted, reactive, ref, shallowRef } from 'vue'
import api from '../api'
import GeoMap from '../components/GeoMap.vue'
import { countryToISO, provinceKey, provinceCenter } from '../utils/geoNames'

const summary = reactive({ total_links: 0, total_visits: 0, active_links: 0, expired_links: 0, pending_links: 0 })
const trend = ref([])
const top = ref([])
const topIPs = ref([])
const loading = ref(false)

const metrics = computed(() => [
  { label: '累计访问', value: summary.total_visits },
  { label: '启用中', value: summary.active_links },
  { label: '已过期', value: summary.expired_links }
])
const trendTotal = computed(() => trend.value.reduce((s, d) => s + d.count, 0))
const trendMax = computed(() => Math.max(1, ...trend.value.map((d) => d.count)))

function barHeight(count) {
  if (!count) return '2px'
  return Math.max(6, Math.round((count / trendMax.value) * 150)) + 'px'
}

// ---------- 访问地图（世界 / 中国两种模式） ----------
const MAP_MODES = [
  { value: 'world', label: '世界' },
  { value: 'china', label: '中国' }
]
const LOCAL = { 内网: true, 未知: true, 局域网: true, 本地: true }
const mapMode = ref('world')
const geoWorld = ref([])
const geoChina = ref([])
const chinaLoaded = ref(false)

// 地图边界数据按模式懒加载（独立 chunk，避免拖大首屏）
const worldMap = shallowRef(null)
const chinaMap = shallowRef(null)
async function ensureMapData(mode) {
  if (mode === 'world' && !worldMap.value) {
    worldMap.value = (await import('../assets/world.json')).default
  }
  if (mode === 'china' && !chinaMap.value) {
    chinaMap.value = (await import('../assets/china.json')).default
  }
}

const featureKeyWorld = (f) => f.properties.ISO_A2
const featureLabelWorld = (f) => f.properties.NAME || ''
const featureKeyChina = (f) => provinceKey(f.properties.name)
const featureLabelChina = (f) => f.properties.name || ''

// 世界模式：国家中文名 → ISO2（匹配地图着色）
const worldPoints = computed(() => geoWorld.value
  .filter((g) => !LOCAL[g.country])
  .map((g) => ({ key: countryToISO(g.country), name: g.country, count: g.count }))
  .filter((p) => p.key))

// 中国模式：GeoIP 省份名 → 归一化省份键；无经纬度时回退省会坐标
const chinaPoints = computed(() => geoChina.value
  .map((g) => {
    const key = provinceKey(g.country)
    const c = (g.lat !== 0 && g.lon !== 0) ? [g.lon, g.lat] : provinceCenter(key)
    return { key, name: g.country, count: g.count }
  })
  .filter((p) => p.key))

const mapPoints = computed(() => (mapMode.value === 'world' ? worldPoints.value : chinaPoints.value))
const mapData = computed(() => (mapMode.value === 'world' ? worldMap.value : chinaMap.value))

// 地图下方的 Top 地区标签（跟随模式）
const chips = computed(() => {
  const src = mapMode.value === 'world' ? geoWorld.value : geoChina.value
  return [...src]
    .sort((a, b) => b.count - a.count)
    .slice(0, 8)
    .map((g) => ({ name: g.country, count: g.count }))
})

const mapNote = computed(() => mapMode.value === 'world'
  ? '说明：需在服务器放置 MaxMind GeoLite2-City.mmdb（GEO_DB_PATH 指定）后，才能把公网 IP 解析到国家/城市；内网与本地访问归入「内网」。'
  : '说明：中国模式基于 GeoIP 省级行政区（subdivision）解析，统计中国内地及港澳台访问；未识别省份不会上图。')

async function switchMode(mode) {
  mapMode.value = mode
  await ensureMapData(mode)
  if (mode === 'china' && !chinaLoaded.value) {
    chinaLoaded.value = true
    const res = await api.get('/api/stats/geo', { params: { days: 30, scope: 'china' } })
    if (res.code === 0) geoChina.value = res.data
  }
}

async function load() {
  loading.value = true
  try {
    const [a, b, c, d, e] = await Promise.all([
      api.get('/api/stats/summary'),
      api.get('/api/stats/trend', { params: { days: 14 } }),
      api.get('/api/stats/top', { params: { limit: 5 } }),
      api.get('/api/stats/top-ips', { params: { limit: 5 } }),
      api.get('/api/stats/geo', { params: { days: 30 } })
    ])
    if (a.code === 0) Object.assign(summary, a.data)
    if (b.code === 0) trend.value = b.data
    if (c.code === 0) top.value = c.data
    if (d.code === 0) geoWorld.value = d.data
    if (e.code === 0) topIPs.value = e.data
    await ensureMapData('world')
  } finally {
    loading.value = false
  }
}

onMounted(load)
</script>

<style scoped>
.overview {
  display: grid;
  grid-template-columns: minmax(0, 1fr) auto;
  align-items: end;
  gap: 40px;
  padding: clamp(40px, 5vw, 68px) 0 40px;
  border-bottom: 1px solid var(--line);
}
.overview h1 { margin-top: 16px; display: flex; align-items: baseline; gap: 16px; flex-wrap: wrap; }
.overview h1 .count { font-size: clamp(50px, 6.6vw, 86px); font-weight: 500; letter-spacing: -0.045em; line-height: 1; }
.overview h1 .count-unit { font-size: 20px; font-weight: 400; color: var(--ink-3); letter-spacing: -0.01em; }
.overview-sub { margin-top: 18px; font-size: 14px; color: var(--ink-3); }
.metrics { display: flex; }
.metric { padding-left: clamp(18px, 2.6vw, 36px); margin-left: clamp(18px, 2.6vw, 36px); border-left: 1px solid var(--line); }
.metric:first-child { border-left: none; margin-left: 0; padding-left: 0; }
.metric dt { font-size: 12px; letter-spacing: 0.04em; color: var(--ink-4); margin-bottom: 8px; }
.metric dd { font-size: 26px; font-weight: 600; letter-spacing: -0.02em; }

.workspace {
  display: grid;
  grid-template-columns: minmax(0, 1fr) 340px;
  gap: clamp(36px, 5vw, 72px);
  align-items: start;
  padding-top: 48px;
}
.main-col, .side-col { display: flex; flex-direction: column; gap: 52px; min-width: 0; }
.section-head { display: flex; align-items: flex-end; justify-content: space-between; margin-bottom: 22px; }
.section-head h2, .section-head h3 { margin-top: 6px; font-size: 24px; font-weight: 600; letter-spacing: -0.02em; }
.section-head.tight { margin-bottom: 18px; }
.section-head h3 { font-size: 18px; }
.count-note { font-size: 12.5px; color: var(--ink-4); font-variant-numeric: tabular-nums; }
.state-note { padding: 40px 0; color: var(--ink-4); font-size: 14px; }
.empty-mini { padding: 40px 0; color: var(--ink-4); font-size: 14px; }

/* 地图 */
.map-tabs { display: flex; gap: 2px; border: 1px solid var(--line); border-radius: 999px; padding: 3px; }
.map-tabs button {
  padding: 5px 16px; font-size: 13px; color: var(--ink-3); background: none;
  border: none; border-radius: 999px; cursor: pointer;
  transition: color var(--t-fast) var(--ease), background var(--t-fast) var(--ease);
}
.map-tabs button:hover { color: var(--ink); }
.map-tabs button.active { color: var(--surface); background: var(--ink); }
.map-wrap { border: 1px solid var(--line); border-radius: var(--r-md); overflow: hidden; }
.local-chips { display: flex; flex-wrap: wrap; gap: 8px; margin-top: 14px; }
.chip {
  font-size: 12.5px; color: var(--ink-2); background: var(--surface);
  border: 1px solid var(--line); border-radius: 999px; padding: 5px 12px;
}
.chip b { color: var(--ink); margin-left: 4px; }
.map-note { margin-top: 12px; font-size: 12px; line-height: 1.8; color: var(--ink-4); }

/* 柱状图 */
.chart { display: flex; gap: 10px; height: 210px; border-bottom: 1px solid var(--line); }
.bar-col { flex: 1; min-width: 0; display: flex; flex-direction: column; }
.bar-area { flex: 1; display: flex; flex-direction: column; align-items: center; justify-content: flex-end; }
.bar-val { font-size: 11px; color: var(--ink-4); margin-bottom: 6px; }
.bar { width: min(22px, 70%); background: var(--ink); border-radius: 2px 2px 0 0; transition: height var(--t-base) var(--ease); min-height: 2px; }
.bar-col:hover .bar { background: var(--ink-2); }
.bar-label { padding-top: 8px; text-align: center; font-size: 10.5px; color: var(--ink-4); white-space: nowrap; overflow: hidden; }

/* Top */
.top-list { border-top: 1px solid var(--line); }
.top-row {
  display: grid;
  grid-template-columns: 28px minmax(0, 1fr) auto;
  gap: 12px; align-items: center; padding: 13px 2px;
  border-bottom: 1px solid var(--line);
}
.top-rank { font-size: 12px; color: var(--ink-4); font-weight: 500; }
.top-main { min-width: 0; }
.top-code { display: block; font-family: var(--font-mono); font-size: 14px; font-weight: 600; }
.top-url { display: block; margin-top: 2px; font-size: 12px; color: var(--ink-4); overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
.top-url:hover { color: var(--ink-3); }
.top-visits { font-size: 13px; color: var(--ink-3); font-variant-numeric: tabular-nums; }

@media (max-width: 1080px) {
  .workspace { grid-template-columns: 1fr; }
  .side-col { flex-direction: row; }
  .side-col > .panel { flex: 1; }
}
@media (max-width: 860px) {
  .overview { grid-template-columns: 1fr; }
  .metrics { flex-wrap: wrap; }
  .metric { border-left: none; margin-left: 0; padding-left: 0; border-top: 1px solid var(--line); padding-top: 14px; margin-right: 36px; }
  .side-col { flex-direction: column; }
}
</style>
