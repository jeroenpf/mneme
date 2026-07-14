import { createApp } from 'vue'
import App from './App.vue'
import { router } from './router'
import { useTheme } from './composables/useTheme'
import './main.css'

// Apply the persisted (or system-preferred) theme before mount so dark-mode
// users never see the paper default flash in.
useTheme().init()

createApp(App).use(router).mount('#app')
