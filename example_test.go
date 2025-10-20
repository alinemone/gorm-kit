package gormkit_test

import (
	"context"
	"testing"
	"time"

	"github.com/alinemone/gorm-kit"
	"gorm.io/gorm"
)

type User struct {
	ID        uint `gorm:"primarykey"`
	Name      string
	CreatedAt time.Time
}

func TestBasicUsage(t *testing.T) {
	manager, err := gormkit.New(&gormkit.Config{
		Driver:   "test",
		LogLevel: "silent",
	})
	if err != nil {
		t.Fatal(err)
	}
	defer manager.Close()

	db := manager.DB()
	db.AutoMigrate(&User{})

	user := User{Name: "Test"}
	if err := db.Create(&user).Error; err != nil {
		t.Errorf("Create failed: %v", err)
	}

	var found User
	if err := db.First(&found, user.ID).Error; err != nil {
		t.Errorf("Find failed: %v", err)
	}

	if found.Name != "Test" {
		t.Errorf("Expected 'Test', got '%s'", found.Name)
	}
}

func TestWithContext(t *testing.T) {
	manager, _ := gormkit.New(&gormkit.Config{
		Driver:   "test",
		LogLevel: "silent",
	})
	defer manager.Close()

	ctx := context.Background()
	db := manager.WithContext(ctx)

	db.AutoMigrate(&User{})

	user := User{Name: "Context Test"}
	if err := db.Create(&user).Error; err != nil {
		t.Errorf("Create failed: %v", err)
	}
}

func TestTransaction(t *testing.T) {
	manager, _ := gormkit.New(&gormkit.Config{
		Driver:   "test",
		LogLevel: "silent",
	})
	defer manager.Close()

	manager.DB().AutoMigrate(&User{})

	ctx := context.Background()
	err := manager.Transaction(ctx, func(tx *gorm.DB) error {
		return tx.Create(&User{Name: "Transaction Test"}).Error
	})

	if err != nil {
		t.Errorf("Transaction failed: %v", err)
	}
}

func TestPagination(t *testing.T) {
	manager, _ := gormkit.New(&gormkit.Config{
		Driver:   "test",
		LogLevel: "silent",
	})
	defer manager.Close()

	db := manager.DB()
	db.AutoMigrate(&User{})

	for i := 1; i <= 25; i++ {
		db.Create(&User{Name: "User"})
	}

	var users []User
	db.Scopes(gormkit.Paginate(2, 10)).Find(&users)

	if len(users) != 10 {
		t.Errorf("Expected 10 users, got %d", len(users))
	}
}

func TestPing(t *testing.T) {
	manager, _ := gormkit.New(&gormkit.Config{
		Driver:   "test",
		LogLevel: "silent",
	})
	defer manager.Close()

	ctx := context.Background()
	if err := manager.Ping(ctx); err != nil {
		t.Errorf("Ping failed: %v", err)
	}
}

func TestStats(t *testing.T) {
	manager, _ := gormkit.New(&gormkit.Config{
		Driver:   "test",
		LogLevel: "silent",
	})
	defer manager.Close()

	stats := manager.Stats()
	if stats.MaxOpenConnections == 0 {
		t.Error("Stats should return connection info")
	}
}
