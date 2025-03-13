package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"github.com/joho/godotenv"
	"log"
	"net/http"
	"survey/internal/models"
	"survey/internal/templates"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

var db *sql.DB

func main() {
	var err error
	godotenv.Load()
	db, err = sql.Open("sqlite3", "survey.db")
	if err != nil {
		log.Fatal("Failed to connect to SQLite:", err)
	}
	defer db.Close()

	createTable()

	serveStaticFiles()
	http.HandleFunc("/", formHandler)
	http.HandleFunc("/submit", submitHandler)

	fmt.Println("Server started at http://localhost:8080")
	http.ListenAndServe("0.0.0.0:3000", nil)
}

func createTable() {
	query := `CREATE TABLE IF NOT EXISTS responses (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		name TEXT,
		answers TEXT,
		created_at TIMESTAMP,
    ip TEXT
	)`

	_, err := db.Exec(query)
	if err != nil {
		log.Fatal("Failed to create table:", err)
	}
}

func formHandler(w http.ResponseWriter, r *http.Request) {
	ip := r.RemoteAddr
	if forwarded := r.Header.Get("X-Forwarded-For"); forwarded != "" {
		ip = forwarded
	}

	log.Printf("Form submission from IP: %s", ip)

	questions := models.ParseJson("questions.json")
	templates.Survey(questions).Render(r.Context(), w)
}

func submitHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Invalid request method", http.StatusMethodNotAllowed)
		return
	}
	r.ParseForm()
	ip := r.RemoteAddr
	if forwarded := r.Header.Get("X-Forwarded-For"); forwarded != "" {
		ip = forwarded
	}
	name := r.FormValue("question_0")

	answers := make(map[string]string)
	for key, values := range r.Form {
		if key == "question_0" {
			continue
		}
		answers[key] = values[0]
	}

	answersJson, err := json.Marshal(answers)
	if err != nil {
		http.Error(w, "Failed to encode data", http.StatusInternalServerError)
		log.Println("Error encoding data to JSON:", err)
		return
	}

	_, err = db.Exec("INSERT INTO responses (name, answers, created_at, ip) VALUES (?, ?, ?, ?)", name, string(answersJson), time.Now(), ip)
	if err != nil {
		http.Error(w, "Failed to insert data", http.StatusInternalServerError)
		log.Println("Error inserting data into SQLite:", err)
		return
	}

	fmt.Println("Received Form Data:")
	fmt.Printf("Name: %s\n", name)
	for key, value := range answers {
		fmt.Printf("%s: %s\n", key, value)
	}

	w.Write([]byte("Form submitted successfully!"))
}

func serveStaticFiles() {
	http.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.Dir("./static"))))
}
