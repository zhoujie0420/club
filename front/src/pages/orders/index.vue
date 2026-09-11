<template><view class="page">
  <view class="heading"><text class="eyebrow">RESERVATIONS</text><text class="title">我的订单</text></view>
  <view v-if="orders.length===0" class="empty"><text class="empty-title">还没有预订</text><text class="muted">选一张喜欢的台，开启今晚。</text><button class="primary" @click="goBooking">去订台</button></view>
  <view v-else><view v-for="order in orders" :key="order.id" class="order">
    <view class="order-top"><text class="id">{{order.id}}</text><text :class="['status',order.status]">{{order.status==='reserved'?'已预订':'已取消'}}</text></view>
    <text class="date">{{order.date}} · {{order.session}}</text><text class="seats">台位 {{order.seats.join('、')}}</text><text class="guest">预订人 {{order.guestName}} · {{mask(order.phone)}}</text>
    <button v-if="order.status==='reserved'" class="cancel" @click="cancel(order.id)">取消预订</button>
  </view></view>
</view></template>
<script setup lang="ts">
import {ref} from 'vue'; import {onShow} from '@dcloudio/uni-app'; import {getOrders,cancelOrder,type ClubOrder} from '../../services/orders'
const orders=ref<ClubOrder[]>([]); onShow(()=>orders.value=getOrders())
const goBooking=()=>uni.navigateTo({url:'/pages/booking/index'}); const mask=(p:string)=>p.replace(/(\d{3})\d{4}(\d+)/,'$1****$2')
const cancel=(id:string)=>uni.showModal({title:'取消预订',content:'确认取消这笔预订吗？',success:r=>{if(r.confirm){cancelOrder(id);orders.value=getOrders()}}})
</script>
<style scoped>
.page{min-height:100vh;background:#08080c;color:#f5f2e9;padding:46rpx 32rpx;box-sizing:border-box}.heading{display:flex;flex-direction:column;border-bottom:1px solid #302f2b;padding-bottom:30rpx}.eyebrow{font-size:18rpx;letter-spacing:5rpx;color:#b6a77b}.title{font-size:58rpx;font-weight:800;margin-top:12rpx}.empty{text-align:center;padding:180rpx 30rpx}.empty-title{display:block;font-size:38rpx;font-weight:700}.muted{display:block;color:#777;margin:20rpx 0 48rpx}.primary{background:#c7b276;color:#08080c;border-radius:0}.order{background:#111116;border:1px solid #2c2b27;padding:30rpx;margin-top:24rpx}.order-top{display:flex;justify-content:space-between}.id{font-family:monospace;color:#aaa}.status{font-size:22rpx;color:#c7b276}.status.cancelled{color:#666}.date{display:block;font-size:34rpx;font-weight:700;margin:28rpx 0 15rpx}.seats,.guest{display:block;color:#aaa;font-size:24rpx;margin-top:10rpx}.cancel{margin:28rpx 0 0;background:transparent;border:1px solid #5d5542;color:#c7b276;border-radius:0;font-size:24rpx}
</style>
