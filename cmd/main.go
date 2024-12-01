package main

import (
	"log"
	"os"
	"path/filepath"
	"strconv"

	"database/sql"

	"github.com/joho/godotenv"

	dbutils "webtasksplannerexample/internal/db"
	models "webtasksplannerexample/internal/models"
	webserverutils "webtasksplannerexample/internal/webserver"
)

const (
	defaultHTTPPort   = 7540
	defaultHTTPWebDir = `web`
	defaultDBFilePath = `dbdata/scheduler.db`
)

func main() {
	var (
		err error
		db  *sql.DB
	)

	log.Println("Старт приложения")

	serviceConfig := initServiceConfig()

	db, err = dbutils.InitDB(serviceConfig.DbFilePath)
	if err != nil {
		log.Fatal("Ошибка инициализации БД:", err)
	}
	defer db.Close()

	err = webserverutils.InitWebServer(serviceConfig)
	if err != nil {
		log.Fatal("Ошибка запуска web-сервера:", err)
	}
}

func initServiceConfig() models.ServiceConfig {
	var (
		s   models.ServiceConfig
		err error
	)

	// Загружаем значения из файла .env
	if err = godotenv.Load(); err != nil {
		log.Println("Не найден файл .env")
	}

	envDBFilePath := os.Getenv("TODO_DBFILE")
	envHttpPort := os.Getenv("TODO_PORT")
	envHttpWebDir := os.Getenv("TODO_WEBDIR")

	//workDir, _ := os.Getwd()
	workDir, err := os.Executable()
	if err != nil {
		log.Fatal(err)
	}

	if envDBFilePath == "" {
		envDBFilePath = filepath.Join(filepath.Dir(workDir), defaultDBFilePath)
	}

	iHttpport := defaultHTTPPort
	if eport, err := strconv.ParseInt(envHttpPort, 10, 32); err == nil && (eport < 65535 && eport > 0) {
		iHttpport = int(eport)
	}

	if envHttpWebDir == "" {
		envHttpWebDir = filepath.Join(workDir, defaultHTTPWebDir)
	}

	s.DbFilePath = envDBFilePath
	s.HTTPServerPort = iHttpport
	s.HTTPWebDir = envHttpWebDir

	return s
}
