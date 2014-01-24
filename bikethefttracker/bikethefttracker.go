package bikethefttracker

/*
API:

getlocation
<server>/getlocation?clientid=########
	Method that the app uses to retrieve the most recent location data and theft status of the bike.

setlocation
<server>/setlocation?clientid=########&x=#.#####&y=#.#####&stolen=[1/0]
	Method that the bike uses to alert the server of a theft and update its most recent location.

*/

/*

ECE Senior Design 2014 for Oregon State University
Team MOSFET:
Russell Barnes
Paul Burris
Nick Voigt

*/

import (
    "fmt"
    "net/http"
	"encoding/json"
	"appengine"
    "appengine/datastore"
	"time"
	"gotwilio"
	"twilioaccount"	// Don't open-source the Twilio account credentials
)

type Location struct {
	X string
	Y string
	Clientid string
	Date time.Time
}

/*type User struct {
	Appid string
	Clientidkey *datastore.Key
}*/

func init() {
	http.HandleFunc("/getlocation", GetLocation)
    http.HandleFunc("/setlocation", SetLocation)
    http.HandleFunc("/addclient", AddClient)
    http.HandleFunc("/updateclient", UpdateClient)
}

func SetLocation(w http.ResponseWriter, r *http.Request) {
	
	// Add points to datastore for user
	newlocation := &Location{
		X: r.FormValue("x"), 
		Y: r.FormValue("y"),
		Clientid: r.FormValue("clientid"),
		Date: time.Now(),
	}
	
	c := appengine.NewContext(r)
	
	// format: datastore.NewIncompleteKey(context, "subkind", *parentKey)
	key := datastore.NewIncompleteKey(c, "User", ParentKey(c))
    if _, err := datastore.Put(c, key, newlocation); err != nil {
        http.Error(w, err.Error(), http.StatusInternalServerError)
    }
	
	// Stolen == 1? Send text and/or push notification - "open app for more details"
	if r.FormValue("stolen") == "1" {
		
		// Provide your own Twilio credentials and phone numbers (twilio.com)
		accountSid, authToken := twilioaccount.GetTwilioAccount()
		from, to := twilioaccount.GetTwilioNumbers()
	    twilio := gotwilio.NewTwilioClient(accountSid, authToken)

	    message := "Your bicycle was just stolen - open the Bike Theft Tracker app to follow"
	    twiresponse, twiexception, twierror := twilio.SendSMS(from, to, message, "", "")
		
		c.Infof("Twilio request finished.\nResponse: ", twiresponse, "\nException: ", twiexception, "\nError: ", twierror)
	}
}

func GetLocation(w http.ResponseWriter, r *http.Request) {
	
	c := appengine.NewContext(r)
	
	// Retrieve the data of the client requested by the app
	clientid := r.FormValue("clientid")
	if clientid == "" {
		// This will only occur in the development version
		clientid = "00000000"
	}
	
	// Fairly flat DB structure - can be optimized in the future by using clientids as parent entities
	query := datastore.NewQuery("User").Ancestor(ParentKey(c)).Filter("Clientid =", clientid).Order("-Date")
	users := make([]Location, 0, 10)	// Ten most recent locations returned
	if _, err := query.GetAll(c, &users); err != nil {
        http.Error(w, err.Error(), http.StatusInternalServerError)
        return
    }
	fmt.Fprint(w, "Users:\n", users) //users[0].y)
	
	// Respond to the HTML request with JSON-formatted location data
	if userbytes, err := json.Marshal(users); err != nil {
		fmt.Fprint(w, "Oops - something went wrong with the JSON. \n")
		//fmt.Fprint(w, "{error: 1}")
		return
	} else {
		fmt.Fprint(w, "\n\nJSON'd data: \n", string(userbytes))
		return
	}
}

func AddClient(w http.ResponseWriter, r *http.Request) {
	
}

func UpdateClient(w http.ResponseWriter, r *http.Request) {

}

// Get the parent key for the particular Location entity group
func ParentKey(c appengine.Context) *datastore.Key {
    // The string "development_locationentitygroup" refers to an instance of a LocationEntityGroupType
	// format: datastore.NewKey(context, "groupkind", "groupkind_instance", 0, nil)
    return datastore.NewKey(c, "LocationEntityGroupType", "development_locationentitygroup", 0, nil)
}