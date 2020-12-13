package handler

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
	"sync"
	"text/template"
	"time"

	"github.com/joho/godotenv"
	"github.com/teojiahao/HireMe/pkg/database"
	"github.com/teojiahao/HireMe/pkg/queue"
	"github.com/teojiahao/HireMe/pkg/security"

	uuid "github.com/satori/go.uuid"
	"golang.org/x/crypto/bcrypt"
	"googlemaps.github.io/maps"
)

var (
	tpl         *template.Template
	mapSessions = map[string]string{}
	baseURL     string
	jobType     []string
	jobCategory []string
	mapHistory  = map[string]*queue.Queue{}
	mapMutex    sync.RWMutex
	wg          sync.WaitGroup
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

	tpl = template.Must(template.ParseGlob("templates/*"))
}

func checkSubstrings(str string, subs []string) bool {
	for _, sub := range subs {
		if strings.Contains(str, sub) {
			return true
		}
	}
	return false
}

func safeMapDelete(filterUser map[string]database.UserJSON, username string) {
	mapMutex.Lock()
	delete(filterUser, username)
	mapMutex.Unlock()
}

// Index page is the main feature of this application
func Index(res http.ResponseWriter, req *http.Request) {
	myUser := getUserFromCookie(res, req)

	userJSON := getUsers("", "")
	filterUser := map[string]database.UserJSON{}
	err := json.Unmarshal([]byte(userJSON), &filterUser)
	if err != nil {
		fmt.Println(err)
	}

	activity := ""
	req.ParseForm()
	if len(req.Form["Type"]) > 0 {
		jType := req.Form["Type"]
		wg.Add(1)
		go func() {
			mapMutex.RLock()
			for k, v := range filterUser {
				if !checkSubstrings(v.JobType, jType) {
					mapMutex.RUnlock()
					safeMapDelete(filterUser, k)
					mapMutex.RLock()
				}
			}
			activity += strings.Join(jType, ", ") + " "

			mapMutex.RUnlock()
			wg.Done()
		}()
	}

	if len(req.Form["Category"]) > 0 {
		cat := req.Form["Category"]
		wg.Add(1)
		go func() {
			mapMutex.RLock()
			for k, v := range filterUser {
				if !checkSubstrings(v.Skill, cat) {
					mapMutex.RUnlock()
					safeMapDelete(filterUser, k)
					mapMutex.RLock()
				}
			}

			activity += strings.Join(cat, ", ") + " "
			mapMutex.RUnlock()
			wg.Done()
		}()
	}

	if req.FormValue("exp") != "" {
		exp, _ := strconv.Atoi(req.FormValue("exp"))
		wg.Add(1)
		go func() {
			mapMutex.RLock()
			for k, v := range filterUser {
				if v.Exp < exp {
					mapMutex.RUnlock()
					safeMapDelete(filterUser, k)
					mapMutex.RLock()
				}
			}
			activity += req.FormValue("exp") + "Years Of Exp "
			mapMutex.RUnlock()
			wg.Done()
		}()
	}

	if req.FormValue("uDays") != "" {
		uDays, _ := strconv.Atoi(req.FormValue("uDays"))
		wg.Add(1)
		go func() {
			mapMutex.RLock()
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
						mapMutex.RUnlock()
						safeMapDelete(filterUser, k)
						mapMutex.RLock()
					}
				}
			}
			activity += req.FormValue("uDays") + "Days unemployed"
			mapMutex.RUnlock()
			wg.Done()
		}()
	}

	if req.FormValue("keyword") != "" {
		keyword := req.FormValue("keyword")
		wg.Add(1)
		go func() {
			mapMutex.RLock()
			for k, v := range filterUser {
				if !strings.Contains(strings.ToLower(v.Message), strings.ToLower(keyword)) {
					mapMutex.RUnlock()
					safeMapDelete(filterUser, k)
					mapMutex.RLock()
				}
			}
			activity += req.FormValue("keyword") + " "
			mapMutex.RUnlock()
			wg.Done()
		}()
	}
	wg.Wait()
	if len(req.Form["Type"]) > 0 || len(req.Form["Category"]) > 0 || req.FormValue("exp") != "" || req.FormValue("uDays") != "" || req.FormValue("keyword") != "" {
		if _, ok := mapHistory[myUser.Username]; ok {
			currentTime := time.Now()
			mapHistory[myUser.Username].Enqueue(queue.History{Time: fmt.Sprintf(currentTime.Format("2006-01-02 3:04PM")), Activity: "Filter: " + activity})
		}
	}

	data := struct {
		MyUser      database.User
		AllUser     map[string]database.UserJSON
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

// reverse the slice of history for html display
func reverse(history []queue.History) []queue.History {
	for i, j := 0, len(history)-1; i < j; i, j = i+1, j-1 {
		history[i], history[j] = history[j], history[i]
	}
	return history
}

// Activity page
func Activity(res http.ResponseWriter, req *http.Request) {
	myUser := getUserFromCookie(res, req)
	allActivity := []queue.History{}

	if _, ok := mapHistory[myUser.Username]; ok {
		allActivity = mapHistory[myUser.Username].AllHistory()
	}

	tpl.ExecuteTemplate(res, "activity.gohtml", reverse(allActivity))
}

// UpdateProfile page helps user to plot on the google map with its details
func UpdateProfile(res http.ResponseWriter, req *http.Request) {

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

			jsonValue, _ = json.Marshal(database.User{
				Username:       myUser.Username,
				Display:        options,
				CoordX:         x,
				CoordY:         y,
				JobType:        strings.Join(jobType, ", "),
				Skill:          strings.Join(category, ", "),
				Exp:            exp,
				UnemployedDate: lastDay,
				Message:        message,
				Email:          email,
			})
		} else {
			jsonValue, _ = json.Marshal(database.User{
				Username: myUser.Username,
				Display:  options,
			})
		}

		request, err := http.NewRequest(http.MethodPut, baseURL+"/"+myUser.Username, bytes.NewBuffer(jsonValue))
		request.Header.Set("Content-Type", "application/json")
		client := &http.Client{}
		_, err = client.Do(request)
		if err != nil {
			http.Error(res, "Internal server error", http.StatusInternalServerError)
		}

		currentTime := time.Now()
		mapHistory[myUser.Username].Enqueue(queue.History{Time: fmt.Sprintf(currentTime.Format("2006-01-02 3:04PM")), Activity: "Updated Profile"})

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

// Signup page create an account and store it into database
func Signup(res http.ResponseWriter, req *http.Request) {
	if alreadyLoggedIn(req) {
		http.Redirect(res, req, "/", http.StatusSeeOther)
		return
	}
	var myUser database.User
	// process form submission
	if req.Method == http.MethodPost {
		// get form values
		username := req.FormValue("username")
		password := req.FormValue("password")
		//postal := req.FormValue("postal")
		if username != "" {

			mapUsers := database.GetUser()

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

			if _, ok := mapHistory[myUser.Username]; !ok {
				mapHistory[username] = &queue.Queue{}
			}
			currentTime := time.Now()
			mapHistory[username].Enqueue(queue.History{Time: fmt.Sprintf(currentTime.Format("2006-01-02 3:04PM")), Activity: "Sign up"})
			bPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.MinCost)
			if err != nil {
				http.Error(res, "Internal server error", http.StatusInternalServerError)
				return
			}
			// save the user details into DB
			jsonValue, _ := json.Marshal(database.User{
				Username: username,
				Password: bPassword,
			})
			jsonResp, err := http.Post(baseURL+"/"+username, "application/json", bytes.NewBuffer(jsonValue))
			if err != nil {
				http.Error(res, "Internal server error", http.StatusInternalServerError)
			}
			if jsonResp.StatusCode == 409 {
				http.Error(res, "Username already taken", http.StatusForbidden)
			}
		}
		// redirect to main index
		http.Redirect(res, req, "/updateProfile", http.StatusSeeOther)
		return
	}
	tpl.ExecuteTemplate(res, "signup.gohtml", myUser)
}

// Login page checks for user input with database
func Login(res http.ResponseWriter, req *http.Request) {
	if alreadyLoggedIn(req) {
		http.Redirect(res, req, "/", http.StatusSeeOther)
		return
	}
	// process form submission
	if req.Method == http.MethodPost {
		username := req.FormValue("username")
		password := req.FormValue("password")
		// check if User exist with username

		mapUsers := database.GetUser()

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

		if _, ok := mapHistory[myUser.Username]; !ok {
			mapHistory[username] = &queue.Queue{}
		}
		currentTime := time.Now()
		mapHistory[username].Enqueue(queue.History{Time: fmt.Sprintf(currentTime.Format("2006-01-02 3:04PM")), Activity: "Login"})

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
func getUserFromCookie(res http.ResponseWriter, req *http.Request) database.User {
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

	mapUsers := database.GetUser()
	var myUser database.User
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

	mapUsers := database.GetUser()

	username := mapSessions[myCookie.Value]
	_, ok := mapUsers[username]
	return ok
}
