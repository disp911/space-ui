<script setup lang="ts">
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import { formatBytes, formatDuration, formatNumber } from '../../i18n/format'
import { XBadge, XButton, XErrorBox, XIcon, XPopover, XSkeletonText, XStatusDot, type Tone } from '../../kit'
import type { ServiceStatus } from '../../stores/server'
import DashCard from './DashCard.vue'
import DashStat from './DashStat.vue'

/** Xray or the Telegram proxy: state with its version, figures, restart and stop. The error popover links to the panel log. */
const props = defineProps<{
  title: string
  /** Undefined until the first status arrives. */
  service?: ServiceStatus
  /** The first status is on its way: grey bars stand in for the state and the figures. */
  loading?: boolean
  errorTitle: string
  busy?: 'restart' | 'stop' | null
  /** Xray versions are shown as v26.4.25, the Telegram proxy's without the v. */
  versionPrefix?: string
}>()
defineEmits<{ restart: []; stop: []; log: [] }>()
const { t } = useI18n()

const state = computed<{ tone: Tone; label: string }>(() => {
  switch (props.service?.state) {
    case 'running': return { tone: 'ok', label: t('dashboard.running') }
    case 'stop': return { tone: 'warn', label: t('dashboard.stopped') }
    case 'error': return { tone: 'err', label: t('dashboard.error') }
    default: return { tone: 'off', label: t('dashboard.unknown') }
  }
})
const running = computed(() => props.service?.state === 'running')
const errorLines = computed(() => (props.service?.errorMsg ?? '').split('\n').filter(Boolean))
const dash = '—'
</script>

<template>
  <DashCard :title="title">
    <div class="service">
      <div class="service__state">
        <div v-if="loading" class="service__status"><XSkeletonText :width="96" /></div>
        <XPopover v-else :disabled="!errorLines.length" :label="errorTitle">
          <template #default="{ props: trigger }">
          <component
            :is="errorLines.length ? 'button' : 'div'"
            v-bind="errorLines.length ? trigger : {}"
            :type="errorLines.length ? 'button' : undefined"
            :class="['service__status', `service__status--${state.tone}`, { 'is-clickable': errorLines.length }]"
            :title="errorLines.length ? t('dashboard.showError') : undefined"
          >
            <XStatusDot :tone="state.tone" blink />
            <span>{{ state.label }}</span>
            <XIcon v-if="errorLines.length" name="caret" :size="14" :stroke="1.6" />
          </component>
          </template>
          <template #content="{ close }">
            <div class="service__error-head">
              <span>{{ errorTitle }}</span>
              <XButton
                variant="ghost"
                icon="doc"
                :label="t('dashboard.panelLog')"
                @click="close(); $emit('log')"
              />
            </div>
            <XErrorBox :text="service?.errorMsg ?? ''" />
          </template>
        </XPopover>
        <XBadge v-if="service?.version" mono>{{ versionPrefix }}{{ service.version }}</XBadge>
      </div>

      <div class="service__stats">
        <DashStat :label="t('dashboard.online')" :value="running ? formatNumber(service!.online) : dash" :loading="loading" />
        <DashStat :label="t('dashboard.uptime')" :value="running ? formatDuration(service!.uptime) : dash" :loading="loading" />
        <DashStat :label="t('dashboard.memory')" :value="running ? formatBytes(service!.mem) : dash" :loading="loading" />
      </div>

      <div class="service__actions">
        <XButton icon="refresh" :label="t('dashboard.restart')" :loading="busy === 'restart'" :disabled="!service || !!busy" @click="$emit('restart')" />
        <XButton
          variant="danger"
          icon="stop"
          :label="t('dashboard.stop')"
          :loading="busy === 'stop'"
          :disabled="!running || !!busy"
          @click="$emit('stop')"
        />
      </div>
    </div>

    <div v-if="$slots.default" class="service__links"><slot /></div>
  </DashCard>
</template>

<style scoped>
.service {
  display: grid;
  grid-template-areas:
    'state state'
    'stats actions';
  grid-template-columns: minmax(0, 1fr) auto;
  align-items: end;
  gap: 12px;
}

.service__state {
  grid-area: state;
  display: flex;
  align-items: center;
  gap: 10px;
  min-width: 0;
}

/* Each figure as wide as it is, one after another from the left with equal gaps. */
.service__stats {
  grid-area: stats;
  display: grid;
  grid-template-columns: repeat(3, auto);
  justify-content: start;
  gap: 12px 35px;
}

.service__actions {
  grid-area: actions;
  display: flex;
  gap: 8px;
}

.service__status {
  display: inline-flex;
  align-items: center;
  gap: 10px;
  padding: 0;
  border: 0;
  border-radius: var(--btn-r);
  background: none;
  color: var(--tone);
  font: 600 var(--fs-18) / var(--lh-tight) var(--font-sans);
  letter-spacing: -.4px;
  outline: none;
}

.service__status.is-clickable {
  cursor: pointer;
}

.service__status:focus-visible {
  box-shadow: 0 0 0 2px var(--acc-line);
}

.service__status--ok { --tone: var(--ok); }
.service__status--warn { --tone: var(--warn); }
.service__status--err { --tone: var(--err); }
.service__status--off { --tone: var(--dim); }

.service__error-head {
  display: flex;
  align-items: center;
  gap: 4px;
  min-height: 28px;
  margin-bottom: 8px;
  color: var(--fg);
  font: 600 var(--fs-13) / var(--lh-tight) var(--font-sans);
}

.service__error-head > span {
  flex: 1;
}

.service__links {
  display: flex;
  flex-direction: column;
  gap: 2px;
  margin-top: 16px;
  padding-top: 10px;
  border-top: 1px solid var(--div);
}

@media (max-width: 767px) {
  .service {
    grid-template-areas:
      'state actions'
      'stats stats';
    align-items: center;
  }

  .service__stats {
    column-gap: 25px;
  }
}
</style>
