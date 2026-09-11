export type OrderStatus = 'reserved' | 'cancelled'
export interface ClubOrder { id:string; seats:string[]; date:string; session:string; guestName:string; phone:string; status:OrderStatus; createdAt:string }
const STORAGE_KEY = 'hole-club-orders'
export const getOrders = (): ClubOrder[] => {
  try { return JSON.parse(localStorage.getItem(STORAGE_KEY) || '[]') } catch { return [] }
}
export const createOrder = (order: Omit<ClubOrder,'id'|'status'|'createdAt'>) => {
  const next: ClubOrder = {...order,id:`HC${Date.now().toString().slice(-8)}`,status:'reserved',createdAt:new Date().toISOString()}
  localStorage.setItem(STORAGE_KEY, JSON.stringify([next,...getOrders()]))
  return next
}
export const cancelOrder = (id:string) => {
  localStorage.setItem(STORAGE_KEY, JSON.stringify(getOrders().map(o=>o.id===id?{...o,status:'cancelled' as const}:o)))
}
