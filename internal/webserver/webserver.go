package webserverutils

import (
	"log"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"net/http"

	utils "webtasksplannerexample/internal/utils"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

const (
	defaultHttpServerPort = 7540 //стандартный порт для запуска сервера
)

func nextDateHandler(w http.ResponseWriter, r *http.Request) {
	nowStr := r.FormValue("now")
	dateStr := r.FormValue("date")
	repeatStr := r.FormValue("repeat")
	nowDate, err := time.Parse("20060102", nowStr)
	result := ""
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

	w.WriteHeader(http.StatusOK)
	w.Header().Set("Content-Type", "application/json")
	log.Printf("Параметры запроса: now=[%s], date=[%s], repeat=[%s], nextdate=[%s]", nowStr, dateStr, repeatStr, result)
	w.Write([]byte(result))
}

func InitWebServer() {

	router := chi.NewRouter()
	router.Use(middleware.Logger)

	port := defaultHttpServerPort
	workDir, _ := os.Getwd()
	envPort := os.Getenv("TODO_DBFILE")
	// проверяем заданый порт
	// Валидный диапазон 1 - 65635, иначе используем defaultHttpServerPort
	if len(envPort) > 0 {
		if eport, err := strconv.ParseInt(envPort, 10, 32); err == nil {
			if 0 < int(eport) && int(eport) <= 65535 {
				port = int(eport)
			}
		}
	}

	filesDir := http.Dir(filepath.Join(workDir, "web"))
	FileServer(router, "/", filesDir)

	router.Route("/api", func(r chi.Router) {
		r.Get("/nextdate", nextDateHandler)
		// Маршруты для задач
		//r.Route("/task", func(rr chi.Router) {
		//	rr.Post("/", createTaskHandler)
		//	rr.Get("/", tasksHandler)
		//		 Методы для одной задачи
		//		rr.Route("/{id}", func(rri chi.Router) {
		//		    rri.Use(TaskCtx)
		//		    rri.Get("/", taskByIDHandler)
		//		 Добавить другие методы, такие как PUT, DELETE
		//		})
		//})
		// Другие маршруты API
	})

	if err := http.ListenAndServe(":"+strconv.Itoa(port), router); err != nil {
		log.Fatalf("Ошибка при запуске сервера: %s", err.Error())
		return
	}
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
