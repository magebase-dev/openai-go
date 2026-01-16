package skew

import (
	"testing"
)

func TestNewClient(t *testing.T) {
	client := NewClient("test-key")
	if client == nil {
		t.Fatal("Expected client to be created")
	}
}

func TestClientWithoutSkewKey(t *testing.T) {
	t.Setenv("SKEW_API_KEY", "")
	client := NewClient("test-key")
	if client == nil {
		t.Fatal("Expected client to work without SKEW_API_KEY")
	}
}
