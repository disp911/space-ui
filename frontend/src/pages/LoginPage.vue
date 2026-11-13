<script setup lang="ts">
import { computed, onBeforeUnmount, onMounted, ref, useId, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { useRoute, useRouter } from 'vue-router'
import { api, ApiError } from '../api'
import { formatMilliseconds } from '../i18n/format'
import { XButton, XInput, XNotice, XStatusDot } from '../kit'
import AppLogo from '../layout/AppLogo.vue'
import LanguageMenu from '../layout/LanguageMenu.vue'
import LoginBackdrop from '../layout/LoginBackdrop.vue'
import ThemeToggle from '../layout/ThemeToggle.vue'
import { useSession } from '../stores/session'

const { t } = useI18n()
const route = useRoute()
const router = useRouter()
const session = useSession()
const id = useId()
const host = location.hostname

const username = ref('')
const password = ref('')
const code = ref('')
const twoFactor = ref(false)

type Problem = 'username' | 'password' | 'code' | 'credentials' | 'connection'
const problem = ref<Problem | null>(null)
const phase = ref<'idle' | 'checking' | 'success'>('idle')

const message = computed(() => {
  switch (problem.value) {
    case 'username': return t('auth.enterUsername')
    case 'password': return t('auth.enterPassword')
    case 'code': return t('auth.enterCode')
    case 'credentials': return t('auth.wrongCredentials')
    case 'connection': return t('auth.noConnection')
    default: return ''
  }
})

// Editing a field takes the error back.
watch([username, password, code], () => (problem.value = null))

const usernameInput = ref<InstanceType<typeof XInput> | null>(null)
const passwordInput = ref<InstanceType<typeof XInput> | null>(null)
const codeInput = ref<InstanceType<typeof XInput> | null>(null)

async function submit() {
  if (phase.value !== 'idle') return
  const empty = !username.value ? 'username' : !password.value ? 'password' : twoFactor.value && !code.value ? 'code' : null
  if (empty) {
    problem.value = empty
    ;({ username: usernameInput, password: passwordInput, code: codeInput })[empty].value?.focus()
    return
  }

  phase.value = 'checking'
  try {
    await session.login(username.value, password.value, code.value)
  } catch (e) {
    phase.value = 'idle'
    // A refusal comes as a normal answer; anything else means the server was not reached.
    problem.value = e instanceof ApiError && e.status === 200 ? 'credentials' : 'connection'
    passwordInput.value?.focus()
    return
  }
  phase.value = 'success'
  const next = typeof route.query.next === 'string' && route.query.next.startsWith('/') ? route.query.next : '/'
  setTimeout(() => router.replace(next), 600)
}

// The 2FA question doubles as the ping: the server answers it without a session.
const ping = ref<number | null>(null)
const pingFailed = ref(false)
let timer = 0

async function check() {
  const started = performance.now()
  try {
    const enabled = await api.post<boolean>('getTwoFactorEnable')
    ping.value = performance.now() - started
    pingFailed.value = false
    twoFactor.value = enabled === true
  } catch (e) {
    // An empty answer means the setting could not be read, not that the server is gone.
    pingFailed.value = !(e instanceof SyntaxError)
    if (!pingFailed.value) ping.value = performance.now() - started
  }
}

const pingTone = computed(() => (pingFailed.value ? 'err' : ping.value == null || ping.value < 150 ? 'ok' : ping.value < 400 ? 'warn' : 'err'))

onMounted(() => {
  // On phones the keyboard would cover the form before the user asked for it.
  if (matchMedia('(min-width: 768px)').matches) usernameInput.value?.focus()
  check()
  timer = window.setInterval(check, 10_000)
})
onBeforeUnmount(() => clearInterval(timer))
</script>

<template>
  <div class="login">
    <LoginBackdrop />

    <header class="login__top">
      <AppLogo large />
      <div class="login__tools">
        <LanguageMenu />
        <ThemeToggle />
      </div>
    </header>

    <main class="login__main">
      <form class="login__card x-ctl-lg" novalidate @submit.prevent="submit">
        <h1 class="login__title">{{ t('auth.title') }}</h1>
        <i18n-t scope="global" keypath="common.server" tag="p" class="login__server">
          <template #host><span class="login__host">{{ host }}</span></template>
        </i18n-t>

        <div class="login__fields">
          <div class="login__field">
            <label class="x-caps x-caps--lg" :for="`${id}-user`">{{ t('auth.username') }}</label>
            <XInput
              :id="`${id}-user`"
              ref="usernameInput"
              v-model="username"
              icon="user"
              name="username"
              autocomplete="username"
              autocapitalize="none"
              spellcheck="false"
              :invalid="problem === 'username' || problem === 'credentials'"
              :aria-describedby="message ? `${id}-error` : undefined"
            />
          </div>
          <div class="login__field">
            <label class="x-caps x-caps--lg" :for="`${id}-pass`">{{ t('auth.password') }}</label>
            <XInput
              :id="`${id}-pass`"
              ref="passwordInput"
              v-model="password"
              icon="lock"
              type="password"
              name="password"
              autocomplete="current-password"
              mono
              :invalid="problem === 'password' || problem === 'credentials'"
              :aria-describedby="message ? `${id}-error` : undefined"
            />
          </div>
          <div v-if="twoFactor" class="login__field">
            <label class="x-caps x-caps--lg" :for="`${id}-code`">{{ t('auth.code') }}</label>
            <XInput
              :id="`${id}-code`"
              ref="codeInput"
              v-model="code"
              name="code"
              :placeholder="t('auth.code')"
              autocomplete="one-time-code"
              inputmode="numeric"
              maxlength="6"
              mono
              :invalid="problem === 'code' || problem === 'credentials'"
              :aria-describedby="message ? `${id}-error` : undefined"
            />
          </div>
        </div>

        <Transition name="login-error">
          <XNotice v-if="message" :id="`${id}-error`" tone="err" class="login__error">
            {{ message }}
            <template #actions>
              <XButton variant="ghost-danger" icon="close" :label="t('kit.close')" @click="problem = null" />
            </template>
          </XNotice>
        </Transition>

        <XButton
          type="submit"
          variant="primary"
          block
          :loading="phase === 'checking'"
          :success="phase === 'success'"
          class="login__submit"
        >
          {{ phase === 'checking' ? t('auth.checking') : phase === 'success' ? t('auth.success') : t('auth.submit') }}
        </XButton>

        <footer class="login__foot" aria-live="polite">
          <XStatusDot :tone="pingTone" size="sm" blink />
          <span v-if="pingFailed">{{ t('auth.noConnection') }}</span>
          <span v-else-if="ping != null">{{ t('auth.ping', { value: formatMilliseconds(ping) }) }}</span>
        </footer>
      </form>
    </main>
  </div>
</template>

<style scoped>
.login {
  position: relative;
  display: flex;
  flex-direction: column;
  min-height: 100vh;
  min-height: 100dvh;
  padding: 32px 40px;
}

.login__top {
  position: relative;
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
}

.login__tools {
  display: flex;
  gap: 8px;
}

.login__main {
  position: relative;
  display: flex;
  flex: 1;
  align-items: center;
  justify-content: center;
  /* The card is centred on the space under the top bar, less 40px at the bottom. */
  padding: 0 0 40px;
}

.login__card {
  display: flex;
  flex-direction: column;
  width: 100%;
  max-width: 396px;
  padding: 32px;
  border: 1px solid var(--line);
  border-radius: var(--card-r);
  background: color-mix(in srgb, var(--card) 86%, transparent);
  box-shadow: var(--shadow-modal);
  backdrop-filter: blur(10px);
}

.login__title {
  margin: 0 0 4px;
  font: 600 var(--fs-20) / var(--lh-tight) var(--font-sans);
  letter-spacing: -.4px;
}

.login__server {
  margin: 0 0 24px;
  color: var(--dim);
  font-size: var(--fs-13);
  line-height: var(--lh-tight);
}

.login__host {
  color: var(--fg2);
  font-family: var(--font-mono);
}

.login__fields {
  display: flex;
  flex-direction: column;
  gap: 14px;
}

.login__field {
  display: flex;
  flex-direction: column;
  gap: 6px;
}

.login__error {
  margin-top: 14px;
}


.login__submit {
  margin-top: 16px;
}

.login__foot {
  display: flex;
  align-items: center;
  gap: 8px;
  /* Keeps its height before the first ping arrives. */
  min-height: 32px;
  margin-top: 20px;
  padding-top: 16px;
  border-top: 1px solid var(--div);
  color: var(--mut);
  font-size: var(--fs-12);
  line-height: var(--lh-tight);
}


.login-error-enter-active,
.login-error-leave-active {
  transition: opacity var(--t-fast), transform var(--t-fast) var(--ease);
}

.login-error-enter-from,
.login-error-leave-to {
  opacity: 0;
  transform: translateY(-4px);
}



@media (max-width: 767px) {
  .login {
    padding: 16px 24px 32px;
  }

  .login__tools {
    gap: 6px;
  }

  .login__main {
    padding: 24px 0 0;
  }

  .login__card {
    padding: 24px 20px;
  }

  .login__title {
    font-size: var(--fs-24);
    letter-spacing: -.7px;
  }
}
</style>
