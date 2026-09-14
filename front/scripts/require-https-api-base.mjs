const base = process.env.VITE_API_BASE || "";
if (!/^https:\/\//i.test(base)) {
  console.error("VITE_API_BASE must be an https:// URL for mp-weixin builds");
  process.exit(1);
}
