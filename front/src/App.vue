<script setup lang="ts">
import { onLaunch, onShow, onHide } from "@dcloudio/uni-app";
onLaunch(() => {
  console.log("App Launch");
});
onShow(() => {
  console.log("App Show");
});
onHide(() => {
  console.log("App Hide");
});
</script>
<style>
/* #ifdef H5 */
html,body,#app,uni-app{min-height:100%;background:#09090d;margin:0}
body{font-family:Inter,-apple-system,BlinkMacSystemFont,"Segoe UI","PingFang SC",sans-serif;color:#f5f3ed}
uni-page-body{display:block;max-width:480px;min-height:100vh;margin:0 auto;background:#0d0d12;box-shadow:0 0 80px rgba(0,0,0,.55)}
/* #endif */
/* #ifndef H5 */
page{min-height:100%;background:#0d0d12;color:#f5f3ed;font-family:Inter,-apple-system,"PingFang SC",sans-serif}
/* #endif */
button::after{border:none}
*{box-sizing:border-box}
</style>
