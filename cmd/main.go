package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"strings"
	"survey/internal/models"
	"survey/internal/templates"
	"time"

	"github.com/gorilla/sessions"
	"github.com/joho/godotenv"
	_ "github.com/mattn/go-sqlite3"
)

var db *sql.DB
var store = sessions.NewCookieStore([]byte("your-secret-key"))

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
	http.HandleFunc("/", redirectHandler)
	http.HandleFunc("/page/", formHandler)
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

func redirectHandler(w http.ResponseWriter, r *http.Request) {
	http.Redirect(w, r, "/page/1", http.StatusFound)
}

func formHandler(w http.ResponseWriter, r *http.Request) {
	session, err := store.Get(r, "survey_data")
	if err != nil {
		http.Error(w, "Session error: "+err.Error(), http.StatusInternalServerError)
		return
	}

	if session.Values == nil {
		session.Values = make(map[interface{}]interface{})
		session.Values["language"] = "Any"
	}

	if r.Method == http.MethodPost {
		log.Println("Session values after GET in POST:")
		for k, v := range session.Values {
			log.Printf("Key: %v, Value: %v (Type: %T)", k, v, v)
		}

		r.ParseForm()
		for key, values := range r.Form {
			if key == "question_6" {
				switch values[0] {
				case "A":
					session.Values["language"] = "Python"
				case "B":
					session.Values["language"] = "R"
				case "C":
					session.Values["language"] = "Java"
				}
			}
			session.Values[key] = values[0]
		}

		if err := session.Save(r, w); err != nil {
			http.Error(w, "Session save error: "+err.Error(), http.StatusInternalServerError)
			return
		}
	}

	ip := r.RemoteAddr
	if forwarded := r.Header.Get("X-Forwarded-For"); forwarded != "" {
		ip = forwarded
	}
	path := strings.Split(r.URL.Path, "/")
	if len(path) < 3 {
		http.Error(w, "page number missing", http.StatusBadRequest)
	}
	pageNum, err := strconv.Atoi(path[2])
	if err != nil {
		http.Error(w, "page number incorrect", http.StatusBadRequest)
	}

	log.Printf("Form submission from IP: %s", ip)
	log.Printf("%d", pageNum)

	questions := models.ParseJson("questions.json")
	questionsByPage := make([]models.HtmlQuestion, 0)
	submitPageNum := 0
	for _, question := range questions {
		questionPage, _ := strconv.Atoi(question.Question.Page)
		if submitPageNum < questionPage {
			submitPageNum = questionPage
		}
		if questionPage == pageNum && (question.Question.Language == session.Values["language"] || question.Question.Language == "any") {
			questionsByPage = append(questionsByPage, question)
		}
	}
	finalPage := false
	if submitPageNum == pageNum {
		finalPage = true
	}
	templates.Survey(questionsByPage, finalPage, pageNum).Render(r.Context(), w)
}
func submitHandler(w http.ResponseWriter, r *http.Request) {
	session, err := store.Get(r, "survey_data")
	if err != nil {
		http.Error(w, "Session error: "+err.Error(), http.StatusInternalServerError)
		return
	}
	for k, v := range session.Values {
		log.Printf("Key: %v, Value: %v (Type: %T)", k, v, v)
	}
	if r.Method != http.MethodPost {
		http.Error(w, "Invalid request method", http.StatusMethodNotAllowed)
		return
	}

	ip := r.RemoteAddr
	if forwarded := r.Header.Get("X-Forwarded-For"); forwarded != "" {
		ip = forwarded
	}

	name, ok := session.Values["question_0"].(string)
	if !ok {
		name = "Anonymous"
	}

	answers := make(map[string]string)
	for key, value := range session.Values {
		keyStr, ok := key.(string)
		if !ok || keyStr == "question_0" {
			continue
		}
		valStr, ok := value.(string)
		if !ok {
			continue
		}
		log.Println(valStr + keyStr)
		answers[keyStr] = valStr
	}

	r.ParseForm()
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

	session.Options.MaxAge = -1
	if err := session.Save(r, w); err != nil {
		log.Printf("Error deleting session: %v", err)
	}

	w.Write([]byte("Form submitted successfully!"))
}

func serveStaticFiles() {
	http.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.Dir("./static"))))
}
