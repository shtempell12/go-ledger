// Команда ledger проверяет подключения к PostgreSQL и Redis, при необходимости загружает
// бюджеты из JSON-файла и печатает сохранённые данные. HTTP API находится в Gateway.
package main

import (
	"bufio"
	"fmt"
	"log"
	"os"

	"github.com/shtempell12/go-ledger/ledger"
)

func main() {
	if err := ledger.Init(); err != nil {
		log.Fatalf("ledger: %v", err)
	}
	defer ledger.Close()

	if len(os.Args) > 1 {
		if err := loadBudgetsFromFile(os.Args[1]); err != nil {
			log.Fatalf("ledger: %v", err)
		}
	}

	budgets, err := ledger.ListBudgets()
	if err != nil {
		log.Fatalf("ledger: %v", err)
	}
	fmt.Println("Бюджеты:")
	for _, b := range budgets {
		fmt.Printf("  %-12s %10.2f\n", b.Category, b.Limit)
	}

	transactions, err := ledger.ListTransactions()
	if err != nil {
		log.Fatalf("ledger: %v", err)
	}
	fmt.Println("Транзакции:")
	for _, tx := range transactions {
		fmt.Printf("  #%d %s %-12s %10.2f  %s\n", tx.ID, tx.Date.Format("2006-01-02"), tx.Category, tx.Amount, tx.Description)
	}
}

func loadBudgetsFromFile(path string) error {
	f, err := os.Open(path)
	if err != nil {
		return fmt.Errorf("open budgets file: %w", err)
	}
	defer f.Close()

	return ledger.LoadBudgets(bufio.NewReader(f))
}
