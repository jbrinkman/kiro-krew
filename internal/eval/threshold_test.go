package eval

import (
	"testing"
)

func TestGetThreshold(t *testing.T) {
	// Test default threshold (no min_score specified)
	tc1 := TestCase{
		Name: "default-test",
	}
	threshold1 := getThreshold(tc1)
	if threshold1 != 80.0 {
		t.Errorf("Expected default threshold 80.0, got %f", threshold1)
	}

	// Test custom threshold
	customScore := 90.0
	tc2 := TestCase{
		Name:     "custom-test",
		MinScore: &customScore,
	}
	threshold2 := getThreshold(tc2)
	if threshold2 != 90.0 {
		t.Errorf("Expected custom threshold 90.0, got %f", threshold2)
	}

	// Test another custom threshold
	lowScore := 60.0
	tc3 := TestCase{
		Name:     "low-threshold-test",
		MinScore: &lowScore,
	}
	threshold3 := getThreshold(tc3)
	if threshold3 != 60.0 {
		t.Errorf("Expected custom threshold 60.0, got %f", threshold3)
	}
}
