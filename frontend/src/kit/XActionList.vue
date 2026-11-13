<script setup lang="ts">
/** A framed group of XActionRow rows split by thin lines: backup export and import. */
</script>

<template>
  <div class="x-action-list"><slot /></div>
</template>

<style scoped>
.x-action-list {
  display: flex;
  flex-direction: column;
  overflow: hidden;
  border: 1px solid var(--field-line);
  border-radius: var(--nest-r);
  background: var(--field);
}

.x-action-list > :deep(* + *) {
  border-top: 1px solid var(--div);
}
</style>
