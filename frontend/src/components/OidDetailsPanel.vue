<script setup>
/**
 * @vue-component
 * @description Displays the details of a selected MIB node.
 *
 * @vue-prop {Object} [node=null] - The MIB node object whose details are to be displayed.
 */
defineProps({
  node: {
    type: Object,
    default: null
  }
})
</script>

<template>
  <div v-if="node" class="details-panel">
    <h3 class="details-title">OID Details</h3>
    <table class="details-table">
      <tbody>
        <tr>
          <th>Name</th>
          <td>{{ node.name }}</td>
        </tr>
        <tr>
          <th>OID</th>
          <td>{{ node.oid }}</td>
        </tr>
        <tr>
          <th>Type</th>
          <td>{{ node.type || 'N/A' }}</td>
        </tr>
        <tr>
          <th>Syntax</th>
          <td>{{ node.syntax || 'N/A' }}</td>
        </tr>
        <tr>
          <th>Access</th>
          <td>{{ node.access || 'N/A' }}</td>
        </tr>
        <tr>
          <th>Status</th>
          <td>{{ node.status || 'N/A' }}</td>
        </tr>
        <tr>
          <th>Description</th>
          <td>{{ node.description || 'No description available.' }}</td>
        </tr>
      </tbody>
    </table>
  </div>
  <div v-else class="details-panel-placeholder">
    <span class="material-symbols-outlined">touch_app</span>
    <p>Select a node to see its details</p>
  </div>
</template>

<style scoped>
.details-panel {
  height: 100%;
  overflow-y: auto;
  padding: var(--spacing-md);
  background-color: var(--md-sys-color-surface-container);
  flex-shrink: 0;
}

.details-title {
  font-size: var(--md-sys-typescale-title-small-size);
  font-weight: 500;
  margin: 0 0 var(--spacing-sm) 0;
  color: var(--md-sys-color-on-surface-variant);
}

.details-table {
  width: 100%;
  border-collapse: collapse;
  font-size: 14px;
}

.details-table th,
.details-table td {
  padding: var(--spacing-xs);
  text-align: left;
  border-bottom: 1px solid var(--md-sys-color-outline-variant);
  vertical-align: top;
}

.details-table th {
  font-weight: 500;
  color: var(--md-sys-color-on-surface-variant);
  width: 90px;
}

.details-table td {
  color: var(--md-sys-color-on-surface);
  word-break: break-all;
}
</style>