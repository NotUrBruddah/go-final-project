package webserverutils

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"strconv"
	"strings"
	"time"

	"net/http"

	dbutils "webtasksplannerexample/internal/db"
	models "webtasksplannerexample/internal/models"
	utils "webtasksplannerexample/internal/utils"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

func InitWebServer(conf models.ServiceConfig) error {

	router := chi.NewRouter()
	router.Use(middleware.Logger)

	FileServer(router, "/", http.Dir(conf.HTTPWebDir))

	router.Route("/api", func(r chi.Router) {
		r.Get("/nextdate", getNextDateHandler)
		r.Route("/task", func(rr chi.Router) {
			rr.Post("/", postTaskHandler)
			rr.Get("/", getTaskHandler)
			rr.Put("/", putTaskHandler)
			rr.Delete("/", deleteTaskHandler)
			rr.Post("/done", doneTaskHandler)
		})
		r.Route("/tasks", func(rr chi.Router) {
			rr.Get("/", getTasksHandler)
		})
	})

	if err := http.ListenAndServe(":"+strconv.Itoa(conf.HTTPServerPort), router); err != nil {
		return err
	}
	return nil
}

// http.FileServer обработчик для отдачи статического контента с http.FileSystem
func FileServer(r chi.Router, path string, root http.FileSystem) {
	if strings.ContainsAny(path, "{}*") {
		panic("FileServer does not permit any URL parameters.")
	}

	if path != "/" && path[len(path)-1] != '/' {
		r.Get(path, http.RedirectHandler(path+"/", http.StatusMovedPermanently).ServeHTTP)
		path += "/"
	}
	path += "*"

	r.Get(path, func(w http.ResponseWriter, r *http.Request) {
		rctx := chi.RouteContext(r.Context())
		pathPrefix := strings.TrimSuffix(rctx.RoutePattern(), "/*")
		fs := http.StripPrefix(pathPrefix, http.FileServer(root))
		fs.ServeHTTP(w, r)
	})
}

var validFormats = []string{
	// Формат "y"
	`^y$`,
	// Формат "d <число от 1 до 400>"
	`^d\s(([1-9])|([1-9]\d)|[1-3]\d{2}|(400))$`,
	// Формат "w <числа от 1 до 7 через запятую>, при этом не более 7 штук
	`^w\s[1-7](,[1-7]){0,6}$`,
	// Формат "m <числа от 1 до 31 через запятую>, при этом не более 31 штук,
	// далее опционально через пробел <числа от 1 до 12 через запятую> не более 12 штук
	`^m\s((\-[12])|([1-9])|([12]\d)|(3[01]))((,\-[12])|(,[1-9])|(,[12]\d)|(,3[01])){0,30}(\s(([1-9])|(1[012]))((,[1-9])|(,1[012])){0,11})?$`,
}

func TaskValidate(t models.FullTask) error {

	if t.ID == "" {
		return errors.New("некорректный формат поля ID")
	} else if _, err := strconv.Atoi(t.ID); err != nil {
		return errors.New("некорректный формат поля ID")
	}

	if t.Date == "" {
		return errors.New("поле Date должно быть заполнено")
	} else if _, err := time.Parse("20060102", t.Date); err != nil {
		return errors.New("ошибка при парсинге поля даты")
	}

	if t.Title == "" {
		return errors.New("поле Title должно быть заполнено")
	}

	if t.Repeat != "" && !utils.IsValidFormat(t.Repeat, validFormats) {
		return errors.New("поле Repeat имеет неверный формат")
	}

	return nil
}

func NextDate(now time.Time, date string, repeat string) (string, error) {
	//преобразуем date к формату time.Time
	startDate, err := time.Parse("20060102", date)
	if err != nil {
		return "", err
	}

	if repeat == "" {
		return "", fmt.Errorf("пустое значение repeat")
	}

	//валидируем repeat переменную
	if !(utils.IsValidFormat(repeat, validFormats)) {
		return "", fmt.Errorf("некорректный формат repeat")
	}

	substrs := strings.Split(repeat, " ")
	switch substrs[0] {
	case "y":
		nextDate := startDate.AddDate(1, 0, 0)
		for nextDate.Before(now) || nextDate.Equal(now) {
			nextDate = nextDate.AddDate(1, 0, 0)
		}
		return nextDate.Format("20060102"), nil
	case "d":
		days, _ := strconv.Atoi(substrs[1])
		nextDate := startDate.AddDate(0, 0, days)
		for nextDate.Before(now) || nextDate.Equal(now) {
			nextDate = nextDate.AddDate(0, 0, days)
		}
		return nextDate.Format("20060102"), nil
	case "w":
	case "m":
	default:
		return "", fmt.Errorf("неподдерживаемый формат")
	}
	return "", fmt.Errorf("unexpected error")
}

func getNextDateHandler(w http.ResponseWriter, r *http.Request) {
	result := ""
	nowStr := r.FormValue("now")
	dateStr := r.FormValue("date")
	repeatStr := r.FormValue("repeat")
	nowDate, err := time.Parse("20060102", nowStr)
	if err != nil {
		log.Println("Ошибка при парсинге поля даты now", err.Error())
	} else {
		nextDate, err := NextDate(nowDate, dateStr, repeatStr)
		if err != nil {
			log.Println("Ошибка при вычислении nextdate", err.Error())
		} else {
			result = nextDate
		}
	}

	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	if _, err := w.Write([]byte(result)); err != nil {
		log.Printf("Ошибка записи ответа: %v", err)
	}
}

func postTaskHandler(w http.ResponseWriter, r *http.Request) {
	var task models.Task

	w.Header().Set("Content-Type", "application/json; charset=UTF-8")

	decoder := json.NewDecoder(r.Body)

	defer r.Body.Close()

	if err := decoder.Decode(&task); err != nil {
		errorMsg := models.HTTPJSONErrorMessageResponse{Error: err.Error()}
		errResp, _ := json.Marshal(errorMsg)
		w.Write(errResp)
		return
	}

	if task.Title == "" {
		errorMsg := models.HTTPJSONErrorMessageResponse{Error: "Обязательное поле 'title' отсутствует"}
		errResp, _ := json.Marshal(errorMsg)
		w.Write(errResp)
		return
	}

	//now := time.Now()
	now, _ := time.Parse("20060102", time.Now().Format("20060102"))

	if task.Date == "" {
		task.Date = now.Format("20060102")
	}
	date, err := time.Parse("20060102", task.Date)
	if err != nil {
		errorMsg := models.HTTPJSONErrorMessageResponse{Error: "Дата имеет неверный формат"}
		errResp, _ := json.Marshal(errorMsg)
		w.Write(errResp)
		return
	}
	nextDate := ""
	if task.Repeat != "" {
		nextDate, err = NextDate(now, date.Format("20060102"), task.Repeat)
		if err != nil {
			errorMsg := models.HTTPJSONErrorMessageResponse{Error: err.Error()}
			errResp, _ := json.Marshal(errorMsg)
			w.Write(errResp)
			return
		}
	} else {
		task.Date = now.Format("20060102")
	}

	if date.Before(now) && nextDate != "" {
		task.Date = nextDate
	}

	id, err := dbutils.AddTask(task)

	if err != nil {
		errorMsg := models.HTTPJSONErrorMessageResponse{Error: err.Error()}
		errResp, _ := json.Marshal(errorMsg)
		w.Write(errResp)
		return
	} else {
		respData := models.HTTPJSONResponseID{ID: id}
		res, _ := json.Marshal(respData)
		w.Write(res)
		return
	}
}

func getTasksHandler(w http.ResponseWriter, r *http.Request) {
	tasks, err := dbutils.GetTasks()
	if err != nil {
		errorMsg := models.HTTPJSONErrorMessageResponse{Error: err.Error()}
		errResp, _ := json.Marshal(errorMsg)
		w.Header().Set("Content-Type", "application/json; charset=UTF-8")
		w.Write(errResp)
		return
	}

	tasksList := models.TasksList{Tasks: tasks}
	jsonResp, err := json.Marshal(tasksList)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json; charset=UTF-8")
	w.Write(jsonResp)

}

func getTaskHandler(w http.ResponseWriter, r *http.Request) {

	idParam := r.URL.Query().Get("id")

	w.Header().Set("Content-Type", "application/json; charset=UTF-8")

	if idParam == "" {
		errorMsg := models.HTTPJSONErrorMessageResponse{Error: "Не указан идентификатор"}
		jsonResp, _ := json.Marshal(errorMsg)
		w.Write(jsonResp)
		return
	}

	_, err := strconv.Atoi(idParam)
	if err != nil {
		errorMsg := models.HTTPJSONErrorMessageResponse{Error: "Не указан идентификатор"}
		jsonResp, _ := json.Marshal(errorMsg)
		w.Write(jsonResp)
		return
	}

	task, err := dbutils.GetTaskByID(idParam)
	if err != nil {
		if err.Error() == "задача не найдена" {
			errorMsg := models.HTTPJSONErrorMessageResponse{Error: "задача не найдена"}
			jsonResp, _ := json.Marshal(errorMsg)
			w.Write(jsonResp)
			return
		}
		errorMsg := models.HTTPJSONErrorMessageResponse{Error: "задача не найдена"}
		jsonResp, _ := json.Marshal(errorMsg)
		w.Write(jsonResp)
		return
	}

	jsonResp, _ := json.Marshal(task)
	w.Write(jsonResp)
}

func putTaskHandler(w http.ResponseWriter, r *http.Request) {
	var task models.FullTask

	w.Header().Set("Content-Type", "application/json")

	err := json.NewDecoder(r.Body).Decode(&task)
	if err != nil {
		errorMsg := models.HTTPJSONErrorMessageResponse{Error: "Задача не найдена"}
		jsonResp, _ := json.Marshal(errorMsg)
		w.Write(jsonResp)
		return
	}

	defer r.Body.Close()

	if err := TaskValidate(task); err != nil {
		errorMsg := models.HTTPJSONErrorMessageResponse{Error: err.Error()}
		jsonResp, _ := json.Marshal(errorMsg)
		w.Write(jsonResp)
		return
	}

	currentTask, err := dbutils.GetTaskByID(task.ID)
	if err != nil {
		errorMsg := models.HTTPJSONErrorMessageResponse{Error: err.Error()}
		jsonResp, _ := json.Marshal(errorMsg)
		w.Write(jsonResp)
		return
	}

	if currentTask.ID == "" {
		errorMsg := models.HTTPJSONErrorMessageResponse{Error: "Задача не найдена"}
		jsonResp, _ := json.Marshal(errorMsg)
		w.Write(jsonResp)
		return
	}

	err = dbutils.UpdateTask(task)
	if err != nil {
		errorMsg := models.HTTPJSONErrorMessageResponse{Error: err.Error()}
		jsonResp, _ := json.Marshal(errorMsg)
		w.Write(jsonResp)
		return
	}

	// Возвращаем пустой JSON-объект в случае успеха
	w.Write([]byte("{}"))
}

func doneTaskHandler(w http.ResponseWriter, r *http.Request) {
	idParam := r.URL.Query().Get("id")

	w.Header().Set("Content-Type", "text/plain; charset=utf-8")

	if idParam == "" {
		errorMsg := models.HTTPJSONErrorMessageResponse{Error: "не указан идентификатор"}
		jsonResp, _ := json.Marshal(errorMsg)
		w.Write(jsonResp)
		return
	}

	_, err := strconv.Atoi(idParam)
	if err != nil {
		errorMsg := models.HTTPJSONErrorMessageResponse{Error: "неверный формат идентификатора"}
		jsonResp, _ := json.Marshal(errorMsg)
		w.Write(jsonResp)
		return
	}

	currentTask, err := dbutils.GetTaskByID(idParam)
	if err != nil {
		if err.Error() == "задача не найдена" {
			errorMsg := models.HTTPJSONErrorMessageResponse{Error: "задача не найдена"}
			jsonResp, _ := json.Marshal(errorMsg)
			w.Write(jsonResp)
			return
		}
		errorMsg := models.HTTPJSONErrorMessageResponse{Error: "задача не найдена"}
		jsonResp, _ := json.Marshal(errorMsg)
		w.Write(jsonResp)
		return
	}

	if currentTask.Repeat == "" {
		err = dbutils.DeleteTaskByID(idParam)
		if err != nil {
			errorMsg := models.HTTPJSONErrorMessageResponse{Error: err.Error()}
			jsonResp, _ := json.Marshal(errorMsg)
			w.Write(jsonResp)
			return
		}

		w.Write([]byte("{}"))
		return
	}

	now, _ := time.Parse("20060102", time.Now().Format("20060102"))
	nextDate, err := NextDate(now, currentTask.Date, currentTask.Repeat)
	if err != nil {
		errorMsg := models.HTTPJSONErrorMessageResponse{Error: err.Error()}
		jsonResp, _ := json.Marshal(errorMsg)
		w.Write(jsonResp)
		return
	}

	task := models.FullTask{
		ID: currentTask.ID,
		Task: models.Task{
			Date:    nextDate,
			Title:   currentTask.Title,
			Comment: currentTask.Comment,
			Repeat:  currentTask.Repeat,
		},
	}

	err = dbutils.UpdateTask(task)
	if err != nil {
		errorMsg := models.HTTPJSONErrorMessageResponse{Error: err.Error()}
		jsonResp, _ := json.Marshal(errorMsg)
		w.Write(jsonResp)
		return
	}

	// Возвращаем пустой JSON-объект в случае успеха
	w.Write([]byte("{}"))

}

func deleteTaskHandler(w http.ResponseWriter, r *http.Request) {
	idParam := r.URL.Query().Get("id")

	w.Header().Set("Content-Type", "text/plain; charset=utf-8")

	if idParam == "" {
		errorMsg := models.HTTPJSONErrorMessageResponse{Error: "Не указан идентификатор"}
		jsonResp, _ := json.Marshal(errorMsg)
		w.Write(jsonResp)
		return
	}

	_, err := strconv.Atoi(idParam)
	if err != nil {
		errorMsg := models.HTTPJSONErrorMessageResponse{Error: "Неверный формат идентификатора"}
		jsonResp, _ := json.Marshal(errorMsg)
		w.Write(jsonResp)
		return
	}

	err = dbutils.DeleteTaskByID(idParam)
	if err != nil {
		errorMsg := models.HTTPJSONErrorMessageResponse{Error: err.Error()}
		jsonResp, _ := json.Marshal(errorMsg)
		w.Write(jsonResp)
		return
	}

	// Возвращаем пустой JSON-объект в случае успеха
	w.Write([]byte("{}"))
}
