<script setup>
import { computed } from 'vue'

defineOptions({ name: 'ModuleTreeNode' })

const props = defineProps({
  node: {
    type: Object,
    required: true
  },
  level: {
    type: Number,
    default: 0
  },
  expanded: {
    type: Object,
    required: true
  }
})

const emit = defineEmits(['toggle'])

const hasChildren = computed(() => Array.isArray(props.node?.children) && props.node.children.length > 0)
const isExpanded = computed(() => props.expanded.has(props.node?.oid))

const handleToggle = (event) => {
  event.stopPropagation()
  if (hasChildren.value) {
    emit('toggle', props.node.oid)
  }
}
</script>

<template>
  <li class="module-tree-node">
    <div class="module-tree-row" :style="{ paddingLeft: `${level * 16}px` }" @click="handleToggle">
      <span
        v-if="hasChildren"
        class="material-symbols-outlined toggle-icon"
      >
        {{ isExpanded ? 'expand_more' : 'chevron_right' }}
      </span>
      <span v-else class="toggle-icon toggle-icon--placeholder"></span>
      <span class="module-tree-name">{{ node.name || node.oid }}</span>
    </div>
    <ul v-if="hasChildren && isExpanded" class="module-tree-children">
      <ModuleTreeNode
        v-for="child in node.children"
        :key="child.oid"
        :node="child"
        :level="level + 1"
        :expanded="expanded"
        @toggle="emit('toggle', $event)"
      />
    </ul>
  </li>
</template>

<style scoped>
.module-tree-node {
  list-style: none;
}

.module-tree-row {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 4px 8px;
  border-radius: 12px;
  cursor: pointer;
  transition: background-color 0.15s ease;
  color: var(--md-sys-color-on-surface);
}

.module-tree-row:hover {
  background-color: var(--md-sys-color-surface-container-high);
}

.module-tree-name {
  font-size: 14px;
}

.toggle-icon {
  font-size: 18px;
  width: 24px;
  height: 24px;
  display: flex;
  align-items: center;
  justify-content: center;
  color: var(--md-sys-color-on-surface-variant);
}

.toggle-icon--placeholder {
  width: 24px;
  height: 24px;
}

.module-tree-children {
  padding-left: 0;
  margin: 0;
}
</style>
