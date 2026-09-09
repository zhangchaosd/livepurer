<script setup lang="ts">
import { ref, watch, nextTick, onBeforeUnmount, useId } from "vue";
import Icon from "./Icon.vue";
const props = defineProps<{ open: boolean; title: string }>();
const emit = defineEmits<{ close: [] }>();
const titleID = useId();
const dialog = ref<HTMLDialogElement>();
let previous: HTMLElement | null = null;
watch(
  () => props.open,
  async (open) => {
    await nextTick();
    if (open) {
      previous = document.activeElement as HTMLElement;
      dialog.value?.showModal();
    } else {
      dialog.value?.close();
      previous?.focus();
    }
  },
  { immediate: true },
);
onBeforeUnmount(() => {
  dialog.value?.close();
  previous?.focus();
});
</script>
<template>
  <dialog
    ref="dialog"
    class="dialog"
    :aria-labelledby="titleID"
    @cancel.prevent="emit('close')"
    @click="$event.target === dialog && emit('close')"
  >
    <div class="dialog-content">
      <header class="section-heading">
        <h2 :id="titleID">{{ title }}</h2>
        <button
          class="icon-button"
          aria-label="关闭对话框"
          @click="emit('close')"
        >
          <Icon name="close" />
        </button>
      </header>
      <slot />
    </div>
  </dialog>
</template>
