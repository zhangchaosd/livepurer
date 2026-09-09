<script setup lang="ts">
import { ref } from "vue";
import { useConsoleContext } from "../composables/useConsole";
import Icon from "../components/Icon.vue";
const {
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
  pending,
} = useConsoleContext();
const section = ref("server");
</script>
<template>
  <div v-if="settingsError" class="inline-alert error" role="alert">
    {{ settingsError
    }}<button class="text-button" @click="loadSettings">重新加载</button>
  </div>
  <div v-if="restartRequired" class="inline-alert" role="status">
    <Icon name="info" /><span
      >设置已保存。请重启二进制程序，使服务与账号设置生效。</span
    >
  </div>
  <div class="settings-layout">
    <nav class="settings-nav" aria-label="设置分类">
      <button
        :class="{ selected: section === 'server' }"
        @click="section = 'server'"
      >
        <Icon name="server" :size="18" />服务设置<span
          v-if="serverDirty"
          class="draft-dot"
        ></span></button
      ><button
        :class="{ selected: section === 'account' }"
        @click="section = 'account'"
      >
        <Icon name="monitor" :size="18" />平台账号<span
          v-if="accountDirty"
          class="draft-dot"
        ></span>
      </button>
      <div class="settings-note">
        <Icon name="info" :size="18" />
        <p>修改后记得保存。服务与账号配置需要重启程序后生效。</p>
      </div>
    </nav>
    <form v-if="section === 'server'" @submit.prevent="saveServer">
      <fieldset :disabled="!serverReady || pending.has('server')">
        <section class="panel settings-section">
          <header class="section-heading">
            <div>
              <h2>基础服务</h2>
              <p>配置本地监听端口与数据存储位置</p>
            </div>
            <Icon name="server" />
          </header>
          <div class="form-grid">
            <label class="field"
              >监听端口<input
                v-model.number="server.port"
                type="number"
                min="1"
                max="65535"
                required
              /><small>范围 1–65535，默认 8800</small></label
            ><label class="field"
              >数据目录<input v-model="server.path" required /><small
                >存储收藏、数据库与内置界面</small
              ></label
            >
          </div>
          <label class="switch-row"
            ><span
              ><strong>调试日志</strong
              ><small>记录更详细的运行信息，便于排查问题</small></span
            ><input
              v-model="server.debug"
              type="checkbox"
              role="switch"
              aria-label="调试日志"
          /></label>
        </section>
        <section class="panel settings-section">
          <label class="switch-row"
            ><span
              ><strong>SOCKS5 代理</strong
              ><small>为直播服务设置网络代理</small></span
            ><input
              v-model="server.socks5.enable"
              type="checkbox"
              role="switch"
              aria-label="SOCKS5 代理"
          /></label>
          <div v-if="server.socks5.enable" class="form-grid proxy-fields">
            <label class="field"
              >代理地址<input
                v-model="server.socks5.host"
                required
                placeholder="127.0.0.1" /></label
            ><label class="field"
              >代理端口<input
                v-model.number="server.socks5.port"
                type="number"
                min="1"
                max="65535"
                required /></label
            ><label class="field"
              >用户名<input
                v-model="server.socks5.user"
                autocomplete="off"
                placeholder="可选" /></label
            ><label class="field"
              >密码<input
                v-model="server.socks5.password"
                type="password"
                autocomplete="new-password"
                placeholder="可选"
            /></label>
          </div>
        </section>
      </fieldset>
      <div class="settings-save">
        <span>{{
          !serverReady
            ? "等待设置加载"
            : serverDirty
              ? "有未保存修改"
              : "所有修改已保存"
        }}</span
        ><button
          class="button primary"
          :disabled="!serverReady || !serverDirty || pending.has('server')"
        >
          <Icon name="save" :size="17" />{{
            pending.has("server") ? "保存中…" : "保存服务设置"
          }}
        </button>
      </div>
    </form>
    <form v-else @submit.prevent="saveAccount">
      <fieldset :disabled="!accountReady || pending.has('account')">
        <section class="panel settings-section">
          <header class="section-heading">
            <div>
              <h2>平台账号</h2>
              <p>凭据字段留空会保留已保存的值</p>
            </div>
            <Icon name="monitor" />
          </header>
          <label class="switch-row"
            ><span
              ><strong>哔哩哔哩</strong
              ><small>使用已登录账号的凭据</small></span
            ><input
              v-model="account.bilibili.enable"
              type="checkbox"
              role="switch"
              aria-label="启用哔哩哔哩账号"
          /></label>
          <div v-if="account.bilibili.enable" class="form-grid account-fields">
            <label class="field"
              >DedeUserID<input
                v-model="account.bilibili.DedeUserID"
                autocomplete="off"
                placeholder="留空则保留原值" /></label
            ><label class="field"
              >DedeUserIDCkMd5<input
                v-model="account.bilibili.DedeUserIDCkMd5"
                type="password"
                autocomplete="new-password"
                placeholder="留空则保留原值" /></label
            ><label class="field"
              >SESSDATA<input
                v-model="account.bilibili.SESSDATA"
                type="password"
                autocomplete="new-password"
                placeholder="留空则保留原值" /></label
            ><label class="field"
              >BiliJCT<input
                v-model="account.bilibili.BiliJCT"
                type="password"
                autocomplete="new-password"
                placeholder="留空则保留原值"
            /></label>
          </div>
          <label class="switch-row"
            ><span
              ><strong>虎牙</strong><small>使用 Cookies 连接平台</small></span
            ><input
              v-model="account.huya.enable"
              type="checkbox"
              role="switch"
              aria-label="启用虎牙账号" /></label
          ><label v-if="account.huya.enable" class="field account-fields"
            >虎牙 Cookies<input
              v-model="account.huya.cookies"
              type="password"
              autocomplete="new-password"
              placeholder="留空则保留原值" /></label
          ><label class="switch-row"
            ><span><strong>斗鱼</strong><small>启用账号功能</small></span
            ><input
              v-model="account.douyu.enable"
              type="checkbox"
              role="switch"
              aria-label="启用斗鱼账号"
          /></label>
        </section>
      </fieldset>
      <div class="settings-save">
        <span>{{
          !accountReady
            ? "等待设置加载"
            : accountDirty
              ? "有未保存修改"
              : "所有修改已保存"
        }}</span
        ><button
          class="button primary"
          :disabled="!accountReady || !accountDirty || pending.has('account')"
        >
          <Icon name="save" :size="17" />{{
            pending.has("account") ? "保存中…" : "保存账号设置"
          }}
        </button>
      </div>
    </form>
  </div>
</template>
