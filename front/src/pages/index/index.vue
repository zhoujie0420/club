<template>
  <view class="page">
    <view class="hero">
      <text class="eyebrow">WUXI · NIGHT EXPERIENCE</text><text class="brand">HOLE</text><text class="club">CLUB</text>
      <text class="tagline">今晚，给自己留一个位置。</text><button class="primary" @click="go('/pages/booking/index')">立即订台</button>
    </view>
    <view class="section-head"><text class="section-title">服务中心</text><text class="section-sub">SERVICE</text></view>
    <view class="grid">
      <view class="card feature" @click="go('/pages/booking/index')"><text class="number">01</text><text class="card-title">房台状态</text><text class="desc">实时查看并选择空闲台位</text></view>
      <view class="card" @click="go('/pages/orders/index')"><text class="number">02</text><text class="card-title">我的订单</text><text class="desc">预订记录与订单管理</text></view>
      <view class="card" @click="go('/pages/orders/index')"><text class="number">03</text><text class="card-title">消费记录</text><text class="desc">查看历史到店与消费</text></view>
      <view class="card" @click="go('/pages/member/index')"><text class="number">04</text><text class="card-title">会员中心</text><text class="desc">权益、等级与专属礼遇</text></view>
    </view>
    <view class="footer"><text>OPEN DAILY 20:00 — 04:00</text><text>HOLE CLUB · MVP</text></view>
  </view>
</template>
<script setup lang="ts">const go=(url:string)=>uni.navigateTo({url})</script>
<style scoped>
.page{min-height:100vh;background:#08080c;color:#f5f2e9;padding:0 36rpx 60rpx;box-sizing:border-box}.hero{min-height:650rpx;display:flex;flex-direction:column;justify-content:center;border-bottom:1px solid #2d2b25}.eyebrow,.section-sub{font-size:20rpx;letter-spacing:6rpx;color:#b6a77b}.brand,.club{display:block;font-size:150rpx;font-weight:900;line-height:.78;letter-spacing:-8rpx}.club{color:transparent;-webkit-text-stroke:2rpx #b6a77b;margin-left:100rpx}.tagline{margin:60rpx 0 34rpx;color:#aaa49a;font-size:28rpx}.primary{margin:0;width:260rpx;background:#c7b276;color:#09090d;border-radius:0;font-weight:700;font-size:28rpx}.section-head{display:flex;justify-content:space-between;align-items:end;padding:70rpx 0 30rpx}.section-title{font-size:42rpx;font-weight:700}.grid{display:grid;grid-template-columns:1fr 1fr;gap:18rpx}.card{min-height:220rpx;border:1px solid #302f2b;padding:28rpx;display:flex;flex-direction:column;background:#101015}.feature{background:#c7b276;color:#09090d}.number{font-size:20rpx;opacity:.6}.card-title{font-size:32rpx;font-weight:700;margin-top:auto}.desc{font-size:21rpx;opacity:.62;margin-top:12rpx}.footer{border-top:1px solid #2d2b25;margin-top:60rpx;padding-top:28rpx;display:flex;justify-content:space-between;color:#666;font-size:18rpx}
</style>
