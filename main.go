package main

import (
	"log"
	"net/http"
	"os"

	_ "github.com/go-sql-driver/mysql"
	"github.com/gorilla/mux"
	"github.com/joho/godotenv"
	"github.com/teojiahao/HireMe/pkg/api"
	"github.com/teojiahao/HireMe/pkg/handler"
)

func init() {
	err := godotenv.Load()
	if err != nil {
		log.Fatal("Error loading .env file")
	}
}

func main() {
	router := mux.NewRouter()
	router.HandleFunc("/", handler.Index)
	router.HandleFunc("/activity", handler.Activity)
	router.HandleFunc("/updateProfile", handler.UpdateProfile)
	router.HandleFunc("/signup", handler.Signup)
	router.HandleFunc("/login", handler.Login)
	router.HandleFunc("/logout", handler.Logout)

	router.HandleFunc("/api/v1/users", api.AllUsers)
	router.HandleFunc("/api/v1/users/{username}", api.User).Methods("GET", "PUT", "POST", "DELETE")

	log.Println("Listening at port", os.Getenv("PORT"))
	log.Fatal(http.ListenAndServeTLS(":"+os.Getenv("PORT"), "cert/cert.pem", "cert/key.pem", router))
}
