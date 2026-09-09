<script setup lang="ts">
import { useConsoleContext } from "../composables/useConsole";
import Icon from "./Icon.vue";
defineProps<{ url: string; compact?: boolean }>();
const { copy, localAddress } = useConsoleContext();
</script>
<template>
  <div class="subscription">
    <div class="copy-field">
      <input
        :value="url"
        readonly
        placeholder="等待获取局域网地址"
        aria-label="M3U 订阅地址"
        @focus="($event.target as HTMLInputElement).select()"
      /><button class="button" :disabled="!url" @click="copy(url)">
        <Icon name="copy" :size="16" />复制地址
      </button>
    </div>
    <p v-if="!compact" class="help-text">
      <Icon name="info" :size="15" />{{
        localAddress
          ? "暂未获取到局域网地址，请连接 Wi-Fi 或有线网络后刷新状态。"
          : "在 APTV、VLC 或其他电视播放器中添加此 M3U 订阅地址。"
      }}
    </p>
  </div>
</template>
