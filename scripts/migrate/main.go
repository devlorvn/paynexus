package main

import (
	"fmt"
	"log"
	"os"

	"paynexus/internal/golibs/configs"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Println("Cách dùng: go run scripts/migrate.go [up|down]")
		os.Exit(1)
	}

	cmd := os.Args[1]

	// Nạp cấu hình DSN PostgreSQL từ biến môi trường
	cfg := configs.LoadConfigFromEnv("migration", ":50051")
	dsn := cfg.Postgres.MigrationConnectionString()

	m, err := migrate.New("file://migrations", dsn)
	if err != nil {
		log.Fatalf("❌ Lỗi kết nối Migration Engine: %v", err)
	}
	defer m.Close()

	switch cmd {
	case "up":
		if err := m.Up(); err != nil && err != migrate.ErrNoChange {
			log.Fatalf("❌ Migration UP thất bại: %v", err)
		}
		fmt.Println("✅ Run PostgreSQL Migrations UP thành công!")
	case "down":
		if err := m.Down(); err != nil && err != migrate.ErrNoChange {
			log.Fatalf("❌ Migration DOWN thất bại: %v", err)
		}
		fmt.Println("✅ Rollback PostgreSQL Migrations DOWN thành công!")
	default:
		fmt.Printf("Command không hợp lệ: %s. Hãy dùng 'up' hoặc 'down'.\n", cmd)
	}
}
