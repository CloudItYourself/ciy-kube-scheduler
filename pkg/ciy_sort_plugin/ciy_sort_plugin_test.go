package ciy_sort_plugin

import (
	"testing"
)

func TestDummyTest(t *testing.T) {
	sortingObject := CiySortPlugin{}
	score, _ := sortingObject.Score(nil, nil, nil, "")
	if score != 100 {
		t.Errorf("expected 100, got %d", score)
	}
}
