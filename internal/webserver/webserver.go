package webserverutils

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"

	dbutils "webtasksplannerexample/internal/db"
	models "webtasksplannerexample/internal/models"
	utils "webtasksplannerexample/internal/utils"
)

const (
	dateTimeFormat string = "20060102"
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

// Слайс из валидных регэкспов для поля repeat
var repeatValidFormats = []string{
	`^y$`, // Формат "y"
	`^d\s(([1-9])|([1-9]\d)|[1-3]\d{2}|(400))$`, // Формат "d <число от 1 до 400>"
	`^w\s[1-7](,[1-7]){0,6}$`,                   // Формат "w <числа от 1 до 7 через запятую>, при этом не более 7 штук
	`^m\s((\-[12])|(0?[1-9])|([12]\d)|(3[01]))((,\-[12])|(,[1-9])|(,[12]\d)|(,3[01])){0,30}(\s((0?[1-9])|(1[012]))((,[1-9])|(,1[012])){0,11})?$`,
	// Формат "m <числа от 1 до 31 через запятую>, при этом не более 31 штуки, числа 1 - 9 могут иметь написание 01 02 и т.д до 09
	// далее опционально через пробел <числа от 1 до 12 через запятую> не более 12 штук
}

func TaskValidate(t models.FullTask) error {
	if t.ID == "" {
		return errors.New("некорректный формат поля ID")
	} else if _, err := strconv.Atoi(t.ID); err != nil {
		return errors.New("некорректный формат поля ID")
	}

	if t.Date == "" {
		return errors.New("поле Date должно быть заполнено")
	} else if _, err := time.Parse(dateTimeFormat, t.Date); err != nil {
		return errors.New("ошибка при парсинге поля даты")
	}

	if t.Title == "" {
		return errors.New("поле Title должно быть заполнено")
	}

	if t.Repeat != "" && !utils.IsValidFormat(t.Repeat, repeatValidFormats) {
		return errors.New("поле Repeat имеет неверный формат")
	}

	return nil
}

// Функция для подсчета даты по правилам повтора
func NextDate(now time.Time, date string, repeat string) (string, error) {
	startDate, err := time.Parse(dateTimeFormat, date)
	if err != nil {
		return "", err
	}
	if repeat == "" {
		return "", fmt.Errorf("пустое значение repeat")
	}
	//валидируем repeat переменную
	if !(utils.IsValidFormat(repeat, repeatValidFormats)) {
		return "", fmt.Errorf("некорректный формат repeat")
	}

	substrs := strings.Split(repeat, " ")
	switch substrs[0] {
	case "y":
		nextDate := startDate.AddDate(1, 0, 0)
		for nextDate.Before(now) || nextDate.Equal(now) {
			nextDate = nextDate.AddDate(1, 0, 0)
		}
		return nextDate.Format(dateTimeFormat), nil
	case "d":
		days, _ := strconv.Atoi(substrs[1])
		nextDate := startDate.AddDate(0, 0, days)
		for nextDate.Before(now) || nextDate.Equal(now) {
			nextDate = nextDate.AddDate(0, 0, days)
		}
		return nextDate.Format(dateTimeFormat), nil
	case "w":
		repeatdays, err := utils.StringToInt(strings.Split(substrs[1], ","))
		if err != nil {
			return "", fmt.Errorf("неподдерживаемый формат")
		}
		var nextDateSlice []time.Time
		for i := 0; i < len(repeatdays); i++ {
			closestDate := startDate
			if closestDate = utils.GetClosestWeekday(repeatdays[i], startDate); closestDate.Before(now) || closestDate.Equal(now) {
				closestDate = utils.GetClosestWeekday(repeatdays[i], now)
			}
			nextDateSlice = append(nextDateSlice, closestDate)
		}
		return utils.FindMinDate(nextDateSlice).Format(dateTimeFormat), nil
	case "m":
		if len(substrs) == 2 {
			repeatmonthdays, err := utils.StringSliceToIntSortAndRemoveDuplicates(strings.Split(substrs[1], ","))
			if err != nil {
				return "", fmt.Errorf("неподдерживаемый формат")
			}
			var nextDateSlice []time.Time
			for i := 0; i < len(repeatmonthdays); i++ {
				closestDate := startDate
				if closestDate = utils.GetClosesDateOfMonth(repeatmonthdays[i], int(startDate.Month()), startDate); closestDate.Before(now) || closestDate.Equal(now) {
					if closestDate = utils.GetClosesDateOfMonth(repeatmonthdays[i], int(now.Month()), now); closestDate.Before(now) || closestDate.Equal(now) {
						closestDate = utils.GetClosesDateOfMonth(repeatmonthdays[i], int(now.Month())+1, now)
					}
				}
				nextDateSlice = append(nextDateSlice, closestDate)
			}
			return utils.FindMinDate(nextDateSlice).Format(dateTimeFormat), nil
		} else if len(substrs) == 3 {

			repeatmonthdays, err := utils.StringSliceToIntSortAndRemoveDuplicates(strings.Split(substrs[1], ","))
			if err != nil {
				return "", fmt.Errorf("неподдерживаемый формат")
			}
			repeatmonths, err := utils.StringSliceToIntSortAndRemoveDuplicates(strings.Split(substrs[2], ","))
			if err != nil {
				return "", fmt.Errorf("неподдерживаемый формат")
			}
			var nextDateSlice []time.Time
			for i := 0; i < len(repeatmonthdays); i++ {
				for j := 0; j < len(repeatmonths); j++ {
					closestDate := startDate
					if closestDate = utils.GetDateOfMonth(repeatmonthdays[i], repeatmonths[j], now, startDate); closestDate.Before(now) || closestDate.Equal(now) {
						continue
					} else {
						nextDateSlice = append(nextDateSlice, closestDate)
					}
				}
			}
			fmt.Println(utils.FindMinDate(nextDateSlice).Format(dateTimeFormat))
			return utils.FindMinDate(nextDateSlice).Format(dateTimeFormat), nil
		}

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
	nowDate, err := time.Parse(dateTimeFormat, nowStr)
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
		if _, err := w.Write(errResp); err != nil {
			http.Error(w, "ошибка записи ответа", http.StatusInternalServerError)
		}
		return
	}

	if task.Title == "" {
		errorMsg := models.HTTPJSONErrorMessageResponse{Error: "Обязательное поле 'title' отсутствует"}
		errResp, _ := json.Marshal(errorMsg)
		if _, err := w.Write(errResp); err != nil {
			http.Error(w, "ошибка записи ответа", http.StatusInternalServerError)
		}
		return
	}

	now, _ := time.Parse(dateTimeFormat, time.Now().Format(dateTimeFormat))

	if task.Date == "" {
		task.Date = now.Format(dateTimeFormat)
	}
	date, err := time.Parse(dateTimeFormat, task.Date)
	if err != nil {
		errorMsg := models.HTTPJSONErrorMessageResponse{Error: "Дата имеет неверный формат"}
		errResp, _ := json.Marshal(errorMsg)
		if _, err := w.Write(errResp); err != nil {
			http.Error(w, "ошибка записи ответа", http.StatusInternalServerError)
		}
		return
	}
	nextDate := ""
	if task.Repeat != "" {
		nextDate, err = NextDate(now, date.Format(dateTimeFormat), task.Repeat)
		if err != nil {
			errorMsg := models.HTTPJSONErrorMessageResponse{Error: err.Error()}
			errResp, _ := json.Marshal(errorMsg)
			if _, err := w.Write(errResp); err != nil {
				http.Error(w, "ошибка записи ответа", http.StatusInternalServerError)
			}
			return
		}
	} else {
		task.Date = now.Format(dateTimeFormat)
	}

	if date.Before(now) && nextDate != "" {
		task.Date = nextDate
	}

	id, err := dbutils.AddTask(task)

	if err != nil {
		errorMsg := models.HTTPJSONErrorMessageResponse{Error: err.Error()}
		errResp, _ := json.Marshal(errorMsg)
		if _, err := w.Write(errResp); err != nil {
			http.Error(w, "ошибка записи ответа", http.StatusInternalServerError)
		}
		return
	} else {
		respData := models.HTTPJSONResponseID{ID: id}
		res, _ := json.Marshal(respData)
		if _, err := w.Write(res); err != nil {
			http.Error(w, "ошибка записи ответа", http.StatusInternalServerError)
		}
		return
	}
}

func getTasksHandler(w http.ResponseWriter, r *http.Request) {
	//проверяем наличие и формат данных в поисковой строке
	searchDateBool := false
	searchStr := ""
	searchStr = r.URL.Query().Get("search")
	searchDate, err := time.Parse("02.01.2006", searchStr)
	if err == nil {
		searchDateBool = true
		searchStr = searchDate.Format(dateTimeFormat)
	}

	tasks, err := dbutils.GetTasks(searchStr, searchDateBool)
	if err != nil {
		errorMsg := models.HTTPJSONErrorMessageResponse{Error: err.Error()}
		errResp, _ := json.Marshal(errorMsg)
		w.Header().Set("Content-Type", "application/json; charset=UTF-8")
		if _, err := w.Write(errResp); err != nil {
			http.Error(w, "ошибка записи ответа", http.StatusInternalServerError)
		}
		return
	}

	tasksList := models.TasksList{Tasks: tasks}
	jsonResp, err := json.Marshal(tasksList)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json; charset=UTF-8")
	if _, err := w.Write(jsonResp); err != nil {
		http.Error(w, "ошибка записи ответа", http.StatusInternalServerError)
	}

}

func getTaskHandler(w http.ResponseWriter, r *http.Request) {

	idParam := r.URL.Query().Get("id")

	w.Header().Set("Content-Type", "application/json; charset=UTF-8")

	if idParam == "" {
		errorMsg := models.HTTPJSONErrorMessageResponse{Error: "Не указан идентификатор"}
		jsonResp, _ := json.Marshal(errorMsg)
		if _, err := w.Write(jsonResp); err != nil {
			http.Error(w, "ошибка записи ответа", http.StatusInternalServerError)
		}
		return
	}

	_, err := strconv.Atoi(idParam)
	if err != nil {
		errorMsg := models.HTTPJSONErrorMessageResponse{Error: "Не указан идентификатор"}
		jsonResp, _ := json.Marshal(errorMsg)
		if _, err := w.Write(jsonResp); err != nil {
			http.Error(w, "ошибка записи ответа", http.StatusInternalServerError)
		}
		return
	}

	task, err := dbutils.GetTaskByID(idParam)
	if err != nil {
		if err.Error() == "задача не найдена" {
			errorMsg := models.HTTPJSONErrorMessageResponse{Error: "задача не найдена"}
			jsonResp, _ := json.Marshal(errorMsg)
			if _, err := w.Write(jsonResp); err != nil {
				http.Error(w, "ошибка записи ответа", http.StatusInternalServerError)
			}
			return
		}
		errorMsg := models.HTTPJSONErrorMessageResponse{Error: "задача не найдена"}
		jsonResp, _ := json.Marshal(errorMsg)
		if _, err := w.Write(jsonResp); err != nil {
			http.Error(w, "ошибка записи ответа", http.StatusInternalServerError)
		}
		return
	}

	jsonResp, _ := json.Marshal(task)
	if _, err := w.Write(jsonResp); err != nil {
		http.Error(w, "ошибка записи ответа", http.StatusInternalServerError)
	}
}

func putTaskHandler(w http.ResponseWriter, r *http.Request) {
	var task models.FullTask

	w.Header().Set("Content-Type", "application/json")

	err := json.NewDecoder(r.Body).Decode(&task)
	if err != nil {
		errorMsg := models.HTTPJSONErrorMessageResponse{Error: "Задача не найдена"}
		jsonResp, _ := json.Marshal(errorMsg)
		if _, err := w.Write(jsonResp); err != nil {
			http.Error(w, "ошибка записи ответа", http.StatusInternalServerError)
		}
		return
	}

	defer r.Body.Close()

	if err := TaskValidate(task); err != nil {
		errorMsg := models.HTTPJSONErrorMessageResponse{Error: err.Error()}
		jsonResp, _ := json.Marshal(errorMsg)
		if _, err := w.Write(jsonResp); err != nil {
			http.Error(w, "ошибка записи ответа", http.StatusInternalServerError)
		}
		return
	}

	currentTask, err := dbutils.GetTaskByID(task.ID)
	if err != nil {
		errorMsg := models.HTTPJSONErrorMessageResponse{Error: err.Error()}
		jsonResp, _ := json.Marshal(errorMsg)
		if _, err := w.Write(jsonResp); err != nil {
			http.Error(w, "ошибка записи ответа", http.StatusInternalServerError)
		}
		return
	}

	if currentTask.ID == "" {
		errorMsg := models.HTTPJSONErrorMessageResponse{Error: "Задача не найдена"}
		jsonResp, _ := json.Marshal(errorMsg)
		if _, err := w.Write(jsonResp); err != nil {
			http.Error(w, "ошибка записи ответа", http.StatusInternalServerError)
		}
		return
	}

	err = dbutils.UpdateTask(task)
	if err != nil {
		errorMsg := models.HTTPJSONErrorMessageResponse{Error: err.Error()}
		jsonResp, _ := json.Marshal(errorMsg)
		if _, err := w.Write(jsonResp); err != nil {
			http.Error(w, "ошибка записи ответа", http.StatusInternalServerError)
		}
		return
	}

	// Возвращаем пустой JSON-объект в случае успеха
	if _, err := w.Write([]byte("{}")); err != nil {
		http.Error(w, "ошибка записи ответа", http.StatusInternalServerError)
	}

}

func doneTaskHandler(w http.ResponseWriter, r *http.Request) {
	idParam := r.URL.Query().Get("id")

	w.Header().Set("Content-Type", "text/plain; charset=utf-8")

	if idParam == "" {
		errorMsg := models.HTTPJSONErrorMessageResponse{Error: "не указан идентификатор"}
		jsonResp, _ := json.Marshal(errorMsg)
		if _, err := w.Write(jsonResp); err != nil {
			http.Error(w, "ошибка записи ответа", http.StatusInternalServerError)
		}
		return
	}

	_, err := strconv.Atoi(idParam)
	if err != nil {
		errorMsg := models.HTTPJSONErrorMessageResponse{Error: "неверный формат идентификатора"}
		jsonResp, _ := json.Marshal(errorMsg)
		if _, err := w.Write(jsonResp); err != nil {
			http.Error(w, "ошибка записи ответа", http.StatusInternalServerError)
		}
		return
	}

	currentTask, err := dbutils.GetTaskByID(idParam)
	if err != nil {
		if err.Error() == "задача не найдена" {
			errorMsg := models.HTTPJSONErrorMessageResponse{Error: "задача не найдена"}
			jsonResp, _ := json.Marshal(errorMsg)
			if _, err := w.Write(jsonResp); err != nil {
				http.Error(w, "ошибка записи ответа", http.StatusInternalServerError)
			}
			return
		}
		errorMsg := models.HTTPJSONErrorMessageResponse{Error: "задача не найдена"}
		jsonResp, _ := json.Marshal(errorMsg)
		if _, err := w.Write(jsonResp); err != nil {
			http.Error(w, "ошибка записи ответа", http.StatusInternalServerError)
		}
		return
	}

	if currentTask.Repeat == "" {
		err = dbutils.DeleteTaskByID(idParam)
		if err != nil {
			errorMsg := models.HTTPJSONErrorMessageResponse{Error: err.Error()}
			jsonResp, _ := json.Marshal(errorMsg)
			if _, err := w.Write(jsonResp); err != nil {
				http.Error(w, "ошибка записи ответа", http.StatusInternalServerError)
			}
			return
		}

		if _, err := w.Write([]byte("{}")); err != nil {
			http.Error(w, "ошибка записи ответа", http.StatusInternalServerError)
		}
		return
	}

	now, _ := time.Parse(dateTimeFormat, time.Now().Format(dateTimeFormat))
	nextDate, err := NextDate(now, currentTask.Date, currentTask.Repeat)
	if err != nil {
		errorMsg := models.HTTPJSONErrorMessageResponse{Error: err.Error()}
		jsonResp, _ := json.Marshal(errorMsg)
		if _, err := w.Write(jsonResp); err != nil {
			http.Error(w, "ошибка записи ответа", http.StatusInternalServerError)
		}
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
		if _, err := w.Write(jsonResp); err != nil {
			http.Error(w, "ошибка записи ответа", http.StatusInternalServerError)
		}
		return
	}

	// Возвращаем пустой JSON-объект в случае успеха
	if _, err := w.Write([]byte("{}")); err != nil {
		http.Error(w, "ошибка записи ответа", http.StatusInternalServerError)
	}

}

func deleteTaskHandler(w http.ResponseWriter, r *http.Request) {
	idParam := r.URL.Query().Get("id")

	w.Header().Set("Content-Type", "text/plain; charset=utf-8")

	if idParam == "" {
		errorMsg := models.HTTPJSONErrorMessageResponse{Error: "Не указан идентификатор"}
		jsonResp, _ := json.Marshal(errorMsg)
		if _, err := w.Write(jsonResp); err != nil {
			http.Error(w, "ошибка записи ответа", http.StatusInternalServerError)
		}
		return
	}

	_, err := strconv.Atoi(idParam)
	if err != nil {
		errorMsg := models.HTTPJSONErrorMessageResponse{Error: "Неверный формат идентификатора"}
		jsonResp, _ := json.Marshal(errorMsg)
		if _, err := w.Write(jsonResp); err != nil {
			http.Error(w, "ошибка записи ответа", http.StatusInternalServerError)
		}
		return
	}

	err = dbutils.DeleteTaskByID(idParam)
	if err != nil {
		errorMsg := models.HTTPJSONErrorMessageResponse{Error: err.Error()}
		jsonResp, _ := json.Marshal(errorMsg)
		if _, err := w.Write(jsonResp); err != nil {
			http.Error(w, "ошибка записи ответа", http.StatusInternalServerError)
		}
		return
	}

	// Возвращаем пустой JSON-объект в случае успеха
	if _, err := w.Write([]byte("{}")); err != nil {
		http.Error(w, "ошибка записи ответа", http.StatusInternalServerError)
	}
}
