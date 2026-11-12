<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { TOTP } from 'otpauth'
import { XButton, XCodeInput, XFieldNote, XModal, XQrCode } from '../../kit'

/**
 * Asks for a code from the authenticator app. enable first shows the new
 * token as a QR code to scan; disable and credentials only ask for the code.
 * The code is checked here against the token, as the old panel did. Then
 * apply runs (enabling or disabling saves at once); on success the button
 * says so for a moment and the window closes, and verified is emitted.
 */
const props = defineProps<{
  mode: 'enable' | 'disable' | 'credentials'
  token: string
  apply?: () => Promise<void>
}>()
const emit = defineEmits<{ verified: [] }>()
const open = defineModel<boolean>('open', { default: false })
const { t } = useI18n()

const totp = computed(() => new TOTP({ issuer: 'Space UI', label: 'Administrator', algorithm: 'SHA1', digits: 6, period: 30, secret: props.token }))

const code = ref('')
const wrong = ref(false)
const busy = ref(false)
const done = ref(false)

watch(open, (on) => {
  if (!on) return
  code.value = ''
  wrong.value = false
  busy.value = false
  done.value = false
})
watch(code, () => (wrong.value = false))

const text = computed(() => ({
  enable: { title: t('settings.enableTitle'), done: t('settings.enabled') },
  disable: { title: t('settings.disableTitle'), done: t('settings.disabled') },
  credentials: { title: t('settings.credentialsTitle'), done: '' },
})[props.mode])

async function confirm() {
  if (code.value.length < 6 || busy.value || done.value) return
  if (totp.value.generate() !== code.value) {
    wrong.value = true
    return
  }
  if (props.apply) {
    busy.value = true
    try {
      await props.apply()
    } catch {
      busy.value = false
      return
    } finally {
      busy.value = false
    }
  }
  emit('verified')
  if (!text.value.done) {
    open.value = false
    return
  }
  done.value = true
  setTimeout(() => (open.value = false), 1600)
}
</script>

<template>
  <XModal v-model:open="open" :title="text.title" :size="mode === 'enable' ? 'sm' : 'xs'" cancel :dismissible="!busy">
    <div class="tfa">
      <template v-if="mode === 'enable'">
        <p class="tfa__lead">{{ t('settings.enableSteps') }}</p>
        <hr class="tfa__line" />
        <p class="tfa__step">{{ t('settings.enableScan') }}</p>
        <div class="tfa__token">
          <XQrCode :value="totp.toString()" :copy="token" :copy-label="t('settings.copyToken')" />
          <span class="tfa__secret">{{ token }}</span>
        </div>
        <hr class="tfa__line" />
        <p class="tfa__step">{{ t('settings.enableCode') }}</p>
      </template>
      <p v-else class="tfa__lead">{{ t(mode === 'disable' ? 'settings.disableText' : 'settings.credentialsText') }}</p>
      <div class="tfa__code">
        <XCodeInput v-model="code" :label="t('settings.code')" :invalid="wrong" @submit="confirm" />
        <XFieldNote v-if="wrong" tone="err">{{ t('settings.codeWrong') }}</XFieldNote>
      </div>
    </div>
    <template #footer>
      <XButton
        variant="primary"
        :disabled="code.length < 6 && !done"
        :loading="busy"
        :success="done"
        :title="code.length < 6 ? t('settings.codeShort') : undefined"
        @click="confirm"
      >
        {{ done ? text.done : t('settings.confirm') }}
      </XButton>
    </template>
  </XModal>
</template>

<style scoped>
.tfa {
  display: flex;
  flex-direction: column;
  gap: 14px;
}

.tfa__lead,
.tfa__step {
  margin: 0;
  font-size: var(--fs-13);
  line-height: var(--lh-text);
  text-wrap: pretty;
}

.tfa__lead {
  color: var(--mut);
}

.tfa__step {
  color: var(--fg2);
}

.tfa__line {
  width: 100%;
  height: 1px;
  margin: 0;
  border: 0;
  background: var(--div);
}

.tfa__token {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 12px;
  padding-top: 2px;
}

.tfa__secret {
  color: var(--fg2);
  font: 500 var(--fs-13) / var(--lh-tight) var(--font-mono);
  letter-spacing: .05em;
  text-align: center;
  word-break: break-all;
  user-select: all;
}

.tfa__code {
  display: flex;
  flex-direction: column;
  gap: 8px;
}
</style>
