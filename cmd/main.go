package main

import (
	"log"

	"github.com/joho/godotenv"

	dbutils "webtasksplannerexample/internal/db"
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

	dbutils.InitDB()
	webserverutils.InitWebServer()

}
