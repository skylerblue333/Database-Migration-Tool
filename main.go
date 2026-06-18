package main

import (
	"fmt"
	"log"
	"os"
	"sort"
	"strings"
)

// Mock Migration represents a database schema change
type Migration struct {
	ID          string
	Description string
	Applied     bool
}

type MigrationManager struct {
	migrations map[string]*Migration
}

func NewMigrationManager() *MigrationManager {
	mm := &MigrationManager{
		migrations: make(map[string]*Migration),
	}
	
	// Load available migrations
	mm.migrations["001_create_users"] = &Migration{"001_create_users", "Create users table", true}
	mm.migrations["002_add_email_idx"] = &Migration{"002_add_email_idx", "Add index on email", false}
	mm.migrations["003_create_posts"] = &Migration{"003_create_posts", "Create posts table", false}
	
	return mm
}

func (mm *MigrationManager) Status() {
	fmt.Println("=== Database Migration Status ===")
	
	var keys []string
	for k := range mm.migrations {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	
	for _, k := range keys {
		m := mm.migrations[k]
		status := "[PENDING]"
		if m.Applied {
			status = "[APPLIED]"
		}
		fmt.Printf("%s %s - %s\n", status, m.ID, m.Description)
	}
}

func (mm *MigrationManager) Migrate() {
	fmt.Println("\nExecuting pending migrations...")
	
	var keys []string
	for k := range mm.migrations {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	
	count := 0
	for _, k := range keys {
		m := mm.migrations[k]
		if !m.Applied {
			log.Printf("Applying: %s...", m.ID)
			// Simulate DB execution
			m.Applied = true
			count++
		}
	}
	
	if count == 0 {
		fmt.Println("Database is up to date.")
	} else {
		fmt.Printf("Successfully applied %d migrations.\n", count)
	}
}

func main() {
	manager := NewMigrationManager()
	
	args := os.Args[1:]
	if len(args) == 0 {
		fmt.Println("Usage: db-migrate [status|up]")
		os.Exit(1)
	}
	
	cmd := strings.ToLower(args[0])
	switch cmd {
	case "status":
		manager.Status()
	case "up":
		manager.Status()
		manager.Migrate()
		fmt.Println()
		manager.Status()
	default:
		fmt.Printf("Unknown command: %s\n", cmd)
	}
}
