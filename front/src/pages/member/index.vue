<template><view class="page">
  <view class="member-card"><text class="label">HOLE MEMBERSHIP</text><text class="level">BLACK</text><text class="name">体验会员</text><text class="no">NO. 000888</text></view>
  <view class="stats"><view><text class="value">{{validCount}}</text><text class="key">累计预订</text></view><view><text class="value">0</text><text class="key">积分</text></view><view><text class="value">2</text><text class="key">可用权益</text></view></view>
  <text class="section">会员权益</text>
  <view class="benefit"><text class="icon">01</text><view><text class="benefit-title">优先留台</text><text class="desc">热门日期享优先预订机会</text></view></view>
  <view class="benefit"><text class="icon">02</text><view><text class="benefit-title">生日礼遇</text><text class="desc">生日月到店获得专属礼遇</text></view></view>
  <text class="tip">当前为产品演示版本，登录与真实积分将在服务端接入后开放。</text>
</view></template>
<script setup lang="ts">
import {computed,ref} from 'vue'; import {onShow} from '@dcloudio/uni-app'; import {getOrders,type ClubOrder} from '../../services/orders'
const orders=ref<ClubOrder[]>([]); onShow(()=>orders.value=getOrders()); const validCount=computed(()=>orders.value.filter(o=>o.status==='reserved').length)
</script>
<style scoped>
.page{min-height:100vh;background:#08080c;color:#f5f2e9;padding:48rpx 32rpx;box-sizing:border-box}.member-card{height:360rpx;background:linear-gradient(135deg,#c9b579,#6d6040);color:#09090d;padding:38rpx;display:flex;flex-direction:column;box-sizing:border-box}.label{font-size:19rpx;letter-spacing:4rpx}.level{font-size:90rpx;font-weight:900;margin-top:40rpx}.name{margin-top:auto;font-size:25rpx}.no{font-family:monospace;font-size:20rpx;margin-top:8rpx}.stats{display:grid;grid-template-columns:repeat(3,1fr);padding:44rpx 0;border-bottom:1px solid #302f2b;text-align:center}.value,.key{display:block}.value{font-size:44rpx;font-weight:700}.key{font-size:20rpx;color:#777;margin-top:8rpx}.section{display:block;font-size:34rpx;font-weight:700;margin:52rpx 0 20rpx}.benefit{display:flex;align-items:center;gap:30rpx;padding:30rpx 0;border-bottom:1px solid #252429}.icon{width:72rpx;height:72rpx;border:1px solid #6d6040;color:#c9b579;display:flex;align-items:center;justify-content:center}.benefit-title,.desc{display:block}.benefit-title{font-size:28rpx}.desc{color:#777;font-size:22rpx;margin-top:8rpx}.tip{display:block;color:#555;font-size:20rpx;line-height:1.6;margin-top:50rpx}
</style>
