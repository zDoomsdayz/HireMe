package api

import (
	"encoding/json"
	"io/ioutil"
	"net/http"

	"github.com/teojiahao/HireMe/pkg/security"

	"github.com/gorilla/mux"
	"github.com/teojiahao/HireMe/pkg/database"
)

// check if the user provide key and check if the key exsit inside db
/*func validKey(req *http.Request) bool {
	v := req.URL.Query()
	if key, ok := v["key"]; ok {
		return CheckAPIKey(key[0])
	}
	return false
}*/

// Login func
func Login(res http.ResponseWriter, req *http.Request) {
	if req.Header.Get("Content-type") == "application/json" {
		if req.Method == "POST" {
			var user database.User
			reqBody, err := ioutil.ReadAll(req.Body)
			if err == nil {
				json.Unmarshal(reqBody, &user)

				// Only accept a proper JSON format
				if user.Username == "" {
					res.WriteHeader(http.StatusUnprocessableEntity)
					res.Write([]byte("422 - Please supply course information in JSON format"))
					return
				}

				// Get all user from db
				dbAllUser := database.GetAllUser()
				// check if user exist in the db
				dbUser, ok := dbAllUser[user.Username]
				if !ok {
					res.WriteHeader(http.StatusForbidden)
					res.Write([]byte("403 - Username and/or password do not match"))
					return
				}

				// compare the password with the db password
				err := security.HashPasswordCompare(user.Password, "", dbUser.Password)
				if err != nil {
					res.WriteHeader(http.StatusForbidden)
					res.Write([]byte("403 - Username and/or password do not match"))
					return
				}

				// write something back to user
				secretKey, _ := security.Encrypt([]byte(dbUser.Username), "")
				res.Write(secretKey)

			} else {
				res.WriteHeader(http.StatusUnprocessableEntity)
				res.Write([]byte("422 - Please supply User information in JSON format"))
			}
		}
	}
}

// display all courses in JSON format
func AllUsers(res http.ResponseWriter, req *http.Request) {
	/*if !validKey(req) {
		res.WriteHeader(http.StatusNotFound)
		res.Write([]byte("404 - invalid key!"))
		return
	}*/

	json.NewEncoder(res).Encode(database.UserInfoJSON())
}

// individual course which require GET PUT POST and DELETE method to access
func User(res http.ResponseWriter, req *http.Request) {
	/*if !validKey(req) {
		res.WriteHeader(http.StatusNotFound)
		res.Write([]byte("404 - invalid key!"))
		return
	}*/

	params := mux.Vars(req)

	if req.Method == "GET" {
		// Get all user from DB
		users := database.GetAllUser()

		// Check if user exist
		if _, ok := users[params["username"]]; ok {
			res.WriteHeader(http.StatusOK)
			res.Write([]byte("200 - User found!"))
		} else {
			res.WriteHeader(http.StatusNotFound)
			res.Write([]byte("404 - No user found!"))
		}
	}

	if req.Header.Get("Content-type") == "application/json" {
		if req.Method == "POST" {
			var newUser database.User
			reqBody, err := ioutil.ReadAll(req.Body)
			if err == nil {
				json.Unmarshal(reqBody, &newUser)

				// Only accept a proper JSON format
				if newUser.Username == "" {
					res.WriteHeader(http.StatusUnprocessableEntity)
					res.Write([]byte("422 - Please supply user information in JSON format"))
					return
				}

				// Attempt to Add user into DB
				insertChan := make(chan error)
				go database.InsertUser(string(params["username"]), newUser.Password, insertChan)
				err = <-insertChan
				if err != nil {
					res.WriteHeader(http.StatusConflict)
					res.Write([]byte("409 - Duplicate username"))
				} else {
					res.WriteHeader(http.StatusCreated)
					res.Write([]byte("201 - User added: " + params["username"]))
				}
			} else {
				res.WriteHeader(http.StatusUnprocessableEntity)
				res.Write([]byte("422 - Please supply User information in JSON format"))
			}
		}

		if req.Method == "PATCH" {
			var newUser database.User
			reqBody, err := ioutil.ReadAll(req.Body)
			if err == nil {
				json.Unmarshal(reqBody, &newUser)

				// Only accept a proper JSON format
				if newUser.Username == "" {
					res.WriteHeader(http.StatusUnprocessableEntity)
					res.Write([]byte("422 - Please supply user information in JSON format"))
					return
				}
				// connect to db and update it
				database.UpdateUser(newUser.Username, newUser.Display, newUser.CoordX, newUser.CoordY, newUser.JobType, newUser.Skill, newUser.Exp, newUser.UnemployedDate, newUser.Message, newUser.Email)
			} else {
				res.WriteHeader(http.StatusUnprocessableEntity)
				res.Write([]byte("422 - Please supply user information in JSON format"))
			}
		}
	}
}
