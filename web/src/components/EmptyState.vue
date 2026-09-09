<script setup lang="ts">
import Icon from "./Icon.vue";
defineProps<{ icon?: string; title: string; description: string }>();
</script>
<template>
  <div class="empty-state">
    <span class="empty-icon"><Icon :name="icon || 'folder'" :size="28" /></span>
    <h3>{{ title }}</h3>
    <p>{{ description }}</p>
    <slot />
  </div>
</template>
