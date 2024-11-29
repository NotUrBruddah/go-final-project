package models

import (
	"database/sql"
	"errors"
	"os"
	"path/filepath"
	"strconv"
	"webtasksplannerexample/internal/utils"
)

type ServiceConfig struct {
	DbFilePath     string
	HTTPServerPort int
	HTTPWebDir     string
	DBobject       *sql.DB
}

func (s *ServiceConfig) Init(dbfilepath string, httpport string, httpwebdir string) {

	workDir, _ := os.Getwd()

	if dbfilepath == "" {
		dbfilepath = filepath.Join(filepath.Dir(workDir), "dbdata/scheduler.db")
	}

	iHttpport := 7540
	if eport, err := strconv.ParseInt(httpport, 10, 32); err == nil && (eport < 65535 && eport > 0) {
		iHttpport = int(eport)
	}

	if httpwebdir == "" {
		httpwebdir = filepath.Join(workDir, "web")
	}

	s.DbFilePath = dbfilepath
	s.HTTPServerPort = iHttpport
	s.HTTPWebDir = httpwebdir
}

type Task struct {
	ID      string `json:"id"`
	Date    string `json:"date"`
	Title   string `json:"title"`
	Comment string `json:"comment,omitempty"` // Опциональный параметр
	Repeat  string `json:"repeat,omitempty"`  // Опциональный параметр
}

func (t *Task) Validate() error {
	if t.Date == "" {
		return errors.New("поле Date должно быть заполнено")
	}
	if t.Title == "" {
		return errors.New("поле Title должно быть заполнено")
	}
	if t.Repeat != "" && !utils.IsValidFormat(t.Repeat) {
		return errors.New("поле Repeat имеет неверный формат")
	}
	return nil
}

type TasksList struct {
	Tasks []Task `json:"tasks"`
}

type HTTPJSONResponse struct {
	ID    int64  `json:"id"`
	Error string `json:"error"`
}

type HTTPJSONErrorMessageResponse struct {
	Error string `json:"error"`
}
