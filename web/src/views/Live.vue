<script setup lang="ts">
import { onBeforeUnmount } from "vue";
import { useConsoleContext } from "../composables/useConsole";
import { platformName } from "../lib/types";
import Icon from "../components/Icon.vue";
import RoomForm from "../components/RoomForm.vue";
import Modal from "../components/Modal.vue";
import EmptyState from "../components/EmptyState.vue";
const {
  player,
  room,
  roomInfo,
  recent,
  watchRoom,
  looking,
  openPicker,
  addChannel,
  channelReady,
  copy,
  sharePlaybackURL,
  pickerOpen,
  selectedIDs,
  favorites,
  favoritesReady,
  favoritesError,
  loadFavorites,
  addFavorites,
  pending,
  navigate,
} = useConsoleContext();
const { video, state, activeRoom, mediaEvent, stop } = player;
const labels = {
  idle: "等待播放",
  connecting: "正在连接",
  playing: "正在播放",
  waiting: "正在缓冲",
  paused: "已暂停",
  stopped: "已停止",
  error: "连接失败",
  unsupported: "浏览器暂不支持",
};
onBeforeUnmount(stop);
</script>
<template>
  <section class="panel player-search"><RoomForm /></section>
  <div class="live-columns">
    <section class="player-panel">
      <div class="player-header">
        <span
          ><i class="status-dot" :class="{ muted: state !== 'playing' }"></i
          >{{
            activeRoom
              ? `${platformName(activeRoom.plat)} · ${activeRoom.room}`
              : "LIVE PLAYER"
          }}</span
        ><span role="status">{{ labels[state] }}</span>
      </div>
      <div class="video-stage">
        <video
          ref="video"
          controls
          autoplay
          muted
          playsinline
          :class="{
            hidden: ['idle', 'stopped', 'error', 'unsupported'].includes(state),
          }"
          @playing="mediaEvent('playing')"
          @waiting="mediaEvent('waiting')"
          @pause="mediaEvent('paused')"
          @ended="mediaEvent('stopped')"
        ></video>
        <div
          v-if="['idle', 'stopped', 'error', 'unsupported'].includes(state)"
          class="player-placeholder"
        >
          <span class="player-empty-icon"
            ><Icon :name="state === 'error' ? 'refresh' : 'play'" :size="34"
          /></span>
          <h2>
            {{
              state === "error"
                ? "暂时无法连接直播"
                : state === "unsupported"
                  ? "使用外部播放器观看"
                  : state === "stopped"
                    ? "直播已停止"
                    : "下一场精彩，等你开启"
            }}
          </h2>
          <p>
            {{
              state === "error"
                ? "确认房间已开播后重试，或复制播放地址到外部播放器。"
                : state === "unsupported"
                  ? "当前浏览器不支持 FLV 播放，可复制地址到 VLC、APTV 等播放器。"
                  : "选择平台，输入房间号，即可开始观看。"
            }}
          </p>
          <button
            v-if="state === 'error' || (state === 'stopped' && activeRoom)"
            class="button player-retry"
            @click="watchRoom(activeRoom)"
          >
            <Icon name="refresh" :size="16" />重新连接
          </button>
        </div>
        <div
          v-if="state === 'connecting'"
          class="player-connecting"
          role="status"
        >
          <span class="spinner"></span>正在连接直播流…
        </div>
      </div>
      <div class="player-toolbar">
        <div>
          <button
            class="button subtle"
            :disabled="
              ['idle', 'stopped', 'error', 'unsupported'].includes(state)
            "
            @click="stop"
          >
            <Icon name="stop" :size="16" />停止</button
          ><button
            class="button subtle"
            :disabled="!activeRoom"
            @click="watchRoom(activeRoom)"
          >
            <Icon name="refresh" :size="16" />重连
          </button>
        </div>
        <button
          class="button subtle"
          :disabled="!activeRoom || !sharePlaybackURL(activeRoom)"
          @click="activeRoom && copy(sharePlaybackURL(activeRoom))"
        >
          <Icon name="copy" :size="16" />复制播放地址
        </button>
      </div>
      <div v-if="activeRoom" class="stream-address">
        <input
          :value="sharePlaybackURL(activeRoom)"
          placeholder="等待获取局域网地址"
          readonly
          aria-label="稳定播放地址"
          @focus="($event.target as HTMLInputElement).select()"
        />
      </div>
      <div class="room-detail">
        <div>
          <span class="room-avatar"><Icon name="monitor" /></span>
          <div>
            <h2>
              {{
                looking ? "正在查询直播间…" : roomInfo?.upper || "直播间信息"
              }}
            </h2>
            <p>
              {{ roomInfo?.title || "查询房间信息后，可添加收藏或加入 IPTV。" }}
            </p>
          </div>
          <span
            v-if="roomInfo"
            class="pill"
            :class="{ neutral: !roomInfo.status }"
            >{{ roomInfo.status ? "直播中" : "未开播" }}</span
          >
        </div>
        <div class="room-actions">
          <button
            class="button"
            :disabled="!roomInfo || !favoritesReady"
            @click="openPicker"
          >
            <Icon name="heart" :size="17" />添加收藏</button
          ><button
            class="button"
            :disabled="!room.room.trim() || !channelReady"
            @click="addChannel(room, roomInfo?.upper)"
          >
            <Icon name="plus" :size="17" />加入 IPTV
          </button>
        </div>
      </div>
    </section>
    <aside class="panel recent-panel">
      <header class="section-heading">
        <h2>最近打开</h2>
        <span class="count-badge">{{ recent.length }}</span>
      </header>
      <div v-if="recent.length" class="room-list">
        <button
          v-for="item in recent"
          :key="`${item.plat}:${item.room}`"
          class="quick-room"
          @click="watchRoom(item)"
        >
          <span class="room-avatar small">{{
            (item.upper || platformName(item.plat)).slice(0, 1)
          }}</span
          ><span
            ><strong>{{ item.upper || `房间 ${item.room}` }}</strong
            ><small
              >{{ platformName(item.plat) }} · {{ item.room }}</small
            ></span
          ><Icon name="play" :size="15" />
        </button>
      </div>
      <EmptyState
        v-else
        icon="clock"
        title="还没有观看记录"
        description="打开过的房间会保存在当前浏览器，方便下次继续。"
      />
      <div class="recent-tip">
        <Icon name="info" :size="17" />
        <p>切换页面会停止当前播放。收藏直播间，下次更快找到。</p>
      </div>
    </aside>
  </div>
  <Modal
    :open="pickerOpen"
    title="添加到收藏夹"
    @close="!pending.has('add-favorites') && (pickerOpen = false)"
    ><p class="dialog-description">
      选择一个或多个收藏夹，保存你喜欢的直播间。
    </p>
    <div v-if="favoritesError" class="inline-alert error">
      {{ favoritesError
      }}<button class="text-button" @click="loadFavorites">重试</button>
    </div>
    <div v-if="favorites.length" class="picker-list">
      <label v-for="list in favorites" :key="list.id"
        ><input
          v-model="selectedIDs"
          type="checkbox"
          :value="list.id"
          :disabled="pending.has('add-favorites')"
        /><Icon name="folder" :size="19" />{{ list.title }}</label
      >
    </div>
    <EmptyState
      v-else
      icon="folder"
      title="先创建一个收藏夹"
      description="给喜欢的直播间一个专属的位置。"
      ><button
        class="button"
        @click="
          pickerOpen = false;
          navigate('favorites');
        "
      >
        去创建收藏夹<Icon name="arrow" :size="16" /></button
    ></EmptyState>
    <div class="dialog-actions">
      <button
        class="button"
        :disabled="pending.has('add-favorites')"
        @click="pickerOpen = false"
      >
        取消</button
      ><button
        class="button primary"
        :disabled="!selectedIDs.length || pending.has('add-favorites')"
        @click="addFavorites"
      >
        {{
          pending.has("add-favorites")
            ? "添加中…"
            : `添加${selectedIDs.length ? `到 ${selectedIDs.length} 个收藏夹` : ""}`
        }}
      </button>
    </div></Modal
  >
</template>
