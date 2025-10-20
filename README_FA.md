# gorm-kit

یک Manager ساده و قابل استفاده مجدد برای GORM

## نصب

```bash
go get github.com/alinemone/gorm-kit
```

## استفاده سریع

```go
package main

import (
    "github.com/alinemone/gorm-kit"
    "log"
)

func main() {
    // ساخت manager
    manager, err := gormkit.New(&gormkit.Config{
        Driver:   "postgres",
        Host:     "localhost",
        Port:     5432,
        User:     "postgres",
        Password: "postgres",
        Database: "mydb",
    })
    if err != nil {
        log.Fatal(err)
    }
    defer manager.Close()

    // استفاده
    db := manager.DB()
    
    type User struct {
        ID   uint
        Name string
    }
    
    db.AutoMigrate(&User{})
    db.Create(&User{Name: "علی"})
}
```

## قابلیت‌ها

- ✅ پشتیبانی PostgreSQL, MySQL, SQLite
- ✅ Connection pooling
- ✅ پشتیبانی Context
- ✅ Transaction helper
- ✅ Pagination ساده
- ✅ تلاش مجدد خودکار

## API

### توابع اصلی

```go
// ساخت manager
manager, _ := gormkit.New(&gormkit.Config{...})

// دریافت database
db := manager.DB()

// با context
db := manager.WithContext(ctx)

// Transaction
manager.Transaction(ctx, func(tx *gorm.DB) error {
    return nil
})

// Pagination
db.Scopes(gormkit.Paginate(page, limit)).Find(&users)

// Migration
manager.Migrate(&User{})

// بستن
manager.Close()
```

## مثال با Fiber

```go
var db *gormkit.Manager

func main() {
    db, _ = gormkit.New(&gormkit.Config{
        Driver: "postgres",
        Host: "localhost",
        Port: 5432,
        Database: "myapp",
    })
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

## تنظیمات

| گزینه | پیش‌فرض | توضیح |
|-------|---------|-------|
| Driver | - | postgres, mysql, sqlite, test |
| Host | - | آدرس سرور |
| Port | - | پورت |
| User | - | نام کاربری |
| Password | - | رمز عبور |
| Database | - | نام دیتابیس |
| MaxOpenConns | 25 | حداکثر اتصالات |
| MaxIdleConns | 5 | اتصالات بیکار |
| LogLevel | info | سطح log |

## استفاده در پروژه

```bash
# نصب
go get github.com/alinemone/gorm-kit

# در کد
import "github.com/alinemone/gorm-kit"
```

## لایسنس

MIT
