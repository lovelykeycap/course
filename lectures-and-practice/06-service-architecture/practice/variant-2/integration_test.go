package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"sync"
	"testing"
	"time"
)

type externalCall struct {
	Kind        string
	Method      string
	Path        string
	ContentType string
	Body        map[string]any
}

type callLog struct {
	mu    sync.Mutex
	calls []externalCall
}

func (l *callLog) add(call externalCall) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.calls = append(l.calls, call)
}

func (l *callLog) snapshot() []externalCall {
	l.mu.Lock()
	defer l.mu.Unlock()
	return append([]externalCall(nil), l.calls...)
}

type testApp struct {
	url    string
	client *http.Client
	calls  *callLog
}

func startTestApp(t *testing.T, availabilityStatus, notifyStatus int) *testApp {
	t.Helper()
	log := &callLog{}
	dependency := func(kind string, status int) *httptest.Server {
		s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			var body map[string]any
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
				t.Errorf("%s received invalid JSON: %v", kind, err)
			}
			log.add(externalCall{Kind: kind, Method: r.Method, Path: r.URL.Path, ContentType: r.Header.Get("Content-Type"), Body: body})
			if status == 0 {
				conn, _, err := w.(http.Hijacker).Hijack()
				if err != nil {
					t.Errorf("hijack: %v", err)
					return
				}
				conn.Close()
				return
			}
			w.WriteHeader(status)
		}))
		t.Cleanup(s.Close)
		return s
	}
	availability := dependency("availability", availabilityStatus)
	notifier := dependency("notify", notifyStatus)
	transport := &http.Transport{Proxy: nil}
	t.Cleanup(transport.CloseIdleConnections)
	client := &http.Client{Transport: transport, Timeout: 5 * time.Second}
	server := httptest.NewServer(newTestHandler(availability.URL+"/check", notifier.URL+"/send", client))
	t.Cleanup(server.Close)
	return &testApp{url: server.URL, client: client, calls: log}
}

func (a *testApp) request(t *testing.T, method, path, body string, wantStatus int) (http.Header, []byte) {
	t.Helper()
	req, err := http.NewRequest(method, a.url+path, strings.NewReader(body))
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := a.client.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	data, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != wantStatus {
		t.Fatalf("%s %s: got %d %q, want %d", method, path, resp.StatusCode, data, wantStatus)
	}
	return resp.Header, data
}

func decodeObject(t *testing.T, data []byte) map[string]any {
	t.Helper()
	var result map[string]any
	if err := json.Unmarshal(data, &result); err != nil {
		t.Fatalf("invalid JSON %q: %v", data, err)
	}
	return result
}

type bookingView struct {
	ID         int64
	RoomID     int64
	EmployeeID int64
	StartAt    time.Time
	EndAt      time.Time
	Status     string
	CreatedAt  time.Time
}

func decodeBooking(t *testing.T, data []byte) bookingView {
	t.Helper()
	fields := decodeObject(t, data)
	for _, name := range []string{"ID", "RoomID", "EmployeeID", "StartAt", "EndAt", "Status", "CreatedAt"} {
		if _, ok := fields[name]; !ok {
			t.Fatalf("missing %s in %s", name, data)
		}
	}
	if len(fields) != 7 {
		t.Fatalf("unexpected fields: %s", data)
	}
	var result bookingView
	if err := json.Unmarshal(data, &result); err != nil {
		t.Fatal(err)
	}
	return result
}

func createBody(roomID, employeeID int64, start, end string) string {
	data, _ := json.Marshal(map[string]any{"room_id": roomID, "employee_id": employeeID, "start_at": start, "end_at": end})
	return string(data)
}

func timeBody(start, end string) string {
	data, _ := json.Marshal(map[string]string{"start_at": start, "end_at": end})
	return string(data)
}

func (a *testApp) create(t *testing.T, roomID, employeeID int64, start, end, wantStatus string) int64 {
	t.Helper()
	_, data := a.request(t, "POST", "/bookings", createBody(roomID, employeeID, start, end), 201)
	result := decodeObject(t, data)
	if len(result) != 2 || result["status"] != wantStatus {
		t.Fatalf("unexpected create response: %s", data)
	}
	id, ok := result["id"].(float64)
	if !ok || id <= 0 || float64(int64(id)) != id {
		t.Fatalf("invalid booking id: %s", data)
	}
	return int64(id)
}

func (a *testApp) get(t *testing.T, id int64) bookingView {
	t.Helper()
	header, data := a.request(t, "GET", fmt.Sprintf("/bookings/%d", id), "", 200)
	if header.Get("Content-Type") != "application/json" {
		t.Fatalf("unexpected content type: %v", header)
	}
	return decodeBooking(t, data)
}

func (a *testApp) reschedule(t *testing.T, id int64, start, end string) bookingView {
	t.Helper()
	header, data := a.request(t, "PATCH", fmt.Sprintf("/bookings/%d/time", id), timeBody(start, end), 200)
	if header.Get("Content-Type") != "application/json" {
		t.Fatalf("unexpected content type: %v", header)
	}
	return decodeBooking(t, data)
}

func assertBody(t *testing.T, got []byte, want string) {
	t.Helper()
	if string(got) != want {
		t.Fatalf("got body %q, want %q", got, want)
	}
}

func TestIntegrationCreateDependencies(t *testing.T) {
	for _, availability := range []struct {
		name   string
		status int
	}{
		{"available", 200}, {"other success", 204}, {"busy", 409}, {"server failure", 500}, {"connection failure", 0},
	} {
		for _, notify := range []struct {
			name   string
			status int
		}{
			{"sent", 200}, {"failed response", 500}, {"connection failure", 0},
		} {
			t.Run(availability.name+"/"+notify.name, func(t *testing.T) {
				a := startTestApp(t, availability.status, notify.status)
				status := "pending_confirmation"
				if availability.status == 200 {
					status = "confirmed"
				}
				start, end := "2026-11-01T10:00:00+03:00", "2026-11-01T11:00:00+03:00"
				before := time.Now()
				id := a.create(t, 7, 42, start, end, status)
				after := time.Now()
				b := a.get(t, id)
				startAt, _ := time.Parse(time.RFC3339, start)
				endAt, _ := time.Parse(time.RFC3339, end)
				if id != 1 || b.ID != id || b.RoomID != 7 || b.EmployeeID != 42 || b.Status != status || !b.StartAt.Equal(startAt) || !b.EndAt.Equal(endAt) {
					t.Fatalf("unexpected saved booking: %+v", b)
				}
				if b.CreatedAt.Before(before) || b.CreatedAt.After(after) {
					t.Fatalf("unexpected creation time: %s", b.CreatedAt)
				}
				want := []externalCall{
					{Kind: "availability", Method: "POST", Path: "/check", ContentType: "application/json", Body: map[string]any{"room_id": float64(7), "start_at": start, "end_at": end}},
					{Kind: "notify", Method: "POST", Path: "/send", ContentType: "application/json", Body: map[string]any{"booking_id": float64(id)}},
				}
				if got := a.calls.snapshot(); !reflect.DeepEqual(got, want) {
					t.Fatalf("external calls: got %+v, want %+v", got, want)
				}
			})
		}
	}
}

func TestIntegrationValidation(t *testing.T) {
	start, end := "2026-11-01T10:00:00Z", "2026-11-01T11:00:00Z"
	cases := []struct{ name, body, message string }{
		{"empty", "", ""},
		{"malformed", "{", ""},
		{"wrong timestamp type", `{"start_at":123}`, ""},
		{"wrong id type", `{"room_id":"one"}`, ""},
		{"missing start", `{}`, "start_at must be a valid timestamp\n"},
		{"invalid start", timeBody("bad", end), "start_at must be a valid timestamp\n"},
		{"missing end", `{"start_at":"2026-11-01T10:00:00Z"}`, "end_at must be a valid timestamp\n"},
		{"invalid end", timeBody(start, "bad"), "end_at must be a valid timestamp\n"},
		{"equal", timeBody(start, start), "end_at must be after start_at\n"},
		{"reversed", timeBody(end, start), "end_at must be after start_at\n"},
	}
	for _, method := range []string{"POST", "PATCH"} {
		for _, tt := range cases {
			if method == "PATCH" && tt.name == "wrong id type" {
				continue
			}
			t.Run(method+"/"+tt.name, func(t *testing.T) {
				a := startTestApp(t, 200, 200)
				path := "/bookings"
				var before bookingView
				calls := 0
				if method == "PATCH" {
					id := a.create(t, 1, 1, start, end, "confirmed")
					before = a.get(t, id)
					path = fmt.Sprintf("/bookings/%d/time", id)
					calls = 2
				}
				_, body := a.request(t, method, path, tt.body, 400)
				assertBody(t, body, tt.message)
				if len(a.calls.snapshot()) != calls {
					t.Fatal("invalid request called dependencies")
				}
				if method == "PATCH" && a.get(t, before.ID) != before {
					t.Fatal("invalid request changed booking")
				}
				id := a.create(t, 2, 1, start, end, "confirmed")
				if id != int64(calls/2+1) {
					t.Fatal("invalid request consumed an id")
				}
			})
		}
	}
}

func TestIntegrationLookupAndValidationOrder(t *testing.T) {
	a := startTestApp(t, 200, 200)
	valid := timeBody("2026-11-01T10:00:00Z", "2026-11-01T11:00:00Z")
	for _, method := range []string{"GET", "PATCH"} {
		for _, id := range []string{"unknown", "9223372036854775808", "0", "-1", "999"} {
			path := "/bookings/" + id
			if method == "PATCH" {
				path += "/time"
			}
			status := 404
			if id == "unknown" || id == "9223372036854775808" {
				status = 400
			}
			_, body := a.request(t, method, path, valid, status)
			assertBody(t, body, "")
		}
	}
	_, body := a.request(t, "PATCH", "/bookings/999/time", `{}`, 400)
	assertBody(t, body, "start_at must be a valid timestamp\n")
	_, body = a.request(t, "PATCH", "/bookings/invalid/time", `{}`, 400)
	assertBody(t, body, "")
	if len(a.calls.snapshot()) != 0 {
		t.Fatal("lookup called dependencies")
	}
}

func TestIntegrationOverlaps(t *testing.T) {
	cases := []struct {
		name               string
		roomID, employeeID int64
		start, end         string
		conflict, cancels  bool
	}{
		{"same interval", 1, 1, "10:00", "11:00", true, false},
		{"overlaps start", 1, 1, "09:30", "10:30", true, false},
		{"overlaps end", 1, 1, "10:30", "11:30", true, false},
		{"inside", 1, 1, "10:15", "10:45", true, false},
		{"contains", 1, 1, "09:00", "12:00", true, false},
		{"unknown employee", 1, 99, "10:00", "11:00", true, false},
		{"zero employee", 1, 0, "10:00", "11:00", true, false},
		{"adjacent before", 1, 1, "09:00", "10:00", false, false},
		{"adjacent after", 1, 1, "11:00", "12:00", false, false},
		{"different room", 2, 1, "10:00", "11:00", false, false},
		{"admin", 1, 2, "10:00", "11:00", false, true},
	}
	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			a := startTestApp(t, 200, 200)
			id := a.create(t, 1, 1, "2026-11-01T10:00:00Z", "2026-11-01T11:00:00Z", "confirmed")
			before := a.get(t, id)
			start, end := "2026-11-01T"+tt.start+":00Z", "2026-11-01T"+tt.end+":00Z"
			if tt.conflict {
				_, body := a.request(t, "POST", "/bookings", createBody(tt.roomID, tt.employeeID, start, end), 409)
				assertBody(t, body, "room is already booked for this time\n")
				if a.get(t, id) != before {
					t.Fatal("conflict changed booking")
				}
				if len(a.calls.snapshot()) != 2 {
					t.Fatal("conflict called dependencies")
				}
				next := a.create(t, 3, 1, start, end, "confirmed")
				if next != 2 {
					t.Fatal("conflict consumed an id")
				}
			} else {
				next := a.create(t, tt.roomID, tt.employeeID, start, end, "confirmed")
				if next != 2 {
					t.Fatalf("unexpected id %d", next)
				}
				if tt.cancels {
					before.Status = "cancelled"
				}
				if a.get(t, id) != before {
					t.Fatal("unexpected original booking change")
				}
			}
		})
	}
}

func TestIntegrationAdminCancelsOneOverlap(t *testing.T) {
	a := startTestApp(t, 200, 200)
	ids := []int64{
		a.create(t, 1, 1, "2026-11-01T09:00:00Z", "2026-11-01T10:00:00Z", "confirmed"),
		a.create(t, 1, 1, "2026-11-01T10:00:00Z", "2026-11-01T11:00:00Z", "confirmed"),
	}
	a.create(t, 1, 2, "2026-11-01T09:30:00Z", "2026-11-01T10:30:00Z", "confirmed")
	cancelled := 0
	for _, id := range ids {
		if a.get(t, id).Status == "cancelled" {
			cancelled++
		}
	}
	if cancelled != 1 {
		t.Fatalf("cancelled %d bookings, want 1", cancelled)
	}
}

func TestIntegrationCancelledBooking(t *testing.T) {
	a := startTestApp(t, 200, 200)
	id := a.create(t, 1, 1, "2026-11-01T09:00:00Z", "2026-11-01T11:00:00Z", "confirmed")
	a.create(t, 1, 2, "2026-11-01T10:00:00Z", "2026-11-01T11:00:00Z", "confirmed")
	before := a.get(t, id)
	if before.Status != "cancelled" {
		t.Fatal("admin did not cancel booking")
	}
	_, body := a.request(t, "PATCH", fmt.Sprintf("/bookings/%d/time", id), timeBody("2026-11-02T09:00:00Z", "2026-11-02T10:00:00Z"), 409)
	assertBody(t, body, "cannot reschedule cancelled booking\n")
	if a.get(t, id) != before {
		t.Fatal("cancelled booking changed")
	}
	_, body = a.request(t, "PATCH", fmt.Sprintf("/bookings/%d/time", id), `{}`, 400)
	assertBody(t, body, "start_at must be a valid timestamp\n")
	if len(a.calls.snapshot()) != 4 {
		t.Fatal("rescheduling called dependencies")
	}
	a.create(t, 1, 1, "2026-11-01T09:00:00Z", "2026-11-01T10:00:00Z", "confirmed")
}

func TestIntegrationRescheduling(t *testing.T) {
	cases := []struct {
		initial string
		offset  time.Duration
		want    string
	}{
		{"confirmed", time.Hour, "rescheduling"},
		{"confirmed", 3 * time.Hour, "confirmed"},
		{"confirmed", -time.Hour, "rescheduling"},
		{"pending_confirmation", time.Hour, "pending_confirmation"},
		{"pending_confirmation", 3 * time.Hour, "pending_confirmation"},
		{"rescheduling", time.Hour, "rescheduling"},
		{"rescheduling", 3 * time.Hour, "rescheduling"},
	}
	for _, tt := range cases {
		t.Run(fmt.Sprintf("%s/%s", tt.initial, tt.offset), func(t *testing.T) {
			availability := 200
			initial := "confirmed"
			if tt.initial == "pending_confirmation" {
				availability = 409
				initial = tt.initial
			}
			a := startTestApp(t, availability, 200)
			base := time.Now().UTC().Truncate(time.Second)
			stamp := func(offset time.Duration) string { return base.Add(offset).Format(time.RFC3339) }
			id := a.create(t, 1, 1, stamp(24*time.Hour), stamp(25*time.Hour), initial)
			if tt.initial == "rescheduling" {
				a.reschedule(t, id, stamp(time.Hour), stamp(2*time.Hour))
			}
			before := a.get(t, id)
			if before.Status != tt.initial {
				t.Fatalf("unexpected initial status %s", before.Status)
			}
			after := a.reschedule(t, id, stamp(tt.offset), stamp(tt.offset+time.Hour))
			if after.Status != tt.want || !after.StartAt.Equal(base.Add(tt.offset)) || !after.EndAt.Equal(base.Add(tt.offset+time.Hour)) {
				t.Fatalf("unexpected reschedule: %+v", after)
			}
			if after.ID != before.ID || after.RoomID != before.RoomID || after.EmployeeID != before.EmployeeID || !after.CreatedAt.Equal(before.CreatedAt) {
				t.Fatal("rescheduling changed unrelated fields")
			}
			if a.get(t, id) != after {
				t.Fatal("reschedule was not saved")
			}
			if len(a.calls.snapshot()) != 2 {
				t.Fatal("rescheduling called dependencies")
			}
		})
	}
}

func TestIntegrationRescheduleAllowsOverlap(t *testing.T) {
	a := startTestApp(t, 200, 200)
	base := time.Now().UTC().Truncate(time.Second).Add(24 * time.Hour)
	stamp := func(hours int) string { return base.Add(time.Duration(hours) * time.Hour).Format(time.RFC3339) }
	first := a.create(t, 1, 1, stamp(0), stamp(1), "confirmed")
	second := a.create(t, 1, 1, stamp(2), stamp(3), "confirmed")
	before := a.get(t, first)
	updated := a.reschedule(t, second, stamp(0), stamp(1))
	if updated.Status != "confirmed" || a.get(t, first) != before {
		t.Fatal("unexpected overlap handling during reschedule")
	}
	if len(a.calls.snapshot()) != 4 {
		t.Fatal("rescheduling called dependencies")
	}
}

func TestIntegrationPermissiveInputs(t *testing.T) {
	a := startTestApp(t, 200, 200)
	id := a.create(t, -1, -10, "2000-01-01T10:00:00Z", "2000-01-01T11:00:00Z", "confirmed")
	b := a.get(t, id)
	if b.RoomID != -1 || b.EmployeeID != -10 {
		t.Fatal("ids changed")
	}
	body := `{"start_at":"2000-01-01T10:00:00Z","end_at":"2000-01-01T11:00:00Z","unknown":true} {"ignored":true}`
	_, data := a.request(t, "POST", "/bookings", body, 201)
	created := decodeObject(t, data)
	if created["status"] != "confirmed" {
		t.Fatalf("unexpected response: %s", data)
	}
	b = a.get(t, int64(created["id"].(float64)))
	if b.RoomID != 0 || b.EmployeeID != 0 {
		t.Fatal("missing ids must default to zero")
	}
}

func TestIntegrationActiveStatusesBlockBooking(t *testing.T) {
	for _, status := range []string{"confirmed", "pending_confirmation", "rescheduling"} {
		t.Run(status, func(t *testing.T) {
			availability, initial := 200, "confirmed"
			if status == "pending_confirmation" {
				availability, initial = 409, status
			}
			a := startTestApp(t, availability, 200)
			base := time.Now().UTC().Truncate(time.Second)
			start, end := base.Add(time.Hour).Format(time.RFC3339), base.Add(2*time.Hour).Format(time.RFC3339)
			id := a.create(t, 1, 1, start, end, initial)
			if status == "rescheduling" {
				a.reschedule(t, id, start, end)
			}
			before := a.get(t, id)
			if before.Status != status {
				t.Fatalf("got %s, want %s", before.Status, status)
			}
			_, body := a.request(t, "POST", "/bookings", createBody(1, 1, start, end), 409)
			assertBody(t, body, "room is already booked for this time\n")
			if len(a.calls.snapshot()) != 2 || a.get(t, id) != before {
				t.Fatal("conflict changed state or called dependencies")
			}
			a.create(t, 1, 2, start, end, initial)
			before.Status = "cancelled"
			if a.get(t, id) != before {
				t.Fatal("admin did not cancel the existing booking")
			}
		})
	}
}

func TestIntegrationAdminCanReplaceOwnBooking(t *testing.T) {
	a := startTestApp(t, 200, 200)
	start, end := "2026-11-01T10:00:00Z", "2026-11-01T11:00:00Z"
	id := a.create(t, 1, 2, start, end, "confirmed")
	a.create(t, 1, 2, start, end, "confirmed")
	if a.get(t, id).Status != "cancelled" {
		t.Fatal("admin's previous booking was not cancelled")
	}
}

func TestIntegrationTimezoneOverlap(t *testing.T) {
	a := startTestApp(t, 200, 200)
	a.create(t, 1, 1, "2026-11-01T10:00:00+03:00", "2026-11-01T11:00:00+03:00", "confirmed")
	_, body := a.request(t, "POST", "/bookings", createBody(1, 1, "2026-11-01T07:30:00Z", "2026-11-01T08:30:00Z"), 409)
	assertBody(t, body, "room is already booked for this time\n")
	if len(a.calls.snapshot()) != 2 {
		t.Fatal("conflict called dependencies")
	}
}
