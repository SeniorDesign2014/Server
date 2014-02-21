package bikethefttracker

/*
API:

getlocation
<server>/getlocation?clientid=########
	Method that the app uses to retrieve the most recent location data and theft status of the bike.

setlocation
<server>/setlocation?clientid=########&x=#.#####&y=#.#####&stolen=[1/0]
	Method that the bike uses to alert the server of a theft and update its most recent location.

Message format for texting Twilio phone number:
{"clientid":"#","x":"#.#####","y":"#.#####","stolen":"[0/1]"}
For complete URL request format, see function twiliorequest

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
	http.HandleFunc("/twiliorequest", TwilioRequest)
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
	    twiresponse, twiexception, twierror := twilio.SendSMS(from, to, message, "", "", c)
		
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
	query := datastore.NewQuery("User").Ancestor(ParentKey(c)).Filter("Clientid =", clientid).Order("-Date").Limit(10)
	users := make([]Location, 0, 10)	// Ten most recent locations returned
	if _, err := query.GetAll(c, &users); err != nil {
        http.Error(w, err.Error(), http.StatusInternalServerError)
        return
    }
	//fmt.Fprint(w, "Users:\n", users) //users[0].y)
	
	// Respond to the HTML request with JSON-formatted location data
	if userbytes, err := json.Marshal(users); err != nil {
		fmt.Fprint(w, "Oops - something went wrong with the JSON. \n")
		//fmt.Fprint(w, "{error: 1}")
		return
	} else {
		fmt.Fprint(w, string(userbytes))	// Print locations in date-descending order as a JSON array
		return
	}
}

func AddClient(w http.ResponseWriter, r *http.Request) {
	
}

func UpdateClient(w http.ResponseWriter, r *http.Request) {

}

/*

Bike Theft Tracker GSM request format:
{"clientid":#,"x":"#.#####","y":"#.#####","stolen":"[0/1]"}
or
{%22clientid%22:%22#%22,%22x%22:%22#.#####%22,%22y%22:%22#.#####%22,%22stolen%22:%22[0/1]%22}

*/

func TwilioRequest(w http.ResponseWriter, r *http.Request) {
	
	c := appengine.NewContext(r)
	
	// Retrieve the body of the message
	
	message := r.FormValue("Body")
	
	if message != "" {
		c.Infof("Twilio SMS: ", message)
	} else {
		c.Infof("message is empty")
		
	}
	
	messagedata := []byte(message)
	
	var jsonmap map[string]interface{}
	
	if err := json.Unmarshal(messagedata, &jsonmap); err != nil {
		c.Infof("Error parsing JSON from GSM: ", messagedata)
	}
	
	/*x := jsonmap["x"].(float64)
	y := jsonmap["y"].(float64)
	clientid := jsonmap["clientid"].(float64)
	
	c.Infof("x: ", x, "y: ", y, "client ID: ", clientid)
	*/
	
	
	newlocation := &Location{
		X: jsonmap["x"].(string), 
		Y: jsonmap["y"].(string),
		Clientid: jsonmap["clientid"].(string),
		Date: time.Now(),
	}
	
	// format: datastore.NewIncompleteKey(context, "subkind", *parentKey)
	key := datastore.NewIncompleteKey(c, "User", ParentKey(c))
    if _, err := datastore.Put(c, key, newlocation); err != nil {
        http.Error(w, err.Error(), http.StatusInternalServerError)
    }
	
	// Stolen == 1? Send text and/or push notification - "open app for more details"
	if jsonmap["stolen"].(string) == "1" {
		
		// Provide your own Twilio credentials and phone numbers (twilio.com)
		accountSid, authToken := twilioaccount.GetTwilioAccount()
		from, to := twilioaccount.GetTwilioNumbers()
	    twilio := gotwilio.NewTwilioClient(accountSid, authToken)

	    message := "Your bicycle was just stolen - open the Bike Theft Tracker app to follow"
	    twiresponse, twiexception, twierror := twilio.SendSMS(from, to, message, "", "", c)
		
		c.Infof("Twilio request finished.\nResponse: ", twiresponse, "\nException: ", twiexception, "\nError: ", twierror)
	}
	
	
	/* Sample URL request:
	<server>/twiliorequest?
	AccountSid=###...
	&MessageSid=###...
	&Body=BODY_OF_MESSAGE
	&ToZip=97###
	&ToCity=PORTLAND
	&FromState=OR
	&ToState=OR
	&SmsSid=###...
	&To=%2B1##########
	&ToCountry=US
	&FromCountry=US
	&SmsMessageSid=###...
	&ApiVersion=2010-04-01
	&FromCity=PORTLAND
	&SmsStatus=received
	&NumMedia=0
	&From=%2B1##########
	&FromZip=97###
	*/
}

// Get the parent key for the particular Location entity group
func ParentKey(c appengine.Context) *datastore.Key {
    // The string "development_locationentitygroup" refers to an instance of a LocationEntityGroupType
	// format: datastore.NewKey(context, "groupkind", "groupkind_instance", 0, nil)
    return datastore.NewKey(c, "LocationEntityGroupType", "development_locationentitygroup", 0, nil)
}