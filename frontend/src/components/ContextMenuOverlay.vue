<template>
  <div>
    <div
      v-if="state.visible"
      class="context-menu-overlay"
      @click="$emit('close')"
      @contextmenu.prevent="$emit('close')"
    ></div>
    <div
      v-if="state.visible"
      class="context-menu"
      data-test="context-menu"
      :style="{ left: `${state.x}px`, top: `${state.y}px` }"
      @contextmenu.stop.prevent
      @click.stop
    >
      <template v-for="item in state.items" :key="item.key">
        <div v-if="item.type === 'divider'" class="context-menu__divider"></div>
        <button
          v-else
          type="button"
          class="context-menu__item"
          @click="$emit('select', item)"
        >
          <span class="material-symbols-outlined context-menu__icon">{{ item.icon }}</span>
          <span class="context-menu__label">{{ item.label }}</span>
        </button>
      </template>
    </div>
  </div>
</template>

<script setup>
defineProps({
  state: {
    type: Object,
    required: true
  }
})
defineEmits(['close', 'select'])
</script>

<style scoped>
.context-menu-overlay {
  position: fixed;
  inset: 0;
  background: transparent;
  z-index: 2147483646;
}

.context-menu {
  position: fixed;
  z-index: 2147483647;
  min-width: 220px;
  max-width: min(280px, calc(100vw - 16px));
  background-color: var(--md-sys-color-surface-container-high, #1f2937);
  border: 1px solid var(--md-sys-color-outline-variant, rgba(255, 255, 255, 0.12));
  border-radius: 12px;
  box-shadow: 0 8px 22px rgba(0, 0, 0, 0.25);
  padding: 4px 0;
  overflow: hidden;
}

.context-menu__item {
  width: 100%;
  display: flex;
  align-items: center;
  gap: 12px;
  padding: 10px 16px;
  background: none;
  border: none;
  color: var(--md-sys-color-on-surface, #f3f4f6);
  font: inherit;
  text-align: left;
  cursor: pointer;
  transition: background-color 0.15s ease;
}

.context-menu__item:hover,
.context-menu__item:focus {
  background-color: var(--md-sys-color-surface-container-highest, rgba(255, 255, 255, 0.06));
}

.context-menu__icon {
  font-size: 20px;
  color: var(--md-sys-color-primary, #8b5cf6);
}

.context-menu__label {
  flex: 1;
  font-size: 14px;
}

.context-menu__divider {
  height: 1px;
  margin: 4px 16px;
  background-color: var(--md-sys-color-outline-variant, rgba(255, 255, 255, 0.12));
}
</style>
