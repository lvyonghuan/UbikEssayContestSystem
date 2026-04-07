import { createApp } from 'vue'
import { createPinia } from 'pinia'
import ElementPlus from 'element-plus'
import 'element-plus/dist/index.css'
import './style.css'
import App from './App.vue'
import router from './router'
import { adminClient, systemClient } from './services/http/client'
import { setupInterceptors } from './services/http/interceptors'

async function prepareMockServer() {
	if (import.meta.env.VITE_USE_MOCK === 'true') {
		const { worker } = await import('./mocks/browser')
		await worker.start({ onUnhandledRequest: 'bypass' })
	}
}

prepareMockServer().finally(() => {
	setupInterceptors([adminClient, systemClient])
	const app = createApp(App)
	app.use(createPinia())
	app.use(router)
	app.use(ElementPlus)
	app.mount('#app')
})
