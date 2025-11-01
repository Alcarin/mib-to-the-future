/**
 * @file Main entry point for the Vue application.
 * This file initializes the Vue app, imports global styles, and mounts the root component.
 */
import { createApp } from 'vue'
import App from './App.vue'
import './style.css'
import './theme.css'
import 'material-symbols/outlined.css'

createApp(App).mount('#app')
