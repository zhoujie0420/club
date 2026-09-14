<template>
  <view class="privacy">
    <text class="title">HOLE CLUB 员工端隐私说明</text>
    <text class="p">本小程序 / H5 仅供门店员工处理订台、到店、加单和线下收款记录。</text>
    <text class="p">登录使用员工账号与口令，不使用微信登录，不读取微信手机号，不发起在线支付。</text>
    <text class="p">员工录入的客户姓名、手机号仅用于到店核验和订单查询。请只填写虚构或经客户同意的测试资料。</text>
    <text class="p">会话保存在服务端，约 12 小时过期；店长重置口令或停用账号后立即失效。</text>
    <text class="p">我们不向第三方出售个人数据。正式营业前将按主体资质补充完整隐私政策。</text>
    <button class="back" @click="back">返回工作台</button>
  </view>
</template>
<script setup lang="ts">
function back() {
  const pages = getCurrentPages();
  if (pages.length > 1) uni.navigateBack();
  else uni.reLaunch({ url: "/pages/console/index" });
}
</script>
<style scoped>
.privacy {
  min-height: 100vh;
  background: #0d0d12;
  color: #f5f3ed;
  padding: 28px 20px 40px;
}
.title {
  display: block;
  font-size: 22px;
  font-weight: 700;
  margin-bottom: 18px;
}
.p {
  display: block;
  font-size: 14px;
  line-height: 1.7;
  color: #c4c4cc;
  margin-bottom: 14px;
}
.back {
  margin-top: 24px;
  height: 48px;
  background: #d7ff3f;
  color: #161811;
  border-radius: 10px;
  font-weight: 700;
}
</style>
