package main

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"
)

func testClub(t *testing.T) (*application, string) {
	t.Helper()
	s, err := newClubStore(filepath.Join(t.TempDir(), "test.db"), "test-code")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { s.db.Close() })
	token := "test-session"
	if _, err = s.db.Exec("INSERT INTO sessions(token,expires,user_id) VALUES (?,?,?)", token, time.Now().Add(time.Hour).Unix(), "owner"); err != nil {
		t.Fatal(err)
	}
	return &application{store: s, allowedOrigins: map[string]struct{}{}}, token
}
func call(t *testing.T, a *application, token, path, key string, body any) *httptest.ResponseRecorder {
	t.Helper()
	method := "POST"
	if body == nil {
		method = "GET"
	}
	return callMethod(t, a, token, method, path, key, body)
}
func callMethod(t *testing.T, a *application, token, method, path, key string, body any) *httptest.ResponseRecorder {
	t.Helper()
	payload, _ := json.Marshal(body)
	r := httptest.NewRequest(method, path, bytes.NewReader(payload))
	r.Header.Set("Authorization", "Bearer "+token)
	r.Header.Set("Idempotency-Key", key)
	w := httptest.NewRecorder()
	a.routes().ServeHTTP(w, r)
	return w
}
func bookingData() map[string]any {
	return map[string]any{"tableId": "V01", "customerName": "测试客户", "phone": "13800000000", "guestCount": 6, "date": businessDate(), "arrivalTime": "22:30", "items": []map[string]any{{"productId": "classic", "qty": 1, "price": 1}}}
}
func editData(o order) map[string]any {
	return map[string]any{"version": o.Version, "customerName": o.CustomerName, "phone": o.Phone, "guestCount": o.GuestCount, "date": o.Date, "arrivalTime": o.ArrivalTime, "durationMinutes": o.DurationMinutes, "remark": o.Remark}
}
func readOrder(t *testing.T, w *httptest.ResponseRecorder) order {
	t.Helper()
	if w.Code != 200 {
		t.Fatalf("status %d: %s", w.Code, w.Body)
	}
	var res struct {
		Order order `json:"order"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &res); err != nil {
		t.Fatal(err)
	}
	return res.Order
}
func staffToken(t *testing.T, a *application, userID string) string {
	t.Helper()
	token := userID + "-token"
	if _, err := a.store.db.Exec("INSERT INTO sessions(token,expires,user_id) VALUES (?,?,?)", token, time.Now().Add(time.Hour).Unix(), userID); err != nil {
		t.Fatal(err)
	}
	return token
}
func TestLifecycleAndServerPricing(t *testing.T) {
	a, token := testClub(t)
	o := readOrder(t, call(t, a, token, "/api/v1/orders", "create-0001", bookingData()))
	if amount(o) != 2880 {
		t.Fatal("client price trusted")
	}
	if o.Status != "reserved" {
		t.Fatal(o.Status)
	}
	for i, step := range []struct {
		action, status string
		extra          map[string]any
	}{{"confirm-arrival", "arrived", nil}, {"open-table", "serving", nil}, {"items", "serving", map[string]any{"items": []map[string]any{{"productId": "beer", "qty": 6}}}}, {"checkout", "cleaning", map[string]any{"paymentMethod": "微信"}}, {"complete-cleaning", "completed", nil}} {
		body := map[string]any{"version": o.Version}
		for k, v := range step.extra {
			body[k] = v
		}
		o = readOrder(t, call(t, a, token, "/api/v1/orders/"+o.ID+"/"+step.action, fmt.Sprintf("step-%04d", i), body))
		if o.Status != step.status {
			t.Fatalf("want %s got %s", step.status, o.Status)
		}
	}
	if o.PaidAmount != 3168 || o.PaymentMethod != "微信" || len(o.Events) != 6 {
		t.Fatalf("missing payment/event data: %+v", o)
	}
	readOrder(t, call(t, a, token, "/api/v1/orders", "create-0002", bookingData()))
}
func TestIdempotencyConflictAndVersions(t *testing.T) {
	a, token := testClub(t)
	first := call(t, a, token, "/api/v1/orders", "same-key-01", bookingData())
	o := readOrder(t, first)
	again := call(t, a, token, "/api/v1/orders", "same-key-01", bookingData())
	if readOrder(t, again).ID != o.ID {
		t.Fatal("duplicate created")
	}
	if w := call(t, a, token, "/api/v1/orders", "different-key", bookingData()); w.Code != 409 {
		t.Fatal(w.Code)
	}
	for _, body := range []map[string]any{{"version": 0}, {"version": 1, "paymentMethod": "现金"}} {
		if w := call(t, a, token, "/api/v1/orders/"+o.ID+"/checkout", randomID(), body); w.Code != 409 {
			t.Fatal("invalid checkout accepted")
		}
	}
	changed := bookingData()
	changed["tableId"] = "V02"
	if w := call(t, a, token, "/api/v1/orders", "same-key-01", changed); w.Code != 409 {
		t.Fatal("reused key accepted")
	}
}
func TestConcurrentBooking(t *testing.T) {
	a, token := testClub(t)
	var wg sync.WaitGroup
	codes := make(chan int, 8)
	for i := 0; i < 8; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			codes <- call(t, a, token, "/api/v1/orders", fmt.Sprintf("parallel-%d", i), bookingData()).Code
		}(i)
	}
	wg.Wait()
	close(codes)
	success := 0
	for code := range codes {
		if code == 200 {
			success++
		} else if code != 409 {
			t.Fatal(code)
		}
	}
	if success != 1 {
		t.Fatalf("created %d concurrent bookings", success)
	}
}
func TestAuthenticationAndValidation(t *testing.T) {
	a, token := testClub(t)
	if w := call(t, a, "invalid", "/api/v1/state", "", nil); w.Code != 401 {
		t.Fatal(w.Code)
	}
	if w := call(t, a, "", "/api/v1/login", "", map[string]string{"code": "wrong", "userId": "owner"}); w.Code != 401 {
		t.Fatal(w.Code)
	}
	if w := call(t, a, "", "/api/v1/login", "", map[string]string{"code": "test-code", "userId": "owner"}); w.Code != 200 {
		t.Fatal(w.Code)
	}
	for key, value := range map[string]any{"phone": "bad", "guestCount": 11, "date": "2020-01-01", "tableId": "missing", "customerName": " "} {
		body := bookingData()
		body[key] = value
		if w := call(t, a, token, "/api/v1/orders", randomID(), body); w.Code != 409 {
			t.Fatalf("accepted invalid %s", key)
		}
	}
	if w := call(t, a, token, "/api/v1/state", "", nil); w.Code != http.StatusOK {
		t.Fatal(w.Code)
	}
}
func TestStateRevisionPolling(t *testing.T) {
	a, token := testClub(t)
	first := call(t, a, token, "/api/v1/state", "", nil)
	if first.Code != 200 {
		t.Fatal(first.Body)
	}
	var payload struct {
		Rev          int64   `json:"rev"`
		Unchanged    bool    `json:"unchanged"`
		Orders       []order `json:"orders"`
		BusinessDate string  `json:"businessDate"`
	}
	if err := json.Unmarshal(first.Body.Bytes(), &payload); err != nil || payload.Rev < 1 || payload.Unchanged {
		t.Fatalf("%v %#v", err, payload)
	}
	again := call(t, a, token, "/api/v1/state?rev="+fmt.Sprintf("%d", payload.Rev), "", nil)
	var poll struct {
		Rev       int64   `json:"rev"`
		Unchanged bool    `json:"unchanged"`
		Orders    []order `json:"orders"`
	}
	if err := json.Unmarshal(again.Body.Bytes(), &poll); err != nil {
		t.Fatal(err)
	}
	if again.Code != 200 || !poll.Unchanged || poll.Rev != payload.Rev || len(poll.Orders) != 0 {
		t.Fatalf("poll %#v %s", poll, again.Body)
	}
	readOrder(t, call(t, a, token, "/api/v1/orders", "rev-create", bookingData()))
	changed := call(t, a, token, "/api/v1/state?rev="+fmt.Sprintf("%d", payload.Rev), "", nil)
	if err := json.Unmarshal(changed.Body.Bytes(), &payload); err != nil {
		t.Fatal(err)
	}
	if payload.Unchanged || payload.Rev <= poll.Rev || len(payload.Orders) == 0 {
		t.Fatalf("expected new state %#v %s", payload, changed.Body)
	}
}
func TestPersistenceAfterReopen(t *testing.T) {
	path := filepath.Join(t.TempDir(), "persist.db")
	s, err := newClubStore(path, "test-code")
	if err != nil {
		t.Fatal(err)
	}
	s.db.Exec("INSERT INTO sessions(token,expires,user_id) VALUES (?,?,?)", "persist-token", time.Now().Add(time.Hour).Unix(), "owner")
	a := &application{store: s}
	o := readOrder(t, call(t, a, "persist-token", "/api/v1/orders", "persist-request", bookingData()))
	s.db.Close()
	s, err = newClubStore(path, "test-code")
	if err != nil {
		t.Fatal(err)
	}
	defer s.db.Close()
	var body string
	if err = s.db.QueryRow("SELECT body FROM orders WHERE id=?", o.ID).Scan(&body); err != nil {
		t.Fatal(err)
	}
	var saved order
	json.Unmarshal([]byte(body), &saved)
	if saved.CustomerName != o.CustomerName {
		t.Fatal("order lost")
	}
}

func TestRolePermissionsAndAudit(t *testing.T) {
	a, _ := testClub(t)
	sales := staffToken(t, a, "sales")
	frontdesk := staffToken(t, a, "frontdesk")
	waiter := staffToken(t, a, "waiter")

	o := readOrder(t, call(t, a, sales, "/api/v1/orders", "role-create-sales", bookingData()))
	if o.CreatedBy != "sales" || o.SalesName != "销售小林" || o.Events[0].Operator != "销售小林" {
		t.Fatalf("creator not recorded: %+v", o)
	}
	o = readOrder(t, call(t, a, frontdesk, "/api/v1/orders/"+o.ID+"/confirm-arrival", "role-arrival", map[string]any{"version": o.Version}))
	if w := call(t, a, frontdesk, "/api/v1/orders/"+o.ID+"/open-table", "role-open-denied", map[string]any{"version": o.Version}); w.Code != http.StatusForbidden {
		t.Fatalf("frontdesk opened table: %d %s", w.Code, w.Body.String())
	}
	o = readOrder(t, call(t, a, waiter, "/api/v1/orders/"+o.ID+"/open-table", "role-open", map[string]any{"version": o.Version}))
	if w := call(t, a, sales, "/api/v1/orders/"+o.ID+"/items", "role-item-denied", map[string]any{"version": o.Version, "items": []map[string]any{{"productId": "beer", "qty": 1}}}); w.Code != http.StatusForbidden {
		t.Fatalf("sales added item: %d", w.Code)
	}
	o = readOrder(t, call(t, a, waiter, "/api/v1/orders/"+o.ID+"/items", "role-item", map[string]any{"version": o.Version, "items": []map[string]any{{"productId": "beer", "qty": 1}}}))
	o = readOrder(t, call(t, a, waiter, "/api/v1/orders/"+o.ID+"/checkout", "role-pay", map[string]any{"version": o.Version, "paymentMethod": "现金"}))
	readOrder(t, call(t, a, waiter, "/api/v1/orders/"+o.ID+"/complete-cleaning", "role-clean", map[string]any{"version": o.Version}))
	var auditCount int
	if err := a.store.db.QueryRow("SELECT COUNT(*) FROM audit_logs WHERE order_id=?", o.ID).Scan(&auditCount); err != nil || auditCount != 6 {
		t.Fatalf("audit count=%d err=%v", auditCount, err)
	}

	body := bookingData()
	body["tableId"] = "V02"
	other := readOrder(t, call(t, a, frontdesk, "/api/v1/orders", "role-create-front", body))
	ownBody := bookingData()
	ownBody["tableId"] = "V03"
	own := readOrder(t, call(t, a, sales, "/api/v1/orders", "role-create-sales-two", ownBody))
	if w := call(t, a, frontdesk, "/api/v1/orders/"+other.ID+"/change-table", "role-change-conflict", map[string]any{"version": other.Version, "tableId": "V03"}); w.Code != http.StatusConflict {
		t.Fatalf("conflicting table change accepted: %d %s", w.Code, w.Body.String())
	}
	other = readOrder(t, call(t, a, frontdesk, "/api/v1/orders/"+other.ID+"/change-table", "role-change-table", map[string]any{"version": other.Version, "tableId": "A01"}))
	if other.TableID != "A01" {
		t.Fatalf("table unchanged: %s", other.TableID)
	}
	if w := call(t, a, sales, "/api/v1/orders/"+other.ID+"/cancel", "role-cancel-other", map[string]any{"version": other.Version}); w.Code != http.StatusForbidden {
		t.Fatalf("sales cancelled another order: %d", w.Code)
	}
	if w := call(t, a, sales, "/api/v1/orders/"+own.ID+"/cancel", "role-cancel-no-reason", map[string]any{"version": own.Version}); w.Code != http.StatusConflict {
		t.Fatalf("cancel without reason: %d", w.Code)
	}
	own = readOrder(t, call(t, a, sales, "/api/v1/orders/"+own.ID+"/cancel", "role-cancel-own", map[string]any{"version": own.Version, "cancelReason": "客户取消"}))
	if own.CancelReason != "客户取消" {
		t.Fatal("cancel reason not stored")
	}
	if w := call(t, a, waiter, "/api/v1/orders", "role-waiter-create", body); w.Code != http.StatusForbidden {
		t.Fatalf("waiter created booking: %d", w.Code)
	}
	stateResponse := call(t, a, sales, "/api/v1/state", "", nil)
	var stateBody struct {
		Orders []order `json:"orders"`
	}
	if stateResponse.Code != 200 || json.Unmarshal(stateResponse.Body.Bytes(), &stateBody) != nil {
		t.Fatal("sales state failed")
	}
	for _, visible := range stateBody.Orders {
		if visible.CreatedBy != "sales" {
			t.Fatalf("sales saw another employee order: %s", visible.ID)
		}
	}
}

func TestAuditAPIAndLogout(t *testing.T) {
	a, owner := testClub(t)
	frontdesk := staffToken(t, a, "frontdesk")
	o := readOrder(t, call(t, a, owner, "/api/v1/orders", "audit-api-create", bookingData()))

	w := call(t, a, owner, "/api/v1/audit-logs", "", nil)
	if w.Code != http.StatusOK {
		t.Fatalf("owner audit status=%d body=%s", w.Code, w.Body.String())
	}
	var result struct {
		Logs []auditLog `json:"logs"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &result); err != nil || len(result.Logs) != 1 {
		t.Fatalf("invalid audit response: %+v err=%v", result, err)
	}
	if result.Logs[0].OrderID != o.ID || result.Logs[0].UserName != "店长" || result.Logs[0].RoleName != "店长" {
		t.Fatalf("audit identity missing: %+v", result.Logs[0])
	}
	if w := call(t, a, frontdesk, "/api/v1/audit-logs", "", nil); w.Code != http.StatusForbidden {
		t.Fatalf("frontdesk read audit: %d", w.Code)
	}
	if w := call(t, a, owner, "/api/v1/logout", "", map[string]any{}); w.Code != http.StatusOK {
		t.Fatalf("logout failed: %d", w.Code)
	}
	if w := call(t, a, owner, "/api/v1/state", "", nil); w.Code != http.StatusUnauthorized {
		t.Fatalf("logged-out session remained valid: %d", w.Code)
	}
}

func TestStaffAccountManagement(t *testing.T) {
	a, owner := testClub(t)
	frontdesk := staffToken(t, a, "frontdesk")
	if w := call(t, a, frontdesk, "/api/v1/staff", "", nil); w.Code != http.StatusForbidden {
		t.Fatalf("frontdesk listed staff: %d", w.Code)
	}
	w := call(t, a, owner, "/api/v1/staff", "", map[string]any{"name": "新销售", "role": "sales", "pin": "sales-7788"})
	if w.Code != http.StatusCreated {
		t.Fatalf("create staff status=%d body=%s", w.Code, w.Body.String())
	}
	var created struct {
		Staff staffAccount `json:"staff"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &created); err != nil || created.Staff.ID == "" || created.Staff.Role != "sales" {
		t.Fatalf("bad staff response: %+v err=%v", created, err)
	}
	loginResponse := call(t, a, "", "/api/v1/login", "", map[string]string{"code": "sales-7788", "userId": created.Staff.ID})
	if loginResponse.Code != http.StatusOK {
		t.Fatalf("new employee login failed: %d %s", loginResponse.Code, loginResponse.Body.String())
	}
	w = callMethod(t, a, owner, http.MethodPut, "/api/v1/staff/"+created.Staff.ID, "", map[string]any{"active": false})
	if w.Code != http.StatusOK {
		t.Fatalf("disable staff status=%d body=%s", w.Code, w.Body.String())
	}
	if w := call(t, a, "", "/api/v1/login", "", map[string]string{"code": "sales-7788", "userId": created.Staff.ID}); w.Code != http.StatusUnauthorized {
		t.Fatalf("disabled employee logged in: %d", w.Code)
	}
	if w := callMethod(t, a, owner, http.MethodPut, "/api/v1/staff/owner", "", map[string]any{"active": false}); w.Code != http.StatusConflict {
		t.Fatalf("owner disabled self: %d", w.Code)
	}
}

func TestDailyReportPermissionsAndSalesScope(t *testing.T) {
	a, owner := testClub(t)
	sales := staffToken(t, a, "sales")
	waiter := staffToken(t, a, "waiter")

	completed := readOrder(t, call(t, a, owner, "/api/v1/orders", "report-owner-create", bookingData()))
	for i, action := range []string{"confirm-arrival", "open-table", "checkout", "complete-cleaning"} {
		body := map[string]any{"version": completed.Version}
		if action == "checkout" {
			body["paymentMethod"] = "现金"
		}
		completed = readOrder(t, call(t, a, owner, "/api/v1/orders/"+completed.ID+"/"+action, fmt.Sprintf("report-owner-%d", i), body))
	}
	salesBody := bookingData()
	salesBody["tableId"] = "V02"
	readOrder(t, call(t, a, sales, "/api/v1/orders", "report-sales-create", salesBody))

	path := "/api/v1/reports/daily?from=" + businessDate() + "&to=" + businessDate()
	if w := call(t, a, waiter, path, "", nil); w.Code != http.StatusForbidden {
		t.Fatalf("waiter read report: %d", w.Code)
	}
	var ownerReport struct {
		Totals reportDay `json:"totals"`
		Scope  string    `json:"scope"`
	}
	w := call(t, a, owner, path, "", nil)
	if w.Code != http.StatusOK || json.Unmarshal(w.Body.Bytes(), &ownerReport) != nil {
		t.Fatalf("owner report failed: %d %s", w.Code, w.Body.String())
	}
	if ownerReport.Totals.Reservations != 2 || ownerReport.Totals.Completed != 1 || ownerReport.Totals.Revenue != 2880 || ownerReport.Scope != "store" {
		t.Fatalf("wrong owner totals: %+v", ownerReport)
	}
	var salesReport struct {
		Totals reportDay `json:"totals"`
		Scope  string    `json:"scope"`
	}
	w = call(t, a, sales, path, "", nil)
	if w.Code != http.StatusOK || json.Unmarshal(w.Body.Bytes(), &salesReport) != nil {
		t.Fatalf("sales report failed: %d %s", w.Code, w.Body.String())
	}
	if salesReport.Totals.Reservations != 1 || salesReport.Totals.Completed != 0 || salesReport.Scope != "own" {
		t.Fatalf("sales saw store report: %+v", salesReport)
	}
	if w := call(t, a, owner, "/api/v1/reports/daily?from=2025-01-01&to=2026-01-01", "", nil); w.Code != http.StatusBadRequest {
		t.Fatalf("oversized report range accepted: %d", w.Code)
	}
}

func TestProductManagementAndServerPricing(t *testing.T) {
	a, owner := testClub(t)
	frontdesk := staffToken(t, a, "frontdesk")
	if w := call(t, a, frontdesk, "/api/v1/products", "", nil); w.Code != http.StatusForbidden {
		t.Fatalf("frontdesk listed products: %d", w.Code)
	}
	w := call(t, a, owner, "/api/v1/products", "", map[string]any{"name": "测试特饮", "price": 66})
	if w.Code != http.StatusCreated {
		t.Fatalf("create product: %d %s", w.Code, w.Body.String())
	}
	var created struct {
		Product item `json:"product"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &created); err != nil || created.Product.ProductID == "" {
		t.Fatalf("bad product: %+v err=%v", created, err)
	}
	body := bookingData()
	body["items"] = []any{}
	o := readOrder(t, call(t, a, owner, "/api/v1/orders", "product-order", body))
	o = readOrder(t, call(t, a, owner, "/api/v1/orders/"+o.ID+"/confirm-arrival", "product-arrive", map[string]any{"version": o.Version}))
	o = readOrder(t, call(t, a, owner, "/api/v1/orders/"+o.ID+"/open-table", "product-open", map[string]any{"version": o.Version}))
	o = readOrder(t, call(t, a, owner, "/api/v1/orders/"+o.ID+"/items", "product-first", map[string]any{"version": o.Version, "items": []map[string]any{{"productId": created.Product.ProductID, "qty": 2, "price": 1}}}))
	if got := o.Items[len(o.Items)-1].Price; got != 66 {
		t.Fatalf("client price trusted: %d", got)
	}
	w = callMethod(t, a, owner, http.MethodPut, "/api/v1/products/"+created.Product.ProductID, "", map[string]any{"price": 88})
	if w.Code != http.StatusOK {
		t.Fatalf("update price: %d %s", w.Code, w.Body.String())
	}
	o = readOrder(t, call(t, a, owner, "/api/v1/orders/"+o.ID+"/items", "product-second", map[string]any{"version": o.Version, "items": []map[string]any{{"productId": created.Product.ProductID, "qty": 1}}}))
	if o.Items[len(o.Items)-2].Price != 66 || o.Items[len(o.Items)-1].Price != 88 {
		t.Fatalf("historical/current prices wrong: %+v", o.Items)
	}
	w = callMethod(t, a, owner, http.MethodPut, "/api/v1/products/"+created.Product.ProductID, "", map[string]any{"active": false})
	if w.Code != http.StatusOK {
		t.Fatalf("disable product: %d", w.Code)
	}
	if w := call(t, a, owner, "/api/v1/orders/"+o.ID+"/items", "product-disabled", map[string]any{"version": o.Version, "items": []map[string]any{{"productId": created.Product.ProductID, "qty": 1}}}); w.Code != http.StatusConflict {
		t.Fatalf("disabled product sold: %d", w.Code)
	}
}

func TestTableScheduleAllowsAdjacentSlotsAndRejectsOverlap(t *testing.T) {
	a, owner := testClub(t)
	first := bookingData()
	first["arrivalTime"] = "18:00"
	first["durationMinutes"] = 120
	readOrder(t, call(t, a, owner, "/api/v1/orders", "slot-first", first))

	adjacent := bookingData()
	adjacent["arrivalTime"] = "20:00"
	adjacent["durationMinutes"] = 120
	readOrder(t, call(t, a, owner, "/api/v1/orders", "slot-adjacent", adjacent))

	overlap := bookingData()
	overlap["arrivalTime"] = "19:30"
	overlap["durationMinutes"] = 120
	if w := call(t, a, owner, "/api/v1/orders", "slot-overlap", overlap); w.Code != http.StatusConflict {
		t.Fatalf("overlapping slot accepted: %d %s", w.Code, w.Body.String())
	}
	invalidDuration := bookingData()
	invalidDuration["tableId"] = "V02"
	invalidDuration["durationMinutes"] = 45
	if w := call(t, a, owner, "/api/v1/orders", "slot-duration", invalidDuration); w.Code != http.StatusConflict {
		t.Fatalf("invalid duration accepted: %d", w.Code)
	}
}

func TestLegacyDatabaseMigratesToTimeSlots(t *testing.T) {
	path := filepath.Join(t.TempDir(), "legacy.db")
	db, err := sql.Open("sqlite", path)
	if err != nil {
		t.Fatal(err)
	}
	_, err = db.Exec(`CREATE TABLE orders(id TEXT PRIMARY KEY, table_id TEXT NOT NULL, date TEXT NOT NULL, status TEXT NOT NULL, body TEXT NOT NULL);
		CREATE UNIQUE INDEX occupied_table ON orders(table_id,date) WHERE status NOT IN ('completed','cancelled');`)
	if err != nil {
		t.Fatal(err)
	}
	legacy := order{ID: "legacy-order", TableID: "V01", Date: businessDate(), ArrivalTime: "22:30", Status: "reserved", CustomerName: "旧订单"}
	raw, _ := json.Marshal(legacy)
	if _, err = db.Exec("INSERT INTO orders VALUES (?,?,?,?,?)", legacy.ID, legacy.TableID, legacy.Date, legacy.Status, string(raw)); err != nil {
		t.Fatal(err)
	}
	db.Close()

	store, err := newClubStore(path, "legacy-code")
	if err != nil {
		t.Fatal(err)
	}
	defer store.db.Close()
	var start, end int
	if err = store.db.QueryRow("SELECT start_minute,end_minute FROM orders WHERE id=?", legacy.ID).Scan(&start, &end); err != nil || start != 22*60+30 || end != 22*60+30+240 {
		t.Fatalf("legacy schedule not backfilled: %d-%d err=%v", start, end, err)
	}
	var oldIndex int
	store.db.QueryRow("SELECT COUNT(*) FROM sqlite_master WHERE type='index' AND name='occupied_table'").Scan(&oldIndex)
	if oldIndex != 0 {
		t.Fatal("legacy all-day unique index remains")
	}
}

func TestBookingEditPermissionsConflictsAndAudit(t *testing.T) {
	a, owner := testClub(t)
	sales := staffToken(t, a, "sales")
	frontdesk := staffToken(t, a, "frontdesk")
	waiter := staffToken(t, a, "waiter")
	firstBody := bookingData()
	firstBody["arrivalTime"], firstBody["durationMinutes"] = "18:00", 120
	first := readOrder(t, call(t, a, sales, "/api/v1/orders", "edit-first", firstBody))
	secondBody := bookingData()
	secondBody["arrivalTime"], secondBody["durationMinutes"] = "20:00", 120
	second := readOrder(t, call(t, a, owner, "/api/v1/orders", "edit-second", secondBody))

	edit := editData(first)
	edit["customerName"], edit["remark"] = "修改后的客户", "靠近舞台"
	first = readOrder(t, call(t, a, sales, "/api/v1/orders/"+first.ID+"/update-booking", "edit-sales-own", edit))
	if first.CustomerName != "修改后的客户" || !strings.Contains(first.Events[len(first.Events)-1].Action, "客户资料") || first.Events[len(first.Events)-1].Operator != "销售小林" {
		t.Fatalf("edit or audit missing: %+v", first)
	}
	if w := call(t, a, waiter, "/api/v1/orders/"+first.ID+"/update-booking", "edit-waiter", editData(first)); w.Code != http.StatusForbidden {
		t.Fatalf("waiter edited booking: %d", w.Code)
	}
	if w := call(t, a, sales, "/api/v1/orders/"+second.ID+"/update-booking", "edit-sales-other", editData(second)); w.Code != http.StatusForbidden {
		t.Fatalf("sales edited another booking: %d", w.Code)
	}
	overlap := editData(first)
	overlap["durationMinutes"] = 240
	if w := call(t, a, frontdesk, "/api/v1/orders/"+first.ID+"/update-booking", "edit-overlap", overlap); w.Code != http.StatusConflict {
		t.Fatalf("overlap edit accepted: %d %s", w.Code, w.Body.String())
	}
	success := editData(first)
	success["guestCount"], success["durationMinutes"] = 5, 90
	first = readOrder(t, call(t, a, frontdesk, "/api/v1/orders/"+first.ID+"/update-booking", "edit-frontdesk", success))
	if first.GuestCount != 5 || first.DurationMinutes != 90 || first.Events[len(first.Events)-1].Operator != "前台小周" {
		t.Fatalf("frontdesk edit failed: %+v", first)
	}
}

func TestPartialRefundPermissionIdempotencyAndNetReport(t *testing.T) {
	a, owner := testClub(t)
	waiter := staffToken(t, a, "waiter")
	o := readOrder(t, call(t, a, owner, "/api/v1/orders", "refund-create", bookingData()))
	for i, action := range []string{"confirm-arrival", "open-table", "checkout"} {
		body := map[string]any{"version": o.Version}
		if action == "checkout" {
			body["paymentMethod"] = "支付宝"
		}
		o = readOrder(t, call(t, a, owner, "/api/v1/orders/"+o.ID+"/"+action, fmt.Sprintf("refund-step-%d", i), body))
	}
	if w := call(t, a, waiter, "/api/v1/orders/"+o.ID+"/refund", "refund-waiter", map[string]any{"version": o.Version, "refundReason": "运营异常"}); w.Code != http.StatusForbidden {
		t.Fatalf("waiter recorded refund: %d", w.Code)
	}
	if w := call(t, a, owner, "/api/v1/orders/"+o.ID+"/refund", "refund-no-reason", map[string]any{"version": o.Version}); w.Code != http.StatusConflict {
		t.Fatalf("refund without reason accepted: %d", w.Code)
	}
	o = readOrder(t, call(t, a, owner, "/api/v1/orders/"+o.ID+"/refund", "refund-owner", map[string]any{"version": o.Version, "refundReason": "客户投诉退款", "refundAmount": 1000}))
	if o.RefundedAmount != 1000 || o.RefundReason != "客户投诉退款" || o.RefundedAt == "" || len(o.Refunds) != 1 || o.Refunds[0].Amount != 1000 || !strings.Contains(o.Events[len(o.Events)-1].Action, "部分退款") {
		t.Fatalf("refund data missing: %+v", o)
	}
	replayed := readOrder(t, call(t, a, owner, "/api/v1/orders/"+o.ID+"/refund", "refund-owner", map[string]any{"version": o.Version - 1, "refundReason": "客户投诉退款", "refundAmount": 1000}))
	if replayed.Version != o.Version || replayed.RefundedAmount != o.RefundedAmount {
		t.Fatal("refund idempotency replay changed result")
	}
	path := "/api/v1/reports/daily?from=" + businessDate() + "&to=" + businessDate()
	var report struct {
		Totals reportDay `json:"totals"`
	}
	w := call(t, a, owner, path, "", nil)
	if w.Code != http.StatusOK || json.Unmarshal(w.Body.Bytes(), &report) != nil {
		t.Fatalf("refund report failed: %d %s", w.Code, w.Body.String())
	}
	if report.Totals.Revenue != o.PaidAmount-1000 || report.Totals.Refunded != 1 || report.Totals.RefundAmount != 1000 || report.Totals.Completed != 1 {
		t.Fatalf("wrong partial-refund report: %+v", report.Totals)
	}
	o = readOrder(t, call(t, a, owner, "/api/v1/orders/"+o.ID+"/refund", "refund-rest", map[string]any{"version": o.Version, "refundReason": "其他", "refundAmount": o.PaidAmount - o.RefundedAmount}))
	if o.RefundedAmount != o.PaidAmount || len(o.Refunds) != 2 || !strings.Contains(o.Events[len(o.Events)-1].Action, "全额退款") {
		t.Fatalf("full refund completion missing: %+v", o)
	}
	if w := call(t, a, owner, "/api/v1/orders/"+o.ID+"/refund", "refund-repeat", map[string]any{"version": o.Version, "refundReason": "其他", "refundAmount": 1}); w.Code != http.StatusConflict {
		t.Fatalf("over-refund accepted: %d", w.Code)
	}
	w = call(t, a, owner, path, "", nil)
	if w.Code != http.StatusOK || json.Unmarshal(w.Body.Bytes(), &report) != nil {
		t.Fatalf("full refund report failed: %d %s", w.Code, w.Body.String())
	}
	if report.Totals.Revenue != 0 || report.Totals.Refunded != 1 || report.Totals.RefundAmount != o.PaidAmount || report.Totals.Completed != 0 {
		t.Fatalf("wrong net report: %+v", report.Totals)
	}
	auditDate := time.Now().Format("2006-01-02")
	auditPath := "/api/v1/audit-logs?from=" + auditDate + "&to=" + auditDate + "&userId=owner&category=refund"
	var audit struct {
		Logs []auditLog `json:"logs"`
	}
	w = call(t, a, owner, auditPath, "", nil)
	if w.Code != http.StatusOK || json.Unmarshal(w.Body.Bytes(), &audit) != nil || len(audit.Logs) != 2 {
		t.Fatalf("filtered refund audit failed: %d %+v %s", w.Code, audit, w.Body.String())
	}
	if w := call(t, a, owner, "/api/v1/audit-logs?category=invalid", "", nil); w.Code != http.StatusBadRequest {
		t.Fatalf("invalid audit category accepted: %d", w.Code)
	}
	o = readOrder(t, call(t, a, owner, "/api/v1/orders/"+o.ID+"/complete-cleaning", "refund-clean", map[string]any{"version": o.Version}))
	if o.Status != "completed" {
		t.Fatal("refunded table could not be cleaned")
	}
}

func TestOrderPaginationSearchAndSalesScope(t *testing.T) {
	a, owner := testClub(t)
	sales := staffToken(t, a, "sales")
	for i := 0; i < 25; i++ {
		body := bookingData()
		body["tableId"] = tables[i%len(tables)].ID
		body["arrivalTime"] = map[bool]string{true: "18:00", false: "12:00"}[i >= len(tables)]
		body["durationMinutes"] = 120
		body["customerName"] = fmt.Sprintf("分页客户%02d", i)
		token := owner
		if i%2 == 0 {
			token = sales
		}
		readOrder(t, call(t, a, token, "/api/v1/orders", fmt.Sprintf("page-create-%02d", i), body))
	}
	var first struct {
		Orders     []order `json:"orders"`
		NextCursor string  `json:"nextCursor"`
	}
	w := call(t, a, owner, "/api/v1/orders?limit=10", "", nil)
	if w.Code != http.StatusOK || json.Unmarshal(w.Body.Bytes(), &first) != nil || len(first.Orders) != 10 || first.NextCursor == "" {
		t.Fatalf("first page failed: %d %+v %s", w.Code, first, w.Body.String())
	}
	var second struct {
		Orders     []order `json:"orders"`
		NextCursor string  `json:"nextCursor"`
	}
	w = call(t, a, owner, "/api/v1/orders?limit=10&cursor="+first.NextCursor, "", nil)
	if w.Code != http.StatusOK || json.Unmarshal(w.Body.Bytes(), &second) != nil || len(second.Orders) != 10 || second.Orders[0].ID == first.Orders[len(first.Orders)-1].ID {
		t.Fatalf("second page failed: %d %+v", w.Code, second)
	}
	var search struct{ Orders []order `json:"orders"` }
	w = call(t, a, owner, "/api/v1/orders?q="+"分页客户17", "", nil)
	if w.Code != http.StatusOK || json.Unmarshal(w.Body.Bytes(), &search) != nil || len(search.Orders) != 1 || search.Orders[0].CustomerName != "分页客户17" {
		t.Fatalf("indexed search failed: %d %+v", w.Code, search)
	}
	var scoped struct{ Orders []order `json:"orders"` }
	w = call(t, a, sales, "/api/v1/orders?limit=100", "", nil)
	if w.Code != http.StatusOK || json.Unmarshal(w.Body.Bytes(), &scoped) != nil || len(scoped.Orders) != 13 {
		t.Fatalf("sales page scope failed: %d count=%d", w.Code, len(scoped.Orders))
	}
	for _, o := range scoped.Orders {
		if o.CreatedBy != "sales" {
			t.Fatalf("sales saw foreign order %s", o.ID)
		}
	}
	if w := call(t, a, owner, "/api/v1/orders?limit=101", "", nil); w.Code != http.StatusBadRequest {
		t.Fatalf("oversized page accepted: %d", w.Code)
	}
}
