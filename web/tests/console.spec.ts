import { test, expect, type Page } from "@playwright/test";
import type { Channel, FavoriteList, Server } from "../src/lib/types";
async function mockAPI(page: Page) {
  const state = {
    channels: [
      { plat: "bilibili", room: "6", name: "测试频道", logo: "" },
    ] as Channel[],
    lists: [
      {
        id: 1,
        title: "常看直播",
        order: 1,
        favorites: [
          { id: 1, plat: "bilibili", room: "6", upper: "测试主播", order: 1 },
        ],
      },
    ] as FavoriteList[],
    server: {
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
    } as Server,
    saves: 0,
    failSave: false,
    failList: false,
  };
  await page.route("**/api/**", async (route) => {
    const request = route.request(),
      url = new URL(request.url()),
      path = url.pathname.replace("/api/v1", "");
    let data: unknown = null;
    if (path === "/settings/channels") {
      if (request.method() === "PUT") {
        if (state.failSave)
          return route.fulfill({
            json: { code: 1, msg: "测试：保存失败", data: null },
          });
        state.channels = request.postDataJSON();
        state.saves++;
      }
      data = state.channels;
    } else if (path === "/settings/server") {
      if (request.method() === "PUT") state.server = request.postDataJSON();
      data = state.server;
    } else if (path === "/settings/account")
      data = {
        bilibili: {
          enable: false,
          DedeUserID: "",
          DedeUserIDCkMd5: "",
          SESSDATA: "",
          BiliJCT: "",
        },
        huya: { enable: false, cookies: "" },
        douyu: { enable: false },
      };
    else if (path === "/os/all")
      data = {
        lan_ip: "192.168.1.42",
        info: { os: "linux", kernel_arch: "amd64" },
        sys_cpu: { percent: 12.4 },
        sys_mem: { total: 100, avl: 75 },
        self_mem: { mem_str: "32 MB" },
      };
    else if (path === "/fav/list/get_all")
      data = state.lists.map(({ favorites, ...list }) => list);
    else if (path === "/fav/list/get")
      data = state.lists.find(
        (l) => l.id === Number(url.searchParams.get("id")),
      );
    else if (path === "/fav/list/add") {
      const list = {
        ...request.postDataJSON(),
        id: state.lists.length + 1,
        favorites: [],
      };
      state.lists.push(list);
      data = list;
    } else if (path === "/fav/list/edit") {
      if (state.failList)
        return route.fulfill({
          json: { code: 1, msg: "测试：收藏夹更新失败" },
        });
      const values = request.postDataJSON();
      Object.assign(
        state.lists.find((l) => l.id === values.id)!,
        values,
      );
    } else if (path === "/fav/list/del")
      state.lists = state.lists.filter(
        (l) => l.id !== request.postDataJSON().id,
      );
    else if (path === "/fav/del")
      state.lists.forEach(
        (l) =>
          (l.favorites = l.favorites?.filter(
            (f) => f.id !== request.postDataJSON().id,
          )),
      );
    else if (path === "/fav/add") {
      const value = request.postDataJSON();
      state.lists
        .find((l) => l.id === value.fid)
        ?.favorites?.push({ ...value, id: 10 });
    } else if (path === "/live/room_info")
      data = {
        upper: `主播 ${url.searchParams.get("room")}`,
        title: "测试直播间",
        status: true,
      };
    else if (path === "/live/play")
      return route.fulfill({ status: 503, body: "Offline fixture" });
    else if (path === "/live/m3u")
      return route.fulfill({
        contentType: "audio/x-mpegurl",
        body: "#EXTM3U\n",
      });
    else throw new Error(`Unexpected request: ${request.method()} ${path}`);
    return route.fulfill({ json: { code: 0, msg: "success", data } });
  });
  return state;
}
async function navigate(page: Page, name: string) {
  await page
    .getByRole("navigation", { name: "主导航" })
    .getByRole("link", { name: new RegExp(`^${name}`) })
    .click();
}

test("all pages remain usable without horizontal overflow", async ({
  page,
}) => {
  const errors: string[] = [];
  page.on("pageerror", (e) => errors.push(e.message));
  await mockAPI(page);
  await page.goto("/");
  for (const name of ["总览", "直播播放", "我的收藏", "IPTV 频道", "设置"]) {
    await navigate(page, name);
    await expect(page.locator("#page-title")).toHaveText(name);
    expect(
      await page.evaluate(
        () => document.documentElement.scrollWidth <= window.innerWidth,
      ),
    ).toBe(true);
  }
  expect(errors).toEqual([]);
});

test("channel validation, failed save, retry, ordering and rollback preserve draft", async ({
  page,
}) => {
  const state = await mockAPI(page);
  await page.goto("/#iptv");
  await page.getByRole("button", { name: "添加频道", exact: true }).click();
  await page.getByRole("button", { name: "保存频道", exact: true }).click();
  await expect(page.getByRole("alert")).toContainText(
    "第 2 个频道还没有填写房间号",
  );
  await page.getByLabel("第 2 个频道房间号", { exact: true }).fill("6");
  await page.getByRole("button", { name: "保存频道", exact: true }).click();
  await expect(page.getByRole("alert")).toContainText("重复");
  await page.getByLabel("第 2 个频道房间号", { exact: true }).fill(" 99 ");
  await page.getByLabel("上移第 2 个频道", { exact: true }).click();
  await navigate(page, "总览");
  await navigate(page, "IPTV 频道");
  await expect(
    page.getByLabel("第 1 个频道房间号", { exact: true }),
  ).toHaveValue(" 99 ");
  state.failSave = true;
  await page.getByRole("button", { name: "保存频道", exact: true }).click();
  await expect(
    page.getByRole("alert").filter({ hasText: "测试：保存失败" }),
  ).toBeVisible();
  expect(state.saves).toBe(0);
  await expect(page.getByRole("button", { name: "撤销修改" })).toBeEnabled();
  state.failSave = false;
  await page.getByRole("button", { name: "关闭提示" }).click();
  await page.getByRole("button", { name: "保存频道", exact: true }).click();
  await expect(
    page.getByRole("button", { name: "保存频道", exact: true }),
  ).toBeDisabled();
  expect(state.channels.map((c) => c.room)).toEqual(["99", "6"]);
  expect(state.channels.every((c) => !("key" in c))).toBe(true);
  await page.getByLabel("移除第 1 个频道", { exact: true }).click();
  await page.getByRole("button", { name: "撤销修改" }).click();
  await expect(
    page.getByLabel("第 1 个频道房间号", { exact: true }),
  ).toHaveValue("99");
});

test("favorites load from direct link, search, create and cancel destructive action", async ({
  page,
}) => {
  const state = await mockAPI(page);
  await page.goto("/#favorites");
  await expect(
    page.getByRole("heading", { name: "测试主播", exact: true }),
  ).toBeVisible();
  await page.getByLabel("搜索收藏的直播间").fill("不存在");
  await expect(page.getByText("没有找到匹配的直播间")).toBeVisible();
  await page.getByLabel("搜索收藏的直播间").fill("");
  await page.getByRole("button", { name: "新建收藏夹", exact: true }).click();
  const dialog = page.getByRole("dialog", { name: "新建收藏夹" });
  await dialog.getByLabel("收藏夹名称").fill("游戏收藏");
  await dialog.getByRole("button", { name: "创建收藏夹", exact: true }).click();
  await expect(dialog).not.toBeVisible();
  await expect(page.getByRole("heading", { name: "游戏收藏" })).toBeVisible();
  await page.getByRole("button", { name: "删除收藏夹", exact: true }).click();
  await page
    .getByRole("dialog", { name: "删除收藏夹" })
    .getByRole("button", { name: "取消", exact: true })
    .click();
  expect(state.lists).toHaveLength(2);
  await page.getByRole("button", { name: "编辑收藏夹" }).click();
  const edit = page.getByRole("dialog", { name: "编辑收藏夹" });
  await edit.getByLabel("名称", { exact: true }).fill("修改后的名称");
  state.failList = true;
  await edit.getByRole("button", { name: "保存修改" }).click();
  await expect(page.getByRole("alert")).toContainText("测试：收藏夹更新失败");
  await expect(edit).toBeVisible();
  await expect(edit.getByLabel("名称", { exact: true })).toHaveValue(
    "修改后的名称",
  );
});

test("dialog keyboard focus and escape restore the trigger", async ({
  page,
}) => {
  await mockAPI(page);
  await page.goto("/#favorites");
  const trigger = page.getByRole("button", { name: "新建收藏夹", exact: true });
  await trigger.click();
  await expect(page.getByRole("dialog", { name: "新建收藏夹" })).toBeVisible();
  await page.keyboard.press("Escape");
  await expect(
    page.getByRole("dialog", { name: "新建收藏夹" }),
  ).not.toBeVisible();
  await expect(trigger).toBeFocused();
});

test("settings drafts survive navigation and refresh, then save", async ({
  page,
}) => {
  const state = await mockAPI(page);
  await page.goto("/#settings");
  await page.getByLabel("监听端口").fill("9900");
  await page.getByRole("switch", { name: "SOCKS5 代理", exact: true }).check();
  await expect(page.getByLabel("代理地址", { exact: true })).toBeVisible();
  await navigate(page, "总览");
  await page.getByRole("button", { name: "刷新服务状态" }).click();
  await navigate(page, "设置");
  await expect(page.getByLabel("监听端口")).toHaveValue("9900");
  await page.getByRole("button", { name: "保存服务设置" }).click();
  await expect(
    page.getByRole("button", { name: "保存服务设置" }),
  ).toBeDisabled();
  expect(state.server.port).toBe(9900);
  await expect(page.getByText("设置已保存。请重启二进制程序")).toBeVisible();
});

test("lookup and favorite picker work, external stream failure can recover", async ({
  page,
}) => {
  await mockAPI(page);
  await page.goto("/#live");
  await page.getByLabel("房间号", { exact: true }).fill("123");
  await page.getByRole("button", { name: "查询信息" }).click();
  await expect(page.getByRole("heading", { name: "主播 123" })).toBeVisible();
  await page.getByRole("button", { name: "添加收藏", exact: true }).click();
  const picker = page.getByRole("dialog", { name: "添加到收藏夹" });
  await picker.getByLabel("常看直播").check();
  await picker.getByRole("button", { name: "添加到 1 个收藏夹" }).click();
  await expect(picker).not.toBeVisible();
  await page.getByRole("button", { name: "立即播放", exact: true }).click();
  await expect(
    page.getByRole("heading", { name: "暂时无法连接直播" }),
  ).toBeVisible();
  await expect(page.getByLabel("稳定播放地址")).toHaveValue(/room=123/);
  await navigate(page, "总览");
  await expect(page.getByText("主播 123", { exact: true })).toBeVisible();
});

test("unavailable service shows retry and never claims healthy status", async ({
  page,
}) => {
  await page.route("**/api/**", (route) =>
    route.fulfill({ status: 503, body: "offline" }),
  );
  await page.goto("/");
  await expect(page.getByText("服务连接失败", { exact: true })).toBeVisible();
  await navigate(page, "IPTV 频道");
  await expect(page.getByRole("alert")).toContainText("频道加载失败");
  await expect(
    page.getByRole("button", { name: "添加频道", exact: true }),
  ).toBeDisabled();
  await expect(
    page.getByRole("button", { name: "重试", exact: true }),
  ).toBeEnabled();
});

test("unsupported metrics remain distinct from zero and service failure", async ({
  page,
}) => {
  await mockAPI(page);
  await page.route("**/api/v1/os/all", (route) =>
    route.fulfill({
      json: {
        code: 0,
        data: {
          info: { os: "darwin", kernel_arch: "arm64" },
          unavailable: ["sys_cpu", "sys_mem"],
        },
      },
    }),
  );
  await page.goto("/");
  await expect(page.getByText("服务已连接", { exact: true })).toBeVisible();
  await expect(
    page.getByText("部分系统指标暂不可用，以 — 显示。"),
  ).toBeVisible();
  await expect(page.locator(".resource strong")).toHaveText(["—", "—"]);
});

test("LAN addresses are shared across subscriptions and playback", async ({
  page,
}) => {
  await mockAPI(page);
  await page.goto("/");
  const origin = new URL(page.url());
  origin.hostname = "192.168.1.42";
  await expect(page.getByLabel("M3U 订阅地址")).toHaveValue(
    `${origin.origin}/api/v1/live/m3u`,
  );
  await navigate(page, "我的收藏");
  await expect(page.getByLabel("M3U 订阅地址")).toHaveValue(
    `${origin.origin}/api/v1/live/m3u?fav_list_id=1`,
  );
  await navigate(page, "直播播放");
  await page.getByLabel("房间号", { exact: true }).fill("123");
  await page.getByRole("button", { name: "立即播放", exact: true }).click();
  await expect(page.getByLabel("稳定播放地址")).toHaveValue(
    `${origin.origin}/api/v1/live/play?plat=bilibili&room=123`,
  );
  await navigate(page, "IPTV 频道");
  await expect(page.getByLabel("M3U 订阅地址")).toHaveValue(
    `${origin.origin}/api/v1/live/m3u`,
  );
  await expect(page.getByRole("link", { name: "下载 M3U" })).toHaveAttribute(
    "href",
    "/api/v1/live/m3u",
  );
});

test("missing LAN address does not expose a loopback subscription", async ({
  page,
}) => {
  await mockAPI(page);
  await page.route("**/api/v1/os/all", (route) =>
    route.fulfill({
      json: { code: 0, data: { lan_ip: "", info: { os: "linux" } } },
    }),
  );
  await page.goto("/");
  await expect(page.getByText("服务已连接", { exact: true })).toBeVisible();
  await expect(page.getByLabel("M3U 订阅地址")).toHaveValue("");
  await expect(
    page.getByRole("button", { name: "复制地址", exact: true }),
  ).toBeDisabled();
  await expect(
    page.getByText(
      "暂未获取到局域网地址，请连接 Wi-Fi 或有线网络后刷新状态。",
      { exact: true },
    ),
  ).toBeVisible();
});
