import { sveltekit } from '@sveltejs/kit/vite';
import { paraglideVitePlugin } from '@inlang/paraglide-js';
import tailwindcss from '@tailwindcss/vite';
import { defineConfig } from 'vite';

export default defineConfig({
	plugins: [
		tailwindcss(),
		sveltekit(),
		paraglideVitePlugin({
			project: './project.inlang',
			outdir: './src/lib/paraglide'
		})
	],
	server: {
		port: 5173,
		proxy: {
			// Proxy API requests to Go backend during development
			'/api': {
				target: 'http://localhost:8080',
				changeOrigin: true
			},
			'/v1': {
				target: 'http://localhost:8080',
				changeOrigin: true
			}
		}
	}
});
