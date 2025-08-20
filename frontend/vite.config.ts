import { defineConfig } from 'vite'
import react from '@vitejs/plugin-react'
import path from "path" // Pastikan 'path' diimpor

export default defineConfig({
  plugins: [react()],
  resolve: {
    alias: {
      // Baris ini secara manual memberitahu Vite bahwa
      // "@" adalah alias untuk folder "/src"
      "@": path.resolve(__dirname, "./src"),
    },
  },
})