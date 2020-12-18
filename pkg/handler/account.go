// Package handler handle all the http
package handler

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"time"

	uuid "github.com/satori/go.uuid"
	"github.com/teojiahao/HireMe/pkg/database"
	"github.com/teojiahao/HireMe/pkg/queue"
	"github.com/teojiahao/HireMe/pkg/security"
)

// Signup page send a POST to REST API
func Signup(res http.ResponseWriter, req *http.Request) {
	if alreadyLoggedIn(req) {
		http.Redirect(res, req, "/", http.StatusSeeOther)
		return
	}
	// process form submission
	if req.Method == http.MethodPost {
		// get form values
		username := bm.Sanitize(req.FormValue("username"))
		password := bm.Sanitize(req.FormValue("password"))

		if username != "" {
			//check password
			if err := security.CheckPassword(password); err != nil {
				//http.Error(res, fmt.Sprintf("%v", err), http.StatusForbidden)
				tpl.ExecuteTemplate(res, "signup.gohtml", fmt.Sprintf("%v", err))
				return
			}

			hashPassword, err := security.HashPassword(password, "")
			if err != nil {
				http.Error(res, "Internal server error", http.StatusInternalServerError)
				return
			}
			// send user details to API
			jsonValue, _ := json.Marshal(database.User{
				Username: username,
				Password: hashPassword,
			})
			jsonResp, err := http.Post(baseURL+"/"+username, "application/json", bytes.NewBuffer(jsonValue))
			if err != nil {
				http.Error(res, "Internal server error", http.StatusInternalServerError)
				return
			}
			if jsonResp.StatusCode == 409 {
				//http.Error(res, "Username already taken", http.StatusForbidden)
				tpl.ExecuteTemplate(res, "signup.gohtml", "Username already taken")
				return
			}

			// get encrypted key from API
			key, _ := ioutil.ReadAll(jsonResp.Body)
			jsonResp.Body.Close()
			secretKey, _ := security.Decrypt(key, "")

			// create session
			id := uuid.NewV4()
			myCookie := &http.Cookie{
				Name:  "myCookie",
				Value: id.String(),
			}
			http.SetCookie(res, myCookie)
			mapSessions[myCookie.Value] = Session{username, string(secretKey)}

			if _, ok := mapHistory[username]; !ok {
				mapHistory[username] = &queue.Queue{}
			}
			currentTime := time.Now()
			mapHistory[username].Enqueue(queue.History{Time: fmt.Sprintf(currentTime.Format("2006-01-02 3:04PM")), Activity: "Sign up"})
		}
		// redirect to main index
		http.Redirect(res, req, "/updateProfile", http.StatusSeeOther)
		return
	}
	tpl.ExecuteTemplate(res, "signup.gohtml", nil)
}

// Login page send a POST to REST API
func Login(res http.ResponseWriter, req *http.Request) {
	if alreadyLoggedIn(req) {
		http.Redirect(res, req, "/", http.StatusSeeOther)
		return
	}
	// process form submission
	if req.Method == http.MethodPost {
		timer := make(chan string, 1)
		go func() {
			time.Sleep(1 * time.Second)
			timer <- "times up"
		}()

		username := bm.Sanitize(req.FormValue("username"))
		password := bm.Sanitize(req.FormValue("password"))

		// check for ASCII
		if !security.IsASCII(username) || !security.IsASCII(password) {
			//http.Error(res, "ASCII Character only", http.StatusForbidden)
			tpl.ExecuteTemplate(res, "login.gohtml", "ASCII Character only")
			return
		}

		// send user details to API
		jsonValue, _ := json.Marshal(database.User{
			Username: username,
			Password: []byte(password),
		})
		jsonResp, err := http.Post("https://localhost:5000/api/v1/login", "application/json", bytes.NewBuffer(jsonValue))
		if err != nil {
			log.Println(err)
			http.Error(res, "Internal server error", http.StatusInternalServerError)
			return
		}
		if jsonResp.StatusCode == 403 {
			currentTime := time.Now()
			if _, ok := mapHistory[username]; !ok {
				mapHistory[username] = &queue.Queue{}
			}
			mapHistory[username].Enqueue(queue.History{Time: fmt.Sprintf(currentTime.Format("2006-01-02 3:04PM")), Activity: `<p style="color:red;">Failed to login</p>`})
			<-timer
			//http.Error(res, "Username and/or password do not match", http.StatusForbidden)
			tpl.ExecuteTemplate(res, "login.gohtml", "Username and/or password do not match")
			return
		}

		// get encrypted key from API
		key, _ := ioutil.ReadAll(jsonResp.Body)
		jsonResp.Body.Close()
		secretKey, _ := security.Decrypt(key, "")

		// create session
		id := uuid.NewV4()
		myCookie := &http.Cookie{
			Name:  "myCookie",
			Value: id.String(),
		}
		http.SetCookie(res, myCookie)
		mapSessions[myCookie.Value] = Session{username, string(secretKey)}

		currentTime := time.Now()
		if _, ok := mapHistory[username]; !ok {
			mapHistory[username] = &queue.Queue{}
		}
		mapHistory[username].Enqueue(queue.History{Time: fmt.Sprintf(currentTime.Format("2006-01-02 3:04PM")), Activity: `<p style="color:green;">Successfully login</p>`})

		http.Redirect(res, req, "/", http.StatusSeeOther)
		return
	}
	tpl.ExecuteTemplate(res, "login.gohtml", nil)
}

// Logout page remove the cookies from the browser
func Logout(res http.ResponseWriter, req *http.Request) {
	if !alreadyLoggedIn(req) {
		http.Redirect(res, req, "/", http.StatusSeeOther)
		return
	}

	myUser := getUserFromCookie(res, req)

	myCookie, _ := req.Cookie("myCookie")
	// delete the session
	delete(mapSessions, myCookie.Value)
	// remove the cookie
	myCookie = &http.Cookie{
		Name:   "myCookie",
		Value:  "",
		MaxAge: -1,
	}
	http.SetCookie(res, myCookie)

	currentTime := time.Now()
	mapHistory[myUser.Username].Enqueue(queue.History{Time: fmt.Sprintf(currentTime.Format("2006-01-02 3:04PM")), Activity: "Logout"})

	http.Redirect(res, req, "/", http.StatusSeeOther)
}

// check if cookie exist
func getUserFromCookie(res http.ResponseWriter, req *http.Request) Session {
	// get current session cookie
	myCookie, err := req.Cookie("myCookie")
	if err != nil {
		id := uuid.NewV4()
		myCookie = &http.Cookie{
			Name:  "myCookie",
			Value: id.String(),
		}
		http.SetCookie(res, myCookie)
	}

	// if the User exists already, get username
	username, ok := mapSessions[myCookie.Value]
	if !ok {
		return username
	}
	return username
}

// check if user already logged in
func alreadyLoggedIn(req *http.Request) bool {
	myCookie, err := req.Cookie("myCookie")
	if err != nil {
		return false
	}
	session := mapSessions[myCookie.Value]
	// send user details to API
	response, err := http.Get(baseURL + "/" + session.Username)
	if err != nil {
		return false
	}
	if response.StatusCode == 404 {
		return false
	}
	return true
}
