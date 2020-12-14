package database

import (
	"database/sql"
	"fmt"
	"log"
	"os"
	"sync"
)

// User struct for db
type User struct {
	Username       string
	Password       []byte
	Display        string
	CoordX         float64
	CoordY         float64
	JobType        string
	Skill          string
	Exp            int
	UnemployedDate string
	Message        string
	Email          string
}

// UserJSON for RESTAPI
type UserJSON struct {
	Username       string
	CoordX         float64
	CoordY         float64
	JobType        string
	Skill          string
	Exp            int
	UnemployedDate string
	Message        string
	Email          string
}

// OpenSQL return a opened db
func OpenSQL() *sql.DB {
	db, err := sql.Open("mysql", os.Getenv("DATABASE_IP"))

	if err != nil {
		log.Panic(err.Error())
	} /*else {
		fmt.Println("Database opened!")
	}*/
	return db
}

// InsertUser takes in the username, password and key and store into db
func InsertUser(username string, pass []byte, errChan chan error) {
	var mutex sync.Mutex
	db := OpenSQL()
	defer db.Close()
	query := fmt.Sprintf("INSERT INTO Users VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)")
	mutex.Lock()
	defer mutex.Unlock()
	statement, _ := db.Prepare(query)
	_, err := statement.Exec(username, pass, "No", 0, 0, "", "", 0, "", "", "")
	if err != nil {
		errChan <- fmt.Errorf("409 - Duplicate Username")
		return
	}
	errChan <- nil
}

// UpdateUser takes in the username, password and key and store into db
func UpdateUser(username string, display string, coordX, coordY float64, jobType string, skill string, exp int, unemployedDate string, message string, email string) {
	db := OpenSQL()
	defer db.Close()
	query := fmt.Sprintf("UPDATE Users SET Display='%s', CoordX='%v', CoordY='%v', JobType='%s', Skill='%s', Exp='%v', UnemployedDate='%s', Message='%s', Email='%s' WHERE Username=?", display, coordX, coordY, jobType, skill, exp, unemployedDate, message, email)

	_, err := db.Query(query, username)

	if err != nil {
		log.Panic(fmt.Sprintf("%s", err.Error()))
	}
}

// GetAllUser get all the users details in db and return back a map of user
func GetAllUser() map[string]User {
	db := OpenSQL()
	defer db.Close()
	results, err := db.Query("Select * from my_db.Users")
	users := map[string]User{}

	if err != nil {
		log.Panic(fmt.Sprintf("%s", err.Error()))
	}
	for results.Next() {
		var user User
		err := results.Scan(&user.Username, &user.Password, &user.Display, &user.CoordX, &user.CoordY, &user.JobType, &user.Skill, &user.Exp, &user.UnemployedDate, &user.Message, &user.Email)
		if err != nil {
			panic(err.Error)
		}
		users[user.Username] = user
	}
	return users
}

// UserInfoJSON get all the users details in db and return back a map of user
func UserInfoJSON() map[string]UserJSON {
	db := OpenSQL()
	defer db.Close()
	results, err := db.Query("Select * from my_db.Users")
	users := map[string]UserJSON{}

	if err != nil {
		log.Panic(fmt.Sprintf("%s", err.Error()))
	}
	for results.Next() {
		var user User
		err := results.Scan(&user.Username, &user.Password, &user.Display, &user.CoordX, &user.CoordY, &user.JobType, &user.Skill, &user.Exp, &user.UnemployedDate, &user.Message, &user.Email)
		if err != nil {
			log.Panic(fmt.Sprintf("%s", err.Error()))
		}
		if user.Display == "Yes" {
			users[user.Username] = UserJSON{user.Username, user.CoordX, user.CoordY, user.JobType, user.Skill, user.Exp, user.UnemployedDate, user.Message, user.Email}
		}
	}
	return users
}
