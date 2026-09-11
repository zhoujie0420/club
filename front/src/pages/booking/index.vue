<template>
  <view class="booking-page">
    <!-- 顶部标题 -->
    <view class="header">
      <text class="title">HOLE CLUB</text>
      <text class="subtitle">Location Map</text>
    </view>

    <view class="booking-info">
      <picker mode="date" :value="bookingDate" @change="onDateChange">
        <view class="field"><text class="field-label">到店日期</text><text>{{ bookingDate }} ›</text></view>
      </picker>
      <view class="field">
        <text class="field-label">到店场次</text>
        <view class="sessions"><text v-for="item in sessions" :key="item" :class="['session',{active:session===item}]" @click="session=item">{{item}}</text></view>
      </view>
      <view class="guest-fields">
        <input v-model="guestName" class="input" placeholder="预订人姓名" placeholder-class="placeholder" />
        <input v-model="phone" class="input" type="number" maxlength="11" placeholder="手机号码" placeholder-class="placeholder" />
      </view>
    </view>

    <!-- 座位地图容器 -->
    <view class="venue-map-container">
      <movable-area class="venue-map" :scale-area="false">
        <movable-view
          v-for="seat in seats"
          :key="seat.id"
          class="movable-seat"
          direction="none"
          :x="seat.x"
          :y="seat.y"
          :style="getSeatStyle(seat)"
          @tap="() => handleSeatSelect(seat)"
        >
          <view
            class="seat"
            :class="[seat.type, seat.status, {
              vertical: seat.vertical,
              circle: seat.shape === 'circle',
              stage: seat.isStage,
              bar: seat.isBar
            }]"
          >
            <text class="seat-name" :class="{ 'stage-text': seat.isStage || seat.isBar }">
              {{ seat.name }}
            </text>
          </view>
        </movable-view>
      </movable-area>
    </view>

    <!-- 底部操作栏 -->
    <view class="footer">
      <view class="status-indicators">
        <view class="indicator">
          <view class="color-box available"></view>
          <text>可选</text>
        </view>
        <view class="indicator">
          <view class="color-box occupied"></view>
          <text>已占</text>
        </view>
        <view class="indicator">
          <view class="color-box selected"></view>
          <text>选中</text>
        </view>
      </view>

      <button
        class="confirm-btn"
        :disabled="selectedSeats.length === 0"
        @click="confirmBooking"
      >
        确认定座 ({{ selectedSeats.length }})
      </button>
    </view>
  </view>
</template>

<script setup lang="ts">
import { ref, onMounted } from 'vue'
import seatConfig from './seat-config.json'
import { createOrder } from '../../services/orders'

// 定义座位状态类型
type SeatStatus = 'available' | 'occupied' | 'selected'

// 定义座位接口
interface Seat {
  id: string
  name: string
  type: string
  status: SeatStatus
  col?: number
  row?: number
  x?: number
  y?: number
  width: number
  height: number
  shape?: string
  vertical?: boolean
  isStage?: boolean
  isBar?: boolean
  transform?: string
}

// 座位数据 - 转换JSON配置为正确类型，支持网格系统
const gridSize = seatConfig.gridSize || 50

// 添加调试日志
console.log('原始座位数据:', seatConfig.seats)

const seats = ref<Seat[]>(seatConfig.seats.map((seat: any) => {
  // 如果使用网格系统(col,row)，则转换为像素坐标
  let x = 0, y = 0
  if (seat.col !== undefined && seat.row !== undefined) {
    x = seat.col * gridSize
    y = seat.row * gridSize
  } else if (seat.x !== undefined && seat.y !== undefined) {
    x = seat.x
    y = seat.y
  } else {
    console.warn('座位缺少位置信息:', seat)
    x = 0
    y = 0
  }

  // 确保必要的属性存在
  const processedSeat = {
    id: seat.id || `seat_${Date.now()}_${Math.random()}`,
    name: seat.name || seat.id || '座位',
    type: seat.type || 'lounge',
    status: (seat.status as SeatStatus) || 'available',
    x: x,
    y: y,
    width: seat.width || 60,
    height: seat.height || 60,
    shape: seat.shape,
    vertical: seat.vertical,
    isStage: seat.isStage,
    isBar: seat.isBar,
    transform: seat.transform,
    col: seat.col,
    row: seat.row
  }

  return processedSeat
}))

// 添加调试日志
console.log('处理后的座位数据:', seats.value.map(s => ({id: s.id, name: s.name, x: s.x, y: s.y, width: s.width, height: s.height})))

// 选中的座位
const selectedSeats = ref<Seat[]>([])
const tomorrow = new Date(Date.now() + 86400000)
const bookingDate = ref(tomorrow.toISOString().slice(0, 10))
const sessions = ['20:00', '22:30', '00:30']
const session = ref(sessions[0])
const guestName = ref('')
const phone = ref('')
const onDateChange = (event: any) => bookingDate.value = event.detail.value

// 获取座位样式
const getSeatStyle = (seat: Seat) => {
  // movable-view 使用 width 和 height 控制尺寸
  return {
    width: `${seat.width}rpx`,
    height: `${seat.height}rpx`
  }
}

// 处理座位选择
const handleSeatSelect = (seat: Seat) => {
  // 如果座位已被占用，则不能选择
  if (seat.status === 'occupied') return

  // 如果座位已选中，则取消选择
  if (seat.status === 'selected') {
    seat.status = 'available'
    selectedSeats.value = selectedSeats.value.filter(s => s.id !== seat.id)
  } else {
    // 选择座位
    seat.status = 'selected'
    selectedSeats.value.push(seat)
  }
}

// 确认定座
const confirmBooking = () => {
  if (selectedSeats.value.length === 0) return
  if (!guestName.value.trim() || !/^1\d{10}$/.test(phone.value)) {
    uni.showToast({ title: '请填写姓名和正确手机号', icon: 'none' })
    return
  }
  const order = createOrder({ seats:selectedSeats.value.map(s=>s.name), date:bookingDate.value, session:session.value, guestName:guestName.value.trim(), phone:phone.value })
  uni.showModal({ title:'预订成功', content:`订单 ${order.id} 已生成`, showCancel:false, success:()=>uni.redirectTo({url:'/pages/orders/index'}) })
}

// 页面加载完成后计算实际尺寸
onMounted(() => {
  // 可以在这里获取容器实际尺寸并调整座位位置
  console.log('座位页面已加载，座位数量:', seats.value.length)
})
</script>

<style scoped>
.booking-page {
  padding: 20rpx;
  background-color: #000;
  color: #fff;
  min-height: 100vh;
}

.header {
  text-align: center;
  margin-bottom: 30rpx;
}
.booking-info{background:#111116;border:1rpx solid #302f2b;margin-bottom:24rpx;padding:0 26rpx}.field{min-height:92rpx;display:flex;align-items:center;justify-content:space-between;border-bottom:1rpx solid #26252a;font-size:25rpx}.field-label{color:#777}.sessions{display:flex;gap:10rpx}.session{padding:10rpx 15rpx;border:1rpx solid #37353b;color:#888}.session.active{background:#c7b276;color:#08080c;border-color:#c7b276}.guest-fields{display:grid;grid-template-columns:1fr 1fr;gap:18rpx;padding:20rpx 0}.input{height:72rpx;border:1rpx solid #343239;padding:0 18rpx;color:#fff;font-size:24rpx}.placeholder{color:#555}

.title {
  font-size: 48rpx;
  font-weight: bold;
  display: block;
}

.subtitle {
  font-size: 28rpx;
  color: #aaa;
  display: block;
  margin-top: 10rpx;
}

.venue-map-container {
  height: 1200rpx;
  border: 2rpx solid #333;
  border-radius: 20rpx;
  margin-bottom: 30rpx;
  overflow: hidden;
}

.venue-map {
  width: 100%;
  height: 100%;
  background: linear-gradient(180deg, #1a1a2e 0%, #16213e 100%);
}

.movable-seat {
  position: absolute;
  display: flex;
  align-items: center;
  justify-content: center;
}

.seat {
  display: flex;
  align-items: center;
  justify-content: center;
  border-radius: 15rpx;
  font-weight: bold;
  cursor: pointer;
  transition: all 0.2s ease;
  box-sizing: border-box;
  width: 100%;
  height: 100%;
}

.seat-name {
  font-size: 24rpx;
  color: white;
  text-align: center;
  font-weight: bold;
}

/* 座位状态样式 */
.seat.available {
  background-color: #9b59b6; /* 浅紫色 */
  border: 2rpx solid #8e44ad;
}

.seat.occupied {
  background-color: #7f8c8d; /* 深灰色 */
  border: 2rpx solid #6c7a89;
  opacity: 0.7;
  cursor: not-allowed;
}

.seat.selected {
  background-color: #f39c12; /* 亮橙色 */
  border: 2rpx solid #e67e22;
  transform: scale(1.1);
  box-shadow: 0 0 15rpx rgba(243, 156, 18, 0.7);
  z-index: 10;
}

/* 圆形座位 (VIP区) */
.seat.circle {
  border-radius: 50%;
}

/* 舞台区域 */
.seat.stage {
  background: linear-gradient(45deg, #ff416c, #ff4b2b);
  border: none;
  border-radius: 10rpx;
}

.seat.bar {
  background: linear-gradient(45deg, #00c9ff, #92fe9d);
  border: none;
  border-radius: 10rpx;
}

.stage-text, .bar-text {
  font-weight: bold;
  font-size: 24rpx;
}

.footer {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 20rpx;
  background-color: #1a1a1a;
  border-radius: 15rpx;
}

.status-indicators {
  display: flex;
  gap: 30rpx;
}

.indicator {
  display: flex;
  align-items: center;
  gap: 10rpx;
}

.color-box {
  width: 30rpx;
  height: 30rpx;
  border-radius: 6rpx;
}

.available {
  background-color: #9b59b6; /* 浅紫色 */
}

.occupied {
  background-color: #7f8c8d; /* 深灰色 */
}

.selected {
  background-color: #f39c12; /* 亮橙色 */
}

.confirm-btn {
  background: linear-gradient(45deg, #ff416c, #ff4b2b);
  color: white;
  border: none;
  padding: 20rpx 40rpx;
  border-radius: 50rpx;
  font-weight: bold;
}

.confirm-btn:disabled {
  background: #333;
  color: #666;
}
</style>
