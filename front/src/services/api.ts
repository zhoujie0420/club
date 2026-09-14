export interface Product {
  productId: string;
  name: string;
  price: number;
  qty: number;
  active?: boolean;
}
export interface Order {
  id: string;
  tableId: string;
  customerName: string;
  phone: string;
  guestCount: number;
  date: string;
  arrivalTime: string;
  durationMinutes: number;
  status: string;
  items: Product[];
  paidAmount: number;
  refundedAmount?: number;
  refundReason?: string;
  refundedAt?: string;
  refunds?: { amount: number; reason: string; at: string; operator: string }[];
  paymentMethod?: string;
  salesName: string;
  createdBy?: string;
  remark?: string;
  cancelReason?: string;
  version: number;
  events: { action: string; at: string; operator?: string }[];
}
export interface Staff {
  id: string;
  name: string;
  role: string;
  roleName: string;
  permissions: string[];
}
export interface Table {
  id: string;
  zone: string;
  capacity: number;
}
export interface State {
  orders: Order[];
  tables: Table[];
  products: Product[];
  businessDate: string;
  updatedAt: string;
  currentUser: Staff;
}
export interface AuditLog {
  id: string;
  orderId: string;
  userId: string;
  userName: string;
  roleName: string;
  action: string;
  createdAt: string;
}
export interface StaffAccount extends Staff {
  active: boolean;
  createdAt: string;
}
export interface ReportDay {
  date: string;
  reservations: number;
  completed: number;
  cancelled: number;
  revenue: number;
  guests: number;
  refunded: number;
  refundAmount: number;
}
export interface DailyReport {
  from: string;
  to: string;
  days: ReportDay[];
  totals: ReportDay;
  scope: "own" | "store";
}
const tokenKey = "club-test-session";
export class ApiError extends Error {
  constructor(
    message: string,
    public status = 0,
  ) {
    super(message);
  }
}
export const session = () => uni.getStorageSync(tokenKey) as string;
export async function logout() {
  try {
    if (session()) await request<{ ok: boolean }>("/logout", "POST", {});
  } finally {
    uni.removeStorageSync(tokenKey);
  }
}
export async function request<T>(
  path: string,
  method: "GET" | "POST" | "PUT" = "GET",
  data?: unknown,
  key?: string,
): Promise<T> {
  return new Promise((resolve, reject) =>
    uni.request({
      url: `/api/v1${path}`,
      method,
      data: data as any,
      timeout: 12000,
      header: {
        "Content-Type": "application/json",
        Authorization: `Bearer ${session()}`,
        ...(key ? { "Idempotency-Key": key } : {}),
      },
      success: (r) => {
        if (r.statusCode >= 200 && r.statusCode < 300) resolve(r.data as T);
        else {
          if (r.statusCode === 401) logout();
          reject(
            new ApiError(
              (r.data as any)?.error || "服务暂时不可用",
              r.statusCode,
            ),
          );
        }
      },
      fail: () =>
        reject(new ApiError("网络连接失败，结果可能尚未确认；请重试同一操作")),
    }),
  );
}
export async function login(code: string, userId: string) {
  const r = await request<{ token: string }>("/login", "POST", {
    code,
    userId,
  });
  uni.setStorageSync(tokenKey, r.token);
}
export const requestKey = () =>
  `${Date.now()}-${Math.random().toString(36).slice(2)}-${Math.random().toString(36).slice(2)}`;
export const total = (o: Order) =>
  o.items.reduce((s, p) => s + p.price * p.qty, 0);
