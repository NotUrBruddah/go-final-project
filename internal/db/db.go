package dbutils

import (
	"database/sql"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"webtasksplannerexample/internal/models"

	_ "modernc.org/sqlite"
)

const (
	maxRowCountLimit int = 50
)

var (
	db *sql.DB
)

func createDirPathIfNotExist(path string) error {
	var (
		err error
	)

	dbDir := filepath.Join(path, "..")
	if _, err = os.Stat(dbDir); os.IsNotExist(err) {
		err = os.MkdirAll(dbDir, 0755)
		if err != nil {
			return err
		}
	}
	if err != nil {
		return err
	}
	return nil
}

func InitDB(dbFilePath string) (*sql.DB, error) {
	var err error

	if err = createDirPathIfNotExist(dbFilePath); err != nil {
		return nil, fmt.Errorf("не удалось создать каталоги для размещения файла базы данных: %s", err.Error())
	}

	if _, err := os.Stat(dbFilePath); errors.Is(err, os.ErrNotExist) {
		// Файла нет, создаем новую базу данных
		f, err := os.Create(dbFilePath)
		if err != nil {
			return nil, fmt.Errorf("не удалось создать файл базы данных: %s", err.Error())
		}
		f.Close()
	}

	// Подключаемся к базе данных
	db, err = sql.Open("sqlite", dbFilePath)
	if err != nil {
		return nil, fmt.Errorf("не удалось открыть базу данных: %w", err)
	}

	// Проверяем наличие таблицы 'scheduler' и создаем ее, если необходимо
	if _, err = db.Exec(`CREATE TABLE IF NOT EXISTS scheduler (
        id INTEGER PRIMARY KEY AUTOINCREMENT,
        date VARCHAR(8) NOT NULL,
        title VARCHAR(64) NOT NULL,
        comment TEXT CHECK(length(comment) <= 512),
        repeat  VARCHAR(128)
    )`); err != nil {
		return nil, fmt.Errorf("не удалось создать таблицу 'scheduler': %w", err)
	}

	// Создаём индекс по полю 'date'
	if _, err = db.Exec(`CREATE INDEX IF NOT EXISTS idx_date ON scheduler(date)`); err != nil {
		return nil, fmt.Errorf("не удалось создать индекс по полю 'date': %w", err)
	}

	return db, nil
}

func AddTask(task models.Task) (int64, error) {
	result, err := db.Exec(
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

func GetTasks(searchString string, searchIsDate bool) ([]models.FullTask, error) {
	var (
		err  error
		rows *sql.Rows
	)
	if searchString != "" && searchIsDate {
		rows, err = db.Query(`
			SELECT id, date, title, comment, repeat
			FROM scheduler WHERE date = ?
			ORDER BY date ASC LIMIT ?`,
			searchString,
			maxRowCountLimit,
		)
	} else if searchString != "" && !searchIsDate {
		searchString = `%` + searchString + `%`
		rows, err = db.Query(`
			SELECT id, date, title, comment, repeat
			FROM scheduler WHERE LOWER(title) LIKE LOWER(?) 
			OR LOWER(comment) LIKE LOWER(?)
			ORDER BY date ASC LIMIT ?`,
			searchString,
			searchString,
			maxRowCountLimit,
		)
	} else {
		rows, err = db.Query(`
			SELECT id, date, title, comment, repeat
			FROM scheduler
			ORDER BY date ASC LIMIT ?`,
			maxRowCountLimit,
		)
	}

	if err != nil {
		return nil, err
	}
	defer rows.Close()

	tasks := []models.FullTask{}
	for rows.Next() {
		var task models.FullTask
		if err := rows.Scan(&task.ID, &task.Date, &task.Title, &task.Comment, &task.Repeat); err != nil {
			return nil, err
		}
		tasks = append(tasks, task)
	}
	if err = rows.Err(); err != nil {
		return nil, err
	}

	return tasks, nil
}

func GetTaskByID(id string) (models.FullTask, error) {

	var task models.FullTask

	row := db.QueryRow(`SELECT id, date, title, comment, repeat FROM scheduler WHERE id = $1`, id)

	err := row.Scan(&task.ID, &task.Date, &task.Title, &task.Comment, &task.Repeat)
	if err != nil {
		if err == sql.ErrNoRows {
			return models.FullTask{}, fmt.Errorf("задача не найдена")
		}
		return models.FullTask{}, err
	}
	return task, nil
}

func UpdateTask(task models.FullTask) error {

	_, err := db.Exec(`UPDATE scheduler SET date = ?, title = ?, comment = ?, repeat = ? WHERE id = ?`,
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

func DeleteTaskByID(id string) error {

	_, err := db.Exec(`DELETE FROM scheduler WHERE id = ?`, id)

	if err != nil {
		return err
	}

	return nil

}
