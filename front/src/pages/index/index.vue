<template>
  <view class="container">
    <!-- 顶部标题 -->
    <view class="header">
      <text class="title">HOLE CLUB</text>
      <text class="subtitle">夜店订台系统</text>
    </view>

    <!-- 导航菜单 -->
    <view class="menu-container">
      <view class="menu-item" @click="goToBooking">
        <view class="icon booking-icon"></view>
        <text class="menu-text">房台状态</text>
      </view>

      <view class="menu-item">
        <view class="icon order-icon"></view>
        <text class="menu-text">我的订单</text>
      </view>

      <view class="menu-item">
        <view class="icon history-icon"></view>
        <text class="menu-text">消费记录</text>
      </view>

      <view class="menu-item">
        <view class="icon member-icon"></view>
        <text class="menu-text">会员中心</text>
      </view>
    </view>

    <!-- 底部信息 -->
    <view class="footer">
      <text class="footer-text">欢迎光临HOLE CLUB</text>
    </view>
  </view>
</template>

<script setup lang="ts">
const goToBooking = () => {
  uni.navigateTo({
    url: '/pages/booking/index'
  })
}
</script>

<style>
.container {
  display: flex;
  flex-direction: column;
  height: 100vh;
  background: linear-gradient(to bottom, #1a1a2e, #16213e);
  color: white;
  padding: 40rpx;
}

.header {
  text-align: center;
  margin-bottom: 100rpx;
  margin-top: 100rpx;
}

.title {
  font-size: 60rpx;
  font-weight: bold;
  display: block;
  margin-bottom: 20rpx;
  background: linear-gradient(45deg, #ff416c, #ff4b2b);
  -webkit-background-clip: text;
  -webkit-text-fill-color: transparent;
}

.subtitle {
  font-size: 32rpx;
  color: #aaa;
  display: block;
}

.menu-container {
  display: grid;
  grid-template-columns: 1fr 1fr;
  gap: 40rpx;
  flex: 1;
}

.menu-item {
  background: rgba(255, 255, 255, 0.1);
  border-radius: 20rpx;
  padding: 60rpx 20rpx;
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  backdrop-filter: blur(10px);
  border: 1rpx solid rgba(255, 255, 255, 0.2);
  transition: all 0.3s ease;
}

.menu-item:active {
  transform: scale(0.95);
  background: rgba(255, 255, 255, 0.2);
}

.icon {
  width: 100rpx;
  height: 100rpx;
  border-radius: 50%;
  margin-bottom: 30rpx;
  background: linear-gradient(45deg, #6a11cb, #2575fc);
  display: flex;
  align-items: center;
  justify-content: center;
  font-size: 50rpx;
}

.booking-icon {
  background: linear-gradient(45deg, #ff416c, #ff4b2b);
}

.order-icon {
  background: linear-gradient(45deg, #11998e, #38ef7d);
}

.history-icon {
  background: linear-gradient(45deg, #6a11cb, #2575fc);
}

.member-icon {
  background: linear-gradient(45deg, #f7971e, #ffd200);
}

.menu-text {
  font-size: 32rpx;
  font-weight: bold;
}

.footer {
  text-align: center;
  margin-top: 60rpx;
}

.footer-text {
  color: #aaa;
  font-size: 28rpx;
}
</style>