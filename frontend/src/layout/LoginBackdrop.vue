<script setup lang="ts">
// Decorative background of the login screen: a glow and three drifting waves, laid in the page under its content.
const waves = [
  'M0 196 C 240 138 480 254 720 196 C 960 138 1200 254 1440 196 C 1680 138 1920 254 2160 196 C 2400 138 2640 254 2880 196 L2880 340 L0 340 Z',
  'M0 232 C 120 190 360 274 480 232 C 600 190 840 274 960 232 C 1080 190 1320 274 1440 232 C 1560 190 1800 274 1920 232 C 2040 190 2280 274 2400 232 C 2520 190 2760 274 2880 232 L2880 340 L0 340 Z',
  'M0 268 C 90 242 270 294 360 268 C 450 242 630 294 720 268 C 810 242 990 294 1080 268 C 1170 242 1350 294 1440 268 C 1530 242 1710 294 1800 268 C 1890 242 2070 294 2160 268 C 2250 242 2430 294 2520 268 C 2610 242 2790 294 2880 268 L2880 340 L0 340 Z',
]
</script>

<template>
  <div class="backdrop" aria-hidden="true">
    <div class="backdrop__waves">
      <div v-for="(d, i) in waves" :key="i" :class="['backdrop__wave', `backdrop__wave--${i + 1}`]">
        <svg viewBox="0 0 2880 340" preserveAspectRatio="none"><path :d="d" /></svg>
      </div>
    </div>
  </div>
</template>

<style scoped>
/* Colours of the mockup; only this screen uses them. */
.backdrop {
  --glow: rgba(10, 132, 255, .2);
  --wave-1: #0A84FF;
  --wave-2: #64B0FF;
  --wave-3: #0B2645;
  --wave-1-opacity: .3;
  --wave-2-opacity: .2;
  --wave-3-opacity: .42;
  /*
   * Part of the page, not fixed to the screen: iOS Safari draws nothing fixed
   * under its floating bottom bar, so fixed waves stopped short of the screen's
   * edge. As tall as the screen with its bars hidden (100lvh), the waves reach
   * the bottom edge and show through the bar.
   */
  position: absolute;
  top: 0;
  right: 0;
  left: 0;
  height: 100%;
  height: max(100%, 100lvh);
  overflow: hidden;
  pointer-events: none;
  background: radial-gradient(120% 90% at 50% 118%, var(--glow) 0%, transparent 62%);
}

:root[data-theme='light'] .backdrop {
  --glow: rgba(0, 122, 255, .14);
  --wave-3: #007AFF;
  --wave-1-opacity: .34;
  --wave-2-opacity: .22;
  --wave-3-opacity: .3;
}

.backdrop__waves {
  position: absolute;
  right: 0;
  bottom: 0;
  left: 0;
  height: 440px;
  overflow: hidden;
}

.backdrop__wave {
  position: absolute;
  left: 0;
  width: 200%;
  height: 100%;
  animation: backdrop-drift linear infinite;
}

.backdrop__wave svg {
  display: block;
  width: 100%;
  height: 100%;
}

.backdrop__wave--1 {
  bottom: -6px;
  opacity: var(--wave-1-opacity);
  fill: var(--wave-1);
  animation-duration: 26s;
}

.backdrop__wave--2 {
  bottom: -14px;
  opacity: var(--wave-2-opacity);
  fill: var(--wave-2);
  animation-duration: 17s;
  animation-direction: reverse;
}

.backdrop__wave--3 {
  bottom: -20px;
  opacity: var(--wave-3-opacity);
  fill: var(--wave-3);
  animation-duration: 11s;
}

@keyframes backdrop-drift {
  from {
    transform: translate3d(0, 0, 0);
  }
  to {
    transform: translate3d(-50%, 0, 0);
  }
}

@media (max-width: 767px) {
  .backdrop {
    background: radial-gradient(130% 80% at 50% 116%, var(--glow) 0%, transparent 64%);
  }

  .backdrop__waves {
    height: 330px;
  }
}

@media (prefers-reduced-motion: reduce) {
  .backdrop__wave {
    animation: none;
  }
}
</style>
