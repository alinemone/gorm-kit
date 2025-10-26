# gorm-kit

A simple, reusable GORM manager for Go projects.

## Installation

```bash
go get github.com/alinemone/gorm-kit
```

## Quick Start

```go
package main

import (
    "github.com/alinemone/gorm-kit"
    "log"
)

func main() {
    // Create manager
    manager, err := gormkit.New(&gormkit.Config{
        Driver:   "postgres",
        Host:     "localhost",
        Port:     5432,
        User:     "postgres",
        Password: "postgres",
        Database: "mydb",
        SSLMode:  "disable",
    })
    if err != nil {
        log.Fatal(err)
    }
    defer manager.Close()

    // Get database
    db := manager.DB()

    // Use it
    type User struct {
        ID   uint
        Name string
    }
    
    db.AutoMigrate(&User{})
    db.Create(&User{Name: "John"})
}
```

## Features

- ✅ PostgreSQL, MySQL, SQLite support
- ✅ Connection pooling
- ✅ Context support
- ✅ Transaction helper
- ✅ Simple pagination
- ✅ Auto-retry on connection failure

## API

### Create Manager

```go
manager, err := gormkit.New(&gormkit.Config{
    Driver:   "postgres",  // postgres, mysql, sqlite, test
    Host:     "localhost",
    Port:     5432,
    User:     "postgres",
    Password: "postgres",
    Database: "mydb",

    // Optional
    Timezone:        "UTC",  // UTC, Asia/Tehran, America/New_York, etc.
    MaxOpenConns:    25,
    MaxIdleConns:    5,
    ConnMaxLifetime: 5 * time.Minute,
    LogLevel:        "info",  // silent, error, info
    AutoMigrate:     true,
})
```

### Use Database

```go
// Get GORM instance
db := manager.DB()

// With context
db := manager.WithContext(ctx)

// Transaction
err := manager.Transaction(ctx, func(tx *gorm.DB) error {
    tx.Create(&user)
    return nil
})

// Pagination
db.Scopes(gormkit.Paginate(page, perPage)).Find(&users)

// Migrate
manager.Migrate(&User{}, &Product{})

// Health check
err := manager.Ping(ctx)

// Stats
stats := manager.Stats()
```

## Examples

### Custom Timezone

```go
// Use Tehran timezone
manager, err := gormkit.New(&gormkit.Config{
    Driver:   "postgres",
    Host:     "localhost",
    Port:     5432,
    User:     "postgres",
    Password: "postgres",
    Database: "mydb",
    Timezone: "Asia/Tehran", // All timestamps will use Tehran timezone
})

// Use New York timezone
manager, err := gormkit.New(&gormkit.Config{
    Driver:   "mysql",
    Host:     "localhost",
    Port:     3306,
    User:     "root",
    Password: "password",
    Database: "myapp",
    Timezone: "America/New_York",
})
```

### With Fiber

```go
package main

import (
    "github.com/gofiber/fiber/v2"
    "github.com/alinemone/gorm-kit"
)

var db *gormkit.Manager

func main() {
	var err error
	db, err = gormkit.New(&gormkit.Config{
		Driver:   "postgres",
		Host:     "localhost",
		User:     "postgres",
		Password: "your-password",
		Port:     5432,
		Database: "myapp",
	})
	if err != nil {
		panic(err)
	}
	defer db.Close()

	app := fiber.New()

	app.Get("/users", func(c *fiber.Ctx) error {
		var users []User
		db.WithContext(c.Context()).Find(&users)
		return c.JSON(users)
	})

	app.Listen(":8080")
}

```

### Pagination

```go
func ListUsers(ctx context.Context, page, perPage int) ([]User, error) {
    var users []User
    err := manager.WithContext(ctx).
        Scopes(gormkit.Paginate(page, perPage)).
        Find(&users).Error
    return users, err
}
```

### Transaction

```go
err := manager.Transaction(ctx, func(tx *gorm.DB) error {
    if err := tx.Create(&user).Error; err != nil {
        return err
    }
    if err := tx.Create(&profile).Error; err != nil {
        return err
    }
    return nil
})
```

## Testing

```go
func TestMyFunc(t *testing.T) {
    // Use in-memory SQLite
    manager, _ := gormkit.New(&gormkit.Config{
        Driver:   "test",
        LogLevel: "silent",
    })
    defer manager.Close()

    // Test with manager.DB()
}
```

## Configuration Options

| Option | Default | Description |
|--------|---------|-------------|
| Driver | - | postgres, mysql, sqlite, test |
| Host | - | Database host |
| Port | - | Database port |
| User | - | Database user |
| Password | - | Database password |
| Database | - | Database name |
| SSLMode | disable | SSL mode for postgres |
| Timezone | UTC | Database timezone (e.g., UTC, Asia/Tehran) |
| MaxOpenConns | 25 | Max open connections |
| MaxIdleConns | 5 | Max idle connections |
| ConnMaxLifetime | 5m | Connection max lifetime |
| LogLevel | info | silent, error, info |
| AutoMigrate | false | Enable auto migration |
| RetryAttempts | 3 | Connection retry attempts |
| ConnectTimeout | 10s | Connection timeout |

## License

MIT
