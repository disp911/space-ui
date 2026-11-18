<script setup lang="ts">
import { ref, useId } from 'vue'
import { useFloating, type Placement } from './floating'

/**
 * Panel of content that opens from a trigger and closes on a click outside
 * or Escape. The trigger binds the slot props:
 *   <XPopover><template #default="{ props }"><button v-bind="props">…</button></template><template #content>…</template></XPopover>
 * It opens under the trigger from its left edge; bottom-end lines it up with
 * the right edge, e.g. a settings button at the end of a header.
 */
const props = withDefaults(defineProps<{ disabled?: boolean; label?: string; placement?: Placement }>(), { placement: 'bottom-start' })

const id = useId()
const anchor = ref<HTMLElement | null>(null)
const panel = ref<HTMLElement | null>(null)
const open = ref(false)
const { style } = useFloating(anchor, panel, open, { placement: props.placement, onDismiss: close })

function close() {
  open.value = false
}

function toggle(e: MouseEvent) {
  if (props.disabled) return
  anchor.value = e.currentTarget as HTMLElement
  open.value = !open.value
}
</script>

<template>
  <slot :open="open" :props="{ 'aria-expanded': disabled ? undefined : open, 'aria-controls': `${id}-panel`, onClick: toggle }" />
  <Teleport to="body">
    <Transition name="x-popup">
      <div v-if="open" :id="`${id}-panel`" ref="panel" role="dialog" :aria-label="label" class="x-popover" :style="style">
        <slot name="content" :close="close" />
      </div>
    </Transition>
  </Teleport>
</template>

<style scoped>
.x-popover {
  z-index: 1000;
  max-width: min(420px, calc(100vw - 16px));
  padding: 10px 12px 12px 14px;
  overflow: auto;
  border: 1px solid var(--line);
  border-radius: var(--pop-r);
  background: var(--dlg);
  box-shadow: var(--shadow-pop);
}

/* As a menu's: the scrollbar clear of the rounded corners. */
.x-popover::-webkit-scrollbar-track {
  margin-block: calc(var(--pop-r) - 1px);
}
</style>
