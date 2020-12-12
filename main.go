package main

import (
	"fmt"
	"log"
	"net/http"
	"text/template"

	"github.com/gorilla/mux"
	"github.com/teojiahao/HireMe/pkg/api"
)

func init() {
	tpl = template.Must(template.ParseGlob("templates/*"))
}

func main() {
	router := mux.NewRouter()
	router.HandleFunc("/", index)
	router.HandleFunc("/activity", activity)
	router.HandleFunc("/updateProfile", updateProfile)
	router.HandleFunc("/signup", signup)
	router.HandleFunc("/login", login)
	router.HandleFunc("/logout", logout)

	router.HandleFunc("/api/v1/users", api.AllUsers)
	router.HandleFunc("/api/v1/users/{username}", api.User).Methods("GET", "PUT", "POST", "DELETE")

	fmt.Println("Listening at port 5000")
	log.Fatal(http.ListenAndServeTLS(":5000", "cert/cert.pem", "cert/key.pem", router))
}
