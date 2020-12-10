package main

import (
	"fmt"
	"log"
	"net/http"
	"text/template"

	"github.com/gorilla/mux"
)

func init() {
	tpl = template.Must(template.ParseGlob("templates/*"))
}

func main() {
	router := mux.NewRouter()
	router.HandleFunc("/", index)
	router.HandleFunc("/updateProfile", updateProfile)
	router.HandleFunc("/signup", signup)
	router.HandleFunc("/login", login)
	router.HandleFunc("/logout", logout)

	router.HandleFunc("/api/v1/users", allUsers)
	router.HandleFunc("/api/v1/users/{username}", user).Methods("GET", "PUT", "POST", "DELETE")

	fmt.Println("Listening at port 5000")
	log.Fatal(http.ListenAndServeTLS(":5000", "cert/cert.pem", "cert/key.pem", router))
}
