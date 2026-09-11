<template>
  <view class="page">
    <view class="page-head"><view><text class="eyebrow">RESERVATION</text><text class="title">选择你的台位</text></view><view class="step">01 / 02</view></view>
    <view class="card form">
      <picker mode="date" :value="bookingDate" @change="onDateChange"><view class="row"><view><text class="small">到店日期</text><text class="value">{{bookingDate}}</text></view><text class="arrow">›</text></view></picker>
      <view class="sessions"><button v-for="item in sessions" :key="item" :class="{active:session===item}" @click="session=item">{{item}}</button></view>
    </view>
    <view class="map-head"><view><text class="small">FLOOR PLAN</text><text class="map-title">选择空闲台位</text></view><view class="legend"><i></i>可选 <i class="busy"></i>已订</view></view>
    <view class="floor">
      <view class="stage"><span></span><text>DJ STAGE</text><span></span></view>
      <view class="zone">VIP AREA</view>
      <view class="seats">
        <button v-for="seat in displaySeats" :key="seat.id" :class="['seat',seat.status,seat.type]" :disabled="seat.status==='occupied'" @click="toggleSeat(seat)">
          <text>{{seat.name}}</text><span>{{seat.type==='vip'?'VIP':'TABLE'}}</span>
        </button>
      </view>
      <view class="bar">BAR · BAR · BAR</view>
    </view>
    <view class="details">
      <text class="details-title">预订信息</text>
      <view class="inputs"><input v-model="guestName" placeholder="姓名" placeholder-class="placeholder"/><input v-model="phone" type="number" maxlength="11" placeholder="手机号码" placeholder-class="placeholder"/></view>
    </view>
    <view class="selection"><view><text class="small">已选台位</text><text class="selected-names">{{selectedSeats.length?selectedSeats.map(s=>s.name).join('、'):'暂未选择'}}</text></view><button :disabled="!selectedSeats.length" @click="confirmBooking">确认预订 <text>→</text></button></view>
  </view>
</template>
<script setup lang="ts">
import {ref} from 'vue'; import seatConfig from './seat-config.json'; import {createOrder} from '../../services/orders'
type SeatStatus='available'|'occupied'|'selected'; interface Seat{id:string;name:string;type:string;status:SeatStatus;isStage?:boolean;isBar?:boolean}
const displaySeats=ref<Seat[]>(seatConfig.seats.filter((s:any)=>!s.isStage&&!s.isBar).slice(0,20).map((s:any,i:number)=>({...s,status:(i===4||i===11)?'occupied':s.status})))
const selectedSeats=ref<Seat[]>([]),tomorrow=new Date(Date.now()+86400000),bookingDate=ref(tomorrow.toISOString().slice(0,10)),sessions=['20:00','22:30','00:30'],session=ref('22:30'),guestName=ref(''),phone=ref('')
const onDateChange=(e:any)=>bookingDate.value=e.detail.value
const toggleSeat=(seat:Seat)=>{if(seat.status==='occupied')return;if(seat.status==='selected'){seat.status='available';selectedSeats.value=selectedSeats.value.filter(s=>s.id!==seat.id)}else{seat.status='selected';selectedSeats.value.push(seat)}}
const confirmBooking=()=>{if(!guestName.value.trim()||!/^1\d{10}$/.test(phone.value)){uni.showToast({title:'请填写姓名和正确手机号',icon:'none'});return}const o=createOrder({seats:selectedSeats.value.map(s=>s.name),date:bookingDate.value,session:session.value,guestName:guestName.value.trim(),phone:phone.value});uni.showModal({title:'预订成功',content:`订单 ${o.id} 已生成`,showCancel:false,success:()=>uni.redirectTo({url:'/pages/orders/index'})})}
</script>
<style scoped>
.page{min-height:100vh;background:#0d0d12;color:#f4f3ed;padding:28px 20px 120px}.page-head{display:flex;justify-content:space-between;align-items:end;margin-bottom:24px}.eyebrow,.small{display:block;font-size:9px;letter-spacing:2px;color:#7a7a84}.eyebrow{color:#d7ff3f}.title{display:block;font-size:28px;font-weight:850;margin-top:6px}.step{font-size:11px;color:#666670}.card{background:#17171e;border:1px solid #24242c;border-radius:12px}.form{padding:0 16px}.row{height:72px;display:flex;align-items:center;justify-content:space-between;border-bottom:1px solid #292930}.value{display:block;font-size:16px;font-weight:700;margin-top:4px}.arrow{font-size:26px;color:#777}.sessions{display:grid;grid-template-columns:repeat(3,1fr);gap:8px;padding:14px 0}.sessions button{height:38px;background:#202028;color:#85858d;border-radius:7px;font-size:12px}.sessions button.active{background:#d7ff3f;color:#09090d;font-weight:800}.map-head{display:flex;justify-content:space-between;align-items:end;margin:30px 0 14px}.map-title{display:block;font-size:18px;font-weight:800;margin-top:4px}.legend{font-size:9px;color:#777}.legend i{display:inline-block;width:7px;height:7px;border-radius:2px;background:#292934;border:1px solid #51515d;margin:0 4px 0 9px}.legend i.busy{background:#24242a;opacity:.4}.floor{border:1px solid #292931;border-radius:14px;padding:16px;background:radial-gradient(circle at 50% 0,rgba(115,52,224,.2),transparent 37%),#121218}.stage{height:44px;border-radius:7px;background:linear-gradient(90deg,#241835,#6f3b99,#241835);display:flex;align-items:center;justify-content:center;gap:10px;color:#e9d9ff;font-size:9px;letter-spacing:3px}.stage span{width:28px;height:1px;background:#8867a4}.zone{text-align:center;font-size:8px;letter-spacing:3px;color:#53535d;margin:15px}.seats{display:grid;grid-template-columns:repeat(4,1fr);gap:9px}.seat{height:58px;margin:0;padding:0;background:#22222a;color:#ddd;border:1px solid #343440;border-radius:9px;display:flex;flex-direction:column;align-items:center;justify-content:center}.seat text{font-size:12px;font-weight:800}.seat span{font-size:7px;letter-spacing:1px;color:#676771;margin-top:4px}.seat.vip{border-color:#583e77;background:#21182a}.seat.selected{background:#d7ff3f;border-color:#d7ff3f;color:#08080c;box-shadow:0 0 18px rgba(215,255,63,.2)}.seat.selected span{color:#59631b}.seat.occupied{opacity:.25}.bar{text-align:center;margin-top:14px;padding:10px;border:1px dashed #373740;border-radius:6px;color:#52525d;font-size:8px;letter-spacing:3px}.details{margin-top:25px}.details-title{font-size:16px;font-weight:800}.inputs{display:grid;grid-template-columns:1fr 1.4fr;gap:9px;margin-top:12px}.inputs input{height:46px;background:#17171e;border:1px solid #292931;border-radius:8px;padding:0 13px;font-size:12px;color:white}.placeholder{color:#53535d}.selection{position:fixed;bottom:0;left:50%;transform:translateX(-50%);width:min(480px,100%);height:88px;background:rgba(18,18,24,.96);backdrop-filter:blur(15px);border-top:1px solid #2a2a31;padding:14px 20px;display:flex;align-items:center;justify-content:space-between}.selected-names{display:block;font-size:13px;font-weight:700;margin-top:5px;max-width:150px;white-space:nowrap;overflow:hidden;text-overflow:ellipsis}.selection button{margin:0;width:160px;height:52px;background:#d7ff3f;color:#09090d;border-radius:6px;font-size:13px;font-weight:800;display:flex;justify-content:space-between;align-items:center;padding:0 18px}.selection button[disabled]{background:#27272f;color:#5c5c65}
</style>
