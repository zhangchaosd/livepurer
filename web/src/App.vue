<script setup lang="ts">
import { computed, provide, watch } from "vue";
import { consoleKey, useConsole } from "./composables/useConsole";
import type { Page } from "./lib/types";
import Icon from "./components/Icon.vue";
import Modal from "./components/Modal.vue";
import Dashboard from "./views/Dashboard.vue";
import Live from "./views/Live.vue";
import Favorites from "./views/Favorites.vue";
import IPTV from "./views/IPTV.vue";
import Settings from "./views/Settings.vue";
const skipToContent = () =>
  document.querySelector<HTMLElement>("#content")?.focus();
const context = useConsole();
provide(consoleKey, context);
const {
  page,
  navigate,
  systemState,
  channelDirty,
  notice,
  confirmRequest,
  acceptConfirm,
  pending,
} = context;
const nav: { id: Page; label: string; icon: string; description: string }[] = [
  {
    id: "dashboard",
    label: "总览",
    icon: "grid",
    description: "你的直播，一处尽览。",
  },
  {
    id: "live",
    label: "直播播放",
    icon: "play",
    description: "专注每一场你喜欢的直播。",
  },
  {
    id: "favorites",
    label: "我的收藏",
    icon: "heart",
    description: "喜欢的直播间，下次一键见。",
  },
  {
    id: "iptv",
    label: "IPTV 频道",
    icon: "tv",
    description: "从小屏到大屏，让直播随处可看。",
  },
  {
    id: "settings",
    label: "设置",
    icon: "settings",
    description: "让服务按照你的习惯运行。",
  },
];
const active = computed(() => nav.find((n) => n.id === page.value)!);
watch(active, (value) => (document.title = `${value.label} · LivePurer`), {
  immediate: true,
});
const statusLabel = computed(
  () =>
    ({ loading: "连接服务中", online: "服务已连接", offline: "服务连接失败" })[
      systemState.value
    ],
);
</script>
<template>
  <a class="skip-link" href="#content" @click.prevent="skipToContent"
    >跳到主内容</a
  >
  <div class="app-shell">
    <aside class="sidebar">
      <a class="brand" href="#dashboard" @click.prevent="navigate('dashboard')"
        ><span class="brand-symbol"><Icon name="play" :size="22" /></span
        ><span>LivePurer<small>让直播回归纯粹</small></span></a
      >
      <div class="nav-caption">工作空间</div>
      <nav aria-label="主导航">
        <a
          v-for="item in nav"
          :key="item.id"
          :href="`#${item.id}`"
          :aria-current="page === item.id ? 'page' : undefined"
          :class="['nav-item', { active: page === item.id }]"
          @click.prevent="navigate(item.id)"
          ><Icon :name="item.icon" :size="19" /><span>{{ item.label }}</span
          ><span
            v-if="item.id === 'iptv' && channelDirty"
            class="draft-dot"
            aria-label="有未保存的频道修改"
          ></span
        ></a>
      </nav>
      <div class="sidebar-bottom">
        <div class="sidebar-note">
          <span class="mini-logo"><Icon name="monitor" :size="20" /></span
          ><strong>直播，本该简单。</strong>
          <p>一个入口，连接你喜欢的世界。</p>
        </div>
        <span class="sidebar-version"
          >LivePurer <span>本地直播控制台</span></span
        >
      </div>
    </aside>
    <div class="workspace">
      <header class="topbar">
        <span class="breadcrumb"
          >工作空间 <Icon name="chevron" :size="13" />
          <strong>{{ active.label }}</strong></span
        ><span class="connection" :class="systemState"
          ><i></i>{{ statusLabel }}</span
        >
      </header>
      <main id="content" class="page-content" tabindex="-1">
        <header class="page-heading">
          <div>
            <p class="eyebrow">
              {{
                page === "live"
                  ? "LIVE PLAYER"
                  : page === "iptv"
                    ? "ON THE BIG SCREEN"
                    : page === "favorites"
                      ? "YOUR COLLECTION"
                      : page === "settings"
                        ? "PREFERENCES"
                        : "YOUR LIVE WORKSPACE"
              }}
            </p>
            <h1 id="page-title" tabindex="-1">{{ active.label }}</h1>
            <p>{{ active.description }}</p>
          </div>
          <span class="heading-decoration"
            ><Icon :name="active.icon" :size="30"
          /></span>
        </header>
        <Dashboard v-if="page === 'dashboard'" /><Live
          v-else-if="page === 'live'"
        /><Favorites v-else-if="page === 'favorites'" /><IPTV
          v-else-if="page === 'iptv'"
        /><Settings v-else />
        <footer class="page-footer">
          <span>LivePurer</span><span>没有打扰，只有直播。</span>
        </footer>
      </main>
    </div>
    <Transition name="toast"
      ><div
        v-if="notice"
        class="toast"
        :class="notice.kind"
        :role="notice.kind === 'error' ? 'alert' : 'status'"
      >
        <Icon
          :name="
            notice.kind === 'error'
              ? 'warning'
              : notice.kind === 'info'
                ? 'info'
                : 'check'
          "
        /><span>{{ notice.message }}</span
        ><button
          class="icon-button"
          aria-label="关闭提示"
          @click="notice = undefined"
        >
          <Icon name="close" :size="17" />
        </button></div
    ></Transition>
    <Modal
      :open="!!confirmRequest"
      :title="confirmRequest?.title || '确认操作'"
      @close="!pending.has('confirm') && (confirmRequest = undefined)"
      ><p class="dialog-description">{{ confirmRequest?.message }}</p>
      <div class="dialog-actions">
        <button
          class="button"
          :disabled="pending.has('confirm')"
          @click="confirmRequest = undefined"
        >
          取消</button
        ><button
          class="button danger-solid"
          :disabled="pending.has('confirm')"
          @click="acceptConfirm"
        >
          {{ pending.has("confirm") ? "处理中…" : "确认删除" }}
        </button>
      </div></Modal
    >
  </div>
</template>
