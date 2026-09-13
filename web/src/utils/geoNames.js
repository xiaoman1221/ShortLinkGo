// 地理名称匹配工具：把后端统计返回的中文国家/省名，映射到地图数据的匹配键。

/**
 * 中文国家名 → ISO 3166-1 alpha-2（对应 world.json 的 ISO_A2 属性）。
 * 覆盖常见访问来源；未收录的国家不在地图上着色，但仍显示于列表。
 */
const COUNTRY_ISO = {
  中国: 'CN', 中国大陆: 'CN', 中国香港: 'HK', 香港: 'HK', 中国台湾: 'TW', 台湾: 'TW',
  中国澳门: 'MO', 澳门: 'MO', 美国: 'US', 日本: 'JP', 韩国: 'KR', 朝鲜: 'KP',
  新加坡: 'SG', 马来西亚: 'MY', 泰国: 'TH', 印度尼西亚: 'ID', 越南: 'VN',
  菲律宾: 'PH', 缅甸: 'MM', 柬埔寨: 'KH', 老挝: 'LA', 文莱: 'BN', 蒙古: 'MN',
  印度: 'IN', 巴基斯坦: 'PK', 孟加拉国: 'BD', 斯里兰卡: 'LK', 尼泊尔: 'NP',
  马尔代夫: 'MV', 不丹: 'BT', 阿富汗: 'AF', 伊朗: 'IR', 伊拉克: 'IQ',
  土耳其: 'TR', 叙利亚: 'SY', 黎巴嫩: 'LB', 以色列: 'IL', 巴勒斯坦: 'PS',
  约旦: 'JO', 沙特阿拉伯: 'SA', 阿联酋: 'AE', 卡塔尔: 'QA', 科威特: 'KW',
  巴林: 'BH', 阿曼: 'OM', 也门: 'YE', 哈萨克斯坦: 'KZ', 乌兹别克斯坦: 'UZ',
  土库曼斯坦: 'TM', 吉尔吉斯斯坦: 'KG', 塔吉克斯坦: 'TJ', 格鲁吉亚: 'GE',
  亚美尼亚: 'AM', 阿塞拜疆: 'AZ', 俄罗斯: 'RU', 乌克兰: 'UA', 白俄罗斯: 'BY',
  摩尔多瓦: 'MD', 德国: 'DE', 法国: 'FR', 英国: 'GB', 爱尔兰: 'IE',
  荷兰: 'NL', 比利时: 'BE', 卢森堡: 'LU', 奥地利: 'AT', 瑞士: 'CH',
  意大利: 'IT', 西班牙: 'ES', 葡萄牙: 'PT', 希腊: 'GR', 瑞典: 'SE',
  挪威: 'NO', 丹麦: 'DK', 芬兰: 'FI', 冰岛: 'IS', 波兰: 'PL',
  捷克: 'CZ', 斯洛伐克: 'SK', 匈牙利: 'HU', 罗马尼亚: 'RO', 保加利亚: 'BG',
  斯洛文尼亚: 'SI', 克罗地亚: 'HR', 塞尔维亚: 'RS', 波黑: 'BA', 黑山: 'ME',
  北马其顿: 'MK', 阿尔巴尼亚: 'AL', 立陶宛: 'LT', 拉脱维亚: 'LV',
  爱沙尼亚: 'EE', 马耳他: 'MT', 塞浦路斯: 'CY', 摩纳哥: 'MC',
  列支敦士登: 'LI', 安道尔: 'AD', 圣马力诺: 'SM', 加拿大: 'CA',
  墨西哥: 'MX', 危地马拉: 'GT', 洪都拉斯: 'HN', 萨尔瓦多: 'SV',
  尼加拉瓜: 'NI', 哥斯达黎加: 'CR', 巴拿马: 'PA', 古巴: 'CU',
  牙买加: 'JM', 海地: 'HT', 多米尼加: 'DO', 波多黎各: 'PR',
  巴哈马: 'BS', 巴巴多斯: 'BB', 特立尼达和多巴哥: 'TT', 百慕大: 'BM',
  巴西: 'BR', 阿根廷: 'AR', 智利: 'CL', 乌拉圭: 'UY', 巴拉圭: 'PY',
  玻利维亚: 'BO', 秘鲁: 'PE', 厄瓜多尔: 'EC', 哥伦比亚: 'CO',
  委内瑞拉: 'VE', 圭亚那: 'GY', 苏里南: 'SR', 澳大利亚: 'AU',
  新西兰: 'NZ', 斐济: 'FJ', 巴布亚新几内亚: 'PG', 埃及: 'EG',
  利比亚: 'LY', 突尼斯: 'TN', 阿尔及利亚: 'DZ', 摩洛哥: 'MA',
  苏丹: 'SD', 南苏丹: 'SS', 埃塞俄比亚: 'ET', 厄立特里亚: 'ER',
  吉布提: 'DJ', 索马里: 'SO', 肯尼亚: 'KE', 乌干达: 'UG',
  坦桑尼亚: 'TZ', 卢旺达: 'RW', 布隆迪: 'BI', 南非: 'ZA',
  纳米比亚: 'NA', 博茨瓦纳: 'BW', 津巴布韦: 'ZW', 赞比亚: 'ZM',
  马拉维: 'MW', 莫桑比克: 'MZ', 马达加斯加: 'MG', 毛里求斯: 'MU',
  安哥拉: 'AO', 刚果: 'CG', 刚果金: 'CD', 加蓬: 'GA',
  喀麦隆: 'CM', 中非: 'CF', 乍得: 'TD', 尼日尔: 'NE',
  尼日利亚: 'NG', 贝宁: 'BJ', 多哥: 'TG', 加纳: 'GH',
  科特迪瓦: 'CI', 象牙海岸: 'CI', 利比里亚: 'LR', 塞拉利昂: 'SL',
  几内亚: 'GN', 塞内加尔: 'SN', 冈比亚: 'GM', 马里: 'ML',
  布基纳法索: 'BF', 毛里塔尼亚: 'MR'
}

/** 世界模式：中文国名 → ISO2（未收录返回空）。兼容「香港特别行政区」「台湾省」等全称写法。 */
export function countryToISO(name) {
  const n = String(name || '').trim()
  if (!n) return ''
  if (COUNTRY_ISO[n]) return COUNTRY_ISO[n]
  return COUNTRY_ISO[n.replace(/(特别行政区|省)$/, '')] || ''
}

/** 省级行政区归一化：去掉「省/市/自治区」等后缀，便于与 GeoJSON 属性名互匹。 */
export function normalizeProvince(name) {
  return String(name || '')
    .replace(/特别行政区|维吾尔自治区|壮族自治区|回族自治区|自治区|省|市/g, '')
    .trim()
}

/** GeoLite2 英文省名 → 归一化中文键（mmdb 缺少 zh-CN subdivision 名时兜底）。 */
const PROVINCE_EN = {
  beijing: '北京', tianjin: '天津', shanghai: '上海', chongqing: '重庆',
  hebei: '河北', shanxi: '山西', liaoning: '辽宁', jilin: '吉林',
  'heilongjiang': '黑龙江', jiangsu: '江苏', zhejiang: '浙江',
  anhui: '安徽', fujian: '福建', jiangxi: '江西', shandong: '山东',
  henan: '河南', hubei: '湖北', hunan: '湖南', guangdong: '广东',
  hainan: '海南', sichuan: '四川', guizhou: '贵州', yunnan: '云南',
  shaanxi: '陕西', gansu: '甘肃', qinghai: '青海', taiwan: '台湾',
  'inner mongolia': '内蒙古', 'nei mongol': '内蒙古', guangxi: '广西',
  xizang: '西藏', tibet: '西藏', ningxia: '宁夏', xinjiang: '新疆',
  'hong kong': '香港', macau: '澳门', macao: '澳门'
}

/** 省级行政区中心坐标 [lon, lat]（API 无经纬度时兜底定位）。 */
export const PROVINCE_CENTER = {
  北京: [116.41, 39.9], 天津: [117.2, 39.08], 河北: [114.51, 38.04],
  山西: [112.55, 37.87], 内蒙古: [111.75, 40.84], 辽宁: [123.43, 41.8],
  吉林: [125.32, 43.82], 黑龙江: [126.53, 45.8], 上海: [121.47, 31.23],
  江苏: [118.8, 32.06], 浙江: [120.15, 30.28], 安徽: [117.28, 31.86],
  福建: [119.3, 26.08], 江西: [115.86, 28.68], 山东: [117.12, 36.65],
  河南: [113.63, 34.75], 湖北: [114.31, 30.59], 湖南: [112.94, 28.23],
  广东: [113.26, 23.13], 广西: [108.32, 22.82], 海南: [110.35, 20.02],
  重庆: [106.55, 29.56], 四川: [104.07, 30.57], 贵州: [106.63, 26.65],
  云南: [102.83, 24.88], 西藏: [91.11, 29.66], 陕西: [108.94, 34.34],
  甘肃: [103.83, 36.06], 青海: [101.78, 36.62], 宁夏: [106.28, 38.47],
  新疆: [87.62, 43.83], 台湾: [121.52, 25.03], 香港: [114.17, 22.32],
  澳门: [113.55, 22.2]
}

/** 中国模式：GeoIP subdivision 名 → 归一化省份键（兼容中文全称与英文名）。 */
export function provinceKey(name) {
  const raw = String(name || '').trim()
  if (!raw || raw === '未知') return ''
  if (/^[\u4e00-\u9fa5]/.test(raw)) return normalizeProvince(raw)
  return PROVINCE_EN[raw.toLowerCase()] || ''
}

/** 省份归一化键 → 中心坐标（未收录返回空）。 */
export function provinceCenter(key) {
  return PROVINCE_CENTER[key] || null
}
