import { defineConfig } from 'vite'
import react from '@vitejs/plugin-react'
import tailwindcss from '@tailwindcss/vite'
import { VitePWA } from 'vite-plugin-pwa'

export default defineConfig({
  plugins: [
    react(),
    tailwindcss(),
    VitePWA({
      registerType: 'autoUpdate',
      manifest: {
        name: 'English Anywhere Lab',
        short_name: 'EA Lab',
        start_url: '/today',
        display: 'standalone',
        background_color: '#ffffff',
        theme_color: '#2563eb',
        icons: [
          { src: '/vite.svg', sizes: '192x192', type: 'image/svg+xml' },
          { src: '/vite.svg', sizes: '512x512', type: 'image/svg+xml' },
        ],
      },
      workbox: {
        globPatterns: ['**/*.{js,css,html,ico,png,svg,woff2}'],
        runtimeCaching: [
          {
            // Pack catalogue — read-only, safe to cache
            urlPattern: /\/api\/v1\/packs(\?.*)?$/i,
            handler: 'NetworkFirst',
            options: {
              cacheName: 'api-packs-list',
              expiration: { maxAgeSeconds: 300, maxEntries: 20 },
              cacheableResponse: { statuses: [0, 200] },
            },
          },
          {
            // Individual pack detail + lessons — read-only
            urlPattern: /\/api\/v1\/packs\/[^/]+$/i,
            handler: 'NetworkFirst',
            options: {
              cacheName: 'api-pack-detail',
              expiration: { maxAgeSeconds: 300, maxEntries: 50 },
              cacheableResponse: { statuses: [0, 200] },
            },
          },
          {
            // Lesson output tasks — read-only
            urlPattern: /\/api\/v1\/lessons\/[^/]+\/output-tasks$/i,
            handler: 'NetworkFirst',
            options: {
              cacheName: 'api-output-tasks',
              expiration: { maxAgeSeconds: 300, maxEntries: 50 },
              cacheableResponse: { statuses: [0, 200] },
            },
          },
        ],
      },
    }),
  ],
  server: {
    proxy: {
      '/api': {
        target: 'http://localhost:8080',
        changeOrigin: true,
      },
    },
  },
})
