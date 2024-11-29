package main

import (
	"log"
	"os"

	"github.com/joho/godotenv"

	dbutils "webtasksplannerexample/internal/db"
	models "webtasksplannerexample/internal/models"
	webserverutils "webtasksplannerexample/internal/webserver"
)

func init() {
	// Загружаем значения из файла .env проекта
	if err := godotenv.Load(); err != nil {
		log.Println("Не найден файл .env")
	}
}

func main() {
	log.Println("Старт приложения")

	var config models.ServiceConfig
	config.Init(
		os.Getenv("TODO_DBFILE"),
		os.Getenv("TODO_PORT"),
		os.Getenv("TODO_WEBDIR"),
	)

	db, err := dbutils.InitDB(config)
	if err != nil {
		log.Fatal("Ошибка инициализации БД:", err)
	}
	defer db.Close()
	config.DBobject = db

	err = webserverutils.InitWebServer(config)
	if err != nil {
		log.Fatal("Ошибка запуска web-сервера:", err)
	}
}
