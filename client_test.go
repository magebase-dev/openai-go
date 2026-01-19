package langmesh

import (
	"testing"
)

func TestNewClient(t *testing.T) {
	client := NewClient("test-key")
	if client == nil {
		t.Fatal("Expected client to be created")
	}
}

func TestClientWithoutlangmeshKey(t *testing.T) {
	t.Setenv("langmesh_API_KEY", "")
	client := NewClient("test-key")
	if client == nil {
		t.Fatal("Expected client to work without langmesh_API_KEY")
	}
}
