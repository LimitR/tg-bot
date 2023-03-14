package main

import (
	"bot/cmd/database"
	"bot/cmd/telegram"
	"os"

	"github.com/joho/godotenv"
	_ "github.com/mattn/go-sqlite3"
)

func main() {
	err := godotenv.Load()
	if err != nil {
		panic(err)
	}
	db := database.NewConnection(os.Getenv("DATABASE_NAME"))
	db.CreateTableIfNotExists()
	bot, err := telegram.NewBot(os.Getenv("TOKEN"), db)
	if err != nil {
		panic(err)
	}
	bot.Run()
}
