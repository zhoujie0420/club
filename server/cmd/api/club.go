package main

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	_ "modernc.org/sqlite"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"golang.org/x/crypto/bcrypt"
)

type clubStore struct {
	db       *sql.DB
	code     string
	rev      atomic.Int64
	mu       sync.Mutex
	attempts map[string][]time.Time
}
type item struct {
	ProductID string `json:"productId"`
	Name      string `json:"name"`
	Qty       int    `json:"qty"`
	Price     int    `json:"price"`
	Active    bool   `json:"active,omitempty"`
}
type event struct {
	Action   string `json:"action"`
	At       string `json:"at"`
	Operator string `json:"operator,omitempty"`
}
type refund struct {
	Amount   int    `json:"amount"`
	Reason   string `json:"reason"`
	At       string `json:"at"`
	Operator string `json:"operator"`
}
type staff struct {
	ID          string   `json:"id"`
	Name        string   `json:"name"`
	Role        string   `json:"role"`
	RoleName    string   `json:"roleName"`
	Permissions []string `json:"permissions"`
}
type staffContextKey struct{}

type auditLog struct {
	ID        string `json:"id"`
	OrderID   string `json:"orderId"`
	UserID    string `json:"userId"`
	UserName  string `json:"userName"`
	RoleName  string `json:"roleName"`
	Action    string `json:"action"`
	CreatedAt string `json:"createdAt"`
}

var defaultStaffProfiles = map[string]staff{
	"sales":     {"sales", "销售小林", "sales", "销售", []string{"order:create", "order:cancel-own", "report:own"}},
	"frontdesk": {"frontdesk", "前台小周", "frontdesk", "前台", []string{"order:create", "order:confirm-arrival", "order:change-table", "order:cancel"}},
	"waiter":    {"waiter", "服务员阿杰", "waiter", "服务员", []string{"order:confirm-arrival", "table:open", "order:add-item", "payment:checkout", "table:clean"}},
	"owner":     {"owner", "店长", "owner", "店长", []string{"*"}},
}

var rolePermissions = map[string]struct {
	Name        string
	Permissions []string
}{
	"sales":     {"销售", []string{"order:create", "order:update-own", "order:cancel-own", "report:own"}},
	"frontdesk": {"前台", []string{"order:create", "order:update", "order:confirm-arrival", "order:change-table", "order:cancel"}},
	"waiter":    {"服务员", []string{"order:confirm-arrival", "table:open", "order:add-item", "payment:checkout", "table:clean"}},
	"owner":     {"店长", []string{"*"}},
}

type staffAccount struct {
	staff
	Active    bool   `json:"active"`
	CreatedAt string `json:"createdAt"`
}

type reportDay struct {
	Date         string `json:"date"`
	Reservations int    `json:"reservations"`
	Completed    int    `json:"completed"`
	Cancelled    int    `json:"cancelled"`
	Revenue      int    `json:"revenue"`
	Guests       int    `json:"guests"`
	Refunded     int    `json:"refunded"`
	RefundAmount int    `json:"refundAmount"`
}

type order struct {
	ID              string   `json:"id"`
	TableID         string   `json:"tableId"`
	CustomerName    string   `json:"customerName"`
	Phone           string   `json:"phone"`
	Wechat          string   `json:"wechat"`
	Remark          string   `json:"remark,omitempty"`
	SalesName       string   `json:"salesName"`
	CreatedBy       string   `json:"createdBy,omitempty"`
	Date            string   `json:"date"`
	ArrivalTime     string   `json:"arrivalTime"`
	DurationMinutes int      `json:"durationMinutes"`
	GuestCount      int      `json:"guestCount"`
	Status          string   `json:"status"`
	Items           []item   `json:"items"`
	PaymentMethod   string   `json:"paymentMethod,omitempty"`
	CancelReason    string   `json:"cancelReason,omitempty"`
	PaidAmount      int      `json:"paidAmount"`
	RefundedAmount  int      `json:"refundedAmount,omitempty"`
	RefundReason    string   `json:"refundReason,omitempty"`
	RefundedAt      string   `json:"refundedAt,omitempty"`
	Refunds         []refund `json:"refunds,omitempty"`
	CreatedAt       string   `json:"createdAt"`
	Events          []event  `json:"events"`
	Version         int      `json:"version"`
}
type table struct {
	ID       string `json:"id"`
	Zone     string `json:"zone"`
	Capacity int    `json:"capacity"`
}

var tables = func() []table {
	out := []table{}
	for _, prefix := range []string{"V", "A", "B"} {
		for _, n := range []int{1, 2, 3, 5, 6, 8} {
			capacity := 6
			zone := "卡座"
			if prefix == "V" {
				capacity = 10
				zone = "VIP"
			}
			out = append(out, table{fmt.Sprintf("%s%02d", prefix, n), zone, capacity})
		}
	}
	return out
}()
var defaultProducts = []item{{"classic", "经典畅饮套餐", 1, 2880, true}, {"champagne", "尊享香槟套餐", 1, 3880, true}, {"beer", "精酿啤酒", 1, 48, true}, {"soda", "苏打水", 1, 20, true}, {"fruit", "时令果盘", 1, 128, true}}

func newClubStore(path, code string) (*clubStore, error) {
	db, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, err
	}
	db.SetMaxOpenConns(4)
	db.SetMaxIdleConns(2)
	_, err = db.Exec(`PRAGMA journal_mode=WAL; PRAGMA busy_timeout=5000; PRAGMA synchronous=NORMAL; PRAGMA temp_store=MEMORY; PRAGMA cache_size=-8000; PRAGMA mmap_size=67108864;
 CREATE TABLE IF NOT EXISTS orders(id TEXT PRIMARY KEY, table_id TEXT NOT NULL, date TEXT NOT NULL, status TEXT NOT NULL, body TEXT NOT NULL, start_minute INTEGER NOT NULL DEFAULT 0, end_minute INTEGER NOT NULL DEFAULT 1440, created_by TEXT NOT NULL DEFAULT '', customer_name TEXT NOT NULL DEFAULT '', phone TEXT NOT NULL DEFAULT '');
 CREATE TABLE IF NOT EXISTS requests(key TEXT PRIMARY KEY, fingerprint TEXT NOT NULL, body TEXT NOT NULL);
 CREATE TABLE IF NOT EXISTS sessions(token TEXT PRIMARY KEY, expires INTEGER NOT NULL, user_id TEXT NOT NULL DEFAULT 'owner');
 CREATE TABLE IF NOT EXISTS audit_logs(id TEXT PRIMARY KEY, order_id TEXT NOT NULL, user_id TEXT NOT NULL, action TEXT NOT NULL, created_at TEXT NOT NULL);
 CREATE TABLE IF NOT EXISTS staff_accounts(id TEXT PRIMARY KEY, name TEXT NOT NULL, role TEXT NOT NULL, pin_hash TEXT NOT NULL, active INTEGER NOT NULL DEFAULT 1, created_at TEXT NOT NULL);
 CREATE TABLE IF NOT EXISTS meta(key TEXT PRIMARY KEY, value TEXT NOT NULL);`)
	if err != nil {
		db.Close()
		return nil, err
	}
	// Upgrade databases created by the first shared-account test release.
	if _, alterErr := db.Exec("ALTER TABLE sessions ADD COLUMN user_id TEXT NOT NULL DEFAULT 'owner'"); alterErr != nil && !strings.Contains(alterErr.Error(), "duplicate column") {
		db.Close()
		return nil, alterErr
	}
	for _, statement := range []string{
		"ALTER TABLE orders ADD COLUMN start_minute INTEGER NOT NULL DEFAULT 0",
		"ALTER TABLE orders ADD COLUMN end_minute INTEGER NOT NULL DEFAULT 1440",
		"ALTER TABLE orders ADD COLUMN created_by TEXT NOT NULL DEFAULT ''",
		"ALTER TABLE orders ADD COLUMN customer_name TEXT NOT NULL DEFAULT ''",
		"ALTER TABLE orders ADD COLUMN phone TEXT NOT NULL DEFAULT ''",
	} {
		if _, alterErr := db.Exec(statement); alterErr != nil && !strings.Contains(alterErr.Error(), "duplicate column") {
			db.Close()
			return nil, alterErr
		}
	}
	if _, err = db.Exec("DROP INDEX IF EXISTS occupied_table; CREATE INDEX IF NOT EXISTS table_schedule ON orders(table_id,date,start_minute,end_minute,status); CREATE INDEX IF NOT EXISTS order_history ON orders(status,date,created_by); CREATE INDEX IF NOT EXISTS order_customer ON orders(customer_name,phone); CREATE INDEX IF NOT EXISTS orders_open ON orders(date,table_id) WHERE status NOT IN ('completed','cancelled'); CREATE INDEX IF NOT EXISTS sessions_expires ON sessions(expires)"); err != nil {
		db.Close()
		return nil, err
	}
	rows, rowsErr := db.Query("SELECT id,body FROM orders WHERE start_minute=0 AND end_minute=1440")
	if rowsErr != nil {
		db.Close()
		return nil, rowsErr
	}
	type scheduleMigration struct {
		id         string
		start, end int
	}
	migrations := []scheduleMigration{}
	for rows.Next() {
		var id, raw string
		var existing order
		if rows.Scan(&id, &raw) == nil && json.Unmarshal([]byte(raw), &existing) == nil {
			start, end, scheduleErr := orderSchedule(existing.ArrivalTime, existing.DurationMinutes)
			if scheduleErr == nil {
				migrations = append(migrations, scheduleMigration{id, start, end})
			}
		}
	}
	rows.Close()
	for _, migration := range migrations {
		if _, err = db.Exec("UPDATE orders SET start_minute=?,end_minute=? WHERE id=?", migration.start, migration.end, migration.id); err != nil {
			db.Close()
			return nil, err
		}
	}
	rows, rowsErr = db.Query("SELECT id,body FROM orders WHERE created_by='' OR customer_name='' OR phone=''")
	if rowsErr != nil {
		db.Close()
		return nil, rowsErr
	}
	type searchMigration struct{ id, createdBy, customerName, phone string }
	searchMigrations := []searchMigration{}
	for rows.Next() {
		var id, raw string
		var existing order
		if rows.Scan(&id, &raw) == nil && json.Unmarshal([]byte(raw), &existing) == nil {
			searchMigrations = append(searchMigrations, searchMigration{id, existing.CreatedBy, existing.CustomerName, existing.Phone})
		}
	}
	rows.Close()
	for _, migration := range searchMigrations {
		if _, err = db.Exec("UPDATE orders SET created_by=?,customer_name=?,phone=? WHERE id=?", migration.createdBy, migration.customerName, migration.phone, migration.id); err != nil {
			db.Close()
			return nil, err
		}
	}
	count := 0
	if err := db.QueryRow("SELECT COUNT(*) FROM staff_accounts").Scan(&count); err != nil {
		db.Close()
		return nil, err
	}
	if count == 0 {
		if code == "" {
			db.Close()
			return nil, errors.New("CLUB_ACCESS_CODE is required to initialize staff accounts")
		}
		hash, hashErr := bcrypt.GenerateFromPassword([]byte(code), bcrypt.DefaultCost)
		if hashErr != nil {
			db.Close()
			return nil, hashErr
		}
		tx, txErr := db.Begin()
		if txErr != nil {
			db.Close()
			return nil, txErr
		}
		for _, profile := range defaultStaffProfiles {
			if _, txErr = tx.Exec("INSERT INTO staff_accounts(id,name,role,pin_hash,active,created_at) VALUES (?,?,?,?,1,?)", profile.ID, profile.Name, profile.Role, string(hash), time.Now().Format(time.RFC3339)); txErr != nil {
				tx.Rollback()
				db.Close()
				return nil, txErr
			}
		}
		if txErr = tx.Commit(); txErr != nil {
			db.Close()
			return nil, txErr
		}
	}
	if _, err = db.Exec(`CREATE TABLE IF NOT EXISTS products(id TEXT PRIMARY KEY, name TEXT NOT NULL, price INTEGER NOT NULL, active INTEGER NOT NULL DEFAULT 1, sort_order INTEGER NOT NULL DEFAULT 0, created_at TEXT NOT NULL)`); err != nil {
		db.Close()
		return nil, err
	}
	for index, product := range defaultProducts {
		if _, err = db.Exec("INSERT OR IGNORE INTO products(id,name,price,active,sort_order,created_at) VALUES (?,?,?,1,?,?)", product.ProductID, product.Name, product.Price, index, time.Now().Format(time.RFC3339)); err != nil {
			db.Close()
			return nil, err
		}
	}
	store := &clubStore{db: db, code: code, attempts: map[string][]time.Time{}}
	var revText string
	if db.QueryRow("SELECT value FROM meta WHERE key='rev'").Scan(&revText) == nil {
		if n, parseErr := strconv.ParseInt(revText, 10, 64); parseErr == nil && n > 0 {
			store.rev.Store(n)
		}
	}
	if store.rev.Load() == 0 {
		store.rev.Store(1)
	}
	return store, nil
}

func (s *clubStore) revision() int64 {
	return s.rev.Load()
}

func (s *clubStore) bump(tx *sql.Tx) {
	n := s.rev.Add(1)
	value := strconv.FormatInt(n, 10)
	const q = "INSERT INTO meta(key,value) VALUES ('rev',?) ON CONFLICT(key) DO UPDATE SET value=excluded.value"
	if tx != nil {
		_, _ = tx.Exec(q, value)
		return
	}
	_, _ = s.db.Exec(q, value)
}

func (s *clubStore) staffByID(id string, requireActive bool) (staff, error) {
	var profile staff
	var active int
	if err := s.db.QueryRow("SELECT id,name,role,active FROM staff_accounts WHERE id=?", id).Scan(&profile.ID, &profile.Name, &profile.Role, &active); err != nil {
		return profile, err
	}
	role, ok := rolePermissions[profile.Role]
	if !ok || (requireActive && active != 1) {
		return profile, sql.ErrNoRows
	}
	profile.RoleName, profile.Permissions = role.Name, role.Permissions
	return profile, nil
}
func randomID() string {
	var b [24]byte
	if _, err := rand.Read(b[:]); err != nil {
		panic(err)
	}
	return hex.EncodeToString(b[:])
}
func orderSchedule(arrival string, duration int) (int, int, error) {
	parsed, err := time.Parse("15:04", arrival)
	if err != nil {
		return 0, 0, errors.New("到店时间不正确")
	}
	if duration == 0 {
		duration = 240
	}
	if duration < 60 || duration > 720 || duration%30 != 0 {
		return 0, 0, errors.New("预计使用时长需为 1–12 小时，并按 30 分钟递增")
	}
	start := parsed.Hour()*60 + parsed.Minute()
	return start, start + duration, nil
}
func fail(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, response{"error": message})
}
func decode(w http.ResponseWriter, r *http.Request, v any) bool {
	r.Body = http.MaxBytesReader(w, r.Body, 32768)
	d := json.NewDecoder(r.Body)
	if d.Decode(v) != nil {
		fail(w, 400, "请求格式不正确")
		return false
	}
	if d.Decode(new(any)) != io.EOF {
		fail(w, 400, "请求包含多余内容")
		return false
	}
	return true
}
func (app *application) secure(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Cache-Control", "no-store")
		token := strings.TrimPrefix(r.Header.Get("Authorization"), "Bearer ")
		var expires int64
		var userID string
		if app.store.db.QueryRow("SELECT expires,user_id FROM sessions WHERE token=?", token).Scan(&expires, &userID) != nil || expires < time.Now().Unix() {
			fail(w, 401, "请先输入测试访问口令")
			return
		}
		profile, err := app.store.staffByID(userID, true)
		if err != nil {
			fail(w, 401, "员工身份已失效，请重新登录")
			return
		}
		next(w, r.WithContext(context.WithValue(r.Context(), staffContextKey{}, profile)))
	}
}
func (app *application) registerClub(mux *http.ServeMux) {
	mux.HandleFunc("POST /api/v1/login", app.login)
	mux.HandleFunc("GET /api/v1/login-options", app.loginOptions)
	mux.HandleFunc("POST /api/v1/logout", app.secure(app.logout))
	mux.HandleFunc("GET /api/v1/state", app.secure(app.state))
	mux.HandleFunc("GET /api/v1/audit-logs", app.secure(app.auditLogs))
	mux.HandleFunc("GET /api/v1/staff", app.secure(app.listStaff))
	mux.HandleFunc("POST /api/v1/staff", app.secure(app.createStaff))
	mux.HandleFunc("PUT /api/v1/staff/{id}", app.secure(app.updateStaff))
	mux.HandleFunc("GET /api/v1/reports/daily", app.secure(app.dailyReport))
	mux.HandleFunc("GET /api/v1/products", app.secure(app.listProducts))
	mux.HandleFunc("POST /api/v1/products", app.secure(app.createProduct))
	mux.HandleFunc("PUT /api/v1/products/{id}", app.secure(app.updateProduct))
	mux.HandleFunc("POST /api/v1/orders", app.secure(app.createBooking))
	mux.HandleFunc("GET /api/v1/orders", app.secure(app.listOrders))
	mux.HandleFunc("POST /api/v1/orders/{id}/{action}", app.secure(app.mutateOrder))
}

func (app *application) dailyReport(w http.ResponseWriter, r *http.Request) {
	u := currentStaff(r)
	if !permitted(u, "report:view") && !permitted(u, "report:own") {
		fail(w, http.StatusForbidden, "当前角色没有查看营业报表的权限")
		return
	}
	to := r.URL.Query().Get("to")
	from := r.URL.Query().Get("from")
	if to == "" {
		to = businessDate()
	}
	toDate, err := time.Parse("2006-01-02", to)
	if err != nil {
		fail(w, 400, "报表结束日期不正确")
		return
	}
	if from == "" {
		from = toDate.AddDate(0, 0, -6).Format("2006-01-02")
	}
	fromDate, err := time.Parse("2006-01-02", from)
	if err != nil || fromDate.After(toDate) || toDate.Sub(fromDate) > 89*24*time.Hour {
		fail(w, 400, "报表日期范围需为 1–90 天")
		return
	}
	rows, err := app.store.db.QueryContext(r.Context(), "SELECT body FROM orders WHERE date BETWEEN ? AND ? ORDER BY date", from, to)
	if err != nil {
		fail(w, 500, "读取报表失败")
		return
	}
	defer rows.Close()
	byDate := map[string]*reportDay{}
	for day := fromDate; !day.After(toDate); day = day.AddDate(0, 0, 1) {
		date := day.Format("2006-01-02")
		byDate[date] = &reportDay{Date: date}
	}
	for rows.Next() {
		var raw string
		var o order
		if rows.Scan(&raw) != nil || json.Unmarshal([]byte(raw), &o) != nil {
			fail(w, 500, "报表订单数据异常")
			return
		}
		if o.DurationMinutes == 0 {
			o.DurationMinutes = 240
		}
		if permitted(u, "report:own") && !permitted(u, "report:view") && o.CreatedBy != u.ID {
			continue
		}
		day := byDate[o.Date]
		if day == nil {
			continue
		}
		day.Reservations++
		if (o.Status == "completed" || o.Status == "cleaning") && o.RefundedAmount < o.PaidAmount {
			day.Completed++
			day.Guests += o.GuestCount
		}
		day.Revenue += o.PaidAmount - o.RefundedAmount
		if o.RefundedAmount > 0 {
			day.Refunded++
			day.RefundAmount += o.RefundedAmount
		}
		if o.Status == "cancelled" {
			day.Cancelled++
		}
	}
	days := make([]reportDay, 0, len(byDate))
	totals := reportDay{Date: "total"}
	for day := fromDate; !day.After(toDate); day = day.AddDate(0, 0, 1) {
		value := *byDate[day.Format("2006-01-02")]
		days = append(days, value)
		totals.Reservations += value.Reservations
		totals.Completed += value.Completed
		totals.Cancelled += value.Cancelled
		totals.Revenue += value.Revenue
		totals.Guests += value.Guests
		totals.Refunded += value.Refunded
		totals.RefundAmount += value.RefundAmount
	}
	writeJSON(w, 200, response{"from": from, "to": to, "days": days, "totals": totals, "scope": map[bool]string{true: "own", false: "store"}[u.Role == "sales"]})
}

func (app *application) readProducts(ctx context.Context, activeOnly bool) ([]item, error) {
	query := "SELECT id,name,price,active FROM products"
	if activeOnly {
		query += " WHERE active=1"
	}
	query += " ORDER BY sort_order,rowid"
	rows, err := app.store.db.QueryContext(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	products := []item{}
	for rows.Next() {
		var product item
		var active int
		if err := rows.Scan(&product.ProductID, &product.Name, &product.Price, &active); err != nil {
			return nil, err
		}
		product.Qty, product.Active = 1, active == 1
		products = append(products, product)
	}
	return products, rows.Err()
}

func validateProduct(name string, price int) error {
	if strings.TrimSpace(name) == "" || len([]rune(strings.TrimSpace(name))) > 40 {
		return errors.New("商品名称需为 1–40 个字符")
	}
	if price < 1 || price > 1000000 {
		return errors.New("商品价格需为 1–1,000,000 元")
	}
	return nil
}

func (app *application) listProducts(w http.ResponseWriter, r *http.Request) {
	if !requireOwner(w, r) {
		return
	}
	products, err := app.readProducts(r.Context(), false)
	if err != nil {
		fail(w, 500, "读取商品失败")
		return
	}
	writeJSON(w, 200, response{"products": products})
}

func (app *application) createProduct(w http.ResponseWriter, r *http.Request) {
	if !requireOwner(w, r) {
		return
	}
	var req struct {
		Name  string `json:"name"`
		Price int    `json:"price"`
	}
	if !decode(w, r, &req) {
		return
	}
	req.Name = strings.TrimSpace(req.Name)
	if err := validateProduct(req.Name, req.Price); err != nil {
		fail(w, 400, err.Error())
		return
	}
	var sortOrder int
	app.store.db.QueryRowContext(r.Context(), "SELECT COALESCE(MAX(sort_order),-1)+1 FROM products").Scan(&sortOrder)
	product := item{ProductID: "product-" + randomID()[:12], Name: req.Name, Qty: 1, Price: req.Price, Active: true}
	if _, err := app.store.db.ExecContext(r.Context(), "INSERT INTO products(id,name,price,active,sort_order,created_at) VALUES (?,?,?,1,?,?)", product.ProductID, product.Name, product.Price, sortOrder, time.Now().Format(time.RFC3339)); err != nil {
		fail(w, 500, "创建商品失败")
		return
	}
	app.store.bump(nil)
	writeJSON(w, http.StatusCreated, response{"product": product})
}

func (app *application) updateProduct(w http.ResponseWriter, r *http.Request) {
	if !requireOwner(w, r) {
		return
	}
	var req struct {
		Name   *string `json:"name"`
		Price  *int    `json:"price"`
		Active *bool   `json:"active"`
	}
	if !decode(w, r, &req) {
		return
	}
	var product item
	var active int
	if err := app.store.db.QueryRowContext(r.Context(), "SELECT id,name,price,active FROM products WHERE id=?", r.PathValue("id")).Scan(&product.ProductID, &product.Name, &product.Price, &active); err != nil {
		fail(w, 404, "商品不存在")
		return
	}
	if req.Name != nil {
		product.Name = strings.TrimSpace(*req.Name)
	}
	if req.Price != nil {
		product.Price = *req.Price
	}
	if req.Active != nil {
		product.Active = *req.Active
	} else {
		product.Active = active == 1
	}
	if err := validateProduct(product.Name, product.Price); err != nil {
		fail(w, 400, err.Error())
		return
	}
	if _, err := app.store.db.ExecContext(r.Context(), "UPDATE products SET name=?,price=?,active=? WHERE id=?", product.Name, product.Price, map[bool]int{true: 1, false: 0}[product.Active], product.ProductID); err != nil {
		fail(w, 500, "更新商品失败")
		return
	}
	app.store.bump(nil)
	product.Qty = 1
	writeJSON(w, 200, response{"product": product})
}
func (app *application) loginOptions(w http.ResponseWriter, r *http.Request) {
	rows, err := app.store.db.QueryContext(r.Context(), "SELECT id,name,role FROM staff_accounts WHERE active=1 ORDER BY rowid")
	if err != nil {
		fail(w, 500, "读取员工账号失败")
		return
	}
	defer rows.Close()
	options := []staff{}
	for rows.Next() {
		var option staff
		if err := rows.Scan(&option.ID, &option.Name, &option.Role); err != nil {
			fail(w, 500, "员工账号数据异常")
			return
		}
		if definition, ok := rolePermissions[option.Role]; ok {
			option.RoleName = definition.Name
			option.Permissions = nil
			options = append(options, option)
		}
	}
	writeJSON(w, 200, response{"staff": options})
}
func (app *application) logout(w http.ResponseWriter, r *http.Request) {
	token := strings.TrimPrefix(r.Header.Get("Authorization"), "Bearer ")
	if _, err := app.store.db.ExecContext(r.Context(), "DELETE FROM sessions WHERE token=?", token); err != nil {
		fail(w, 500, "退出登录失败")
		return
	}
	writeJSON(w, 200, response{"ok": true})
}
func (app *application) login(w http.ResponseWriter, r *http.Request) {
	s := app.store
	s.mu.Lock()
	defer s.mu.Unlock()
	now := time.Now()
	recent := []time.Time{}
	for _, t := range s.attempts["login"] {
		if now.Sub(t) < time.Minute {
			recent = append(recent, t)
		}
	}
	s.attempts["login"] = recent
	if len(recent) >= 30 {
		fail(w, 429, "尝试过于频繁，请一分钟后重试")
		return
	}
	s.attempts["login"] = append(recent, now)
	var req struct {
		Code   string `json:"code"`
		UserID string `json:"userId"`
	}
	if !decode(w, r, &req) {
		return
	}
	var hash string
	var active int
	if err := s.db.QueryRow("SELECT pin_hash,active FROM staff_accounts WHERE id=?", req.UserID).Scan(&hash, &active); err != nil || active != 1 || bcrypt.CompareHashAndPassword([]byte(hash), []byte(req.Code)) != nil {
		fail(w, 401, "员工账号或登录口令不正确")
		return
	}
	profile, err := s.staffByID(req.UserID, true)
	if err != nil {
		fail(w, 401, "员工账号已停用")
		return
	}
	token := randomID()
	if _, err := s.db.Exec("INSERT INTO sessions(token,expires,user_id) VALUES (?,?,?)", token, time.Now().Add(12*time.Hour).Unix(), profile.ID); err != nil {
		fail(w, 500, "登录失败")
		return
	}
	writeJSON(w, 200, response{"token": token, "user": profile})
}
func currentStaff(r *http.Request) staff { return r.Context().Value(staffContextKey{}).(staff) }
func permitted(u staff, permission string) bool {
	for _, p := range u.Permissions {
		if p == "*" || p == permission {
			return true
		}
	}
	return false
}
func (app *application) auditLogs(w http.ResponseWriter, r *http.Request) {
	u := currentStaff(r)
	if !permitted(u, "audit:view") {
		fail(w, 403, "当前角色没有查看审计记录的权限")
		return
	}
	query := `SELECT a.id,a.order_id,a.user_id,COALESCE(s.name,a.user_id),COALESCE(s.role,''),a.action,a.created_at
		FROM audit_logs a LEFT JOIN staff_accounts s ON s.id=a.user_id WHERE 1=1`
	args := []any{}
	from, to := r.URL.Query().Get("from"), r.URL.Query().Get("to")
	if from != "" || to != "" {
		fromDate, fromErr := time.Parse("2006-01-02", from)
		toDate, toErr := time.Parse("2006-01-02", to)
		if fromErr != nil || toErr != nil || fromDate.After(toDate) || toDate.Sub(fromDate) > 89*24*time.Hour {
			fail(w, 400, "审计日期范围需为 1–90 天")
			return
		}
		query += " AND a.created_at>=? AND a.created_at<?"
		args = append(args, from+"T00:00:00", toDate.AddDate(0, 0, 1).Format("2006-01-02")+"T00:00:00")
	}
	if userID := strings.TrimSpace(r.URL.Query().Get("userId")); userID != "" {
		if len(userID) > 64 {
			fail(w, 400, "员工筛选条件不正确")
			return
		}
		query += " AND a.user_id=?"
		args = append(args, userID)
	}
	switch category := r.URL.Query().Get("category"); category {
	case "", "all":
	case "payment":
		query += " AND (a.action LIKE '%收款%' OR a.action LIKE '%退款%')"
	case "refund":
		query += " AND a.action LIKE '%退款%'"
	default:
		fail(w, 400, "审计类型筛选条件不正确")
		return
	}
	query += " ORDER BY a.rowid DESC LIMIT 500"
	rows, err := app.store.db.QueryContext(r.Context(), query, args...)
	if err != nil {
		fail(w, 500, "读取审计记录失败")
		return
	}
	defer rows.Close()
	logs := []auditLog{}
	for rows.Next() {
		var log auditLog
		var role string
		if err := rows.Scan(&log.ID, &log.OrderID, &log.UserID, &log.UserName, &role, &log.Action, &log.CreatedAt); err != nil {
			fail(w, 500, "审计记录数据异常")
			return
		}
		if definition, ok := rolePermissions[role]; ok {
			log.RoleName = definition.Name
		}
		logs = append(logs, log)
	}
	if rows.Err() != nil {
		fail(w, 500, "读取审计记录失败")
		return
	}
	writeJSON(w, 200, response{"logs": logs, "from": from, "to": to})
}

func requireOwner(w http.ResponseWriter, r *http.Request) bool {
	if !permitted(currentStaff(r), "staff:manage") {
		fail(w, http.StatusForbidden, "只有店长可以管理员工账号")
		return false
	}
	return true
}

func (app *application) listStaff(w http.ResponseWriter, r *http.Request) {
	if !requireOwner(w, r) {
		return
	}
	rows, err := app.store.db.QueryContext(r.Context(), "SELECT id,name,role,active,created_at FROM staff_accounts ORDER BY active DESC, rowid")
	if err != nil {
		fail(w, 500, "读取员工失败")
		return
	}
	defer rows.Close()
	accounts := []staffAccount{}
	for rows.Next() {
		var account staffAccount
		var active int
		if err := rows.Scan(&account.ID, &account.Name, &account.Role, &active, &account.CreatedAt); err != nil {
			fail(w, 500, "员工数据异常")
			return
		}
		role := rolePermissions[account.Role]
		account.RoleName, account.Permissions, account.Active = role.Name, role.Permissions, active == 1
		accounts = append(accounts, account)
	}
	writeJSON(w, 200, response{"staff": accounts})
}

func validateStaffInput(name, role, pin string) error {
	if strings.TrimSpace(name) == "" || len([]rune(strings.TrimSpace(name))) > 30 {
		return errors.New("员工姓名需为 1–30 个字符")
	}
	if _, ok := rolePermissions[role]; !ok {
		return errors.New("员工角色不正确")
	}
	if len(pin) < 6 || len(pin) > 32 {
		return errors.New("登录口令需为 6–32 个字符")
	}
	return nil
}

func (app *application) createStaff(w http.ResponseWriter, r *http.Request) {
	if !requireOwner(w, r) {
		return
	}
	var req struct {
		Name string `json:"name"`
		Role string `json:"role"`
		Pin  string `json:"pin"`
	}
	if !decode(w, r, &req) {
		return
	}
	req.Name = strings.TrimSpace(req.Name)
	if err := validateStaffInput(req.Name, req.Role, req.Pin); err != nil {
		fail(w, 400, err.Error())
		return
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(req.Pin), bcrypt.DefaultCost)
	if err != nil {
		fail(w, 500, "生成员工口令失败")
		return
	}
	id := "staff-" + randomID()[:12]
	createdAt := time.Now().Format(time.RFC3339)
	if _, err = app.store.db.ExecContext(r.Context(), "INSERT INTO staff_accounts(id,name,role,pin_hash,active,created_at) VALUES (?,?,?,?,1,?)", id, req.Name, req.Role, string(hash), createdAt); err != nil {
		fail(w, 500, "创建员工失败")
		return
	}
	profile, _ := app.store.staffByID(id, true)
	writeJSON(w, http.StatusCreated, response{"staff": staffAccount{staff: profile, Active: true, CreatedAt: createdAt}})
}

func (app *application) updateStaff(w http.ResponseWriter, r *http.Request) {
	if !requireOwner(w, r) {
		return
	}
	var req struct {
		Name   *string `json:"name"`
		Role   *string `json:"role"`
		Pin    *string `json:"pin"`
		Active *bool   `json:"active"`
	}
	if !decode(w, r, &req) {
		return
	}
	id := r.PathValue("id")
	var name, role, hash, createdAt string
	var active int
	if err := app.store.db.QueryRowContext(r.Context(), "SELECT name,role,pin_hash,active,created_at FROM staff_accounts WHERE id=?", id).Scan(&name, &role, &hash, &active, &createdAt); err != nil {
		fail(w, 404, "员工不存在")
		return
	}
	if req.Name != nil {
		name = strings.TrimSpace(*req.Name)
	}
	if req.Role != nil {
		role = *req.Role
	}
	pin := "valid-placeholder"
	if req.Pin != nil {
		pin = *req.Pin
	}
	if err := validateStaffInput(name, role, pin); err != nil {
		fail(w, 400, err.Error())
		return
	}
	if req.Pin != nil {
		encoded, err := bcrypt.GenerateFromPassword([]byte(*req.Pin), bcrypt.DefaultCost)
		if err != nil {
			fail(w, 500, "生成员工口令失败")
			return
		}
		hash = string(encoded)
	}
	if req.Active != nil {
		if *req.Active {
			active = 1
		} else {
			if id == currentStaff(r).ID {
				fail(w, 409, "不能停用当前登录账号")
				return
			}
			if role == "owner" {
				var owners int
				app.store.db.QueryRow("SELECT COUNT(*) FROM staff_accounts WHERE role='owner' AND active=1").Scan(&owners)
				if owners <= 1 {
					fail(w, 409, "至少保留一个启用的店长账号")
					return
				}
			}
			active = 0
		}
	}
	if _, err := app.store.db.ExecContext(r.Context(), "UPDATE staff_accounts SET name=?,role=?,pin_hash=?,active=? WHERE id=?", name, role, hash, active, id); err != nil {
		fail(w, 500, "更新员工失败")
		return
	}
	if req.Pin != nil || active == 0 {
		app.store.db.ExecContext(r.Context(), "DELETE FROM sessions WHERE user_id=?", id)
	}
	profile, _ := app.store.staffByID(id, false)
	writeJSON(w, 200, response{"staff": staffAccount{staff: profile, Active: active == 1, CreatedAt: createdAt}})
}
func businessDate() string {
	loc := time.FixedZone("Asia/Shanghai", 8*3600)
	return time.Now().In(loc).Add(-6 * time.Hour).Format("2006-01-02")
}
func (app *application) state(w http.ResponseWriter, r *http.Request) {
	u := currentStaff(r)
	day := businessDate()
	rev := app.store.revision()
	if raw := strings.TrimSpace(r.URL.Query().Get("rev")); raw != "" {
		if parsed, err := strconv.ParseInt(raw, 10, 64); err == nil && parsed > 0 && parsed == rev {
			writeJSON(w, 200, response{"unchanged": true, "rev": rev, "businessDate": day})
			return
		}
	}
	query := `SELECT body FROM orders WHERE (status NOT IN ('completed','cancelled') OR date=?)`
	args := []any{day}
	if u.Role == "sales" {
		query += " AND created_by=?"
		args = append(args, u.ID)
	}
	query += " ORDER BY rowid DESC LIMIT 500"
	rows, err := app.store.db.QueryContext(r.Context(), query, args...)
	if err != nil {
		fail(w, 500, "读取订单失败")
		return
	}
	defer rows.Close()
	orders := []order{}
	for rows.Next() {
		var raw string
		var o order
		if rows.Scan(&raw) != nil || json.Unmarshal([]byte(raw), &o) != nil {
			fail(w, 500, "订单数据异常")
			return
		}
		if o.DurationMinutes == 0 {
			o.DurationMinutes = 240
		}
		if u.Role == "sales" && o.CreatedBy != u.ID {
			continue
		}
		orders = append(orders, o)
	}
	if rows.Err() != nil {
		fail(w, 500, "读取订单失败")
		return
	}
	products, err := app.readProducts(r.Context(), true)
	if err != nil {
		fail(w, 500, "读取商品失败")
		return
	}
	writeJSON(w, 200, response{"orders": orders, "tables": tables, "products": products, "businessDate": day, "updatedAt": time.Now().Format(time.RFC3339), "mode": "role-test", "currentUser": u, "rev": rev, "unchanged": false})
}

func escapeLike(value string) string {
	value = strings.ReplaceAll(value, `\`, `\\`)
	value = strings.ReplaceAll(value, `%`, `\%`)
	return strings.ReplaceAll(value, `_`, `\_`)
}

func (app *application) listOrders(w http.ResponseWriter, r *http.Request) {
	u := currentStaff(r)
	limit := 20
	if raw := r.URL.Query().Get("limit"); raw != "" {
		parsed, err := strconv.Atoi(raw)
		if err != nil || parsed < 1 || parsed > 100 {
			fail(w, 400, "分页数量需为 1–100")
			return
		}
		limit = parsed
	}
	query := "SELECT rowid,body FROM orders WHERE 1=1"
	args := []any{}
	if raw := r.URL.Query().Get("cursor"); raw != "" {
		cursor, err := strconv.ParseInt(raw, 10, 64)
		if err != nil || cursor < 1 {
			fail(w, 400, "分页游标不正确")
			return
		}
		query += " AND rowid<?"
		args = append(args, cursor)
	}
	switch status := r.URL.Query().Get("status"); status {
	case "", "all":
	case "active":
		query += " AND status IN ('arrived','serving','cleaning')"
	case "archived":
		query += " AND status IN ('completed','cancelled')"
	case "reserved", "arrived", "serving", "cleaning", "completed", "cancelled":
		query += " AND status=?"
		args = append(args, status)
	default:
		fail(w, 400, "订单状态筛选条件不正确")
		return
	}
	from, to := r.URL.Query().Get("from"), r.URL.Query().Get("to")
	if from != "" || to != "" {
		fromDate, fromErr := time.Parse("2006-01-02", from)
		toDate, toErr := time.Parse("2006-01-02", to)
		if fromErr != nil || toErr != nil || fromDate.After(toDate) || toDate.Sub(fromDate) > 365*24*time.Hour {
			fail(w, 400, "订单日期范围需为 1–366 天")
			return
		}
		query += " AND date BETWEEN ? AND ?"
		args = append(args, from, to)
	}
	if keyword := strings.TrimSpace(r.URL.Query().Get("q")); keyword != "" {
		if len([]rune(keyword)) > 40 {
			fail(w, 400, "搜索关键词最多 40 个字符")
			return
		}
		like := "%" + escapeLike(keyword) + "%"
		query += ` AND (customer_name LIKE ? ESCAPE '\' OR phone LIKE ? ESCAPE '\' OR table_id LIKE ? ESCAPE '\' OR id LIKE ? ESCAPE '\')`
		args = append(args, like, like, like, like)
	}
	if u.Role == "sales" {
		query += " AND created_by=?"
		args = append(args, u.ID)
	}
	query += " ORDER BY rowid DESC LIMIT ?"
	args = append(args, limit+1)
	rows, err := app.store.db.QueryContext(r.Context(), query, args...)
	if err != nil {
		fail(w, 500, "读取订单列表失败")
		return
	}
	defer rows.Close()
	orders := []order{}
	rowIDs := []int64{}
	for rows.Next() {
		var rowID int64
		var raw string
		var o order
		if rows.Scan(&rowID, &raw) != nil || json.Unmarshal([]byte(raw), &o) != nil {
			fail(w, 500, "订单列表数据异常")
			return
		}
		if o.DurationMinutes == 0 {
			o.DurationMinutes = 240
		}
		orders, rowIDs = append(orders, o), append(rowIDs, rowID)
	}
	nextCursor := ""
	if len(orders) > limit {
		nextCursor = strconv.FormatInt(rowIDs[limit-1], 10)
		orders = orders[:limit]
	}
	writeJSON(w, 200, response{"orders": orders, "nextCursor": nextCursor})
}
func (app *application) priced(tx *sql.Tx, input []item) ([]item, error) {
	out := []item{}
	if len(input) > 50 {
		return nil, errors.New("单次最多添加 50 项")
	}
	for _, v := range input {
		found := false
		var p item
		var active int
		err := tx.QueryRow("SELECT id,name,price,active FROM products WHERE id=?", v.ProductID).Scan(&p.ProductID, &p.Name, &p.Price, &active)
		if err == nil && active == 1 {
			if v.Qty < 1 || v.Qty > 99 {
				return nil, errors.New("商品数量需为 1–99")
			}
			p.Qty = v.Qty
			p.Active = false
			out = append(out, p)
			found = true
		}
		if !found {
			return nil, errors.New("商品不存在")
		}
	}
	return out, nil
}
func amount(o order) int {
	n := 0
	for _, v := range o.Items {
		n += v.Price * v.Qty
	}
	return n
}
func (app *application) writeOrder(w http.ResponseWriter, r *http.Request, action func(*sql.Tx) (order, error)) {
	key := r.Header.Get("Idempotency-Key")
	if len(key) < 8 || len(key) > 128 {
		fail(w, 400, "缺少有效请求标识")
		return
	}
	raw, err := io.ReadAll(io.LimitReader(r.Body, 32769))
	if err != nil || len(raw) > 32768 {
		fail(w, 400, "请求过大")
		return
	}
	r.Body = io.NopCloser(strings.NewReader(string(raw)))
	sum := sha256.Sum256(append([]byte(r.URL.Path), raw...))
	finger := hex.EncodeToString(sum[:])
	s := app.store
	s.mu.Lock()
	defer s.mu.Unlock()
	tx, err := s.db.BeginTx(r.Context(), nil)
	if err != nil {
		fail(w, 500, "数据库暂不可用")
		return
	}
	defer tx.Rollback()
	var previous, body string
	err = tx.QueryRow("SELECT fingerprint,body FROM requests WHERE key=?", key).Scan(&previous, &body)
	if err == nil {
		if previous != finger {
			fail(w, 409, "请求标识已被使用")
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(body))
		return
	}
	if err != sql.ErrNoRows {
		fail(w, 500, "读取请求失败")
		return
	}
	o, err := action(tx)
	if err != nil {
		status := http.StatusConflict
		if strings.Contains(err.Error(), "权限") {
			status = http.StatusForbidden
		}
		fail(w, status, err.Error())
		return
	}
	startMinute, endMinute, scheduleErr := orderSchedule(o.ArrivalTime, o.DurationMinutes)
	if scheduleErr != nil {
		fail(w, http.StatusConflict, scheduleErr.Error())
		return
	}
	if o.DurationMinutes == 0 {
		o.DurationMinutes = 240
	}
	if o.Status != "completed" && o.Status != "cancelled" {
		var overlap int
		if err = tx.QueryRow(`SELECT COUNT(*) FROM orders WHERE table_id=? AND date=? AND id<>? AND status NOT IN ('completed','cancelled') AND start_minute<? AND end_minute>?`, o.TableID, o.Date, o.ID, endMinute, startMinute).Scan(&overlap); err != nil {
			fail(w, 500, "检查台位时段失败")
			return
		}
		if overlap > 0 {
			fail(w, http.StatusConflict, "该台位在所选时段已被占用，请调整时间或台位")
			return
		}
	}
	payload, err := json.Marshal(o)
	if err != nil {
		fail(w, 500, "保存失败")
		return
	}
	_, err = tx.Exec("INSERT INTO orders(id,table_id,date,status,body,start_minute,end_minute,created_by,customer_name,phone) VALUES (?,?,?,?,?,?,?,?,?,?) ON CONFLICT(id) DO UPDATE SET table_id=excluded.table_id,date=excluded.date,status=excluded.status,body=excluded.body,start_minute=excluded.start_minute,end_minute=excluded.end_minute,created_by=excluded.created_by,customer_name=excluded.customer_name,phone=excluded.phone", o.ID, o.TableID, o.Date, o.Status, string(payload), startMinute, endMinute, o.CreatedBy, o.CustomerName, o.Phone)
	if err != nil {
		fail(w, 409, "该台位在所选营业日已被占用，请重新选择")
		return
	}
	u := currentStaff(r)
	latest := o.Events[len(o.Events)-1]
	if _, err = tx.Exec("INSERT INTO audit_logs VALUES (?,?,?,?,?)", randomID(), o.ID, u.ID, latest.Action, latest.At); err != nil {
		fail(w, 500, "操作日志保存失败")
		return
	}
	result, _ := json.Marshal(response{"order": o})
	if _, err = tx.Exec("INSERT INTO requests VALUES (?,?,?)", key, finger, string(result)); err != nil {
		fail(w, 500, "保存请求失败")
		return
	}
	if err = tx.Commit(); err != nil {
		fail(w, 500, "提交失败，请重试")
		return
	}
	s.bump(nil)
	writeJSON(w, 200, response{"order": o})
}
func (app *application) createBooking(w http.ResponseWriter, r *http.Request) {
	u := currentStaff(r)
	if !permitted(u, "order:create") {
		fail(w, 403, "当前角色没有创建预订权限")
		return
	}
	app.writeOrder(w, r, func(tx *sql.Tx) (order, error) {
		var o order
		d := json.NewDecoder(r.Body)
		if d.Decode(&o) != nil {
			return o, errors.New("预订格式不正确")
		}
		o.CustomerName = strings.TrimSpace(o.CustomerName)
		o.Remark = strings.TrimSpace(o.Remark)
		if o.CustomerName == "" || len([]rune(o.CustomerName)) > 40 || !regexp.MustCompile(`^1\d{10}$`).MatchString(o.Phone) {
			return o, errors.New("请填写姓名和 11 位手机号")
		}
		if len([]rune(o.Remark)) > 200 {
			return o, errors.New("订单备注最多 200 字")
		}
		day, err := time.Parse("2006-01-02", o.Date)
		if err != nil || o.Date < businessDate() || day.After(time.Now().AddDate(0, 0, 90)) {
			return o, errors.New("请选择未来 90 天内的营业日")
		}
		if _, _, err = orderSchedule(o.ArrivalTime, o.DurationMinutes); err != nil {
			return o, err
		}
		found := false
		for _, t := range tables {
			if t.ID == o.TableID {
				found = true
				if o.GuestCount < 1 || o.GuestCount > t.Capacity {
					return o, fmt.Errorf("该台建议人数为 1–%d 人", t.Capacity)
				}
			}
		}
		if !found {
			return o, errors.New("台位不存在")
		}
		o.Items, err = app.priced(tx, o.Items)
		if err != nil {
			return o, err
		}
		o.ID = "HC" + randomID()[:16]
		o.Status = "reserved"
		o.SalesName = u.Name
		o.CreatedBy = u.ID
		o.CreatedAt = time.Now().Format(time.RFC3339)
		o.Version = 1
		if o.DurationMinutes == 0 {
			o.DurationMinutes = 240
		}
		o.PaidAmount = 0
		o.PaymentMethod = ""
		o.Events = []event{{"创建预订", o.CreatedAt, u.Name}}
		return o, nil
	})
}
func (app *application) mutateOrder(w http.ResponseWriter, r *http.Request) {
	u := currentStaff(r)
	app.writeOrder(w, r, func(tx *sql.Tx) (order, error) {
		var o order
		var raw string
		if tx.QueryRow("SELECT body FROM orders WHERE id=?", r.PathValue("id")).Scan(&raw) != nil {
			return o, errors.New("订单不存在")
		}
		if json.Unmarshal([]byte(raw), &o) != nil {
			return o, errors.New("订单数据异常")
		}
		var req struct {
			Version         int    `json:"version"`
			Items           []item `json:"items"`
			PaymentMethod   string `json:"paymentMethod"`
			TableID         string `json:"tableId"`
			CancelReason    string `json:"cancelReason"`
			RefundReason    string `json:"refundReason"`
			RefundAmount    int    `json:"refundAmount"`
			CustomerName    string `json:"customerName"`
			Phone           string `json:"phone"`
			GuestCount      int    `json:"guestCount"`
			Date            string `json:"date"`
			ArrivalTime     string `json:"arrivalTime"`
			DurationMinutes int    `json:"durationMinutes"`
			Remark          string `json:"remark"`
		}
		if json.NewDecoder(r.Body).Decode(&req) != nil {
			return o, errors.New("请求格式不正确")
		}
		if req.Version != o.Version {
			return o, errors.New("订单已被更新，请刷新后重试")
		}
		action := r.PathValue("action")
		permission := map[string]string{"update-booking": "order:update", "confirm-arrival": "order:confirm-arrival", "open-table": "table:open", "change-table": "order:change-table", "cancel": "order:cancel", "complete-cleaning": "table:clean", "items": "order:add-item", "checkout": "payment:checkout", "refund": "payment:refund"}[action]
		allowed := permitted(u, permission)
		if action == "update-booking" && permitted(u, "order:update-own") && o.CreatedBy == u.ID {
			allowed = true
		}
		if action == "cancel" && permitted(u, "order:cancel-own") && o.CreatedBy == u.ID {
			allowed = true
		}
		if permission == "" || !allowed {
			return o, errors.New("当前角色没有执行此操作的权限")
		}
		label := ""
		valid := false
		switch action {
		case "update-booking":
			valid = o.Status == "reserved"
			name, remark := strings.TrimSpace(req.CustomerName), strings.TrimSpace(req.Remark)
			if name == "" || len([]rune(name)) > 40 || !regexp.MustCompile(`^1\d{10}$`).MatchString(req.Phone) {
				return o, errors.New("请填写姓名和 11 位手机号")
			}
			if len([]rune(remark)) > 200 {
				return o, errors.New("订单备注最多 200 字")
			}
			day, parseErr := time.Parse("2006-01-02", req.Date)
			if parseErr != nil || req.Date < businessDate() || day.After(time.Now().AddDate(0, 0, 90)) {
				return o, errors.New("请选择未来 90 天内的营业日")
			}
			if _, _, parseErr = orderSchedule(req.ArrivalTime, req.DurationMinutes); parseErr != nil {
				return o, parseErr
			}
			capacity := 0
			for _, table := range tables {
				if table.ID == o.TableID {
					capacity = table.Capacity
				}
			}
			if req.GuestCount < 1 || req.GuestCount > capacity {
				return o, fmt.Errorf("该台建议人数为 1–%d 人", capacity)
			}
			changes := []string{}
			if o.CustomerName != name || o.Phone != req.Phone {
				changes = append(changes, "客户资料")
			}
			if o.GuestCount != req.GuestCount {
				changes = append(changes, "人数")
			}
			if o.Date != req.Date || o.ArrivalTime != req.ArrivalTime || o.DurationMinutes != req.DurationMinutes {
				changes = append(changes, "到店时段")
			}
			if o.Remark != remark {
				changes = append(changes, "备注")
			}
			if len(changes) == 0 {
				return o, errors.New("预订资料没有变化")
			}
			o.CustomerName, o.Phone, o.GuestCount = name, req.Phone, req.GuestCount
			o.Date, o.ArrivalTime, o.DurationMinutes, o.Remark = req.Date, req.ArrivalTime, req.DurationMinutes, remark
			label = "修改预订 · " + strings.Join(changes, "、")
		case "confirm-arrival":
			valid = o.Status == "reserved"
			o.Status = "arrived"
			label = "确认到店"
		case "open-table":
			valid = o.Status == "arrived"
			o.Status = "serving"
			label = "开台"
		case "cancel":
			valid = o.Status == "reserved"
			reasons := map[string]bool{"客户取消": true, "未按时到店": true, "重复预订": true, "其他": true}
			if !reasons[req.CancelReason] {
				return o, errors.New("请选择取消原因")
			}
			o.CancelReason = req.CancelReason
			o.Status = "cancelled"
			label = "取消预订 · " + req.CancelReason
		case "change-table":
			valid = o.Status == "reserved" || o.Status == "arrived"
			if req.TableID == o.TableID {
				return o, errors.New("请选择其他台位")
			}
			found := false
			for _, t := range tables {
				if t.ID == req.TableID {
					found = true
					if o.GuestCount > t.Capacity {
						return o, fmt.Errorf("新台位最多容纳 %d 人", t.Capacity)
					}
				}
			}
			if !found {
				return o, errors.New("新台位不存在")
			}
			old := o.TableID
			o.TableID = req.TableID
			label = "改台 · " + old + " → " + req.TableID
		case "complete-cleaning":
			valid = o.Status == "cleaning"
			o.Status = "completed"
			label = "完成清台"
		case "items":
			valid = o.Status == "serving"
			items, err := app.priced(tx, req.Items)
			if err != nil {
				return o, err
			}
			if len(items) == 0 {
				return o, errors.New("请选择商品")
			}
			o.Items = append(o.Items, items...)
			if len(o.Items) > 500 {
				return o, errors.New("订单明细已达上限")
			}
			label = "添加商品"
		case "checkout":
			valid = o.Status == "serving"
			if req.PaymentMethod != "微信" && req.PaymentMethod != "支付宝" && req.PaymentMethod != "现金" {
				return o, errors.New("请选择收款方式")
			}
			o.PaymentMethod = req.PaymentMethod
			o.PaidAmount = amount(o)
			o.Status = "cleaning"
			label = "确认线下收款 · " + req.PaymentMethod
		case "refund":
			valid = (o.Status == "cleaning" || o.Status == "completed") && o.PaidAmount > 0 && o.RefundedAmount < o.PaidAmount
			reasons := map[string]bool{"客户投诉退款": true, "重复收款": true, "运营异常": true, "其他": true}
			if !reasons[req.RefundReason] {
				return o, errors.New("请选择退款原因")
			}
			remaining := o.PaidAmount - o.RefundedAmount
			amount := req.RefundAmount
			if amount == 0 {
				amount = remaining
			}
			if amount < 1 || amount > remaining {
				return o, fmt.Errorf("退款金额需为 1–%d 元", remaining)
			}
			o.RefundedAmount += amount
			o.RefundReason = req.RefundReason
			o.RefundedAt = time.Now().Format(time.RFC3339)
			o.Refunds = append(o.Refunds, refund{Amount: amount, Reason: req.RefundReason, At: o.RefundedAt, Operator: u.Name})
			kind := "部分退款"
			if o.RefundedAmount == o.PaidAmount {
				kind = "全额退款"
			}
			label = fmt.Sprintf("登记%s ¥%d · %s", kind, amount, req.RefundReason)
		}
		if !valid {
			return o, errors.New("当前订单状态不允许此操作")
		}
		o.Version++
		now := time.Now().Format(time.RFC3339)
		o.Events = append(o.Events, event{label, now, u.Name})
		return o, nil
	})
}
