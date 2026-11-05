package ledger

import "testing"

// reset очищает хранилище перед тестом.
func reset(t *testing.T) {
	t.Helper()
	transactions = nil
	budgets = make(map[string]Budget)
	nextID = 1
}
