import { defineConfig } from 'vite'
import path from 'path'
import { fileURLToPath } from 'url'
import tailwindcss from '@tailwindcss/vite'
import react from '@vitejs/plugin-react'

// ESM 下没有 __dirname，从 import.meta.url 推导
const __dirname = path.dirname(fileURLToPath(import.meta.url))

function figmaAssetResolver() {
  return {
    name: 'figma-asset-resolver',
    resolveId(id: string) {
      if (id.startsWith('figma:asset/')) {
        const filename = id.replace('figma:asset/', '')
        return path.resolve(__dirname, 'src/assets', filename)
      }
    },
  }
}

export default defineConfig({
  plugins: [
    figmaAssetResolver(),
    // The React and Tailwind plugins are both required for Make, even if
    // Tailwind is not being actively used – do not remove them
    react(),
    tailwindcss(),
  ],
  resolve: {
    alias: {
      // Alias @ to the src directory
      '@': path.resolve(__dirname, './src'),
    },
  },

  server: {
    // 显式绑 IPv4，否则 Vite 默认只监听 ::1(IPv6)，Windows 上 localhost/127.0.0.1 会连接被拒
    host: '127.0.0.1',
    port: 5173,
    strictPort: true,
    // 开发时把后端请求透传到 Go 服务(8091)，前端直接调 /api、/health 即可，免跨域
    proxy: {
      '/api': 'http://127.0.0.1:8091',
      '/health': 'http://127.0.0.1:8091',
    },
  },

  // File types to support raw imports. Never add .css, .tsx, or .ts files to this.
  assetsInclude: ['**/*.svg', '**/*.csv'],
})
