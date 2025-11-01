<script setup>
/**
 * @vue-component
 * @description A header component with a title, a search input, and a DeLorean animation.
 *
 * @vue-prop {String} [search=''] - The current search query.
 *
 * @vue-event {string} update:search - Emitted when the search query is updated. The payload is the new search string.
 */
import { ref, onBeforeUnmount } from 'vue'
import { gsap } from 'gsap'
import '@material/web/iconbutton/icon-button.js'
import '@material/web/textfield/outlined-text-field.js'

const props = defineProps({
  search: {
    type: String,
    default: ''
  }
})

const emit = defineEmits(['update:search'])

const sidebarHeader = ref(null)
const sidebarTitle = ref(null)
const fireTrail = ref(null)
const trailBlocker = ref(null)
const deloreanIcon = ref(null)
const titleTextEl = ref(null)

const flameSegmentCount = 7
const flameSegments = Array.from({ length: flameSegmentCount }, (_, index) => ({
  index,
  upperJitter: ((index % 2 === 0 ? 1 : -1) * 4.5) + (Math.random() - 0.5) * 2,
  lowerJitter: ((index % 2 === 0 ? -1 : 1) * 2.5) + (Math.random() - 0.5) * 1.5,
  upperHueShift: (Math.random() - 0.5) * 20,
  lowerHueShift: (Math.random() - 0.5) * 18
}))

let deloreanTimeline = null
let blockerColor = null
let trailBaseLeft = null

/**
 * @function updateSearch
 * @description Handles updating the search query.
 * Emits an `update:search` event with the new input value.
 * @param {Event} event - The `input` event triggered by the text field.
 */
const updateSearch = (event) => {
  emit('update:search', event.target.value)
}

/**
 * @function startFireTrail
 * @description Starts the DeLorean fire trail animation.
 * Uses the GSAP library to create a complex animation timeline that
 * hides the title, starts the car, and shows the fiery trail.
 * The animation does not start if it is already running.
 */
const startFireTrail = () => {
  const headerEl = sidebarHeader.value
  const titleEl = titleTextEl.value
  const trailEl = fireTrail.value
  const carEl = deloreanIcon.value

  if (!headerEl || !titleEl || !trailEl || !carEl) return
  if (deloreanTimeline && deloreanTimeline.isActive()) return

  const trailBlockerEl = trailBlocker.value

  const headerStyles = getComputedStyle(headerEl)
  blockerColor = headerStyles.backgroundColor || headerStyles.color || 'transparent'
  if (trailBlockerEl) {
    trailBlockerEl.style.backgroundColor = blockerColor
  }

  const burnLine = trailEl.querySelector('.ember-line')
  const sparks = Array.from(trailEl.querySelectorAll('.spark'))
  const upperSegments = Array.from(trailEl.querySelectorAll('.lane-upper .flame-segment'))
  const lowerSegments = Array.from(trailEl.querySelectorAll('.lane-lower .flame-segment'))
  const totalSegments = upperSegments.length || 1

  trailBaseLeft = trailEl.getBoundingClientRect().left

  const containerWidth = headerEl.offsetWidth
  const carWidth = carEl.offsetWidth || 48
  const travelDistance = containerWidth + carWidth * 1.8
  const trailingGap = Math.max(60, carWidth * 1.6)

  const baseSpacing = Math.max(40, (travelDistance - trailingGap) / (totalSegments + 1))
  const clampedSpacing = Math.min(baseSpacing, 96)
  const segmentWidth = Math.min(clampedSpacing * 1.05, clampedSpacing + 18, 110)
  const maxTrailLength = segmentWidth + Math.max(0, totalSegments - 1) * clampedSpacing
  const effectiveTrailWidth = Math.max(maxTrailLength, containerWidth)

  trailEl.style.setProperty('--segment-spacing', `${clampedSpacing.toFixed(2)}px`)
  trailEl.style.setProperty('--segment-width', `${segmentWidth.toFixed(2)}px`)

  const trailProxy = { value: 0 }
  /**
   * @function updateTrail
   * @description Nested callback function for the GSAP timeline. It is executed on every 'tick' of the animation.
   * Calculates the visible width of the fire trail based on the car's position and a proxy value,
   * and applies the corresponding styles to the DOM elements. It also handles the positioning of a "blocker" element
   * to realistically hide parts of the trail.
   */
  const updateTrail = () => {
    const rawWidth = Math.max(0, trailProxy.value - trailingGap)
    const carRect = carEl.getBoundingClientRect()
    const carFront = carRect.left + carRect.width
    const carOffset = Math.max(0, Math.min(carFront - trailBaseLeft, effectiveTrailWidth))
    const revealWidth = Math.min(rawWidth, effectiveTrailWidth)
    const visibleWidth = Math.min(revealWidth, carOffset)

    trailEl.style.setProperty('--active-width', `${visibleWidth}px`)
    gsap.set(trailEl, { width: effectiveTrailWidth, x: 0 })

    if (trailBlockerEl) {
      const blockerWidth = Math.max(0, effectiveTrailWidth - carOffset)
      gsap.set(trailBlockerEl, { left: carOffset, width: blockerWidth, backgroundColor: blockerColor })
    }
  }

  deloreanTimeline = gsap.timeline({
    defaults: { ease: 'power2.out' },
    /**
     * @callback onComplete
     * @description Function executed upon completion of the entire animation timeline.
     * It is responsible for resetting the animation state, hiding the effects, and restoring
     * the visual elements to their original state, ready for a new execution.
     */
    onComplete: () => {
      deloreanTimeline = null
      gsap.set(carEl, { x: 0, rotation: 0 })
      gsap.set(titleEl, { opacity: 1, y: 0 })
      gsap.set(trailEl, { opacity: 0, width: 0 })
      trailEl.classList.remove('is-active')
      trailEl.style.removeProperty('--segment-spacing')
      trailEl.style.removeProperty('--segment-width')
      trailEl.style.removeProperty('--active-width')
      if (trailBlockerEl) {
        gsap.set(trailBlockerEl, { left: 0, width: 0 })
      }
      if (burnLine) gsap.set(burnLine, { opacity: 0, scaleX: 0.3 })
      if (sparks.length > 0) sparks.forEach((spark) => gsap.set(spark, { opacity: 0 }))
      trailBaseLeft = null
    }
  })

  deloreanTimeline.add('launch')
  deloreanTimeline.set(titleEl, { transformOrigin: 'left center' }, 'launch')
  deloreanTimeline.set(carEl, { transformOrigin: 'center center' }, 'launch')
  deloreanTimeline.set(trailEl, { transformOrigin: 'left center', opacity: 0, width: effectiveTrailWidth }, 'launch')

  deloreanTimeline.to(titleEl, { opacity: 0, y: -6, duration: 0.22, ease: 'power1.out' }, 'launch')
  deloreanTimeline.call(() => trailEl.classList.add('is-active'), null, 'launch+=0.02')
  deloreanTimeline.to(trailEl, { opacity: 1, duration: 0.16, ease: 'power1.out' }, 'launch+=0.02')

  deloreanTimeline.to(trailProxy, {
    value: travelDistance,
    duration: 1.18,
    ease: 'power3.in',
    onUpdate: updateTrail,
    onComplete: updateTrail
  }, 'launch+=0.02')

  const launchDuration = 1.18
  deloreanTimeline.to(carEl, { x: travelDistance, rotation: -6.5, duration: launchDuration, ease: 'power3.in' }, 'launch+=0.04')

  if (upperSegments.length > 0) {
    const revealWindow = Math.max(0.4, launchDuration - 0.32)
    const segmentStagger = revealWindow / upperSegments.length

    upperSegments.forEach((segment, index) => {
      const revealAt = 0.06 + index * segmentStagger
      deloreanTimeline.to(segment, {
        opacity: 1,
        scaleY: () => gsap.utils.random(1.1, 1.3),
        scaleX: () => gsap.utils.random(0.9, 1.02),
        filter: 'brightness(1.32)',
        duration: 0.18,
        ease: 'power1.out'
      }, `launch+=${revealAt}`)

      deloreanTimeline.to(segment, {
        scaleY: () => gsap.utils.random(0.96, 1.05),
        scaleX: () => gsap.utils.random(0.9, 1.02),
        filter: 'brightness(1)',
        duration: 0.28,
        ease: 'sine.inOut'
      }, `launch+=${revealAt + 0.16}`)

      deloreanTimeline.to(segment, {
        opacity: () => gsap.utils.random(0.2, 0.35),
        filter: 'brightness(0.58)',
        duration: 0.6,
        ease: 'power1.out'
      }, `launch+=${0.94 + index * 0.07}`)
    })
  }

  if (lowerSegments.length > 0) {
    const revealWindowLower = Math.max(0.45, launchDuration - 0.25)
    const segmentStaggerLower = revealWindowLower / lowerSegments.length

    lowerSegments.forEach((segment, index) => {
      const revealAt = 0.12 + index * segmentStaggerLower
      deloreanTimeline.to(segment, {
        opacity: 1,
        scaleY: () => gsap.utils.random(1.05, 1.18),
        scaleX: () => gsap.utils.random(1.02, 1.12),
        filter: 'brightness(1.15)',
        duration: 0.2,
        ease: 'power1.out'
      }, `launch+=${revealAt}`)

      deloreanTimeline.to(segment, {
        scaleY: () => gsap.utils.random(0.95, 1.04),
        scaleX: () => gsap.utils.random(1.02, 1.1),
        filter: 'brightness(0.98)',
        duration: 0.32,
        ease: 'sine.inOut'
      }, `launch+=${revealAt + 0.18}`)

      deloreanTimeline.to(segment, {
        opacity: () => gsap.utils.random(0.26, 0.38),
        filter: 'brightness(0.6)',
        duration: 0.64,
        ease: 'power1.out'
      }, `launch+=${1.02 + index * 0.08}`)
    })
  }

  if (sparks.length > 0) {
    deloreanTimeline.to(sparks, {
      opacity: () => gsap.utils.random(0.85, 1),
      x: () => gsap.utils.random(-28, 28),
      y: () => gsap.utils.random(-32, -10),
      scale: () => gsap.utils.random(0.7, 1.15),
      rotate: () => gsap.utils.random(-28, 28),
      duration: 0.18,
      repeat: 6,
      repeatRefresh: true,
      yoyo: true,
      stagger: { each: 0.05 },
      ease: 'sine.inOut'
    }, 'launch+=0.06')
  }

  if (burnLine) {
    deloreanTimeline.to(burnLine, { opacity: 0.95, scaleX: 1, duration: 0.48, ease: 'power1.out' }, 'launch+=0.34')
    deloreanTimeline.to(burnLine, { opacity: 0.18, duration: 0.55, ease: 'power1.out' }, 'launch+=0.88')
  }

  deloreanTimeline.to(trailEl, { opacity: 0, duration: 0.5, ease: 'power1.in' }, 'launch+=0.78')
  const fadeStart = launchDuration + 0.1
  deloreanTimeline.to(trailProxy, {
    value: trailingGap,
    duration: 0.48,
    ease: 'power1.in',
    onUpdate: updateTrail
  }, `launch+=${fadeStart}`)

  const returnStart = launchDuration + 1
  deloreanTimeline.add('return', `launch+=${returnStart}`)
  deloreanTimeline.set(carEl, { x: -travelDistance * 0.55, rotation: 0 }, 'return')
  deloreanTimeline.call(() => {
    trailEl.classList.remove('is-active')
    gsap.set(trailEl, { opacity: 0, width: 0 })
    trailEl.style.removeProperty('--segment-spacing')
    trailEl.style.removeProperty('--segment-width')
    trailEl.style.removeProperty('--active-width')
    if (trailBlockerEl) {
      gsap.set(trailBlockerEl, { left: 0, width: 0 })
    }
    if (burnLine) gsap.set(burnLine, { opacity: 0, scaleX: 0.3 })
    if (sparks.length > 0) gsap.set(sparks, { opacity: 0 })
  }, null, 'return')
  deloreanTimeline.call(() => {
    trailProxy.value = 0
  }, null, 'return')
  deloreanTimeline.to(carEl, { x: 0, duration: 0.45, ease: 'power3.out' }, 'return+=0.12')
  deloreanTimeline.to(titleEl, { opacity: 1, y: 0, duration: 0.26, ease: 'power1.out' }, 'return+=0.4')
}

onBeforeUnmount(() => {
  if (deloreanTimeline) {
    deloreanTimeline.kill()
    deloreanTimeline = null
  }
})

</script>

<template>
  <div class="sidebar-header" ref="sidebarHeader">
    <h2 class="sidebar-title" ref="sidebarTitle" @mouseenter="startFireTrail">
      <div ref="fireTrail" class="fire-trail">
        <div class="trail-lane lane-upper">
          <span
            v-for="segment in flameSegments"
            :key="segment.index"
            class="flame-segment"
            :style="{
              '--seg-index': segment.index,
              '--seg-jitter': `${segment.upperJitter}`,
              '--hue-shift': `${segment.upperHueShift}`
            }"
          ></span>
        </div>
        <div class="trail-lane lane-lower">
          <span
            v-for="segment in flameSegments"
            :key="`lower-${segment.index}`"
            class="flame-segment flame-segment--lower"
            :style="{
              '--seg-index': segment.index,
              '--seg-jitter': `${segment.lowerJitter}`,
              '--hue-shift': `${segment.lowerHueShift}`
            }"
          ></span>
        </div>
        <div ref="trailBlocker" class="trail-blocker"></div>
        <span class="spark spark-a"></span>
        <span class="spark spark-b"></span>
        <span class="spark spark-c"></span>
        <span class="spark spark-d"></span>
        <span class="ember-line"></span>
      </div>
      <img
        ref="deloreanIcon"
        src="../assets/images/delorean.png"
        alt="DeLorean"
        class="delorean-icon"
      />
      <span ref="titleTextEl" class="title-text">MIB to the Future</span>
    </h2>
    <md-outlined-text-field
      class="search-input"
      placeholder="Search OID or name..."
      :value="props.search"
      @input="updateSearch"
    >
      <span slot="leading-icon" class="material-symbols-outlined">search</span>
    </md-outlined-text-field>
  </div>
</template>

<style scoped>
.sidebar-header {
  padding: var(--spacing-md);
  border-bottom: 1px solid var(--md-sys-color-outline-variant);
  background-color: var(--md-sys-color-surface-container);
}

.sidebar-title {
  display: flex;
  align-items: center;
  gap: var(--spacing-sm);
  font-size: var(--md-sys-typescale-title-medium-size);
  font-weight: 500;
  color: var(--md-sys-color-on-surface);
  margin: 0 0 var(--spacing-md) 0;
  cursor: pointer;
  position: relative;
  overflow: hidden;
}

.search-input {
  width: 100%;
}

.delorean-icon {
  width: 38px;
  height: 38px;
  object-fit: contain;
  position: absolute;
  left: 8px;
  top: 50%;
  transform: translateY(-50%);
  pointer-events: none;
  z-index: 5;
  will-change: transform;
}

.title-text {
  display: inline-flex;
  align-items: center;
  margin-left: 56px;
  font-weight: 600;
  letter-spacing: 0.01em;
  transition: opacity 0.2s ease, transform 0.2s ease;
}

.fire-trail {
  position: absolute;
  top: calc(50% + 6px);
  left: 56px;
  height: 30px;
  width: 0;
  opacity: 0;
  transform: translateY(-50%);
  pointer-events: none;
  z-index: 1;
  --segment-spacing: 54px;
  --segment-width: 72px;
  --active-width: 0px;
  overflow: visible;
  will-change: width, opacity;
}

.fire-trail .trail-lane {
  position: absolute;
  top: 0;
  bottom: 0;
  left: 0;
  width: var(--active-width);
  pointer-events: none;
  mix-blend-mode: screen;
  overflow: hidden;
}

.fire-trail .flame-segment {
  position: absolute;
  bottom: 0;
  left: calc(var(--seg-index, 0) * var(--segment-spacing));
  width: var(--segment-width);
  height: 100%;
  border-radius: 999px;
  background:
    linear-gradient(
      180deg,
      hsl(calc(34 + var(--hue-shift, 0)) 87% 65%) 0%,
      hsl(calc(25 + var(--hue-shift, 0)) 92% 54%) 48%,
      hsl(calc(22 + var(--hue-shift, 0)) 85% 32%) 100%
    );
  clip-path: polygon(0% 100%, 0% 58%, 10% 34%, 22% 52%, 36% 24%, 48% 54%, 62% 20%, 76% 50%, 90% 32%, 100% 58%, 100% 100%);
  box-shadow:
    0 0 16px rgba(255, 132, 0, 0.32),
    0 0 32px rgba(255, 70, 0, 0.18);
  filter: brightness(0.78);
  opacity: 0;
  transform: translateY(calc(var(--seg-jitter, 0) * 1px)) scale(0.88);
  transform-origin: center bottom;
  mix-blend-mode: screen;
  will-change: transform, opacity, filter;
  pointer-events: none;
}

.fire-trail .flame-segment::before {
  content: '';
  position: absolute;
  inset: 18% 22% 26%;
  border-radius: inherit;
  background: radial-gradient(circle at 50% 0%, rgba(255, 255, 255, 0.92) 0%, rgba(255, 194, 90, 0.52) 58%, rgba(255, 64, 0, 0.16) 100%);
  opacity: 0.9;
  mix-blend-mode: screen;
}

.fire-trail .flame-segment::after {
  content: '';
  position: absolute;
  inset: 42% 12% -6%;
  border-radius: inherit;
  background: radial-gradient(ellipse at 50% 0%, rgba(255, 192, 70, 0.4) 0%, rgba(255, 100, 0, 0.2) 55%, rgba(0, 0, 0, 0) 100%);
  opacity: 0.5;
  mix-blend-mode: screen;
}

.fire-trail .flame-segment--lower {
  transform-origin: center bottom;
}

.fire-trail .flame-segment--lower::after {
  inset: 58% 18% -16%;
  background: radial-gradient(ellipse at 50% 0%, rgba(255, 180, 60, 0.32) 0%, rgba(255, 110, 0, 0.18) 60%, rgba(0, 0, 0, 0) 100%);
}

.fire-trail .spark {
  position: absolute;
  bottom: 18px;
  width: 12px;
  height: 12px;
  border-radius: 50%;
  background: radial-gradient(circle, rgba(255, 249, 228, 0.95) 0%, rgba(255, 196, 90, 0.75) 55%, rgba(255, 90, 0, 0.2) 100%);
  box-shadow:
    0 0 12px rgba(255, 192, 80, 0.85),
    0 0 22px rgba(255, 110, 0, 0.5);
  opacity: 0;
  transform: translate(-50%, 0) scale(0.6);
  mix-blend-mode: screen;
  pointer-events: none;
  z-index: 2;
}

.spark-a { left: 20%; }
.spark-b { left: 45%; }
.spark-c { left: 70%; }
.spark-d { left: 90%; }

.fire-trail .spark::after {
  content: '';
  position: absolute;
  inset: -3px -2px;
  border-radius: 50%;
  background: radial-gradient(circle, rgba(255, 255, 255, 0.85) 0%, rgba(255, 255, 255, 0) 70%);
}

.trail-blocker {
  position: absolute;
  top: -12px;
  bottom: -10px;
  left: 0;
  background: var(--md-sys-color-surface-container);
  box-shadow: none;
  pointer-events: none;
  mix-blend-mode: normal;
  z-index: 4;
}

.fire-trail .ember-line {
  position: absolute;
  left: 0;
  right: 0;
  bottom: -8px;
  height: 6px;
  border-radius: 999px;
  background: linear-gradient(90deg, rgba(255, 128, 0, 0.9) 0%, rgba(255, 212, 150, 0.72) 45%, rgba(255, 110, 0, 0.85) 100%);
  box-shadow: 0 0 12px rgba(255, 118, 0, 0.55);
  opacity: 0;
  transform: scaleX(0.3);
  transform-origin: left center;
  filter: blur(0.5px);
  pointer-events: none;
}

.sidebar-title .material-symbols-outlined {
  color: var(--md-sys-color-primary);
}
</style>
