package main

import (
	"context"
	"fmt"
	"github.com/joho/godotenv"
	"log"
	"net/http"
	"os"
	"survey/internal/models"
	"survey/internal/templates"

	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

var client *mongo.Client
var responsesCollection *mongo.Collection

func main() {
	var err error
	godotenv.Load()
	mongoURI := os.Getenv("MONGODB_URI")
	if mongoURI == "" {
		log.Fatal("MONGODB_URI environment variable not set")
	}

	clientOptions := options.Client().ApplyURI(mongoURI)
	client, err = mongo.Connect(context.Background(), clientOptions)
	if err != nil {
		log.Fatal("Failed to connect to MongoDB:", err)
	}
	defer client.Disconnect(context.Background())

	db := client.Database("survey")
	responsesCollection = db.Collection("responses")

	serveStaticFiles()
	http.HandleFunc("/", formHandler)
	http.HandleFunc("/submit", submitHandler)

	fmt.Println("Server started at http://localhost:8080")
	http.ListenAndServe("0.0.0.0:3000", nil)
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

	name := r.FormValue("question_0")

	answers := make(map[string]string)
	for key, values := range r.Form {
		if key == "question_0" {
			continue
		}
		answers[key] = values[0]
	}

	responseDoc := bson.M{
		"name":       name,
		"answers":    answers,
		"created_at": time.Now(),
	}

	_, err := responsesCollection.InsertOne(context.Background(), responseDoc)
	if err != nil {
		http.Error(w, "Failed to insert data", http.StatusInternalServerError)
		log.Println("Error inserting data into MongoDB:", err)
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
