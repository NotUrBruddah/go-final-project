# Итоговый проект ЯндексПрактикума курс Go-Basic ![Go](https://img.shields.io/badge/go-%2300ADD8.svg?style=for-the-badge&logo=go&logoColor=white)

Проект является финальным испытанием на программе обучения Яндекс практикум Go-Basic когорта 7
Проект представляет с собой планировщик задач, состоящий из Web-Сервера и БД. позволяет создавать задачи, редактировать, 
удалять и отображать в отсортированном по дате виде.

## Список реализованых задач со звездочкой
- задание со `*`, реализована возможность управлять портом запуска веб-сервера
    через переменную окружения `TODO_PORT`
- задание со `*`, возможность определять путь к БД
    через переменную окружения `TODO_DBFILE` 
- задание со `*`, реализованы правила повторения задач `Еженедельно` и `Ежемесячно`
- задание со `*`, реализован `Поиск` 
- задание со `*`, создан `Dockerfile` для сборки образа и запуска приложения в Docker(ниже см. описание сборки и запуска)  

## Доступны переменные окружения:
- `TODO_PORT` - порт, вебсервера для запуска
- `TODO_DBFILE` - каталог(путь) хранения файла БД (scheduler.db)
- `TODO_WEBDIR` - каталог веб-приложения

Файл `.env` для загрузки переменных окружения (https://github.com/joho/godotenv)

## Реализовано:
- веб-сервер `http://localhost:7540/` или использовать порт в env
- реализован обработчик для `GET /api/nextdate`
- реализован обработчик для `POST /api/task` и функция для добавления данных в БД
- реализован обработчик для `GET /api/tasks`, возвращает список ближайших задач из БД
- реализован обработчик для `GET /api/task?id=<id>` - возвращающий данные по задаче из БД
- реализован обработчик для `PUT /api/task`, изменение задачи в БД
- реализован обработчик для `POST /api/task/done?id=<id>`, который реализует логику отметки о выполнении
- реализован обработчик для `DELETE /api/task/done?id=<id>`

## Успешно пройдены тесты
- успешно пройден тест `go test -run ^TestApp$ ./tests`
- успешно пройден тест `go test -run ^TestDB$ ./tests`
- успешно пройден тест `go test -run ^TestNextDate$ ./tests`
- успешно пройден тест `go test -run ^TestAddTask$ ./tests`
- успешно пройден тест `go test -run ^TestTasks$ ./tests`
- успешно пройден тест `go test -run ^TestTask$ ./tests`
- успешно пройден тест `go test -run ^TestEditTask$ ./tests`
- успешно пройден тест `go test -run ^TestDone$ ./tests`
- успешно пройден тест `go test -run ^TestDelTask$ ./tests`
- Все тесты пройдены успешно `go test ./tests`

## Настройки для тестов
```
package tests

var Port = 7540
var DBFile = "../dbdata/scheduler.db"
var FullNextDate = true
var Search = true
var Token = ``
```

## Docker
- сборка образа
```
docker build -t my-go-app:latest .

```
- запуск контейнера
```
docker run -d -p 7540:7540 --name my-go-app -e TODO_PORT="7540" -e TODO_DBFILE="dbdata/scheduler.db" -e TODO_WEBDIR="web" my-go-app:latest

```

## TO DO List(НЕ ВЫПОЛНЕНО, задачи со *):
- авторизация


