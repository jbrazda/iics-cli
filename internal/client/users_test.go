package client

import (
	"context"
	"encoding/json"
	"net/http"
	"testing"
)

func TestListUsers(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("expected GET, got %s", r.Method)
		}
		users := []User{
			{ID: "u1", UserName: "alice", Email: "alice@example.com", State: "Active"},
			{ID: "u2", UserName: "bob", Email: "bob@example.com", State: "Active"},
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(users)
	})

	c := newTestClient(handler)
	users, err := c.ListUsers(context.Background(), UserListOptions{Limit: 10})
	if err != nil {
		t.Fatalf("ListUsers() error: %v", err)
	}
	if len(users) != 2 {
		t.Errorf("expected 2 users, got %d", len(users))
	}
	if users[0].UserName != "alice" {
		t.Errorf("expected 'alice', got %s", users[0].UserName)
	}
}

func TestGetUser(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("expected GET, got %s", r.Method)
		}
		if r.URL.Path != "/public/core/v3/users" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		users := []User{
			{ID: "u999", UserName: "bob", Email: "bob@example.com"},
			{ID: "u123", UserName: "alice", Email: "alice@example.com"},
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(users)
	})

	c := newTestClient(handler)
	user, err := c.GetUser(context.Background(), "u123")
	if err != nil {
		t.Fatalf("GetUser() error: %v", err)
	}
	if user.ID != "u123" {
		t.Errorf("expected ID u123, got %s", user.ID)
	}
	if user.UserName != "alice" {
		t.Errorf("expected userName alice, got %s", user.UserName)
	}
}

func TestGetUserNotFound(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode([]User{})
	})

	c := newTestClient(handler)
	_, err := c.GetUser(context.Background(), "missing")
	if err == nil {
		t.Fatal("expected error for missing user, got nil")
	}
}

func TestGetUserByName(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if q := r.URL.Query().Get("q"); q != "userName==Alice@example.com" {
			t.Errorf("expected q=userName==Alice@example.com, got %q", q)
		}
		users := []User{
			{ID: "u1", UserName: "alice@example.com"},
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(users)
	})

	c := newTestClient(handler)
	user, err := c.GetUserByName(context.Background(), "Alice@example.com")
	if err != nil {
		t.Fatalf("GetUserByName() error: %v", err)
	}
	if user.ID != "u1" {
		t.Errorf("expected ID u1, got %s", user.ID)
	}
}

func TestGetUserByNameNotFound(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode([]User{})
	})

	c := newTestClient(handler)
	_, err := c.GetUserByName(context.Background(), "nobody@example.com")
	if err == nil {
		t.Fatal("expected error for missing user, got nil")
	}
}

func TestSearchUsers(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		users := []User{
			{ID: "u1", UserName: "alice@example.com"},
			{ID: "u2", UserName: "bob@example.com", FirstName: "Bob", LastName: "Alison"},
			{ID: "u3", UserName: "a.smith@example.com", FirstName: "Alice", LastName: "Smith"},
			{ID: "u4", UserName: "carol", Email: "Carol.White@example.com"},
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(users)
	})

	c := newTestClient(handler)
	results, err := c.SearchUsers(context.Background(), "ALI")
	if err != nil {
		t.Fatalf("SearchUsers() error: %v", err)
	}
	if len(results) != 3 {
		t.Errorf("expected 3 results for 'ALI' (user name, last name, first name), got %d", len(results))
	}
	for q, want := range map[string]int{"alice smith": 1, "white@": 1, "zzz": 0} {
		got, _ := c.SearchUsers(context.Background(), q)
		if len(got) != want {
			t.Errorf("SearchUsers(%q) = %d results, want %d", q, len(got), want)
		}
	}
}

func TestCreateUser(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}
		var req createUserRequest
		json.NewDecoder(r.Body).Decode(&req)
		if req.Name != "newuser" {
			t.Errorf("expected name 'newuser', got %q", req.Name)
		}
		resp := User{ID: "new123", UserName: req.Name, Email: req.Email}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(resp)
	})

	c := newTestClient(handler)
	user, err := c.CreateUser(context.Background(), &User{UserName: "newuser", Email: "new@example.com"})
	if err != nil {
		t.Fatalf("CreateUser() error: %v", err)
	}
	if user.ID != "new123" {
		t.Errorf("expected ID new123, got %s", user.ID)
	}
}

func TestDeleteUser(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodDelete {
			t.Errorf("expected DELETE, got %s", r.Method)
		}
		if r.URL.Path != "/public/core/v3/users/u123" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		w.WriteHeader(http.StatusNoContent)
	})

	c := newTestClient(handler)
	if err := c.DeleteUser(context.Background(), "u123"); err != nil {
		t.Fatalf("DeleteUser() error: %v", err)
	}
}

func TestChangePasswordOwnPassword(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}
		if r.URL.Path != "/public/core/v3/Users/ChangePassword" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		var req ChangePasswordRequest
		json.NewDecoder(r.Body).Decode(&req)
		if req.NewPassword != "newpass" {
			t.Errorf("expected newPassword 'newpass', got %s", req.NewPassword)
		}
		if req.OldPassword != "oldpass" {
			t.Errorf("expected oldPassword 'oldpass', got %s", req.OldPassword)
		}
		if req.UserID != "" {
			t.Errorf("expected no userId, got %s", req.UserID)
		}
		w.WriteHeader(http.StatusOK)
	})

	c := newTestClient(handler)
	err := c.ChangePassword(context.Background(), &ChangePasswordRequest{
		NewPassword: "newpass",
		OldPassword: "oldpass",
	})
	if err != nil {
		t.Fatalf("ChangePassword() error: %v", err)
	}
}

func TestChangePasswordAdminChange(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}
		var req ChangePasswordRequest
		json.NewDecoder(r.Body).Decode(&req)
		if req.NewPassword != "newpass" {
			t.Errorf("expected newPassword 'newpass', got %s", req.NewPassword)
		}
		if req.UserID != "u999" {
			t.Errorf("expected userId 'u999', got %s", req.UserID)
		}
		if req.OldPassword != "" {
			t.Errorf("expected no oldPassword, got %s", req.OldPassword)
		}
		w.WriteHeader(http.StatusOK)
	})

	c := newTestClient(handler)
	err := c.ChangePassword(context.Background(), &ChangePasswordRequest{
		NewPassword: "newpass",
		UserID:      "u999",
	})
	if err != nil {
		t.Fatalf("ChangePassword() admin error: %v", err)
	}
}

func TestResetPassword(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}
		if r.URL.Path != "/public/core/v3/Users/ResetPassword" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		var req ResetPasswordRequest
		json.NewDecoder(r.Body).Decode(&req)
		if req.UserID != "u123" {
			t.Errorf("expected userId 'u123', got %s", req.UserID)
		}
		if req.SecurityAnswer != "Simba" {
			t.Errorf("expected securityAnswer 'Simba', got %s", req.SecurityAnswer)
		}
		if req.NewPassword != "newpass" {
			t.Errorf("expected newPassword 'newpass', got %s", req.NewPassword)
		}
		w.WriteHeader(http.StatusOK)
	})

	c := newTestClient(handler)
	err := c.ResetPassword(context.Background(), &ResetPasswordRequest{
		UserID:         "u123",
		SecurityAnswer: "Simba",
		NewPassword:    "newpass",
	})
	if err != nil {
		t.Fatalf("ResetPassword() error: %v", err)
	}
}
