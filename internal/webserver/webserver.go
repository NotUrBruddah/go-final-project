package webserverutils

import (
	"encoding/json"
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

var serviceconf models.ServiceConfig

func InitWebServer(conf models.ServiceConfig) error {

	router := chi.NewRouter()
	router.Use(middleware.Logger)

	serviceconf = conf

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

func getNextDateHandler(w http.ResponseWriter, r *http.Request) {
	result := ""
	nowStr := r.FormValue("now")
	dateStr := r.FormValue("date")
	repeatStr := r.FormValue("repeat")
	nowDate, err := time.Parse("20060102", nowStr)
	if err != nil {
		log.Println("Ошибка при парсинге поля даты now", err.Error())
	} else {
		nextDate, err := utils.NextDate(nowDate, dateStr, repeatStr)
		if err != nil {
			log.Println("Ошибка при вычислении nextdate", err.Error())
		} else {
			result = nextDate
		}
	}

	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	log.Printf("Параметры запроса: now=[%s], date=[%s], repeat=[%s], nextdate=[%s]", nowStr, dateStr, repeatStr, result)
	if _, err := w.Write([]byte(result)); err != nil {
		log.Printf("Ошибка записи ответа: %v", err)
	}
}

func postTaskHandler(w http.ResponseWriter, r *http.Request) {
	var task models.Task

	decoder := json.NewDecoder(r.Body)

	defer r.Body.Close()

	if err := decoder.Decode(&task); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	if task.Title == "" {
		http.Error(w, "Обязательное поле 'title' отсутствует", http.StatusBadRequest)
		return
	}

	now := time.Now()
	if task.Date == "" {
		task.Date = now.Format("20060102")
	}
	date, err := time.Parse("20060102", task.Date)
	if err != nil {
		http.Error(w, "Дата имеет неверный формат", http.StatusBadRequest)
		return
	}
	nextDate := ""
	if task.Repeat != "" {
		nextDate, err = utils.NextDate(now, date.Format("20060102"), task.Repeat)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
	} else {
		task.Date = now.Format("20060102")
	}

	if date.Before(now) && nextDate != "" {
		task.Date = nextDate
	}

	id, err := dbutils.DbAddTask(serviceconf, task)

	var resp models.HTTPJSONResponse

	if err != nil {
		resp.Error = err.Error()
	} else {
		resp.ID = id
	}

	responseBytes, err := json.Marshal(resp)

	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json; charset=UTF-8")

	if _, err := w.Write(responseBytes); err != nil {
		log.Printf("Ошибка записи ответа: %v", err)
	}
}

func getTasksHandler(w http.ResponseWriter, r *http.Request) {
	tasks, err := dbutils.DbGetTasks(serviceconf)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	tasksList := models.TasksList{Tasks: tasks}
	jsonResp, err := json.Marshal(tasksList)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Write(jsonResp)

}

func getTaskHandler(w http.ResponseWriter, r *http.Request) {

	idParam := r.URL.Query().Get("id")

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

	task, err := dbutils.DBGetTaskByID(serviceconf, idParam)
	if err != nil {
		if err.Error() == "Задача не найдена" {
			errorMsg := models.HTTPJSONErrorMessageResponse{Error: "Задача не найдена"}
			jsonResp, _ := json.Marshal(errorMsg)
			w.Write(jsonResp)
			return
		}
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	jsonResp, err := json.Marshal(task)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Write(jsonResp)
}

func putTaskHandler(w http.ResponseWriter, r *http.Request) {
	var task models.Task

	err := json.NewDecoder(r.Body).Decode(&task)

	defer r.Body.Close()

	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Type", "application/json")

	if err := task.Validate(); err != nil {
		errorMsg := models.HTTPJSONErrorMessageResponse{Error: err.Error()}
		jsonResp, _ := json.Marshal(errorMsg)
		w.Write(jsonResp)
		return
	}

	currentTask, err := dbutils.DBGetTaskByID(serviceconf, task.ID)
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

	err = dbutils.DBUpdateTask(serviceconf, task)
	if err != nil {
		errorMsg := models.HTTPJSONErrorMessageResponse{Error: err.Error()}
		jsonResp, _ := json.Marshal(errorMsg)
		w.Write(jsonResp)
		return
	}

	// Возвращаем пустой JSON-объект в случае успеха
	w.Write([]byte{})
}

func doneTaskHandler(w http.ResponseWriter, r *http.Request) {
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

	currentTask, err := dbutils.DBGetTaskByID(serviceconf, idParam)
	if err != nil {
		if err.Error() == "Задача не найдена" {
			errorMsg := models.HTTPJSONErrorMessageResponse{Error: "Задача не найдена"}
			jsonResp, _ := json.Marshal(errorMsg)
			w.Write(jsonResp)
			return
		}
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	if currentTask.Repeat == "" {
		err = dbutils.DBDeleteTaskByID(serviceconf, idParam)
		if err != nil {
			errorMsg := models.HTTPJSONErrorMessageResponse{Error: err.Error()}
			jsonResp, _ := json.Marshal(errorMsg)
			w.Write(jsonResp)
			return
		}

		w.Write([]byte{})
		return
	}

	nextDate, err := utils.NextDate(time.Now(), currentTask.Date, currentTask.Repeat)
	if err != nil {
		errorMsg := models.HTTPJSONErrorMessageResponse{Error: err.Error()}
		jsonResp, _ := json.Marshal(errorMsg)
		w.Write(jsonResp)
		return
	}

	task := models.Task{
		ID:      currentTask.ID,
		Date:    nextDate,
		Title:   currentTask.Title,
		Comment: currentTask.Comment,
		Repeat:  currentTask.Repeat,
	}

	err = dbutils.DBUpdateTask(serviceconf, task)
	if err != nil {
		errorMsg := models.HTTPJSONErrorMessageResponse{Error: err.Error()}
		jsonResp, _ := json.Marshal(errorMsg)
		w.Write(jsonResp)
		return
	}

	// Возвращаем пустой JSON-объект в случае успеха
	w.Write([]byte{})

}

func deleteTaskHandler(w http.ResponseWriter, r *http.Request) {
	idParam := r.URL.Query().Get("id")

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

	err = dbutils.DBDeleteTaskByID(serviceconf, idParam)
	if err != nil {
		errorMsg := models.HTTPJSONErrorMessageResponse{Error: err.Error()}
		jsonResp, _ := json.Marshal(errorMsg)
		w.Write(jsonResp)
		return
	}

	// Возвращаем пустой JSON-объект в случае успеха
	w.Write([]byte{})
}
