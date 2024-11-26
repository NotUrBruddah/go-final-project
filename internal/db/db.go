package dbutils

import (
	"database/sql"
	"errors"
	"fmt"

	"log"
	"os"

	"path/filepath"

	_ "modernc.org/sqlite"
)

type Task struct {
	ID      int
	Date    string
	Title   string
	Comment string
	Repeat  string
}

func InitDB() (*sql.DB, error) {
	log.Println("Инициазация БД")

	workDir, _ := os.Getwd()

	dbFile := filepath.Join(filepath.Dir(workDir), "dbdata/scheduler.db")
	envDBfile := os.Getenv("TODO_DBFILE")
	if len(envDBfile) > 0 {
		dbFile = envDBfile
	}

	if _, err := os.Stat(dbFile); errors.Is(err, os.ErrNotExist) {
		// Файла нет, создаем новую базу данных
		f, err := os.Create(dbFile)
		if err != nil {
			log.Fatalf("не удалось создать файл базы данных: %s", err.Error())
		}
		f.Close()
	}

	// Подключаемся к базе данных
	log.Println("Подключение к БД")
	db, err := sql.Open("sqlite", dbFile)
	if err != nil {
		log.Println("Не удалось открыть БД " + dbFile)
		return nil, fmt.Errorf("не удалось открыть базу данных: %w", err)
	}

	// Проверяем наличие таблицы 'scheduler' и создаем ее, если необходимо

	if _, err := db.Exec(`CREATE TABLE IF NOT EXISTS scheduler (
        id INTEGER PRIMARY KEY AUTOINCREMENT,
        date TEXT NOT NULL,
        title TEXT NOT NULL,
        comment TEXT,
        repeat TEXT
    )`); err != nil {
		log.Println("не удалось создать таблицу scheduler")
		return nil, fmt.Errorf("не удалось создать таблицу 'scheduler': %w", err)
	}

	// Создаём индекс по полю 'date'
	log.Println("Создаем Индекс")
	if _, err := db.Exec(`CREATE INDEX IF NOT EXISTS idx_date ON scheduler(date)`); err != nil {
		log.Println("не удалось создать индекс по полю date")
		return nil, fmt.Errorf("не удалось создать индекс по полю 'date': %w", err)
	}

	return db, nil
}
