import { createApp } from 'vue'
import { createPinia } from 'pinia'
import { createNotivue } from 'notivue'

import 'notivue/notification.css'
import 'notivue/animations.css'

import App from './App.vue'
import router from './router'

const notivue = createNotivue({
	position: 'bottom-right',
	limit: 5,
	enqueue: true,
	avoidDuplicates: true,
	notifications: {
		global: {
			duration: 6000,
		},
		error: {
			duration: 8000,
			ariaLive: 'assertive',
			ariaRole: 'alert',
		},
	},
})

const app = createApp(App)

app.use(createPinia())
app.use(notivue)
app.use(router)

app.mount('#app')
