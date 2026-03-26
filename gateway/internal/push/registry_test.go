package push

import (
	"testing"
)

func TestRegistry_RegisterAndCount(t *testing.T) {
	r := NewConnectionRegistry()
	r.Register("dev-1", nil, map[string]string{"model": "Pixel 7"})
	r.Register("dev-2", nil, map[string]string{"model": "Samsung"})

	if r.ConnectedCount() != 2 {
		t.Errorf("Expected 2 connected, got %d", r.ConnectedCount())
	}
}

func TestRegistry_Unregister(t *testing.T) {
	r := NewConnectionRegistry()
	r.Register("dev-1", nil, nil)
	r.Unregister("dev-1")

	if r.ConnectedCount() != 0 {
		t.Error("Expected 0 after unregister")
	}
}

func TestRegistry_GetConnection_Exists(t *testing.T) {
	r := NewConnectionRegistry()
	r.Register("dev-1", nil, nil)

	conn, ok := r.GetConnection("dev-1")
	if !ok {
		t.Error("Expected connection to exist")
	}
	_ = conn
}

func TestRegistry_GetConnection_Missing(t *testing.T) {
	r := NewConnectionRegistry()

	_, ok := r.GetConnection("nonexistent")
	if ok {
		t.Error("Expected connection to not exist")
	}
}

func TestRegistry_GetConnectedDeviceIDs(t *testing.T) {
	r := NewConnectionRegistry()
	r.Register("dev-1", nil, nil)
	r.Register("dev-2", nil, nil)
	r.Register("dev-3", nil, nil)

	ids := r.GetConnectedDeviceIDs([]string{"dev-1", "dev-3", "dev-5"})
	if len(ids) != 2 {
		t.Errorf("Expected 2 connected from filter, got %d", len(ids))
	}
}
