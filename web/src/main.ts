import { createApp, ref, onMounted, onUnmounted, watch, nextTick } from 'vue'
import flvjs from 'flv.js'
import './style.css'

type ApiResponse<T> = { code: number; msg: string; data: T }
const api = async <T>(path: string, init?: RequestInit): Promise<T> => {
  const response = await fetch(`/api/v1${path}`, { signal: AbortSignal.timeout(20000), headers: { 'Content-Type': 'application/json' }, ...init })
  const body = await response.json() as ApiResponse<T>
  if (!response.ok || body.code !== 0) throw new Error(body.msg || '请求失败')
  return body.data
}

const App = {
  setup() {
    const tab = ref('dashboard'), notice = ref('')
    const origin = window.location.origin, playbackStatus = ref('尚未播放'), busy = ref(false), channelDirty = ref(false)
    let statusTimer: ReturnType<typeof setInterval>, noticeTimer: ReturnType<typeof setTimeout>
    const stop = () => { flvPlayer?.destroy(); flvPlayer = undefined; playbackStatus.value = '已停止' }
    watch(tab, () => { stop(); showFavoritePicker.value = false })
    const server = ref<any>({ socks5: {} }), account = ref<any>({ bilibili: {}, huya: {}, douyu: {} })
    const channels = ref<any[]>([]), favorites = ref<any[]>([]), system = ref<any>({}), rooms = ref<any[]>([])
    const currentFavList = ref<any>(), newFavListTitle = ref('')
    const room = ref({ plat: 'bilibili', room: '' }), loading = ref(false), player = ref<HTMLVideoElement>()
    const showFavoritePicker = ref(false), selectedFavoriteIDs = ref<number[]>([])
    watch(() => [room.value.plat, room.value.room], () => { rooms.value = []; showFavoritePicker.value = false })
    let flvPlayer: flvjs.Player | undefined
    const report = (message: string) => { clearTimeout(noticeTimer); notice.value = message; noticeTimer = setTimeout(() => notice.value = '', 6000) }
    const safely = async (action: () => Promise<void>) => { if (busy.value) return; busy.value = true; try { await action() } catch(error) { report((error as Error).message) } finally { busy.value = false } }
    const load = async () => {
      try {
        ;[server.value, account.value, channels.value, favorites.value, system.value] = await Promise.all([
          api('/settings/server'), api('/settings/account'), api<any[]>('/settings/channels').then(value => value || []), api('/fav/list/get_all'), api('/os/all')
        ])
      } catch (error) { report((error as Error).message) }
    }
    const saveServer = () => safely(async () => { await api('/settings/server', { method: 'PUT', body: JSON.stringify(server.value) }); report('服务配置已保存，重启后生效') })
    const saveAccount = () => safely(async () => { await api('/settings/account', { method: 'PUT', body: JSON.stringify(account.value) }); report('账号配置已保存，重启后生效') })
    const saveChannels = () => safely(async () => {
      const keys = new Set<string>()
      for (const channel of channels.value) {
        channel.room = channel.room.trim()
        if (!channel.room) throw new Error('请填写所有频道的房间号')
        const key = channel.plat + ':' + channel.room
        if (keys.has(key)) throw new Error('存在重复频道：' + key)
        keys.add(key)
      }
      await api('/settings/channels', { method: 'PUT', body: JSON.stringify(channels.value) })
      channelDirty.value = false
      report('频道配置已保存，M3U 已更新')
    })
    const moveChannel = (index: number, offset: number) => {
      const target = index + offset
      if (target < 0 || target >= channels.value.length) return
      const [channel] = channels.value.splice(index, 1)
      channels.value.splice(target, 0, channel)
      channelDirty.value = true
    }
    const addRoomChannel = () => {
      if (!room.value.room.trim()) return
      if (channels.value.some(c => c.plat === room.value.plat && c.room === room.value.room.trim())) { report('频道已存在'); return }
      channels.value.push({ ...room.value, room: room.value.room.trim(), name: rooms.value[0]?.upper || '', logo: '' })
      channelDirty.value = true
      report('已加入频道草稿，请在 IPTV 页保存')
    }
    const watchRoom = async (item: any) => { tab.value = 'live'; room.value = {plat: item.plat, room: item.room}; rooms.value = []; await nextTick(); play(); await resolve() }
    const copyText = async (value: string) => {
      try {
        if (navigator.clipboard && window.isSecureContext) await navigator.clipboard.writeText(value)
        else {
          const input = document.createElement('textarea'); input.value = value; document.body.appendChild(input); input.select()
          const copied = document.execCommand('copy'); input.remove(); if (!copied) throw new Error()
        }
        report('地址已复制')
      } catch { report('复制失败，请手动复制显示的地址') }
    }
    const addChannel = () => { channels.value.push({ plat: 'bilibili', room: '', name: '', logo: '' }); channelDirty.value = true }
    const memPercent = () => system.value.sys_mem?.total ? ((1 - system.value.sys_mem.avl / system.value.sys_mem.total) * 100).toFixed(1) : '-'
    const resolve = async () => {
      if (!room.value.room.trim()) { report('请输入房间号'); return }
      room.value.room = room.value.room.trim()
      loading.value = true
      rooms.value = []
      try { rooms.value = [await api(`/live/room_info?plat=${encodeURIComponent(room.value.plat)}&room=${encodeURIComponent(room.value.room)}`)] } catch (error) { report((error as Error).message) } finally { loading.value = false }
    }
    const play = () => {
      if (!player.value) return
      const url = `/api/v1/live/play?plat=${encodeURIComponent(room.value.plat)}&room=${encodeURIComponent(room.value.room)}`
      stop()
      if (!flvjs.isSupported()) { window.open(url, '_blank'); return }
      playbackStatus.value = '正在连接'
      flvPlayer = flvjs.createPlayer({ type: 'flv', url, isLive: true }, { enableStashBuffer: true, stashInitialSize: 384 * 1024, lazyLoad: false, autoCleanupSourceBuffer: true, autoCleanupMaxBackwardDuration: 30, autoCleanupMinBackwardDuration: 10 })
      flvPlayer.on(flvjs.Events.ERROR, () => { playbackStatus.value = '播放失败'; report('直播流暂不可用，请确认开播后重试；也可复制地址到电视播放器') })
      flvPlayer.attachMediaElement(player.value)
      flvPlayer.load()
      const activePlayer = flvPlayer
      player.value.play().catch((error) => { if (activePlayer === flvPlayer && error.name === 'NotAllowedError') report('点击播放按钮开始直播') })
    }
    const openFavoritePicker = () => {
      if (!rooms.value.length) { report('请先查询直播间'); return }
      selectedFavoriteIDs.value = []
      showFavoritePicker.value = true
    }
    const addToSelectedFavorites = async () => {
      if (!selectedFavoriteIDs.value.length) { report('请选择至少一个收藏夹'); return }
      try {
        await Promise.all(selectedFavoriteIDs.value.map(async (id) => {
          const result = await api<any>(`/fav/list/get?id=${id}`)
          if (result.favorites?.some((item: any) => item.plat === room.value.plat && item.room === room.value.room)) return
          await api('/fav/add', { method: 'POST', body: JSON.stringify({ fid: result.id, order: (result.favorites?.length || 0) + 1, plat: room.value.plat, room: room.value.room, upper: rooms.value[0]?.upper || room.value.room }) })
        }))
        showFavoritePicker.value = false
        report(`已添加到 ${selectedFavoriteIDs.value.length} 个收藏夹`)
      } catch (error) { report((error as Error).message) }
    }
    const refreshFavoriteLists = async () => {
      try { favorites.value = await api('/fav/list/get_all') } catch(error) { report((error as Error).message) }
    }
    const selectFavoriteList = async (id: number) => {
      try { currentFavList.value = await api(`/fav/list/get?id=${id}`) } catch (error) { report((error as Error).message) }
    }
    const createFavoriteList = async () => {
      if (newFavListTitle.value.trim().length < 2) { report('收藏夹名称至少需要两个字符'); return }
      try {
        const created = await api<any>('/fav/list/add', { method: 'POST', body: JSON.stringify({ title: newFavListTitle.value.trim(), order: favorites.value.length + 1 }) })
        newFavListTitle.value = ''
        await refreshFavoriteLists()
        await selectFavoriteList(created.id)
        report('已创建收藏夹')
      } catch (error) { report((error as Error).message) }
    }
    const saveFavoriteList = async () => {
      if (!currentFavList.value) return
      try {
        await api('/fav/list/edit', { method: 'POST', body: JSON.stringify({ id: currentFavList.value.id, title: currentFavList.value.title, order: currentFavList.value.order || 1 }) })
        await refreshFavoriteLists()
        report('收藏夹已保存')
      } catch (error) { report((error as Error).message) }
    }
    const deleteFavoriteList = async () => {
      if (!currentFavList.value || !window.confirm('删除收藏夹及其中的收藏？')) return
      try {
        await api('/fav/list/del', { method: 'POST', body: JSON.stringify({ id: currentFavList.value.id }) })
        currentFavList.value = undefined
        await refreshFavoriteLists()
        report('收藏夹已删除')
      } catch (error) { report((error as Error).message) }
    }
    const deleteFavorite = async (id: number) => {
      try {
        await api('/fav/del', { method: 'POST', body: JSON.stringify({ id }) })
        await selectFavoriteList(currentFavList.value.id)
        report('直播间已从收藏夹移除')
      } catch (error) { report((error as Error).message) }
    }
    const playlistURL = (id: number) => `${window.location.origin}/api/v1/live/m3u?fav_list_id=${id}`
    const copyPlaylistURL = async (id: number) => {
      await copyText(playlistURL(id))
    }
    onMounted(() => { load(); statusTimer = setInterval(async () => { if (tab.value !== 'dashboard') return; try { system.value = await api('/os/all') } catch {} }, 10000) })
    onUnmounted(() => { stop(); clearInterval(statusTimer); clearTimeout(noticeTimer) })
    return { origin, playbackStatus, busy, channelDirty, stop, moveChannel, addRoomChannel, watchRoom, copyText, tab, notice, server, account, channels, favorites, currentFavList, newFavListTitle, system, room, rooms, loading, player, showFavoritePicker, selectedFavoriteIDs, load, saveServer, saveAccount, saveChannels, addChannel, memPercent, resolve, play, openFavoritePicker, addToSelectedFavorites, refreshFavoriteLists, selectFavoriteList, createFavoriteList, saveFavoriteList, deleteFavoriteList, deleteFavorite, playlistURL, copyPlaylistURL, report }
  },
  template: `
  <main><aside><h1>Pure Live</h1><button v-for="item in [['dashboard','概览'],['live','直播'],['favorites','收藏'],['iptv','IPTV'],['settings','设置']]" :class="{active:tab===item[0]}" @click="tab=item[0]">{{item[1]}}</button></aside>
  <section><p v-if="notice" role="status" class="notice">{{notice}}</p>
  <template v-if="tab==='dashboard'"><h2>概览</h2><p class="muted">直播服务控制台 · 系统状态每 10 秒更新</p><div class="cards"><article><b>CPU</b><strong>{{system.sys_cpu?.percent?.toFixed?.(1) ?? '-'}}%</strong></article><article><b>内存</b><strong>{{memPercent()}}%</strong></article><article><b>数据目录</b><strong>{{server.path || '-'}}</strong></article></div><button @click="load">刷新状态</button><h3>电视播放</h3><p>在电视播放器中添加下方 M3U 地址。电视与本机需连接同一局域网，请使用本机局域网 IP 替换 localhost。</p><p class="playlist"><code>{{origin}}/api/v1/live/m3u</code><button @click="copyText(origin+'/api/v1/live/m3u')">复制订阅地址</button></p><div class="cards"><article><b>IPTV 频道</b><strong>{{channels.length}}</strong></article><article><b>收藏夹</b><strong>{{favorites.length}}</strong></article><article><b>控制台</b><strong>运行中</strong></article></div></template>
  <template v-else-if="tab==='live'"><h2>直播播放</h2><div class="form row"><label>平台<select v-model="room.plat"><option>bilibili</option><option>douyu</option><option>huya</option><option>inke</option></select></label><label>房间号<input v-model="room.room" placeholder="例如 6" @keyup.enter="resolve" /></label><button @click="resolve" :disabled="loading">查询</button><button @click="play" :disabled="!room.room">播放</button><button @click="openFavoritePicker" :disabled="!rooms.length">添加</button></div><div class="toolbar"><button @click="stop">停止播放</button><button @click="play" :disabled="!room.room">重新连接</button><button @click="addRoomChannel" :disabled="!room.room">加入 IPTV</button><button :disabled="!room.room" @click="copyText(origin+'/api/v1/live/play?plat='+encodeURIComponent(room.plat)+'&room='+encodeURIComponent(room.room))">复制播放地址</button><span role="status">{{playbackStatus}}</span></div><video ref="player" controls autoplay muted playsinline class="player" @playing="playbackStatus='播放中'" @waiting="playbackStatus='缓冲中'" @pause="playbackStatus='已暂停'"></video><article v-for="item in rooms" class="room"><b>{{item.upper}}</b><span>{{item.title}}</span><em :class="item.status ? 'online' : ''">{{item.status ? '直播中' : '未开播'}}</em></article><div v-if="showFavoritePicker" class="modal-backdrop" @click.self="showFavoritePicker=false"><section class="modal"><h3>添加到收藏夹</h3><p>可选择多个收藏夹：</p><label v-for="list in favorites" class="choice"><input type="checkbox" :value="list.id" v-model="selectedFavoriteIDs"/> {{list.title}}</label><p v-if="!favorites.length">暂无收藏夹，请先创建收藏夹。</p><div><button @click="showFavoritePicker=false">取消</button><button class="primary" @click="addToSelectedFavorites">确认添加</button></div></section></div></template>
  <template v-else-if="tab==='favorites'"><h2>收藏夹</h2><div class="form row"><label>新收藏夹名称<input v-model="newFavListTitle" placeholder="例如：常看直播"/></label><button @click="createFavoriteList">新建收藏夹</button><button @click="refreshFavoriteLists">刷新</button></div><div class="fav-layout"><div class="fav-lists"><div v-for="list in favorites" class="fav-list-item"><button :class="{active:currentFavList?.id===list.id}" @click="selectFavoriteList(list.id)">{{list.title}}</button><a :href="playlistURL(list.id)" target="_blank" title="打开 M3U">M3U</a></div></div><section v-if="currentFavList" class="fav-detail"><div class="form row"><label>名称<input v-model="currentFavList.title"/></label><label>排序<input type="number" v-model.number="currentFavList.order"/></label><button @click="saveFavoriteList">保存</button><button @click="deleteFavoriteList">删除收藏夹</button></div><h3>{{currentFavList.title}}（{{currentFavList.favorites?.length || 0}}）</h3><p class="playlist"><code>{{playlistURL(currentFavList.id)}}</code><button @click="copyPlaylistURL(currentFavList.id)">复制 M3U 地址</button></p><p v-if="!currentFavList.favorites?.length">此收藏夹暂无直播间。可在“直播”页查询后点击“添加”。</p><article v-for="favorite in currentFavList.favorites" class="favorite"><div><b>{{favorite.upper}}</b><span>{{favorite.plat}} · {{favorite.room}}</span></div><button @click="watchRoom(favorite)">播放</button><button @click="deleteFavorite(favorite.id)">移除</button></article></section><section v-else class="fav-detail"><p>请选择或新建一个收藏夹。</p></section></div></template>
  <template v-else-if="tab==='iptv'"><h2>IPTV 频道</h2><p>播放列表：<code>{{origin}}/api/v1/live/m3u</code></p><div class="toolbar"><button @click="copyText(origin+'/api/v1/live/m3u')">复制 M3U 地址</button><a :href="origin+'/api/v1/live/m3u'" download="channels.m3u">下载播放列表</a><span>{{channels.length}} 个频道 · {{channelDirty ? '有未保存修改' : '已保存'}}</span></div><p v-if="!channels.length">暂无频道，添加房间号后保存即可在电视播放。</p><div v-for="(channel,index) in channels" class="channel" @input="channelDirty=true" @change="channelDirty=true"><select v-model="channel.plat"><option>bilibili</option><option>douyu</option><option>huya</option><option>inke</option></select><input v-model="channel.room" placeholder="房间号"/><input v-model="channel.name" placeholder="显示名称（可选）"/><input v-model="channel.logo" placeholder="Logo URL（可选）"/><div class="toolbar"><button @click="watchRoom(channel)" :disabled="!channel.room">播放</button><button @click="moveChannel(index,-1)" :disabled="index===0" aria-label="上移">↑</button><button @click="moveChannel(index,1)" :disabled="index===channels.length-1" aria-label="下移">↓</button><button @click="channels.splice(index,1);channelDirty=true">删除</button></div></div><button @click="addChannel">添加频道</button><button class="primary" @click="saveChannels" :disabled="busy">保存频道</button></template>
  <template v-else><h2>设置</h2><h3>服务</h3><div class="form"><label>端口<input type="number" v-model.number="server.port"/></label><label>数据目录<input v-model="server.path"/></label><label><input type="checkbox" v-model="server.debug"/> Debug 日志</label><label><input type="checkbox" v-model="server.socks5.enable"/> SOCKS5</label><label>代理地址<input v-model="server.socks5.host"/></label><label>代理端口<input type="number" v-model.number="server.socks5.port"/></label><label>代理用户名<input v-model="server.socks5.user"/></label><label>代理密码<input type="password" v-model="server.socks5.password" placeholder="留空则保留原值"/></label></div><button class="primary" @click="saveServer" :disabled="busy">保存服务设置</button><h3>账号</h3><div class="form"><label><input type="checkbox" v-model="account.bilibili.enable"/> 启用 Bilibili</label><label>DedeUserID<input v-model="account.bilibili.DedeUserID" placeholder="留空则保留原值"/></label><label>DedeUserIDCkMd5<input v-model="account.bilibili.DedeUserIDCkMd5" placeholder="留空则保留原值"/></label><label>SESSDATA<input type="password" v-model="account.bilibili.SESSDATA" placeholder="留空则保留原值"/></label><label>BiliJCT<input type="password" v-model="account.bilibili.BiliJCT" placeholder="留空则保留原值"/></label><label><input type="checkbox" v-model="account.huya.enable"/> 启用虎牙</label><label>虎牙 Cookies<textarea v-model="account.huya.cookies" placeholder="留空则保留原值"></textarea></label><label><input type="checkbox" v-model="account.douyu.enable"/> 启用斗鱼</label></div><button class="primary" @click="saveAccount" :disabled="busy">保存账号设置</button></template>
  </section></main>`
}
const app = createApp(App)
app.mount('#app')
