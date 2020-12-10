package main

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"sort"
	"strconv"
	"strings"
	"text/template"
	"time"

	"github.com/joho/godotenv"
	uuid "github.com/satori/go.uuid"
	"github.com/teojiahao/HireMe/pkg/security"
	"golang.org/x/crypto/bcrypt"
	"googlemaps.github.io/maps"
)

var (
	tpl         *template.Template
	mapSessions = map[string]string{}
	baseURL     string
	jobType     []string
	jobCategory []string
)

// init load up env file
func init() {
	err := godotenv.Load()
	if err != nil {
		log.Fatal("Error loading .env file")
	}

	baseURL = os.Getenv("API")
	jobType = []string{"Full–time", "Part-time", "Contractor", "Internship"}
	jobCategory = []string{"Restaurant and Hospitality", "Sales and Retail", "Education", "Admin and Office", "Healthcare", "Cleaning and Facilities", "Transportation and Logistics", "Manufacturing and Warehouse", "Customer Service", "Personal Care and Services", "Art, Fashion and Design", "Human Resources", "Advertising and Marketing", "Management", "Accounting and Finance", "Business Operations", "Protective Services", "Science and Engineering", "Animal Care", "Computer and IT", "Sports Fitness and Recreation", "Installation, Maintenance and Repair", "Legal", "Media, Communications and Writing", "Construction", "Entertainment and Travel", "Farming and Outdoors", "Energy and Mining", "Property", "Social Services and Non-Profit"}
	sort.Strings(jobCategory)
}

func checkSubstrings(str string, subs []string) bool {
	for _, sub := range subs {
		if strings.Contains(str, sub) {
			return true
		}
	}
	return false
}

func index(res http.ResponseWriter, req *http.Request) {
	myUser := getUserFromCookie(res, req)

	userJSON := getUsers("", "")
	filterUser := map[string]UserJSON{}
	err := json.Unmarshal([]byte(userJSON), &filterUser)
	if err != nil {
		fmt.Println(err)
	}

	req.ParseForm()
	if len(req.Form["Type"]) > 0 {
		jType := req.Form["Type"]

		for k, v := range filterUser {
			if !checkSubstrings(v.JobType, jType) {
				delete(filterUser, k)
			}
		}
	}

	if len(req.Form["Category"]) > 0 {
		cat := req.Form["Category"]

		for k, v := range filterUser {
			if !checkSubstrings(v.Skill, cat) {
				delete(filterUser, k)
			}
		}
	}

	if req.FormValue("exp") != "" {
		exp, _ := strconv.Atoi(req.FormValue("exp"))

		for k, v := range filterUser {
			if v.Exp < exp {
				delete(filterUser, k)
			}
		}
	}

	if req.FormValue("uDays") != "" {
		uDays, _ := strconv.Atoi(req.FormValue("uDays"))

		for k, v := range filterUser {
			if v.UnemployedDate != "" {
				then, err := time.Parse("2006-01-02", v.UnemployedDate)
				if err != nil {
					fmt.Println(err)
					return
				}
				duration := time.Since(then)
				durationDays := int(duration.Hours() / 24)

				dateToDays := v
				dateToDays.UnemployedDate = fmt.Sprintf("%s (%v Days)", v.UnemployedDate, durationDays)
				filterUser[k] = dateToDays

				if durationDays < uDays {
					delete(filterUser, k)
				}
			}
		}
	}

	if req.FormValue("keyword") != "" {
		keyword := req.FormValue("keyword")

		for k, v := range filterUser {
			if !strings.Contains(strings.ToLower(v.Message), strings.ToLower(keyword)) {
				delete(filterUser, k)
			}
		}
	}

	data := struct {
		MyUser      User
		AllUser     map[string]UserJSON
		Type        []string
		Category    []string
		GoogleAPI   string
		GoogleMapID string
	}{
		myUser,
		filterUser,
		jobType,
		jobCategory,
		os.Getenv("GOOGLE_API"),
		os.Getenv("GOOGLE_MAP_ID"),
	}

	tpl.ExecuteTemplate(res, "index.gohtml", data)
}

func updateProfile(res http.ResponseWriter, req *http.Request) {

	myUser := getUserFromCookie(res, req)

	if req.Method == http.MethodPost {
		options := req.FormValue("options")
		jsonValue := []byte{}
		if options == "Yes" {
			req.ParseForm()
			postal := req.FormValue("postal")
			jobType := req.Form["Type"]
			category := req.Form["Category"]
			exp, _ := strconv.Atoi(req.FormValue("exp"))
			lastDay := req.FormValue("lastDay")
			message := req.FormValue("message")
			email := req.FormValue("email")

			// check if postal code valid
			x, y, err := getCoordFromPostal(postal)
			if err != nil {
				http.Error(res, "Invalid Postal Code", http.StatusForbidden)
				return
			}

			if len(jobType) == 0 {
				http.Error(res, "Please select at least 1 Job Type", http.StatusForbidden)
				return
			}

			if len(category) == 0 {
				http.Error(res, "Please select at least 1 Category", http.StatusForbidden)
				return
			}

			if req.FormValue("exp") == "" {
				http.Error(res, "Please enter your years of experience", http.StatusForbidden)
				return
			}

			if lastDay == "" {
				http.Error(res, "Please enter your last day of work", http.StatusForbidden)
				return
			}
			// check if selected date valid
			then, err := time.Parse("2006-01-02", lastDay)
			if err != nil {
				fmt.Println(err)
				return
			}
			duration := time.Since(then)
			durationDays := int(duration.Hours() / 24)

			if durationDays < 0 {
				http.Error(res, "Unemployed Date cannot be in the future", http.StatusForbidden)
				return
			}

			//check email
			if err := security.CheckEmail(email); err != nil {
				http.Error(res, fmt.Sprintf("%v", err), http.StatusForbidden)
				return
			}

			jsonValue, _ = json.Marshal(User{myUser.Username, []byte{}, options, x, y, strings.Join(jobType, ","), strings.Join(category, ","), exp, lastDay, message, email})
		} else {
			jsonValue, _ = json.Marshal(User{myUser.Username, []byte{}, options, 0, 0, "", "", 0, "", "", ""})
		}

		request, err := http.NewRequest(http.MethodPut, baseURL+"/"+myUser.Username, bytes.NewBuffer(jsonValue))
		request.Header.Set("Content-Type", "application/json")
		client := &http.Client{}
		_, err = client.Do(request)
		if err != nil {
			http.Error(res, "Internal server error", http.StatusInternalServerError)
		}

		// redirect to main index
		http.Redirect(res, req, "/", http.StatusSeeOther)
		return
	}

	data := struct {
		Type     []string
		Category []string
	}{
		jobType,
		jobCategory,
	}

	tpl.ExecuteTemplate(res, "updateProfile.gohtml", data)
}

func getCoordFromPostal(postal string) (float64, float64, error) {
	c, err := maps.NewClient(maps.WithAPIKey(os.Getenv("GOOGLE_API")))
	if err != nil {
		log.Fatalf("fatal error: %s", err)
	}
	r := &maps.GeocodingRequest{
		Address: postal,
		Region:  "SG",
	}
	resp, err := c.Geocode(context.Background(), r)
	if len(resp) > 0 {
		return resp[0].Geometry.Location.Lat, resp[0].Geometry.Location.Lng, nil
	}
	return 0, 0, fmt.Errorf("invalid postal code")
}

// Accessing the REST API and return back the JSON as string
func getUsers(code, key string) string {
	url := baseURL

	if code != "" {
		url = baseURL + "/" + code + "?key=" + key
	} else {
		url = baseURL + "?key=" + key
	}
	http.DefaultTransport.(*http.Transport).TLSClientConfig = &tls.Config{InsecureSkipVerify: true}
	response, err := http.Get(url)
	if err != nil {
		fmt.Println("Error")
	} else {
		data, _ := ioutil.ReadAll(response.Body)
		//fmt.Println(string(data))
		response.Body.Close()
		return string(data)
	}
	return ""
}

// signup page
func signup(res http.ResponseWriter, req *http.Request) {
	if alreadyLoggedIn(req) {
		http.Redirect(res, req, "/", http.StatusSeeOther)
		return
	}
	var myUser User
	// process form submission
	if req.Method == http.MethodPost {
		// get form values
		username := req.FormValue("username")
		password := req.FormValue("password")
		//postal := req.FormValue("postal")
		if username != "" {

			mapUsers := GetUser()

			// check if username exist/ taken
			if _, ok := mapUsers[username]; ok {
				http.Error(res, "Username already taken", http.StatusForbidden)
				return
			}

			// create session
			id := uuid.NewV4()
			myCookie := &http.Cookie{
				Name:  "myCookie",
				Value: id.String(),
			}
			http.SetCookie(res, myCookie)
			mapSessions[myCookie.Value] = username
			bPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.MinCost)
			if err != nil {
				http.Error(res, "Internal server error", http.StatusInternalServerError)
				return
			}
			// save the user details into DB
			jsonValue, _ := json.Marshal(User{username, bPassword, "", 0, 0, "", "", 0, "", "", ""})
			_, err = http.Post(baseURL+"/"+username, "application/json", bytes.NewBuffer(jsonValue))
			if err != nil {
				http.Error(res, "Internal server error", http.StatusInternalServerError)
			}
		}
		// redirect to main index
		http.Redirect(res, req, "/updateProfile", http.StatusSeeOther)
		return
	}
	tpl.ExecuteTemplate(res, "signup.gohtml", myUser)
}

// login page
func login(res http.ResponseWriter, req *http.Request) {
	if alreadyLoggedIn(req) {
		http.Redirect(res, req, "/", http.StatusSeeOther)
		return
	}
	// process form submission
	if req.Method == http.MethodPost {
		username := req.FormValue("username")
		password := req.FormValue("password")
		// check if User exist with username

		mapUsers := GetUser()

		myUser, ok := mapUsers[username]
		if !ok {
			http.Error(res, "Username and/or password do not match", http.
				StatusForbidden)
			return
		}
		// Matching of password entered
		err := bcrypt.CompareHashAndPassword(myUser.Password, []byte(password))
		if err != nil {
			http.Error(res, "Username and/or password do not match", http.
				StatusForbidden)
			return
		}
		// create session
		id := uuid.NewV4()
		myCookie := &http.Cookie{
			Name:  "myCookie",
			Value: id.String(),
		}
		http.SetCookie(res, myCookie)
		mapSessions[myCookie.Value] = username
		http.Redirect(res, req, "/", http.StatusSeeOther)
		return
	}
	tpl.ExecuteTemplate(res, "login.gohtml", nil)
}

// logout page
func logout(res http.ResponseWriter, req *http.Request) {
	if !alreadyLoggedIn(req) {
		http.Redirect(res, req, "/", http.StatusSeeOther)
		return
	}
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
	http.Redirect(res, req, "/", http.StatusSeeOther)
}

// check if cookie exist
func getUserFromCookie(res http.ResponseWriter, req *http.Request) User {
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

	// if the User exists already, get User

	mapUsers := GetUser()
	var myUser User
	if username, ok := mapSessions[myCookie.Value]; ok {
		myUser = mapUsers[username]
	}
	return myUser
}

// check if user already logged in
func alreadyLoggedIn(req *http.Request) bool {
	myCookie, err := req.Cookie("myCookie")
	if err != nil {
		return false
	}

	mapUsers := GetUser()

	username := mapSessions[myCookie.Value]
	_, ok := mapUsers[username]
	return ok
}
