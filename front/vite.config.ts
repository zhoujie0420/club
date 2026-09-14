import { defineConfig } from "vite";
import uni from "@dcloudio/vite-plugin-uni";

// https://vitejs.dev/config/
export default defineConfig({
  plugins: [uni()],
  base: "/club/",
  server: { proxy: { '/api': 'http://127.0.0.1:18080' } },
});
