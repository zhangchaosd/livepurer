<script setup lang="ts">
import { useConsoleContext } from "../composables/useConsole";
import { platforms } from "../lib/types";
import Icon from "./Icon.vue";
defineProps<{ compact?: boolean }>();
const { room, looking, lookupError, watchRoom, resolveRoom, resetLookup } =
  useConsoleContext();
</script>
<template>
  <form class="room-form" @submit.prevent="watchRoom()">
    <div
      v-if="!compact"
      class="platform-selector"
      role="group"
      aria-label="选择直播平台"
    >
      <button
        v-for="platform in platforms"
        :key="platform.id"
        type="button"
        :aria-pressed="room.plat === platform.id"
        :class="{ selected: room.plat === platform.id }"
        @click="
          room.plat = platform.id;
          resetLookup();
        "
      >
        <span class="platform-mark" :style="{ '--platform': platform.color }">{{
          platform.short
        }}</span
        >{{ platform.name
        }}<Icon v-if="room.plat === platform.id" name="check" :size="16" />
      </button>
    </div>
    <div class="room-input-row">
      <label v-if="compact" class="platform-field"
        ><span class="sr-only">直播平台</span
        ><select v-model="room.plat" @change="resetLookup">
          <option
            v-for="platform in platforms"
            :key="platform.id"
            :value="platform.id"
          >
            {{ platform.name }}
          </option>
        </select></label
      >
      <label class="room-field"
        ><span class="sr-only">房间号</span><Icon name="search" /><input
          v-model="room.room"
          aria-label="房间号"
          :aria-invalid="!!lookupError"
          aria-describedby="room-error"
          placeholder="输入直播间房间号"
          autocomplete="off"
          @input="resetLookup"
        /><kbd>↵</kbd></label
      >
      <button
        class="button primary"
        type="submit"
        :disabled="!room.room.trim()"
      >
        <Icon name="play" :size="17" />立即播放
      </button>
      <button
        v-if="!compact"
        class="button"
        type="button"
        :disabled="looking || !room.room.trim()"
        @click="resolveRoom"
      >
        <Icon name="search" :size="17" />{{ looking ? "查询中…" : "查询信息" }}
      </button>
    </div>
    <p v-if="lookupError" id="room-error" class="field-error" role="alert">
      {{ lookupError }}
    </p>
  </form>
</template>
