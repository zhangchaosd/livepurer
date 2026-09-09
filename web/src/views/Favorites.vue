<script setup lang="ts">
import { watch, ref } from "vue";
import { useConsoleContext } from "../composables/useConsole";
import { platformName } from "../lib/types";
import Icon from "../components/Icon.vue";
import Modal from "../components/Modal.vue";
import EmptyState from "../components/EmptyState.vue";
import PlaylistLink from "../components/PlaylistLink.vue";
const {
  favorites,
  currentList,
  favoritesReady,
  favoritesError,
  detailError,
  favoriteSearch,
  newListTitle,
  createListOpen,
  visibleFavorites,
  loadFavorites,
  selectList,
  createList,
  saveList,
  deleteList,
  deleteFavorite,
  pending,
  watchRoom,
  navigate,
  playlistURL,
} = useConsoleContext();
const editOpen = ref(false),
  editTitle = ref(""),
  editOrder = ref(1);
function edit() {
  if (!currentList.value) return;
  editTitle.value = currentList.value.title;
  editOrder.value = currentList.value.order;
  editOpen.value = true;
}
watch(
  favorites,
  () => {
    if (
      !currentList.value &&
      favorites.value[0] &&
      !pending.value.has("favorite-detail")
    )
      void selectList(favorites.value[0].id);
  },
  { immediate: true },
);
async function submitEdit() {
  if (await saveList(editTitle.value, editOrder.value)) editOpen.value = false;
}
</script>
<template>
  <div class="content-toolbar">
    <p>
      <strong>{{ favorites.length }}</strong> 个收藏夹，收纳你的喜欢
    </p>
    <button
      class="button primary"
      :disabled="!favoritesReady"
      @click="createListOpen = true"
    >
      <Icon name="plus" :size="17" />新建收藏夹
    </button>
  </div>
  <div v-if="favoritesError" class="inline-alert error" role="alert">
    收藏夹加载失败：{{ favoritesError
    }}<button class="text-button" @click="loadFavorites">重试</button>
  </div>
  <div v-else-if="!favoritesReady" class="panel loading-state" role="status">
    <span class="spinner"></span>正在加载收藏夹…
  </div>
  <section v-else-if="!favorites.length" class="panel">
    <EmptyState
      icon="heart"
      title="把喜欢的直播间留在这里"
      description="先创建收藏夹，再到直播页查询房间并添加收藏。"
      ><button class="button primary" @click="createListOpen = true">
        <Icon name="plus" :size="17" />创建第一个收藏夹
      </button></EmptyState
    >
  </section>
  <div v-else class="favorites-layout">
    <aside class="panel folder-panel">
      <p class="nav-caption">全部收藏夹</p>
      <button
        v-for="list in favorites"
        :key="list.id"
        class="folder-item"
        :class="{ selected: currentList?.id === list.id }"
        @click="selectList(list.id)"
      >
        <Icon name="folder" :size="19" /><span>{{ list.title }}</span
        ><Icon name="chevron" :size="15" />
      </button>
    </aside>
    <section class="panel favorite-detail">
      <div
        v-if="pending.has('favorite-detail')"
        class="loading-state"
        role="status"
      >
        <span class="spinner"></span>加载直播间…
      </div>
      <div v-else-if="detailError" class="inline-alert error" role="alert">
        {{ detailError }}<span>请重新选择收藏夹。</span>
      </div>
      <template v-else-if="currentList"
        ><header class="section-heading">
          <div>
            <h2>
              {{ currentList.title }}
              <span class="count-badge">{{
                currentList.favorites?.length || 0
              }}</span>
            </h2>
            <p>收藏夹也可以作为独立的电视订阅</p>
          </div>
          <div class="button-group">
            <button class="icon-button" aria-label="编辑收藏夹" @click="edit">
              <Icon name="edit" :size="18" /></button
            ><button
              class="icon-button danger"
              aria-label="删除收藏夹"
              @click="deleteList"
            >
              <Icon name="trash" :size="18" />
            </button>
          </div>
        </header>
        <PlaylistLink :url="playlistURL(currentList.id)" compact /><label
          class="search-field"
          ><Icon name="search" :size="17" /><input
            v-model="favoriteSearch"
            aria-label="搜索收藏的直播间"
            placeholder="搜索主播、平台或房间号"
        /></label>
        <div v-if="visibleFavorites.length" class="favorite-grid">
          <article
            v-for="favorite in visibleFavorites"
            :key="favorite.id"
            class="favorite-card"
          >
            <div class="favorite-card-top">
              <span class="platform-label">{{
                platformName(favorite.plat)
              }}</span
              ><button
                class="icon-button danger"
                :aria-label="`移除 ${favorite.upper}`"
                @click="deleteFavorite(favorite.id)"
              >
                <Icon name="trash" :size="16" />
              </button>
            </div>
            <span class="room-avatar large">{{
              favorite.upper.slice(0, 1)
            }}</span>
            <h3>{{ favorite.upper }}</h3>
            <p>房间号 {{ favorite.room }}</p>
            <button class="button" @click="watchRoom(favorite)">
              <Icon name="play" :size="17" />立即播放
            </button>
          </article>
        </div>
        <EmptyState
          v-else
          :icon="favoriteSearch ? 'search' : 'heart'"
          :title="
            favoriteSearch ? '没有找到匹配的直播间' : '收藏夹里还没有直播间'
          "
          :description="
            favoriteSearch
              ? '试试其他主播名称或房间号。'
              : '到直播页查询一个房间，点击「添加收藏」即可收纳。'
          "
          ><button
            v-if="!favoriteSearch"
            class="button"
            @click="navigate('live')"
          >
            去发现直播<Icon
              name="arrow"
              :size="16"
            /></button></EmptyState></template
      ><EmptyState
        v-else
        title="选择一个收藏夹"
        description="从左侧选择，查看你的直播间。"
      />
    </section>
  </div>
  <Modal
    :open="createListOpen"
    title="新建收藏夹"
    @close="!pending.has('create-list') && (createListOpen = false)"
    ><form @submit.prevent="createList">
      <label class="field"
        >收藏夹名称<input
          v-model="newListTitle"
          autofocus
          required
          minlength="2"
          maxlength="40"
          placeholder="例如：每天都想看的直播"
      /></label>
      <p class="help-text">2–40 个字符，之后可以随时修改。</p>
      <div class="dialog-actions">
        <button
          type="button"
          class="button"
          :disabled="pending.has('create-list')"
          @click="createListOpen = false"
        >
          取消</button
        ><button class="button primary" :disabled="pending.has('create-list')">
          {{ pending.has("create-list") ? "创建中…" : "创建收藏夹" }}
        </button>
      </div>
    </form></Modal
  >
  <Modal
    :open="editOpen"
    title="编辑收藏夹"
    @close="!pending.has('edit-list') && (editOpen = false)"
    ><form @submit.prevent="submitEdit">
      <label class="field"
        >名称<input
          v-model="editTitle"
          required
          minlength="2"
          maxlength="40" /></label
      ><label class="field"
        >排序<input
          v-model.number="editOrder"
          type="number"
          min="1"
          max="100"
          required
      /></label>
      <p class="help-text">数字越小，排序越靠前。</p>
      <div class="dialog-actions">
        <button type="button" class="button" @click="editOpen = false">
          取消</button
        ><button class="button primary" :disabled="pending.has('edit-list')">
          保存修改
        </button>
      </div>
    </form></Modal
  >
</template>
