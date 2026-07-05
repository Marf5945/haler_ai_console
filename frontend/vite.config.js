import { defineConfig } from 'vite'
import react from '@vitejs/plugin-react'

// https://vitejs.dev/config/
export default defineConfig(({ mode }) => ({
  base: './',
  build: {
    // 禁止把小圖內聯成 data: URL——Pixi 用 fetch 載圖時 data: 會被嚴格 CSP 的
    // connect-src 擋下（眼睛/嘴巴/耳朵等 <4KB 小部件全滅，臉破洞）。
    // 全部改走一般檔案請求（connect-src 'self' 本來就放行）。
    assetsInlineLimit: 0,
  },
  plugins: [react({ fastRefresh: false })],
  test: {
    environment: 'jsdom',
    globals: true,
    setupFiles: './src/test/setup.js',
  },
}))
