import {
  computed,
  inject,
  nextTick,
  onBeforeUnmount,
  onMounted,
  ref,
  type InjectionKey,
} from "vue";
import { api, json } from "../lib/api";
import {
  channelPayload,
  playbackURL,
  roomKey,
  validateChannels,
  platforms,
  type Account,
  type Channel,
  type FavoriteList,
  type Page,
  type Room,
  type RoomInfo,
  type Server,
  type System,
} from "../lib/types";
import { usePlayer } from "./usePlayer";
let channelSequence = 0;
const channelID = () => `channel-${++channelSequence}`;
const clone = <T>(value: T): T => JSON.parse(JSON.stringify(value));
const serialize = (value: unknown) => JSON.stringify(value);
export function useConsole() {
  const pageIDs: Page[] = [
    "dashboard",
    "live",
    "favorites",
    "iptv",
    "settings",
  ];
  const fromHash = (): Page =>
    pageIDs.find((p) => location.hash === `#${p}`) || "dashboard";
  const page = ref<Page>(fromHash());
  const notice = ref<{ message: string; kind: "success" | "error" | "info" }>();
  let noticeTimer: ReturnType<typeof setTimeout>;
  function report(
    message: string,
    kind: "success" | "error" | "info" = "success",
  ) {
    clearTimeout(noticeTimer);
    notice.value = { message, kind };
    if (kind !== "error")
      noticeTimer = setTimeout(() => (notice.value = undefined), 5000);
  }
  const pending = ref(new Set<string>());
  async function perform(key: string, action: () => Promise<void>) {
    if (pending.value.has(key)) return;
    pending.value.add(key);
    try {
      await action();
      return true;
    } catch (error) {
      report(
        error instanceof Error ? error.message : "操作失败，请重试。",
        "error",
      );
      return false;
    } finally {
      pending.value.delete(key);
    }
  }
  const player = usePlayer(report);
  const room = ref<Room>({ plat: "bilibili", room: "" }),
    roomInfo = ref<RoomInfo>();
  const lookupError = ref(""),
    looking = ref(false);
  let lookupSequence = 0,
    lookupAbort: AbortController | undefined;
  const recent = ref<(Room & { upper?: string })[]>([]);
  try {
    const value = JSON.parse(localStorage.getItem("livepurer.recent") || "[]");
    if (Array.isArray(value))
      recent.value = value
        .filter(
          (r) =>
            platforms.some((p) => p.id === r.plat) &&
            typeof r.room === "string",
        )
        .slice(0, 8);
  } catch {
    /* Private browsing can disable storage. */
  }
  function remember(value: Room, upper?: string) {
    recent.value = [
      { ...value, upper },
      ...recent.value.filter((r) => roomKey(r) !== roomKey(value)),
    ].slice(0, 8);
    try {
      localStorage.setItem("livepurer.recent", serialize(recent.value));
    } catch {
      /* History is optional. */
    }
  }
  function resetLookup() {
    lookupSequence++;
    lookupAbort?.abort();
    looking.value = false;
    roomInfo.value = undefined;
    lookupError.value = "";
  }
  function navigate(next: Page) {
    if (next === page.value) return;
    player.stop();
    resetLookup();
    pickerOpen.value = false;
    page.value = next;
    if (location.hash !== `#${next}`) location.hash = next;
    nextTick(() => {
      document.querySelector<HTMLElement>("#page-title")?.focus();
      window.scrollTo(0, 0);
    });
  }
  async function resolveRoom() {
    if (!room.value.room.trim()) {
      lookupError.value = "请输入直播间房间号。";
      return;
    }
    lookupAbort?.abort();
    lookupAbort = new AbortController();
    const controller = lookupAbort,
      seq = ++lookupSequence;
    const target = { ...room.value, room: room.value.room.trim() };
    room.value = target;
    looking.value = true;
    lookupError.value = "";
    roomInfo.value = undefined;
    try {
      const result = await api<RoomInfo>(
        `/live/room_info?${new URLSearchParams(target)}`,
        {
          signal: AbortSignal.any([
            controller.signal,
            AbortSignal.timeout(20000),
          ]),
        },
      );
      if (seq !== lookupSequence || roomKey(target) !== roomKey(room.value))
        return;
      roomInfo.value = result;
      remember(target, result.upper);
    } catch (error) {
      if (seq === lookupSequence && !controller.signal.aborted)
        lookupError.value =
          error instanceof Error ? error.message : "查询失败，请重试。";
    } finally {
      if (seq === lookupSequence) looking.value = false;
    }
  }
  async function watchRoom(target: Room = room.value) {
    if (!target.room.trim()) {
      lookupError.value = "请输入直播间房间号。";
      return;
    }
    navigate("live");
    room.value = { plat: target.plat, room: target.room.trim() };
    await nextTick();
    player.play(room.value);
    remember(room.value);
    void resolveRoom();
  }
  const channels = ref<Channel[]>([]),
    savedChannels = ref<Channel[]>([]);
  const channelReady = ref(false),
    channelsError = ref("");
  const channelDirty = computed(
    () =>
      serialize(channelPayload(channels.value)) !==
      serialize(channelPayload(savedChannels.value)),
  );
  const channelError = ref("");
  async function loadChannels() {
    channelsError.value = "";
    try {
      const result = (await api<Channel[]>("/settings/channels")) || [];
      savedChannels.value = clone(result);
      channels.value = result.map((c) => ({ ...c, key: channelID() }));
      channelReady.value = true;
    } catch (error) {
      channelsError.value = (error as Error).message;
    }
  }
  function addChannel(target?: Room, name = "") {
    if (!channelReady.value) return;
    if (target && channels.value.some((c) => roomKey(c) === roomKey(target))) {
      report("这个直播间已经在 IPTV 频道中。", "info");
      return;
    }
    channels.value.push({
      plat: target?.plat || "bilibili",
      room: target?.room.trim() || "",
      name,
      logo: "",
      key: channelID(),
    });
    channelError.value = "";
    if (target) report("已加入 IPTV 草稿，保存频道后即可在电视观看。", "info");
  }
  function moveChannel(index: number, offset: number) {
    const destination = index + offset;
    if (destination < 0 || destination >= channels.value.length) return;
    const [item] = channels.value.splice(index, 1);
    channels.value.splice(destination, 0, item);
  }
  function saveChannels() {
    channelError.value = validateChannels(channels.value);
    if (channelError.value) return;
    return perform("channels", async () => {
      const payload = channelPayload(channels.value);
      await api("/settings/channels", json("PUT", payload));
      savedChannels.value = clone(payload);
      report("频道已保存，电视订阅已更新。");
    });
  }
  function discardChannels() {
    channels.value = savedChannels.value.map((c) => ({
      ...clone(c),
      key: channelID(),
    }));
    channelError.value = "";
  }
  const favorites = ref<FavoriteList[]>([]),
    currentList = ref<FavoriteList>();
  const favoritesReady = ref(false),
    favoritesError = ref(""),
    detailError = ref("");
  const favoriteSearch = ref(""),
    newListTitle = ref(""),
    createListOpen = ref(false);
  let favoriteSequence = 0;
  async function loadFavorites() {
    favoritesError.value = "";
    try {
      favorites.value = (await api<FavoriteList[]>("/fav/list/get_all")) || [];
      favoritesReady.value = true;
    } catch (error) {
      favoritesError.value = (error as Error).message;
    }
  }
  async function selectList(id: number) {
    const seq = ++favoriteSequence;
    currentList.value = undefined;
    detailError.value = "";
    pending.value.add("favorite-detail");
    try {
      const result = await api<FavoriteList>(`/fav/list/get?id=${id}`);
      if (seq === favoriteSequence) currentList.value = result;
    } catch (error) {
      if (seq === favoriteSequence)
        detailError.value = (error as Error).message;
    } finally {
      if (seq === favoriteSequence) pending.value.delete("favorite-detail");
    }
  }
  const visibleFavorites = computed(
    () =>
      currentList.value?.favorites?.filter((f) =>
        `${f.upper} ${f.room} ${f.plat}`
          .toLowerCase()
          .includes(favoriteSearch.value.toLowerCase()),
      ) || [],
  );
  function createList() {
    const title = newListTitle.value.trim();
    if (title.length < 2 || title.length > 40) {
      report("收藏夹名称需要 2–40 个字符。", "error");
      return;
    }
    return perform("create-list", async () => {
      const result = await api<FavoriteList>(
        "/fav/list/add",
        json("POST", {
          title,
          order: Math.min(favorites.value.length + 1, 100),
        }),
      );
      newListTitle.value = "";
      createListOpen.value = false;
      await loadFavorites();
      await selectList(result.id);
      report("收藏夹已创建。");
    });
  }
  function saveList(title: string, order: number) {
    if (!currentList.value) return;
    if (
      title.trim().length < 2 ||
      title.trim().length > 40 ||
      !Number.isInteger(order) ||
      order < 1 ||
      order > 100
    ) {
      report("名称需要 2–40 个字符，排序需要是 1–100 的整数。", "error");
      return;
    }
    const id = currentList.value.id;
    return perform("edit-list", async () => {
      await api(
        "/fav/list/edit",
        json("POST", { id, title: title.trim(), order }),
      );
      await loadFavorites();
      await selectList(id);
      report("收藏夹已更新。");
    });
  }
  const confirmRequest = ref<{
    title: string;
    message: string;
    action: () => Promise<void>;
  }>();
  function confirm(
    title: string,
    message: string,
    action: () => Promise<void>,
  ) {
    confirmRequest.value = { title, message, action };
  }
  function acceptConfirm() {
    if (!confirmRequest.value) return;
    const request = confirmRequest.value;
    return perform("confirm", async () => {
      await request.action();
      confirmRequest.value = undefined;
    });
  }
  function deleteList() {
    const list = currentList.value;
    if (!list) return;
    confirm(
      "删除收藏夹",
      `将删除「${list.title}」及其中的全部收藏，操作无法撤销。`,
      async () => {
        await api("/fav/list/del", json("POST", { id: list.id }));
        currentList.value = undefined;
        await loadFavorites();
        if (favorites.value[0]) await selectList(favorites.value[0].id);
        report("收藏夹已删除。");
      },
    );
  }
  function deleteFavorite(id: number) {
    const list = currentList.value;
    if (!list) return;
    confirm(
      "移除直播间",
      "将此直播间从当前收藏夹移除，其他收藏夹不受影响。",
      async () => {
        await api("/fav/del", json("POST", { id }));
        await selectList(list.id);
        report("直播间已移除。");
      },
    );
  }
  const pickerOpen = ref(false),
    selectedIDs = ref<number[]>([]);
  function openPicker() {
    selectedIDs.value = [];
    pickerOpen.value = true;
  }
  function addFavorites() {
    if (!selectedIDs.value.length || !roomInfo.value) return;
    const target = { ...room.value },
      upper = roomInfo.value.upper || target.room;
    return perform("add-favorites", async () => {
      // Sequential writes make a retry safe when a later folder fails.
      for (const id of selectedIDs.value) {
        const list = await api<FavoriteList>(`/fav/list/get?id=${id}`);
        if (list.favorites?.some((f) => roomKey(f) === roomKey(target)))
          continue;
        await api(
          "/fav/add",
          json("POST", {
            fid: id,
            order: Math.min((list.favorites?.length || 0) + 1, 100),
            ...target,
            upper,
          }),
        );
      }
      pickerOpen.value = false;
      if (currentList.value) await selectList(currentList.value.id);
      report("直播间已加入所选收藏夹。");
    });
  }
  const server = ref<Server>({
    port: 8800,
    path: "./data",
    debug: false,
    socks5: {
      enable: false,
      host: "127.0.0.1",
      port: 1080,
      user: "",
      password: "",
    },
  });
  const account = ref<Account>({
    bilibili: {
      enable: false,
      DedeUserID: "",
      DedeUserIDCkMd5: "",
      SESSDATA: "",
      BiliJCT: "",
    },
    huya: { enable: false, cookies: "" },
    douyu: { enable: false },
  });
  const serverSnapshot = ref(""),
    accountSnapshot = ref(""),
    settingsError = ref(""),
    serverReady = ref(false),
    accountReady = ref(false),
    restartRequired = ref(false);
  const serverDirty = computed(
    () => serverReady.value && serialize(server.value) !== serverSnapshot.value,
  );
  const accountDirty = computed(
    () =>
      accountReady.value && serialize(account.value) !== accountSnapshot.value,
  );
  async function loadSettings() {
    settingsError.value = "";
    const result = await Promise.allSettled([
      serverReady.value
        ? Promise.resolve()
        : api<Server>("/settings/server").then((r) => {
            server.value = r;
            serverSnapshot.value = serialize(r);
            serverReady.value = true;
          }),
      accountReady.value
        ? Promise.resolve()
        : api<Account>("/settings/account").then((r) => {
            account.value = r;
            accountSnapshot.value = serialize(r);
            accountReady.value = true;
          }),
    ]);
    if (result.some((r) => r.status === "rejected"))
      settingsError.value = "部分设置加载失败，请重试。";
  }
  function saveServer() {
    return perform("server", async () => {
      const payload = clone(server.value);
      await api("/settings/server", json("PUT", payload));
      serverSnapshot.value = serialize(payload);
      restartRequired.value = true;
      report("服务设置已保存，重启程序后生效。");
    });
  }
  function saveAccount() {
    return perform("account", async () => {
      const payload = clone(account.value);
      await api("/settings/account", json("PUT", payload));
      accountSnapshot.value = serialize(payload);
      restartRequired.value = true;
      report("账号设置已保存，重启程序后生效。");
    });
  }
  const system = ref<System>({}),
    systemState = ref<"loading" | "online" | "offline">("loading"),
    updated = ref("");
  async function refreshSystem() {
    if (pending.value.has("system")) return;
    pending.value.add("system");
    try {
      system.value = await api<System>("/os/all");
      systemState.value = "online";
      updated.value = new Date().toLocaleTimeString("zh-CN", { hour12: false });
    } catch {
      systemState.value = "offline";
    } finally {
      pending.value.delete("system");
    }
  }
  const memoryPercent = computed(() =>
    system.value.sys_mem?.total
      ? Math.max(
          0,
          Math.min(
            100,
            (1 - system.value.sys_mem.avl / system.value.sys_mem.total) * 100,
          ),
        )
      : 0,
  );
  const shareOrigin = computed(() => {
    const url = new URL(location.origin);
    if (system.value.lan_ip) url.hostname = system.value.lan_ip;
    else if (["localhost", "127.0.0.1", "[::1]"].includes(url.hostname))
      return "";
    return url.origin;
  });
  const playlistURL = (id?: number) =>
    shareOrigin.value
      ? `${shareOrigin.value}/api/v1/live/m3u${id ? `?fav_list_id=${id}` : ""}`
      : "";
  const sharePlaybackURL = (room: Room) =>
    shareOrigin.value ? playbackURL(room, shareOrigin.value) : "";
  async function copy(value: string) {
    try {
      if (navigator.clipboard && window.isSecureContext)
        await navigator.clipboard.writeText(value);
      else {
        const focused = document.activeElement as HTMLElement;
        const input = document.createElement("textarea");
        input.value = value;
        input.style.cssText = "position:fixed;opacity:0;top:0";
        document.body.appendChild(input);
        let copied = false;
        try {
          input.select();
          copied = document.execCommand("copy");
        } finally {
          input.remove();
          focused?.focus();
        }
        if (!copied) throw new Error();
      }
      report("地址已复制。");
    } catch {
      report("复制失败，请选中页面上的地址手动复制。", "error");
    }
  }
  const localAddress = computed(() => !shareOrigin.value);
  const anyDirty = computed(
    () => channelDirty.value || serverDirty.value || accountDirty.value,
  );
  function beforeUnload(event: BeforeUnloadEvent) {
    if (anyDirty.value) {
      event.preventDefault();
      event.returnValue = "";
    }
  }
  function hashChange() {
    navigate(fromHash());
  }
  let statusTimer: ReturnType<typeof setInterval>;
  onMounted(() => {
    void loadChannels();
    void loadFavorites();
    void loadSettings();
    void refreshSystem();
    statusTimer = setInterval(() => {
      if (document.visibilityState === "visible") void refreshSystem();
    }, 15000);
    window.addEventListener("beforeunload", beforeUnload);
    window.addEventListener("hashchange", hashChange);
  });
  onBeforeUnmount(() => {
    clearInterval(statusTimer);
    clearTimeout(noticeTimer);
    lookupAbort?.abort();
    window.removeEventListener("beforeunload", beforeUnload);
    window.removeEventListener("hashchange", hashChange);
  });
  return {
    page,
    navigate,
    notice,
    report,
    pending,
    perform,
    player,
    room,
    roomInfo,
    lookupError,
    looking,
    recent,
    resetLookup,
    resolveRoom,
    watchRoom,
    channels,
    savedChannels,
    channelReady,
    channelsError,
    channelDirty,
    channelError,
    loadChannels,
    addChannel,
    moveChannel,
    saveChannels,
    discardChannels,
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
    confirmRequest,
    confirm,
    acceptConfirm,
    pickerOpen,
    selectedIDs,
    openPicker,
    addFavorites,
    server,
    account,
    serverReady,
    accountReady,
    serverDirty,
    accountDirty,
    settingsError,
    restartRequired,
    loadSettings,
    saveServer,
    saveAccount,
    system,
    systemState,
    updated,
    refreshSystem,
    memoryPercent,
    playlistURL,
    sharePlaybackURL,
    copy,
    localAddress,
  };
}
export type Console = ReturnType<typeof useConsole>;
export const consoleKey: InjectionKey<Console> = Symbol("console");
export function useConsoleContext() {
  const context = inject(consoleKey);
  if (!context) throw new Error("Console context is missing");
  return context;
}
