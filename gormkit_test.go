package gormkit_test

import (
	"context"
	"fmt"
	"sync"
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

// =============================================================================
// Basic Functionality Tests
// =============================================================================

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

// =============================================================================
// Connection Pool Tests
// =============================================================================

func TestConnectionPoolConfiguration(t *testing.T) {
	tests := []struct {
		name        string
		maxOpen     int
		maxIdle     int
		maxLifetime time.Duration
	}{
		{"Default", 0, 0, 0},
		{"Custom", 50, 10, 10 * time.Minute},
		{"HighLoad", 100, 25, 15 * time.Minute},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			manager, err := gormkit.New(&gormkit.Config{
				Driver:          "test",
				LogLevel:        "silent",
				MaxOpenConns:    tt.maxOpen,
				MaxIdleConns:    tt.maxIdle,
				ConnMaxLifetime: tt.maxLifetime,
			})
			if err != nil {
				t.Fatalf("Failed to create manager: %v", err)
			}
			defer manager.Close()

			stats := manager.Stats()

			expectedOpen := tt.maxOpen
			if expectedOpen == 0 {
				expectedOpen = 25 // default
			}

			if stats.MaxOpenConnections != expectedOpen {
				t.Errorf("Expected MaxOpenConns=%d, got=%d", expectedOpen, stats.MaxOpenConnections)
			}

			t.Logf("Pool stats: Open=%d, Idle=%d, InUse=%d, MaxOpen=%d",
				stats.OpenConnections, stats.Idle, stats.InUse, stats.MaxOpenConnections)
		})
	}
}

func TestConcurrentConnections(t *testing.T) {
	manager, err := gormkit.New(&gormkit.Config{
		Driver:       "test",
		Database:     "file::memory:?cache=shared",
		LogLevel:     "silent",
		MaxOpenConns: 10,
		MaxIdleConns: 5,
	})
	if err != nil {
		t.Fatal(err)
	}
	defer manager.Close()

	db := manager.DB()
	if err := db.AutoMigrate(&User{}); err != nil {
		t.Fatal(err)
	}

	if err := db.Create(&User{Name: "Init"}).Error; err != nil {
		t.Fatal(err)
	}

	const numGoroutines = 50
	const opsPerGoroutine = 10

	var wg sync.WaitGroup
	errors := make(chan error, numGoroutines*opsPerGoroutine)

	start := time.Now()

	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()

			for j := 0; j < opsPerGoroutine; j++ {
				user := User{Name: fmt.Sprintf("User-%d-%d", id, j)}

				if err := db.Create(&user).Error; err != nil {
					errors <- fmt.Errorf("goroutine %d: %w", id, err)
					return
				}

				var found User
				if err := db.First(&found, user.ID).Error; err != nil {
					errors <- fmt.Errorf("goroutine %d read: %w", id, err)
					return
				}
			}
		}(i)
	}

	wg.Wait()
	close(errors)

	duration := time.Since(start)
	totalOps := numGoroutines * opsPerGoroutine

	for err := range errors {
		t.Error(err)
	}

	stats := manager.Stats()
	t.Logf("Concurrent test completed in %v", duration)
	t.Logf("Total operations: %d (%.0f ops/sec)", totalOps*2, float64(totalOps*2)/duration.Seconds())
	t.Logf("Pool stats: Open=%d, Idle=%d, InUse=%d, WaitCount=%d, WaitDuration=%v",
		stats.OpenConnections, stats.Idle, stats.InUse, stats.WaitCount, stats.WaitDuration)

	if stats.OpenConnections > 10 {
		t.Errorf("Connection leak detected: %d connections open (max: 10)", stats.OpenConnections)
	}
}

func TestConnectionPoolExhaustion(t *testing.T) {
	manager, err := gormkit.New(&gormkit.Config{
		Driver:       "test",
		LogLevel:     "silent",
		MaxOpenConns: 5,
		MaxIdleConns: 2,
	})
	if err != nil {
		t.Fatal(err)
	}
	defer manager.Close()

	db := manager.DB()
	db.AutoMigrate(&User{})

	var wg sync.WaitGroup
	blockChan := make(chan struct{})

	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()

			ctx := context.Background()
			err := manager.Transaction(ctx, func(tx *gorm.DB) error {
				tx.Create(&User{Name: fmt.Sprintf("User-%d", id)})
				<-blockChan
				return nil
			})

			if err != nil {
				t.Logf("Transaction %d error: %v", id, err)
			}
		}(i)
	}

	time.Sleep(100 * time.Millisecond)

	stats := manager.Stats()
	t.Logf("During exhaustion: Open=%d, InUse=%d, WaitCount=%d",
		stats.OpenConnections, stats.InUse, stats.WaitCount)

	if stats.WaitCount == 0 {
		t.Logf("Note: No waits detected (pool may not be saturated)")
	}

	close(blockChan)
	wg.Wait()

	time.Sleep(50 * time.Millisecond)

	finalStats := manager.Stats()
	t.Logf("After release: Open=%d, Idle=%d, InUse=%d",
		finalStats.OpenConnections, finalStats.Idle, finalStats.InUse)

	if finalStats.InUse > 0 {
		t.Errorf("Connection leak: %d connections still in use", finalStats.InUse)
	}
}

func TestConnectionLeaks(t *testing.T) {
	manager, err := gormkit.New(&gormkit.Config{
		Driver:       "test",
		LogLevel:     "silent",
		MaxOpenConns: 10,
		MaxIdleConns: 5,
	})
	if err != nil {
		t.Fatal(err)
	}
	defer manager.Close()

	db := manager.DB()
	db.AutoMigrate(&User{})

	initialStats := manager.Stats()

	const iterations = 100
	for i := 0; i < iterations; i++ {
		ctx := context.Background()
		manager.Transaction(ctx, func(tx *gorm.DB) error {
			return tx.Create(&User{Name: "Leak Test"}).Error
		})
	}

	time.Sleep(100 * time.Millisecond)

	finalStats := manager.Stats()

	t.Logf("Initial: Open=%d, Idle=%d, InUse=%d",
		initialStats.OpenConnections, initialStats.Idle, initialStats.InUse)
	t.Logf("Final: Open=%d, Idle=%d, InUse=%d",
		finalStats.OpenConnections, finalStats.Idle, finalStats.InUse)

	if finalStats.InUse > 0 {
		t.Errorf("Connection leak: %d connections in use after %d iterations", finalStats.InUse, iterations)
	}

	if finalStats.OpenConnections > initialStats.OpenConnections+2 {
		t.Errorf("Possible connection leak: connections grew from %d to %d",
			initialStats.OpenConnections, finalStats.OpenConnections)
	}
}

func TestLongRunningConnections(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping long running test")
	}

	manager, err := gormkit.New(&gormkit.Config{
		Driver:          "test",
		LogLevel:        "silent",
		MaxOpenConns:    5,
		MaxIdleConns:    2,
		ConnMaxLifetime: 1 * time.Second,
	})
	if err != nil {
		t.Fatal(err)
	}
	defer manager.Close()

	db := manager.DB()
	db.AutoMigrate(&User{})

	start := time.Now()
	for time.Since(start) < 3*time.Second {
		db.Create(&User{Name: "Long Running"})
		time.Sleep(100 * time.Millisecond)
	}

	stats := manager.Stats()
	t.Logf("After 3s: Open=%d, MaxLifetimeClosed=%d",
		stats.OpenConnections, stats.MaxLifetimeClosed)

	if stats.MaxLifetimeClosed == 0 {
		t.Error("No connections closed due to MaxLifetime")
	}
}

func TestContextCancellation(t *testing.T) {
	manager, err := gormkit.New(&gormkit.Config{
		Driver:       "test",
		LogLevel:     "silent",
		MaxOpenConns: 5,
	})
	if err != nil {
		t.Fatal(err)
	}
	defer manager.Close()

	db := manager.DB()
	db.AutoMigrate(&User{})

	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Millisecond)
	defer cancel()

	time.Sleep(10 * time.Millisecond)

	err = db.WithContext(ctx).Create(&User{Name: "Test"}).Error
	if err == nil {
		t.Error("Expected context timeout error")
	}

	stats := manager.Stats()
	if stats.InUse > 0 {
		t.Errorf("Leaked connection after context cancel: %d in use", stats.InUse)
	}
}

// =============================================================================
// Benchmarks
// =============================================================================

func BenchmarkConnectionPoolPerformance(b *testing.B) {
	manager, _ := gormkit.New(&gormkit.Config{
		Driver:       "test",
		LogLevel:     "silent",
		MaxOpenConns: 25,
		MaxIdleConns: 10,
	})
	defer manager.Close()

	db := manager.DB()
	db.AutoMigrate(&User{})

	b.ResetTimer()

	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			user := User{Name: "Benchmark"}
			db.Create(&user)

			var found User
			db.First(&found, user.ID)
		}
	})

	stats := manager.Stats()
	b.Logf("Pool stats: Open=%d, Idle=%d, MaxOpen=%d, WaitCount=%d, WaitDuration=%v",
		stats.OpenConnections, stats.Idle, stats.MaxOpenConnections,
		stats.WaitCount, stats.WaitDuration)
}

func BenchmarkWithDifferentPoolSizes(b *testing.B) {
	sizes := []int{5, 10, 25, 50, 100}

	for _, size := range sizes {
		b.Run(fmt.Sprintf("PoolSize-%d", size), func(b *testing.B) {
			manager, _ := gormkit.New(&gormkit.Config{
				Driver:       "test",
				LogLevel:     "silent",
				MaxOpenConns: size,
				MaxIdleConns: size / 2,
			})
			defer manager.Close()

			db := manager.DB()
			db.AutoMigrate(&User{})

			b.ResetTimer()

			b.RunParallel(func(pb *testing.PB) {
				for pb.Next() {
					db.Create(&User{Name: "Bench"})
				}
			})

			stats := manager.Stats()
			b.Logf("Size %d: WaitCount=%d, WaitDuration=%v",
				size, stats.WaitCount, stats.WaitDuration)
		})
	}
}
