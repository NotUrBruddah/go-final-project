package models

type ServiceConfig struct {
	DbFilePath     string
	HTTPServerPort int
	HTTPWebDir     string
}

type Task struct {
	Date    string `json:"date"`
	Title   string `json:"title"`
	Comment string `json:"comment,omitempty"` // Опциональный параметр
	Repeat  string `json:"repeat,omitempty"`  // Опциональный параметр
}

type FullTask struct {
	ID string `json:"id"`
	Task
}

type TasksList struct {
	Tasks []FullTask `json:"tasks"`
}

type HTTPJSONResponseID struct {
	ID int64 `json:"id"`
}

type HTTPJSONErrorMessageResponse struct {
	Error string `json:"error"`
}
