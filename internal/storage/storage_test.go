package storage

import (
	"testing"
)

func TestNew(t *testing.T) {
	tests := []struct {
		name        string
		storageType string
		wantErr     bool
		wantType    string
	}{
		{
			name:        "keychain storage",
			storageType: "keychain",
			wantErr:     false,
			wantType:    "*storage.KeychainStorage",
		},
		{
			name:        "gopass storage",
			storageType: "gopass",
			wantErr:     false,
			wantType:    "*storage.GopassStorage",
		},
		{
			name:        "invalid storage type",
			storageType: "invalid",
			wantErr:     true,
		},
		{
			name:        "empty storage type",
			storageType: "",
			wantErr:     true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := New(tt.storageType)

			if (err != nil) != tt.wantErr {
				t.Errorf("New() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if tt.wantErr {
				if got != nil {
					t.Errorf("New() returned non-nil storage on error: %v", got)
				}
				return
			}

			if got == nil {
				t.Fatal("New() returned nil storage")
			}

			// Type assertion to check correct type
			switch tt.storageType {
			case "keychain":
				if _, ok := got.(*KeychainStorage); !ok {
					t.Errorf("New(%q) returned wrong type: %T", tt.storageType, got)
				}
			case "gopass":
				if _, ok := got.(*GopassStorage); !ok {
					t.Errorf("New(%q) returned wrong type: %T", tt.storageType, got)
				}
			}
		})
	}
}

func TestStorageInterface(t *testing.T) {
	// Verify that our storage implementations satisfy the Storage interface
	var _ Storage = (*KeychainStorage)(nil)
	var _ Storage = (*GopassStorage)(nil)
	var _ Storage = (*MockStorage)(nil)
}

func TestMockStorage(t *testing.T) {
	t.Run("store and retrieve", func(t *testing.T) {
		mock := NewMockStorage()

		// Store a value
		err := mock.Store("test-cluster", "token", `{"token":"test123"}`)
		if err != nil {
			t.Fatalf("Store() error = %v", err)
		}

		// Retrieve the value
		value, err := mock.Get("test-cluster", "token")
		if err != nil {
			t.Fatalf("Get() error = %v", err)
		}

		if value != `{"token":"test123"}` {
			t.Errorf("Get() = %q, want %q", value, `{"token":"test123"}`)
		}
	})

	t.Run("get non-existent key", func(t *testing.T) {
		mock := NewMockStorage()

		value, err := mock.Get("test-cluster", "nonexistent")
		if err != nil {
			t.Fatalf("Get() error = %v", err)
		}

		if value != "" {
			t.Errorf("Get() = %q, want empty string", value)
		}
	})

	t.Run("call counts", func(t *testing.T) {
		mock := NewMockStorage()

		mock.Store("cluster1", "key1", "value1")
		mock.Store("cluster1", "key2", "value2")
		mock.Get("cluster1", "key1")
		mock.Get("cluster1", "key2")
		mock.Get("cluster1", "key3")

		if mock.StoreCallCount() != 2 {
			t.Errorf("StoreCallCount() = %d, want 2", mock.StoreCallCount())
		}

		if mock.GetCallCount() != 3 {
			t.Errorf("GetCallCount() = %d, want 3", mock.GetCallCount())
		}
	})

	t.Run("clear", func(t *testing.T) {
		mock := NewMockStorage()

		mock.Store("cluster1", "key1", "value1")
		mock.Get("cluster1", "key1")

		mock.Clear()

		if mock.StoreCallCount() != 0 {
			t.Errorf("StoreCallCount() after clear = %d, want 0", mock.StoreCallCount())
		}

		if mock.GetCallCount() != 0 {
			t.Errorf("GetCallCount() after clear = %d, want 0", mock.GetCallCount())
		}

		value, _ := mock.Get("cluster1", "key1")
		if value != "" {
			t.Errorf("Get() after clear = %q, want empty", value)
		}
	})
}
