package components

import "testing"

func TestVisibleColumnsKeepLowestPriorityNumbers(t *testing.T) {
	table := Table{Columns: []Column{
		{Title: "Status", MinWidth: 12, Priority: 0},
		{Title: "Application", MinWidth: 16, Priority: 0},
		{Title: "Branch", MinWidth: 12, Priority: 2},
		{Title: "Domain", MinWidth: 20, Priority: 4},
	}}

	_, indexes := table.visibleColumns(42)
	if len(indexes) != 3 {
		t.Fatalf("visible column indexes = %v", indexes)
	}
	for _, want := range []int{0, 1} {
		if !containsIndex(indexes, want) {
			t.Fatalf("visible column indexes = %v, missing priority-0 column %d", indexes, want)
		}
	}
	if containsIndex(indexes, 3) {
		t.Fatalf("visible column indexes = %v, kept weakest column", indexes)
	}
}

func containsIndex(indexes []int, target int) bool {
	for _, index := range indexes {
		if index == target {
			return true
		}
	}
	return false
}
