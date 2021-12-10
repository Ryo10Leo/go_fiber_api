package main

import (
	"admin/src/database"
	"admin/src/routes"
	"fmt"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/joho/godotenv"
)

func main() {
	// Connection To Mysql
	database.Connect()
	// Migration
	database.AutoMigrate()
	// Redis
	database.SetupRedis()
	database.SetupCacheChannel()
	loadEnv()

	app := fiber.New()

	routes.Setup(app)
	// 認証にcookieなどの情報を必要とするかどうか
	app.Use(cors.New(cors.Config{
		// 認証にcookieなどの情報を必要とするかどうか
		AllowCredentials: true,
	}))

	app.Listen(":3000")
}

func loadEnv() {
	err := godotenv.Load(".env")

	if err != nil {
		fmt.Printf("読み込み出来ませんでした: %v", err)
	}
}
