package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/joho/godotenv"

	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

const (
	defaultHttpServerPort = 7540 //стандартный порт для запуска сервера
)

func init() {
	// Загружаем значения из файла .env проекта
	if err := godotenv.Load(); err != nil {
		fmt.Printf("Не найден файл .env")
	}
}

func main() {
	r := chi.NewRouter()
	r.Use(middleware.Logger)

	port := defaultHttpServerPort
	workDir, _ := os.Getwd()
	envPort := os.Getenv("TODO_PORT")
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
	FileServer(r, "/", filesDir)

	if err := http.ListenAndServe(":"+strconv.Itoa(port), r); err != nil {
		fmt.Printf("Ошибка при запуске сервера: %s", err.Error())
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
