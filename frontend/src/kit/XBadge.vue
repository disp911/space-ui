<script setup lang="ts">
/**
 * Small static label. fill — protocol, outline — transport,
 * accent — security or a highlighted property.
 */
withDefaults(defineProps<{
  kind?: 'fill' | 'outline' | 'accent'
  mono?: boolean
  /** outline only: a coloured frame and text, e.g. an outbound in the connection log. */
  tone?: 'err' | 'upload'
}>(), { kind: 'fill' })
</script>

<template>
  <span :class="['x-badge', `x-badge--${kind}`, { 'x-badge--mono': mono }, kind === 'outline' && tone && `x-badge--${tone}`]"><slot /></span>
</template>

<style scoped>
.x-badge {
  display: inline-flex;
  flex: none;
  align-items: center;
  height: var(--badge-h);
  padding: 0 6px;
  border: 1px solid transparent;
  border-radius: var(--r-6);
  font: 600 var(--fs-11) / 1 var(--font-sans);
  white-space: nowrap;
}

.x-badge--mono {
  font-family: var(--font-mono);
  font-weight: 500;
}

.x-badge--fill {
  border-color: var(--chip-line);
  background: var(--chip);
  color: var(--fg2);
}

.x-badge--outline {
  border-color: var(--field-line);
  color: var(--mut);
}

.x-badge--err {
  border-color: var(--err-line);
  color: var(--err-t);
}

.x-badge--upload {
  border-color: var(--upload-line);
  color: var(--upload-t);
}

.x-badge--accent {
  border-color: var(--acc-line);
  color: var(--acc-t);
}
</style>
