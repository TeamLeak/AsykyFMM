package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"

	_ "github.com/mattn/go-sqlite3"
)

var secretKey = "my_secret_key"

var db *sql.DB

type File struct {
	Name string `json:"name"`
	Hash string `json:"hash"`
}

func initDB() {
	var err error
	db, err = sql.Open("sqlite3", "./files.db")
	if err != nil {
		log.Fatalf("Ошибка подключения к БД: %v", err)
	}
}

func tableExists(tableName string) bool {
	var exists bool
	query := fmt.Sprintf("SELECT count(*) FROM sqlite_master WHERE type='table' AND name='%s';", tableName)
	err := db.QueryRow(query).Scan(&exists)
	if err != nil {
		log.Printf("Ошибка при проверке существования таблицы: %v", err)
		return false
	}
	return exists
}

func createTable(tableName string) error {
	createTableQuery := fmt.Sprintf(`
		CREATE TABLE IF NOT EXISTS %s (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			name TEXT NOT NULL,
			hash TEXT NOT NULL
		);`, tableName)

	_, err := db.Exec(createTableQuery)
	return err
}

func createListHandler(w http.ResponseWriter, r *http.Request) {
	listType := r.URL.Query().Get("type")
	if listType == "" {
		http.Error(w, "Не указан параметр 'type'", http.StatusBadRequest)
		return
	}

	key := r.URL.Query().Get("secret_key")
	if key != secretKey {
		http.Error(w, "Неверный secret_key", http.StatusForbidden)
		return
	}

	if tableExists(listType) {
		http.Error(w, "Список уже существует", http.StatusBadRequest)
		return
	}

	err := createTable(listType)
	if err != nil {
		http.Error(w, "Ошибка при создании списка", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusCreated)
	_, _ = fmt.Fprintf(w, "Список %s успешно создан", listType)
}

func getFilesHandler(w http.ResponseWriter, r *http.Request) {
	listType := r.URL.Query().Get("type")
	if listType == "" {
		http.Error(w, "Не указан параметр 'type'", http.StatusBadRequest)
		return
	}

	if !tableExists(listType) {
		http.Error(w, "Список не существует", http.StatusNotFound)
		return
	}

	rows, err := db.Query(fmt.Sprintf("SELECT name, hash FROM %s", listType))
	if err != nil {
		http.Error(w, "Ошибка чтения данных из БД", http.StatusInternalServerError)
		return
	}
	defer func(rows *sql.Rows) {
		_ = rows.Close()
	}(rows)

	var files []File
	for rows.Next() {
		var file File
		err := rows.Scan(&file.Name, &file.Hash)
		if err != nil {
			http.Error(w, "Ошибка обработки данных", http.StatusInternalServerError)
			return
		}
		files = append(files, file)
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(files)
}

func addFileHandler(w http.ResponseWriter, r *http.Request) {
	listType := r.URL.Query().Get("type")
	if listType == "" {
		http.Error(w, "Не указан параметр 'type'", http.StatusBadRequest)
		return
	}

	key := r.URL.Query().Get("secret_key")
	if key != secretKey {
		http.Error(w, "Неверный secret_key", http.StatusForbidden)
		return
	}

	if !tableExists(listType) {
		http.Error(w, "Список не существует", http.StatusNotFound)
		return
	}

	var newFile File
	err := json.NewDecoder(r.Body).Decode(&newFile)
	if err != nil {
		http.Error(w, "Неверный формат данных", http.StatusBadRequest)
		return
	}

	_, err = db.Exec(fmt.Sprintf("INSERT INTO %s (name, hash) VALUES (?, ?)", listType), newFile.Name, newFile.Hash)
	if err != nil {
		http.Error(w, "Ошибка добавления файла в БД", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusCreated)
	_, _ = fmt.Fprintf(w, "Файл добавлен успешно")
}

func deleteFileHandler(w http.ResponseWriter, r *http.Request) {
	listType := r.URL.Query().Get("type")
	name := r.URL.Query().Get("name")
	if listType == "" || name == "" {
		http.Error(w, "Не указан параметр 'type' или 'name'", http.StatusBadRequest)
		return
	}

	key := r.URL.Query().Get("secret_key")
	if key != secretKey {
		http.Error(w, "Неверный secret_key", http.StatusForbidden)
		return
	}

	if !tableExists(listType) {
		http.Error(w, "Список не существует", http.StatusNotFound)
		return
	}

	_, err := db.Exec(fmt.Sprintf("DELETE FROM %s WHERE name = ?", listType), name)
	if err != nil {
		http.Error(w, "Ошибка удаления файла из БД", http.StatusInternalServerError)
		return
	}

	_, _ = fmt.Fprintf(w, "Файл %s удален из списка %s", name, listType)
}

func deleteListHandler(w http.ResponseWriter, r *http.Request) {
	listType := r.URL.Query().Get("type")
	if listType == "" {
		http.Error(w, "Не указан параметр 'type'", http.StatusBadRequest)
		return
	}

	key := r.URL.Query().Get("secret_key")
	if key != secretKey {
		http.Error(w, "Неверный secret_key", http.StatusForbidden)
		return
	}

	if !tableExists(listType) {
		http.Error(w, "Список не существует", http.StatusNotFound)
		return
	}

	_, err := db.Exec(fmt.Sprintf("DROP TABLE IF EXISTS %s", listType))
	if err != nil {
		http.Error(w, "Ошибка удаления списка", http.StatusInternalServerError)
		return
	}

	_, _ = fmt.Fprintf(w, "Список %s успешно удален", listType)
}

func moveFileHandler(w http.ResponseWriter, r *http.Request) {
	fromType := r.URL.Query().Get("from")
	toType := r.URL.Query().Get("to")
	name := r.URL.Query().Get("name")
	if fromType == "" || toType == "" || name == "" {
		http.Error(w, "Не указан параметр 'from', 'to' или 'name'", http.StatusBadRequest)
		return
	}

	key := r.URL.Query().Get("secret_key")
	if key != secretKey {
		http.Error(w, "Неверный secret_key", http.StatusForbidden)
		return
	}

	if !tableExists(fromType) {
		http.Error(w, fmt.Sprintf("Список '%s' не существует", fromType), http.StatusNotFound)
		return
	}

	if !tableExists(toType) {
		http.Error(w, fmt.Sprintf("Список '%s' не существует", toType), http.StatusNotFound)
		return
	}

	var file File
	err := db.QueryRow(fmt.Sprintf("SELECT name, hash FROM %s WHERE name = ?", fromType), name).Scan(&file.Name, &file.Hash)
	if err != nil {
		http.Error(w, "Файл не найден в списке источника", http.StatusNotFound)
		return
	}

	_, err = db.Exec(fmt.Sprintf("INSERT INTO %s (name, hash) VALUES (?, ?)", toType), file.Name, file.Hash)
	if err != nil {
		http.Error(w, "Ошибка добавления файла в список назначения", http.StatusInternalServerError)
		return
	}

	_, err = db.Exec(fmt.Sprintf("DELETE FROM %s WHERE name = ?", fromType), file.Name)
	if err != nil {
		http.Error(w, "Ошибка удаления файла из списка источника", http.StatusInternalServerError)
		return
	}

	_, _ = fmt.Fprintf(w, "Файл %s перемещен из %s в %s", name, fromType, toType)
}

func main() {
	initDB()

	port := os.Getenv("PORT")
	secretKey = os.Getenv("SECRET_KEY")

	http.HandleFunc("/verify/list", getFilesHandler)
	http.HandleFunc("/verify/add", addFileHandler)
	http.HandleFunc("/verify/remove", deleteFileHandler)
	http.HandleFunc("/verify/remove_list", deleteListHandler)
	http.HandleFunc("/verify/move", moveFileHandler)
	http.HandleFunc("/verify/create_list", createListHandler)

	if port == "" {
		port = "8080"
	}

	fmt.Printf("Сервер запущен на порту %s\n", port)
	log.Fatal(http.ListenAndServe(":"+port, nil))
}
