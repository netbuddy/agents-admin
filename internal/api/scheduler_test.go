package api

import (
	"testing"
)

func TestScheduler_SelectNode(t *testing.T) {
	s := &Scheduler{}

	// Test with empty nodes
	node := s.selectNode(nil)
	if node != nil {
		t.Error("Expected nil for empty nodes")
	}

	// Test with nodes - should return first one (simple strategy)
	// More complex tests would require mocking the storage layer
}

func TestNewScheduler(t *testing.T) {
	s := NewScheduler(nil, nil)
	if s == nil {
		t.Error("Expected non-nil scheduler")
	}

	if s.stopCh == nil {
		t.Error("Expected stopCh to be initialized")
	}
}
