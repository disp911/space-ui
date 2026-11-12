<script setup lang="ts">
defineProps<{ large?: boolean }>()
</script>

<template>
  <div :class="['logo', { 'logo--large': large }]">
    <svg viewBox="0 0 32 32" aria-hidden="true">
      <rect width="32" height="32" rx="9" fill="var(--acc)" />
      <ellipse cx="16" cy="16" rx="11" ry="4.6" transform="rotate(-32 16 16)" fill="none" stroke="var(--acc-ink)" stroke-width="1.7" opacity=".9" />
      <circle cx="16" cy="16" r="3.2" fill="var(--acc-ink)" />
      <circle cx="25.2" cy="10.6" r="1.9" fill="var(--acc-ink)" />
    </svg>
    <span class="logo__name">Space <span class="logo__panel">panel</span></span>
  </div>
</template>

<style scoped>
.logo {
  display: flex;
  align-items: center;
  gap: 10px;
  min-width: 0;
}

.logo svg {
  display: block;
  flex: none;
  width: 26px;
  height: 26px;
}

.logo--large {
  gap: 12px;
}

.logo--large svg {
  width: 32px;
  height: 32px;
}

.logo--large .logo__name {
  font-size: var(--fs-16);
}

@media (max-width: 767px) {
  .logo--large svg {
    width: 28px;
    height: 28px;
  }

  .logo--large .logo__name {
    font-size: var(--fs-14);
  }
}

.logo__name {
  font: 600 var(--fs-14) / 1 var(--font-sans);
  letter-spacing: -.3px;
  white-space: nowrap;
}

.logo__panel {
  color: var(--dim);
  font-weight: 400;
}
</style>
