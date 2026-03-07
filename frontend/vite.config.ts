import { sveltekit } from '@sveltejs/kit/vite';
import { defineConfig } from 'vite';

export default defineConfig({
	plugins: [sveltekit()],
	server: {
		proxy: {
			'/api': {
				target: process.env.PUBLIC_API_BASE || 'http://localhost:8080',
				changeOrigin: true
			},
			'/ws': {
				target: process.env.PUBLIC_API_BASE || 'http://localhost:8080',
				changeOrigin: true,
				ws: true
			}
		}
	}
});
