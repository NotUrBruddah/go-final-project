package dbutils

import (
	"database/sql"
	"errors"
	"fmt"
	"os"
	"webtasksplannerexample/internal/models"

	_ "modernc.org/sqlite"
)

func InitDB(conf models.ServiceConfig) (*sql.DB, error) {

	if _, err := os.Stat(conf.DbFilePath); errors.Is(err, os.ErrNotExist) {
		// Файла нет, создаем новую базу данных
		f, err := os.Create(conf.DbFilePath)
		if err != nil {
			return nil, fmt.Errorf("не удалось создать файл базы данных: %s", err.Error())
		}
		f.Close()
	}

	// Подключаемся к базе данных
	db, err := sql.Open("sqlite", conf.DbFilePath)
	if err != nil {
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
		return nil, fmt.Errorf("не удалось создать таблицу 'scheduler': %w", err)
	}

	// Создаём индекс по полю 'date'
	if _, err := db.Exec(`CREATE INDEX IF NOT EXISTS idx_date ON scheduler(date)`); err != nil {
		return nil, fmt.Errorf("не удалось создать индекс по полю 'date': %w", err)
	}

	return db, nil
}

func DbAddTask(conf models.ServiceConfig, task models.Task) (int64, error) {
	result, err := conf.DBobject.Exec(
		"INSERT INTO scheduler (date, title, comment, repeat) VALUES (?, ?, ?, ?)",
		task.Date,
		task.Title,
		task.Comment,
		task.Repeat,
	)
	if err != nil {
		return 0, err
	}

	id, err := result.LastInsertId()
	if err != nil {
		return 0, err
	}

	return id, nil
}

func DbGetTasks(conf models.ServiceConfig) ([]models.Task, error) {
	rows, err := conf.DBobject.Query(`
		SELECT id, date, title, comment, repeat
		FROM scheduler
		ORDER BY date ASC LIMIT 50
	`)

	if err != nil {
		return nil, err
	}
	defer rows.Close()

	tasks := []models.Task{}
	for rows.Next() {
		var task models.Task
		if err := rows.Scan(&task.ID, &task.Date, &task.Title, &task.Comment, &task.Repeat); err != nil {
			return nil, err
		}
		tasks = append(tasks, task)
	}

	return tasks, nil
}

func DBGetTaskByID(conf models.ServiceConfig, id string) (models.Task, error) {

	row := conf.DBobject.QueryRow(`SELECT id, date, title, comment, repeat FROM scheduler WHERE id = $1`, id)

	task := models.Task{ID: id}

	err := row.Scan(&task.ID, &task.Date, &task.Title, &task.Comment, &task.Repeat)
	if err != nil {
		if err == sql.ErrNoRows {
			return models.Task{}, fmt.Errorf("задача не найдена")
		}
		return models.Task{}, err
	}
	return task, nil
}

func DBUpdateTask(conf models.ServiceConfig, task models.Task) error {

	_, err := conf.DBobject.Exec(`UPDATE scheduler SET date = ?, title = ?, comment = ?, repeat = ? WHERE id = ?`,
		task.Date,
		task.Title,
		task.Comment,
		task.Repeat,
		task.ID,
	)

	if err != nil {
		return err
	}

	return nil
}

func DBDeleteTaskByID(conf models.ServiceConfig, id string) error {

	_, err := conf.DBobject.Exec(`DELETE FROM scheduler WHERE id = ?`, id )

	if err != nil {
		return err
	}

	return nil

}