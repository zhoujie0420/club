export type Role='sales'|'waiter'|'frontdesk'|'owner'
export type OrderStatus='reserved'|'arrived'|'serving'|'cleaning'|'completed'|'cancelled'
export interface OrderItem{name:string;qty:number;price:number}
export interface ClubOrder{id:string;tableId:string;customerName:string;phone:string;wechat?:string;salesName:string;date:string;arrivalTime:string;guestCount:number;status:OrderStatus;items:OrderItem[];paymentMethod?:'微信'|'支付宝'|'现金';createdAt:string}
const KEY='hole-club-v2-orders'
const seed:ClubOrder[]=[
 {id:'HC-DEMO-01',tableId:'V08',customerName:'演示客户 A',phone:'DEMO',salesName:'销售 A',date:'2026-09-18',arrivalTime:'22:30',guestCount:8,status:'reserved',items:[{name:'经典畅饮套餐',qty:1,price:2880}],createdAt:'2026-09-11T10:00:00Z'},
 {id:'HC-DEMO-02',tableId:'A06',customerName:'演示客户 B',phone:'DEMO',salesName:'销售 A',date:'2026-09-18',arrivalTime:'21:00',guestCount:6,status:'arrived',items:[{name:'尊享香槟套餐',qty:1,price:3880}],createdAt:'2026-09-11T10:20:00Z'},
 {id:'HC-DEMO-03',tableId:'B03',customerName:'演示客户 C',phone:'DEMO',salesName:'销售 B',date:'2026-09-18',arrivalTime:'20:30',guestCount:5,status:'serving',items:[{name:'精酿啤酒',qty:12,price:48}],createdAt:'2026-09-11T10:40:00Z'}
]
export const getOrders=():ClubOrder[]=>{try{const raw=localStorage.getItem(KEY);if(raw)return JSON.parse(raw);localStorage.setItem(KEY,JSON.stringify(seed));return seed}catch{return seed}}
export const saveOrders=(orders:ClubOrder[])=>localStorage.setItem(KEY,JSON.stringify(orders))
export const createOrder=(data:Omit<ClubOrder,'id'|'createdAt'>)=>{const order:ClubOrder={...data,id:`HC${Date.now().toString().slice(-8)}`,createdAt:new Date().toISOString()};saveOrders([order,...getOrders()]);return order}
export const updateStatus=(id:string,status:OrderStatus)=>saveOrders(getOrders().map(o=>o.id===id?{...o,status}:o))
export const addItem=(id:string,item:OrderItem)=>saveOrders(getOrders().map(o=>o.id===id?{...o,items:[...o.items,item]}:o))
export const total=(o:ClubOrder)=>o.items.reduce((sum,item)=>sum+item.price*item.qty,0)
export const currentRole=():Role=>(localStorage.getItem('hole-role') as Role)||'sales'
export const setRole=(role:Role)=>localStorage.setItem('hole-role',role)
export const roleName:Record<Role,string>={sales:'销售',waiter:'服务员',frontdesk:'前台',owner:'老板'}
