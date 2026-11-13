<script setup lang="ts">
import { computed } from 'vue'

/** Error output of a program, line by line in mono on a red tint; scrolls when long. */
const props = defineProps<{ text: string }>()
const lines = computed(() => props.text.split('\n').filter(Boolean))
</script>

<template>
  <div class="x-error-box">
    <div v-for="(line, i) in lines" :key="i">{{ line }}</div>
  </div>
</template>

<style scoped>
.x-error-box {
  display: flex;
  flex-direction: column;
  gap: 2px;
  max-height: 200px;
  padding: 8px 12px;
  overflow: auto;
  border: 1px solid var(--err-line);
  border-radius: var(--nest-r);
  background: var(--err-bg);
  color: var(--err-t);
  font: 400 var(--fs-11) / var(--lh-code) var(--font-mono);
  word-break: break-word;
}
</style>
