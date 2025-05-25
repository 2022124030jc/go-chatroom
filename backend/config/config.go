package config

import (
    "gorm.io/driver/mysql"
    "gorm.io/gorm"
    "github.com/go-redis/redis/v8"
)

var DB *gorm.DB
var RDB *redis.Client

func InitDB() error {
    dsn := "root:jc20031224@tcp(127.0.0.1:3306)/chatroom?charset=utf8mb4&parseTime=True&loc=Local"
    var err error
    DB, err = gorm.Open(mysql.Open(dsn), &gorm.Config{})
    return err
}

func InitRedis() {
    RDB = redis.NewClient(&redis.Options{
        Addr:     "localhost:6379",
        Password: "", // no password set
        DB:       0,  // use default DB
    })
}