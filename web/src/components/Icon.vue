<script setup lang="ts">
defineProps<{ name: string; size?: number }>();
const paths: Record<string, string> = {
  grid: "M3 3h7v7H3z M14 3h7v7h-7z M3 14h7v7H3z M14 14h7v7h-7z",
  play: "m9 5 11 7-11 7V5Z",
  monitor: "M3 4h18v13H3z M8 21h8 M12 17v4",
  heart:
    "M20.8 4.6a5.5 5.5 0 0 0-7.8 0L12 5.7l-1.1-1.1a5.5 5.5 0 0 0-7.8 7.8L12 21l8.8-8.6a5.5 5.5 0 0 0 0-7.8Z",
  tv: "M3 7h18v14H3z M8 2l4 5 4-5",
  settings:
    "M12 8a4 4 0 1 0 0 8 4 4 0 0 0 0-8 M12 2v3 M12 19v3 M2 12h3 M19 12h3 M5 5l2 2 M17 17l2 2 M5 19l2-2 M17 7l2-2",
  arrow: "M4 12h16 M14 6l6 6-6 6",
  chevron: "m9 5 7 7-7 7",
  plus: "M12 5v14 M5 12h14",
  search: "M10.5 3a7.5 7.5 0 1 0 0 15 7.5 7.5 0 0 0 0-15 M16 16l5 5",
  refresh: "M20 7A9 9 0 0 0 4 5 M4 1v4h4 M4 17a9 9 0 0 0 16 2 M20 23v-4h-4",
  copy: "M8 8h13v13H8z M16 8V3H3v13h5",
  check: "m5 12 4 4L19 6",
  close: "m6 6 12 12 M6 18 18 6",
  trash: "M3 6h18 M9 6V3h6v3 M5 6l1 15h12l1-15 M10 10v7 M14 10v7",
  folder: "M3 5h6l2 3h10v12H3V5Z",
  clock: "M12 3a9 9 0 1 0 0 18 9 9 0 0 0 0-18 M12 7v5l3 2",
  stop: "M6 6h12v12H6z",
  external: "M14 3h7v7 M21 3 10 14 M10 3H3v18h18v-7",
  download: "M12 3v12 M7 10l5 5 5-5 M4 16v5h16v-5",
  save: "M4 3h13l4 4v14H3V3h1 M7 3v6h10V3 M7 21v-8h10v8",
  up: "m6 15 6-6 6 6",
  down: "m6 9 6 6 6-6",
  info: "M12 3a9 9 0 1 0 0 18 9 9 0 0 0 0-18 M12 11v6 M12 7v1",
  warning: "M12 3 2 21h20L12 3Z M12 9v5 M12 17v1",
  cpu: "M6 6h12v12H6z M9 9h6v6H9z M9 2v4 M15 2v4 M9 18v4 M15 18v4 M2 9h4 M2 15h4 M18 9h4 M18 15h4",
  memory:
    "M3 6h18v12H3z M7 10v4 M12 10v4 M17 10v4 M6 18v3 M10 18v3 M14 18v3 M18 18v3",
  server: "M3 3h18v7H3z M3 14h18v7H3z M7 6v1 M7 17v1",
  edit: "m15 3 6 6-12 12H3v-6L15 3Z M12 6l6 6",
  link: "m10 13 4-4 M8 16l-2 2a4 4 0 0 1-6-6l5-5a4 4 0 0 1 6 0 M16 8l2-2a4 4 0 0 1 6 6l-5 5a4 4 0 0 1-6 0",
};
</script>
<template>
  <svg
    :width="size || 20"
    :height="size || 20"
    viewBox="0 0 24 24"
    fill="none"
    stroke="currentColor"
    stroke-width="1.65"
    stroke-linecap="round"
    stroke-linejoin="round"
    aria-hidden="true"
  >
    <path :d="paths[name] || paths.monitor" />
  </svg>
</template>
