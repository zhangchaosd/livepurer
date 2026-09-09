<script setup lang="ts">
import { ref } from "vue";
import { useConsoleContext } from "../composables/useConsole";
import { platforms } from "../lib/types";
import Icon from "../components/Icon.vue";
import PlaylistLink from "../components/PlaylistLink.vue";
import EmptyState from "../components/EmptyState.vue";
const {
  channels,
  channelReady,
  channelsError,
  channelDirty,
  channelError,
  loadChannels,
  addChannel,
  moveChannel,
  saveChannels,
  discardChannels,
  pending,
  watchRoom,
  playlistURL,
  report,
} = useConsoleContext();
const expanded = ref<string>();
function remove(index: number) {
  channels.value.splice(index, 1);
  report("已从草稿移除。保存后更新订阅，也可撤销全部修改。", "info");
}
</script>
<template>
  <section class="panel subscription-panel">
    <div class="section-heading">
      <div class="title-with-icon">
        <span class="metric-icon"><Icon name="tv" :size="24" /></span>
        <div>
          <h2>你的电视订阅</h2>
          <p>添加频道并保存后，播放列表会自动更新。</p>
        </div>
      </div>
      <a class="button" href="/api/v1/live/m3u" download="channels.m3u"
        ><Icon name="download" :size="17" />下载 M3U</a
      >
    </div>
    <PlaylistLink :url="playlistURL()" />
  </section>
  <div class="content-toolbar">
    <div class="title-with-icon">
      <h2>
        频道列表 <span class="count-badge">{{ channels.length }}</span>
      </h2>
      <span class="pill" :class="channelDirty ? 'warning' : 'neutral'"
        ><i v-if="channelDirty" class="draft-dot"></i
        >{{ channelDirty ? "有未保存修改" : "已与订阅同步" }}</span
      >
    </div>
    <button
      class="button"
      :disabled="!channelReady || pending.has('channels')"
      @click="addChannel()"
    >
      <Icon name="plus" :size="17" />添加频道
    </button>
  </div>
  <div v-if="channelsError" class="inline-alert error" role="alert">
    频道加载失败：{{ channelsError
    }}<button class="text-button" @click="loadChannels">重试</button>
  </div>
  <div v-else-if="!channelReady" class="panel loading-state" role="status">
    <span class="spinner"></span>正在加载频道…
  </div>
  <section v-else class="panel channel-panel">
    <EmptyState
      v-if="!channels.length"
      icon="tv"
      title="把喜欢的直播带到电视上"
      description="添加平台与房间号，保存后即可在电视播放器订阅。"
      ><button class="button primary" @click="addChannel()">
        <Icon name="plus" :size="17" />添加第一个频道
      </button></EmptyState
    ><template v-else
      ><div class="channel-table-heading">
        <span>#</span><span>直播平台</span><span>房间号</span
        ><span>显示名称</span><span>操作</span>
      </div>
      <fieldset
        v-for="(channel, index) in channels"
        :key="channel.key"
        class="channel-row"
        :disabled="pending.has('channels')"
      >
        <legend class="sr-only">第 {{ index + 1 }} 个频道</legend>
        <span class="channel-index">{{
          String(index + 1).padStart(2, "0")
        }}</span
        ><label
          ><span class="mobile-label">平台</span
          ><select
            v-model="channel.plat"
            :aria-label="`第 ${index + 1} 个频道平台`"
          >
            <option
              v-for="platform in platforms"
              :key="platform.id"
              :value="platform.id"
            >
              {{ platform.name }}
            </option>
          </select></label
        ><label
          ><span class="mobile-label">房间号</span
          ><input
            v-model="channel.room"
            :aria-label="`第 ${index + 1} 个频道房间号`"
            placeholder="填写房间号" /></label
        ><label
          ><span class="mobile-label">显示名称</span
          ><input
            v-model="channel.name"
            :aria-label="`第 ${index + 1} 个频道显示名称`"
            placeholder="选填，默认使用直播标题"
        /></label>
        <div class="channel-actions">
          <button
            class="icon-button"
            :disabled="!channel.room.trim()"
            :aria-label="`播放第 ${index + 1} 个频道`"
            title="播放"
            @click="watchRoom(channel)"
          >
            <Icon name="play" :size="17" /></button
          ><button
            class="icon-button"
            :disabled="index === 0"
            :aria-label="`上移第 ${index + 1} 个频道`"
            title="上移"
            @click="moveChannel(index, -1)"
          >
            <Icon name="up" :size="17" /></button
          ><button
            class="icon-button"
            :disabled="index === channels.length - 1"
            :aria-label="`下移第 ${index + 1} 个频道`"
            title="下移"
            @click="moveChannel(index, 1)"
          >
            <Icon name="down" :size="17" /></button
          ><button
            class="icon-button"
            :aria-label="`设置第 ${index + 1} 个频道图标`"
            :aria-expanded="expanded === channel.key"
            title="图标地址"
            @click="
              expanded = expanded === channel.key ? undefined : channel.key
            "
          >
            <Icon name="settings" :size="17" /></button
          ><button
            class="icon-button danger"
            :aria-label="`移除第 ${index + 1} 个频道`"
            title="移除"
            @click="remove(index)"
          >
            <Icon name="trash" :size="17" />
          </button>
        </div>
        <label v-if="expanded === channel.key" class="channel-logo field"
          >频道图标地址<input
            v-model="channel.logo"
            type="url"
            placeholder="https://example.com/logo.png（可选）"
        /></label></fieldset
    ></template>
  </section>
  <div v-if="channelError" class="inline-alert error" role="alert">
    <Icon name="warning" />{{ channelError }}
  </div>
  <div v-if="channelReady" class="save-bar" :class="{ dirty: channelDirty }">
    <div>
      <Icon :name="channelDirty ? 'info' : 'check'" :size="18" /><span>{{
        channelDirty
          ? "草稿尚未同步到电视订阅"
          : "频道已保存，电视订阅使用当前列表"
      }}</span>
    </div>
    <div class="button-group">
      <button
        class="button"
        :disabled="!channelDirty || pending.has('channels')"
        @click="discardChannels"
      >
        撤销修改</button
      ><button
        class="button primary"
        :disabled="!channelDirty || pending.has('channels')"
        @click="saveChannels"
      >
        <Icon name="save" :size="17" />{{
          pending.has("channels") ? "保存中…" : "保存频道"
        }}
      </button>
    </div>
  </div>
  <div class="iptv-guide">
    <span>01 <strong>添加频道</strong></span
    ><Icon name="arrow" :size="18" /><span
      >02 <strong>保存播放列表</strong></span
    ><Icon name="arrow" :size="18" /><span
      >03 <strong>在电视添加订阅</strong></span
    >
  </div>
</template>
