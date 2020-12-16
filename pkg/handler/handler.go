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

	"github.com/microcosm-cc/bluemonday"
	"googlemaps.github.io/maps"
)

var (
	tpl         *template.Template
	mapSessions = map[string]Session{}
	baseURL     string
	jobType     []string
	jobCategory []string
	mapHistory  = map[string]*queue.Queue{}
	mapMutex    sync.RWMutex
	wg          sync.WaitGroup
	bm          = bluemonday.UGCPolicy()
)

// Session struct
type Session struct {
	Username  string
	Accesskey string
}

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
	http.DefaultTransport.(*http.Transport).TLSClientConfig = &tls.Config{InsecureSkipVerify: true}
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
		log.Println(err)
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
		exp, _ := strconv.Atoi(bm.Sanitize(req.FormValue("exp")))
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
			activity += bm.Sanitize(req.FormValue("exp")) + "Years Of Exp "
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
						log.Println(err)
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
		keyword := bm.Sanitize(req.FormValue("keyword"))
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
			activity += keyword + " "
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
		MyUser      string
		AllUser     map[string]database.UserJSON
		Type        []string
		Category    []string
		GoogleAPI   string
		GoogleMapID string
	}{
		myUser.Username,
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
			postal := bm.Sanitize(req.FormValue("postal"))
			jobType := req.Form["Type"]
			category := req.Form["Category"]
			exp, _ := strconv.Atoi(bm.Sanitize(req.FormValue("exp")))
			lastDay := bm.Sanitize(req.FormValue("lastDay"))
			message := bm.Sanitize(req.FormValue("message"))
			email := bm.Sanitize(req.FormValue("email"))

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
				log.Println(err)
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

		request, err := http.NewRequest(http.MethodPatch, baseURL+"/"+myUser.Username+"?accessKey="+myUser.Accesskey, bytes.NewBuffer(jsonValue))
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
	if len(resp) == 0 {
		return 0, 0, fmt.Errorf("invalid postal code")
	}
	return resp[0].Geometry.Location.Lat, resp[0].Geometry.Location.Lng, nil
}

// Accessing the REST API and return back the JSON as string
func getUsers(code, key string) string {
	url := baseURL

	if code != "" {
		url = baseURL + "/" + code + "?accessKey=" + key
	} else {
		url = baseURL + "?accessKey=" + key
	}

	response, err := http.Get(url)
	if err != nil {
		log.Println("Error:", err)
		return ""
	}
	data, _ := ioutil.ReadAll(response.Body)
	response.Body.Close()
	return string(data)
}
