<script setup lang="ts">
/**
 * Settings sections while they load: cards with a title bar and rows of a
 * name, a description and a field, sized like XSettingsSection and XSettingRow.
 * rows gives the number of rows in each card.
 */
withDefaults(defineProps<{ rows?: number[] }>(), { rows: () => [4, 2] })

const titles = [150, 110]
const names = [46, 34, 52, 40]
const notes = [72, 60, 80, 66]
</script>

<template>
  <div class="x-settings-skeleton" aria-hidden="true">
    <div v-for="(n, i) in rows" :key="i" class="x-settings-skeleton__card">
      <div class="x-settings-skeleton__head">
        <span class="x-skeleton x-settings-skeleton__chevron" />
        <span class="x-skeleton x-settings-skeleton__bar" :style="{ width: `${titles[i % titles.length]}px`, height: '12px' }" />
      </div>
      <div v-for="j in n" :key="j" class="x-settings-skeleton__row">
        <div class="x-settings-skeleton__text">
          <span class="x-skeleton x-settings-skeleton__bar" :style="{ width: `${names[(j - 1) % names.length]}%`, height: '11px' }" />
          <span class="x-skeleton x-settings-skeleton__bar x-settings-skeleton__bar--late" :style="{ width: `${notes[(j - 1) % notes.length]}%`, height: '9px' }" />
        </div>
        <span class="x-skeleton x-settings-skeleton__field" />
      </div>
    </div>
  </div>
</template>

<style scoped>
.x-settings-skeleton {
  display: flex;
  flex-direction: column;
  gap: 12px;
}

.x-settings-skeleton__card {
  overflow: hidden;
  border: 1px solid var(--line);
  border-radius: var(--card-r);
  background: var(--card);
}

.x-settings-skeleton__head {
  display: flex;
  align-items: center;
  gap: 10px;
  height: 45px;
  padding: 0 20px;
}

.x-settings-skeleton__chevron {
  width: 14px;
  height: 14px;
  border-radius: var(--r-4);
  background: var(--skel);
}

.x-settings-skeleton__row {
  display: grid;
  grid-template-columns: minmax(0, 1fr) 400px;
  align-items: center;
  gap: 12px 40px;
  padding: 16px 20px;
  border-top: 1px solid var(--div);
}

.x-settings-skeleton__text {
  display: flex;
  flex-direction: column;
  gap: 8px;
}

.x-settings-skeleton__bar {
  display: block;
  border-radius: var(--r-4);
  background: var(--skel);
}

.x-settings-skeleton__bar--late {
  animation-delay: .2s;
}

.x-settings-skeleton__field {
  height: var(--ctl-h);
  border-radius: var(--ctl-r);
  background: var(--skel);
}

@media (max-width: 767px) {
  .x-settings-skeleton__head,
  .x-settings-skeleton__row {
    padding-inline: 16px;
  }

  .x-settings-skeleton__row {
    grid-template-columns: minmax(0, 1fr);
  }
}
</style>
