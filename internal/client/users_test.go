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
		if r.URL.Path != "/public/core/v3/users/u123" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		user := User{ID: "u123", UserName: "alice", Email: "alice@example.com"}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(user)
	})

	c := newTestClient(handler)
	user, err := c.GetUser(context.Background(), "u123")
	if err != nil {
		t.Fatalf("GetUser() error: %v", err)
	}
	if user.ID != "u123" {
		t.Errorf("expected ID u123, got %s", user.ID)
	}
}

func TestCreateUser(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}
		var u User
		json.NewDecoder(r.Body).Decode(&u)
		if u.UserName != "newuser" {
			t.Errorf("expected userName 'newuser', got %s", u.UserName)
		}
		u.ID = "new123"
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(u)
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

func TestUpdateUser(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPut {
			t.Errorf("expected PUT, got %s", r.Method)
		}
		var u User
		json.NewDecoder(r.Body).Decode(&u)
		u.ID = "u123"
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(u)
	})

	c := newTestClient(handler)
	user, err := c.UpdateUser(context.Background(), "u123", &User{Email: "updated@example.com"})
	if err != nil {
		t.Fatalf("UpdateUser() error: %v", err)
	}
	if user.ID != "u123" {
		t.Errorf("expected ID u123, got %s", user.ID)
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
