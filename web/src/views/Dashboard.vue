<script setup lang="ts">
import { useConsoleContext } from "../composables/useConsole";
import { platformName } from "../lib/types";
import Icon from "../components/Icon.vue";
import RoomForm from "../components/RoomForm.vue";
import PlaylistLink from "../components/PlaylistLink.vue";
import EmptyState from "../components/EmptyState.vue";
const {
  navigate,
  system,
  systemState,
  updated,
  refreshSystem,
  pending,
  memoryPercent,
  savedChannels,
  channelReady,
  channelsError,
  loadChannels,
  favorites,
  favoritesReady,
  favoritesError,
  loadFavorites,
  recent,
  watchRoom,
  playlistURL,
} = useConsoleContext();
</script>
<template>
  <section class="welcome-panel">
    <div>
      <span class="pill"><i class="status-dot"></i>简单 · 纯粹 · 自由</span>
      <h2>打开直播，<br />回到你喜欢的世界。</h2>
      <p>聚合多个平台的直播间，收藏精彩，也把精彩带到大屏。</p>
      <RoomForm compact />
    </div>
    <div class="welcome-art" aria-hidden="true">
      <div class="orbit orbit-one"></div>
      <div class="orbit orbit-two"></div>
      <div class="art-monitor">
        <div class="art-top"><i></i><i></i><i></i></div>
        <div class="art-screen">
          <span class="art-play"><Icon name="play" :size="32" /></span
          ><span class="art-live">LIVE</span>
        </div>
        <div class="art-progress"><i></i></div>
      </div>
      <span class="art-chip chip-one"
        ><Icon name="heart" :size="16" />我的喜欢</span
      ><span class="art-chip chip-two"
        ><Icon name="tv" :size="17" />随处可看</span
      >
    </div>
  </section>
  <div class="metric-grid">
    <button class="metric-card" @click="navigate('iptv')">
      <span class="metric-icon"><Icon name="tv" /></span>
      <div>
        <span>已保存频道</span
        ><strong
          >{{
            channelReady
              ? savedChannels.length.toString().padStart(2, "0")
              : "—"
          }}<small>个频道</small></strong
        >
      </div>
      <Icon name="chevron" :size="17" /></button
    ><button class="metric-card" @click="navigate('favorites')">
      <span class="metric-icon rose"><Icon name="heart" /></span>
      <div>
        <span>我的收藏夹</span
        ><strong
          >{{
            favoritesReady ? favorites.length.toString().padStart(2, "0") : "—"
          }}<small>个收藏夹</small></strong
        >
      </div>
      <Icon name="chevron" :size="17" />
    </button>
    <div class="metric-card">
      <span class="metric-icon blue"><Icon name="server" /></span>
      <div>
        <span>运行环境</span
        ><strong class="metric-text">{{
          system.info
            ? `${system.info.os} / ${system.info.kernel_arch}`
            : "等待连接"
        }}</strong
        ><small>{{
          systemState === "online"
            ? "本地服务已就绪"
            : systemState === "offline"
              ? "暂时无法获取状态"
              : "正在读取服务状态"
        }}</small>
      </div>
    </div>
  </div>
  <div
    v-if="channelsError || favoritesError"
    class="inline-alert error"
    role="alert"
  >
    <Icon name="warning" /><span
      >{{ channelsError ? "频道加载失败。" : ""
      }}{{ favoritesError ? "收藏夹加载失败。" : "" }}</span
    ><button
      class="text-button"
      @click="
        channelsError && loadChannels();
        favoritesError && loadFavorites();
      "
    >
      重试
    </button>
  </div>
  <div class="dashboard-columns">
    <section class="panel">
      <header class="section-heading">
        <div>
          <h2>最近打开</h2>
          <p>继续上次的精彩</p>
        </div>
        <button class="text-button" @click="navigate('live')">
          全部记录<Icon name="arrow" :size="16" />
        </button>
      </header>
      <div v-if="recent.length" class="room-list">
        <button
          v-for="item in recent.slice(0, 4)"
          :key="`${item.plat}:${item.room}`"
          class="quick-room"
          @click="watchRoom(item)"
        >
          <span class="room-avatar">{{
            (item.upper || platformName(item.plat)).slice(0, 1)
          }}</span
          ><span
            ><strong>{{ item.upper || `房间 ${item.room}` }}</strong
            ><small
              >{{ platformName(item.plat) }} · {{ item.room }}</small
            ></span
          ><span class="play-circle"><Icon name="play" :size="16" /></span>
        </button>
      </div>
      <EmptyState
        v-else
        icon="clock"
        title="精彩，从第一场直播开始"
        description="在上方输入房间号，打开过的直播间会出现在这里。"
      />
    </section>
    <section class="panel system-panel">
      <header class="section-heading">
        <div>
          <h2>服务状态</h2>
          <p>{{ updated ? `更新于 ${updated}` : "正在连接服务" }}</p>
        </div>
        <button
          class="icon-button"
          aria-label="刷新服务状态"
          :disabled="pending.has('system')"
          @click="refreshSystem"
        >
          <Icon
            name="refresh"
            :class="{ spinning: pending.has('system') }"
            :size="18"
          />
        </button>
      </header>
      <div
        v-if="systemState === 'offline'"
        class="inline-alert error"
        role="alert"
      >
        服务状态读取失败，请检查程序是否正在运行。
      </div>
      <p
        v-if="system.unavailable?.length && systemState === 'online'"
        class="help-text"
      >
        部分系统指标暂不可用，以 — 显示。
      </p>
      <div class="resource">
        <div>
          <span><Icon name="cpu" :size="17" />CPU 使用率</span
          ><strong>{{
            systemState === "online" && system.sys_cpu
              ? system.sys_cpu.percent.toFixed(1) + "%"
              : "—"
          }}</strong>
        </div>
        <div class="progress-track">
          <i
            :style="{
              width: `${systemState === 'online' ? system.sys_cpu?.percent || 0 : 0}%`,
            }"
          ></i>
        </div>
      </div>
      <div class="resource">
        <div>
          <span><Icon name="memory" :size="17" />内存使用率</span
          ><strong>{{
            systemState === "online" && system.sys_mem?.total
              ? memoryPercent.toFixed(1) + "%"
              : "—"
          }}</strong>
        </div>
        <div class="progress-track blue">
          <i
            :style="{
              width: `${systemState === 'online' ? memoryPercent : 0}%`,
            }"
          ></i>
        </div>
      </div>
      <div class="system-meta">
        <span>进程内存</span
        ><strong>{{
          systemState === "online" ? system.self_mem?.mem_str || "—" : "—"
        }}</strong>
      </div>
      <p class="help-text">状态每 15 秒自动更新</p>
    </section>
  </div>
  <section class="panel television-panel">
    <div class="section-heading">
      <div class="title-with-icon">
        <span class="metric-icon"><Icon name="tv" :size="24" /></span>
        <div>
          <h2>在大屏上，继续精彩</h2>
          <p>一个订阅地址，连接你的全部 IPTV 频道。</p>
        </div>
      </div>
      <button class="text-button" @click="navigate('iptv')">
        管理频道<Icon name="arrow" :size="16" />
      </button>
    </div>
    <PlaylistLink :url="playlistURL()" />
  </section>
</template>
