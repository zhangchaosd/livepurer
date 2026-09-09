import { ref, onBeforeUnmount } from "vue";
import flvjs from "flv.js";
import { playbackURL, type Room } from "../lib/types";
export function usePlayer(
  report: (message: string, kind?: "success" | "error" | "info") => void,
) {
  const video = ref<HTMLVideoElement>();
  const state = ref<
    | "idle"
    | "connecting"
    | "playing"
    | "waiting"
    | "paused"
    | "stopped"
    | "error"
    | "unsupported"
  >("idle");
  const activeRoom = ref<Room>();
  let instance: flvjs.Player | undefined;
  const destroy = () => {
    const old = instance;
    instance = undefined;
    try {
      old?.destroy();
    } catch {
      /* A partially initialized player may already be detached. */
    }
    if (video.value) {
      video.value.pause();
      video.value.removeAttribute("src");
      video.value.load();
    }
  };
  const stop = () => {
    state.value = "stopped";
    destroy();
  };
  const play = (room: Room) => {
    if (!video.value || !room.room.trim()) return;
    destroy();
    activeRoom.value = { ...room, room: room.room.trim() };
    if (!flvjs.isSupported()) {
      state.value = "unsupported";
      return;
    }
    state.value = "connecting";
    try {
      const current = flvjs.createPlayer(
        { type: "flv", url: playbackURL(room), isLive: true },
        {
          enableStashBuffer: true,
          stashInitialSize: 384 * 1024,
          lazyLoad: false,
          autoCleanupSourceBuffer: true,
          autoCleanupMaxBackwardDuration: 30,
          autoCleanupMinBackwardDuration: 10,
        },
      );
      instance = current;
      current.on(flvjs.Events.ERROR, () => {
        if (instance !== current) return;
        state.value = "error";
        destroy();
        report(
          "直播暂时无法播放。请确认房间已开播，或复制地址到外部播放器。",
          "error",
        );
      });
      current.attachMediaElement(video.value);
      current.load();
      video.value.play().catch((error: Error) => {
        if (instance !== current) return;
        if (error.name === "NotAllowedError") {
          state.value = "paused";
          report("浏览器暂停了自动播放，请点击播放器中的播放按钮。", "info");
        }
      });
    } catch {
      state.value = "error";
      destroy();
      report("播放器初始化失败，请重新连接。", "error");
    }
  };
  const mediaEvent = (value: typeof state.value) => {
    if (instance && !["error", "stopped"].includes(state.value))
      state.value = value;
  };
  onBeforeUnmount(destroy);
  return { video, state, activeRoom, play, stop, mediaEvent };
}
