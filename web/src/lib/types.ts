export type Platform = "bilibili" | "douyu" | "huya" | "inke";
export type Page = "dashboard" | "live" | "favorites" | "iptv" | "settings";
export type Room = { plat: Platform; room: string };
export type RoomInfo = {
  cover?: string;
  avatar?: string;
  upper: string;
  title: string;
  status: boolean;
};
export type Channel = Room & { name: string; logo: string; key?: string };
export type Favorite = Room & { id: number; upper: string; order: number };
export type FavoriteList = {
  id: number;
  title: string;
  order: number;
  favorites?: Favorite[];
};
export type Server = {
  port: number;
  path: string;
  debug: boolean;
  socks5: {
    enable: boolean;
    host: string;
    port: number;
    user: string;
    password: string;
  };
};
export type Account = {
  bilibili: {
    enable: boolean;
    DedeUserID: string;
    DedeUserIDCkMd5: string;
    SESSDATA: string;
    BiliJCT: string;
  };
  huya: { enable: boolean; cookies: string };
  douyu: { enable: boolean };
};
export type System = {
  lan_ip?: string;
  unavailable?: string[];
  info?: { os: string; kernel_arch: string; uptime: number };
  sys_cpu?: { percent: number };
  sys_mem?: { total: number; avl: number; total_str: string };
  self_mem?: { mem_str: string };
};
export const platforms: {
  id: Platform;
  name: string;
  short: string;
  color: string;
}[] = [
  { id: "bilibili", name: "哔哩哔哩", short: "哔", color: "#dd6d9b" },
  { id: "douyu", name: "斗鱼", short: "斗", color: "#dd843a" },
  { id: "huya", name: "虎牙", short: "虎", color: "#be902f" },
  { id: "inke", name: "映客", short: "映", color: "#429c9f" },
];
export const platformName = (plat: string) =>
  platforms.find((p) => p.id === plat)?.name || plat;
export const roomKey = (room: Room) => `${room.plat}:${room.room.trim()}`;
export const playbackURL = (room: Room, origin = location.origin) =>
  `${origin}/api/v1/live/play?${new URLSearchParams({ plat: room.plat, room: room.room.trim() })}`;
export const channelPayload = (channels: Channel[]) =>
  channels.map(({ plat, room, name, logo }) => ({
    plat,
    room: room.trim(),
    name: name.trim(),
    logo: logo.trim(),
  }));
export function validateChannels(channels: Channel[]): string {
  if (channels.length > 100) return "最多可以添加 100 个频道。";
  const seen = new Set<string>();
  for (const [i, channel] of channels.entries()) {
    if (!channel.room.trim()) return `第 ${i + 1} 个频道还没有填写房间号。`;
    if (!platforms.some((p) => p.id === channel.plat))
      return `第 ${i + 1} 个频道的平台无效。`;
    if (seen.has(roomKey(channel)))
      return `第 ${i + 1} 个频道与已有频道重复，请检查平台和房间号。`;
    seen.add(roomKey(channel));
    if (channel.logo.trim()) {
      try {
        if (!["http:", "https:"].includes(new URL(channel.logo).protocol))
          throw new Error();
      } catch {
        return `第 ${i + 1} 个频道的图标地址需要以 http:// 或 https:// 开头。`;
      }
    }
  }
  return "";
}
