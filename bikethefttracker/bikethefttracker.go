package bikethefttracker

/*
API:

getlocation
<server>/getlocation?clientid=########
	Method that the app uses to retrieve the most recent location data and theft status of the bike.

setlocation
<server>/setlocation?clientid=########&x=#.#####&y=#.#####&stolen=[1/0]
	Method that the bike uses to alert the server of a theft and update its most recent location.

updateclient
<server>/updateclient?clientid=########[&email=[0/1]&address=abc@123.com][&sms=[0/1]&phonenumber=1503#######][&push=[0/1]]
	E.g. <server>/updateclient?clientid=1&email=1&address=abc@123.com&sms=0&push=1
	Save notification settings for the client.  Choose from email, sms and/or push methods.

Message format for texting Twilio phone number:
{"clientid":"########","x":"####","xm":"##.####","y":"###","ym":"##.####","vel":"#.###","deg":"##.##","stolen":"#"}
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
	"strconv"
	"appengine/mail"
	"gotwilio"
	"twilioaccount"	// Don't open-source the Twilio account credentials
)

type Location struct {
	X string
	Y string
	Clientid string
	Date time.Time
}

type AlertMethod struct {
	Email bool
	Sms bool
	Push bool
	
	Address string
	Phonenumber string
	
	Clientid string
	Date time.Time
}

type Pushdevice struct {
	Pushtoken string
	Clientid string
	Date time.Time
}

func init() {
	http.HandleFunc("/getlocation", GetLocation)
    http.HandleFunc("/setlocation", SetLocation)
    http.HandleFunc("/addclient", AddClient)
    http.HandleFunc("/updateclient", UpdateClient)
	http.HandleFunc("/twiliorequest", TwilioRequest)
	http.HandleFunc("/setpushtoken", SetPushToken)
}


/* 
	"PUBLIC" FUNCTIONS 
*/


func SetLocation(w http.ResponseWriter, r *http.Request) {
	
	c := appengine.NewContext(r)
	
	clientid := r.FormValue("clientid")
	if clientid == "" {
		// This will only occur in the development version
		clientid = "00000000"
	}
	
	// Add points to datastore for user
	newlocation := &Location{
		X: r.FormValue("x"), 
		Y: r.FormValue("y"),
		Clientid: clientid,
		Date: time.Now(),
	}
	
	
	
	// format: datastore.NewIncompleteKey(context, "subkind", *parentKey)
	key := datastore.NewIncompleteKey(c, "Location", ParentKey(c))
    if _, err := datastore.Put(c, key, newlocation); err != nil {
        http.Error(w, err.Error(), http.StatusInternalServerError)
    }
	
	if r.FormValue("stolen") != "1" {
		return
	}
	// Module is stolen
	
	
	// Get the client's theft notification preferences
	clientprefs := _GetClientAlertPrefs(c, w, r, clientid); if clientprefs.Address == "error" {
		c.Errorf("Client preferences could not be found for clientid: ", clientid)
		return
	}
	
	// Send text, email and/or push notification - "open app to follow"
	if clientprefs.Email {
		_SendEmail(c, w, r, clientprefs.Address);	
	}
	if clientprefs.Sms {
		_SendSMS(c, w, r, clientprefs.Phonenumber);	
	}
	if clientprefs.Push {
		_SendPush(c, w, r)
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
	query := datastore.NewQuery("Location").Ancestor(ParentKey(c)).Filter("Clientid =", clientid).Order("-Date").Limit(500)
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
	
	c := appengine.NewContext(r)
	
	// Create notification preferences for client
	newAlertMethod := &AlertMethod{
		Email: r.FormValue("email") == "1",
		Sms: r.FormValue("sms") == "1",
		Push: r.FormValue("email") == "1",
		
		Clientid: r.FormValue("clientid"),
		Date: time.Now(),
	}
	if (newAlertMethod.Email) {
		newAlertMethod.Address = r.FormValue("address")
	}
	if (newAlertMethod.Sms) {
		newAlertMethod.Phonenumber = r.FormValue("phonenumber")
	}
	
	if newAlertMethod.Clientid == "" {
		// This will only occur in the development version
		newAlertMethod.Clientid = "00000000"
	}
	
	// Delete all previous preferences for this client
	query := datastore.NewQuery("AlertMethod").Ancestor(ParentKey(c)).Filter("Clientid =", newAlertMethod.Clientid)
	for t := query.Run(c); ; {
		var x AlertMethod
		key, err := t.Next(&x)
		if err == datastore.Done {
			c.Infof("Preferences deleted for client ", newAlertMethod.Clientid)
			break
		}
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		err = datastore.Delete(c, key)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	}
	
	// Add newAlertMethod notification preferences to database
	
	// format: datastore.NewIncompleteKey(context, "subkind", *parentKey)
	key := datastore.NewIncompleteKey(c, "AlertMethod", ParentKey(c))
    if _, err := datastore.Put(c, key, newAlertMethod); err != nil {
        http.Error(w, err.Error(), http.StatusInternalServerError)
    }
}

/*

Bike Theft Tracker GSM request format:
{"clientid":#,"x":"#.#####","y":"#.#####","stolen":"[0/1]"}
or
{%22clientid%22:%22#%22,%22x%22:%22#.#####%22,%22y%22:%22#.#####%22,%22stolen%22:%22[0/1]%22}

*/

func TwilioRequest(w http.ResponseWriter, r *http.Request) {
	
	/* 
		Message format for texting Twilio phone number:
		{"clientid":"########","x":"####","xm":"##.####","y":"###","ym":"##.####","vel":"#.###","deg":"##.##","stolen":"#"}
	*/
	
	c := appengine.NewContext(r)
	
	// Retrieve the body of the message
	
	message := r.FormValue("Body")
	if message != "" {
		c.Infof("Twilio SMS: ", message)
	} else {
		c.Errorf("message is empty")
		return
	}
	
	messagedata := []byte(message)
	
	var jsonmap map[string]interface{}
	if err := json.Unmarshal(messagedata, &jsonmap); err != nil {
		c.Errorf("Error parsing JSON from GSM: ", messagedata)
		return
	}
	
	
	// Convert degree coordinates to decimal coordinates
	x_str := _ConvertCoordinate(jsonmap["x"].(string), jsonmap["xm"].(string))
	y_str := _ConvertCoordinate(jsonmap["y"].(string), jsonmap["ym"].(string))
	if x_str == "error" || y_str == "error" {
		c.Errorf("String(s) not converted to or from float: ", 
			jsonmap["x"].(string), " ", jsonmap["xm"].(string), " ", 
			jsonmap["y"].(string), " ", jsonmap["ym"].(string))
		return
	}
	
	/*x := jsonmap["x"].(float64)
	y := jsonmap["y"].(float64)
	clientid := jsonmap["clientid"].(float64)
	
	c.Infof("x: ", x, "y: ", y, "client ID: ", clientid)
	*/
	
	// Save the new set of coordinates in the database
	
	clientid := jsonmap["clientid"].(string)
	if clientid == "" {
		clientid = "00000000"	// This will only occur in the development version
	}
	
	newlocation := &Location{
		X: x_str, 
		Y: y_str,
		Clientid: clientid,
		Date: time.Now(),
	}
	
	// format: datastore.NewIncompleteKey(context, "subkind", *parentKey)
	key := datastore.NewIncompleteKey(c, "Location", ParentKey(c))
    if _, err := datastore.Put(c, key, newlocation); err != nil {
        http.Error(w, err.Error(), http.StatusInternalServerError)
    }
	
	// Get the client's theft notification preferences
	clientprefs := _GetClientAlertPrefs(c, w, r, clientid)
	if clientprefs.Address == "error" {
		c.Errorf("Client preferences could not be found for clientid: ", clientid)
		return
	}
	
	// Send text, email and/or push notification - "open app to follow"
	if clientprefs.Email {
		_SendEmail(c, w, r, clientprefs.Address);	
	}
	if clientprefs.Sms {
		_SendSMS(c, w, r, clientprefs.Phonenumber);	
	}
	if clientprefs.Push {
		_SendPush(c, w, r)
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
	
	/* SMS message format:
	{"clientid":"########",
	"x":"####",
	"xm":"##.####",
	"y":"###",
	"ym":"##.####",
	"vel":"#.###",
	"deg":"##.##",
	"stolen":"#"}
	*/
}


func SetPushToken(w http.ResponseWriter, r *http.Request) {
	
	c := appengine.NewContext(r)
	
	clientid := r.FormValue("clientid")
	if clientid == "" {
		clientid = "00000000"
	}
	
	// Create notification preferences for client
	newPushdevice := &Pushdevice{
		Pushtoken: r.FormValue("pushtoken"),
		
		Clientid: clientid,
		Date: time.Now(),
	}
	
	if newPushdevice.Pushtoken == "" {
		c.Errorf("Empty push token for clientid: ", clientid)
	}
	
	// Delete all previous pushdevices for this client
	query := datastore.NewQuery("Pushdevice").Ancestor(ParentKey(c)).Filter("Clientid =", newPushdevice.Clientid)
	for t := query.Run(c); ; {
		var x Pushdevice
		key, err := t.Next(&x)
		if err == datastore.Done {
			c.Infof("Preferences deleted for client ", newPushdevice.Clientid)
			break
		}
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		err = datastore.Delete(c, key)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	}
	
	// Add newAlertMethod notification preferences to database
	
	// format: datastore.NewIncompleteKey(context, "subkind", *parentKey)
	key := datastore.NewIncompleteKey(c, "Pushdevice", ParentKey(c))
    if _, err := datastore.Put(c, key, newPushdevice); err != nil {
        http.Error(w, err.Error(), http.StatusInternalServerError)
    }
}


/* 
	HELPER FUNCTIONS 
*/

func _GetPushdevice(c appengine.Context, w http.ResponseWriter, r *http.Request, clientid string) (Pushdevice) {
	
	// Retrieve the data of the client requested by the app
	
	query := datastore.NewQuery("Pushdevice").Ancestor(ParentKey(c)).Filter("Clientid =", clientid).Order("-Date").Limit(1)
	pushdevices := make([]Pushdevice, 0, 1)	// Most recent alert preferences struct is returned
	if _, err := query.GetAll(c, &pushdevices); err != nil {
        http.Error(w, err.Error(), http.StatusInternalServerError)
		pushdevices[0].Pushtoken = "error"
        return pushdevices[0]
    }
	
	return pushdevices[0]
}

func _ConvertCoordinate(deg string, min string) (string) {
	// Convert degree coordinates to decimal coordinates
	var coord     float64
	var coord_min float64
	var err1 error
	var err2 error
	
	coord, err1     = strconv.ParseFloat(deg, 64)
	coord_min, err2 = strconv.ParseFloat(min, 64)
	if err1 != nil || err2 != nil {
		return "error"
	}
	
	if coord >= 0 {
		coord += (coord_min / 60)
	} else {
		coord -= (coord_min / 60)
	}
	
	coord_str := strconv.FormatFloat(coord, 'f', 5, 64)
	return coord_str
}

func _GetClientAlertPrefs(c appengine.Context, w http.ResponseWriter, r *http.Request, clientid string) (AlertMethod) {
	
	// Retrieve the data of the client requested by the app
	
	query := datastore.NewQuery("AlertMethod").Ancestor(ParentKey(c)).Filter("Clientid =", clientid).Order("-Date").Limit(1)
	alertmethods := make([]AlertMethod, 0, 1)	// Most recent alert preferences struct is returned
	if _, err := query.GetAll(c, &alertmethods); err != nil {
        http.Error(w, err.Error(), http.StatusInternalServerError)
		alertmethods[0].Address = "error"
        return alertmethods[0]
    }
	
	return alertmethods[0]
}

func _SendEmail(c appengine.Context, w http.ResponseWriter, r *http.Request, address string) {
	
	if address == "" {
		c.Errorf("Client email address not set")
		return
	}
	
    msg := &mail.Message{
            Sender:  "Bike Theft Tracker <alert@bikethefttracker.appspotmail.com>",
            To:      []string{address},
            Subject: "Your Bike Has Been Stolen",
            Body:    fmt.Sprintf(emailMessage),
    }
    if err := mail.Send(c, msg); err != nil {
            c.Errorf("Couldn't send email: %v", err)
    }
}

// Email message to send client
const emailMessage = `
So sorry: your bike has been stolen.

`

func _SendSMS(c appengine.Context, w http.ResponseWriter, r *http.Request, phonenumber string) {
	
	// Provide your own Twilio credentials and phone numbers (twilio.com)
	accountSid, authToken := twilioaccount.GetTwilioAccount()
	from, to := twilioaccount.GetTwilioNumbers()
    twilio := gotwilio.NewTwilioClient(accountSid, authToken)
	
	if (phonenumber != "") {
		to = phonenumber
	}
	
    message := "Your bicycle was just stolen - open the Bike Theft Tracker app to follow"
    twiresponse, twiexception, twierror := twilio.SendSMS(from, to, message, "", "", c)
	
	c.Infof("Twilio request finished.\nResponse: ", twiresponse, "\nException: ", twiexception, "\nError: ", twierror)
}

func _SendPush(c appengine.Context, clientid string) {
	
}

// Get the parent key for the particular Location entity group
func ParentKey(c appengine.Context) *datastore.Key {
    // The string "development_locationentitygroup" refers to an instance of a LocationEntityGroupType
	// format: datastore.NewKey(context, "groupkind", "groupkind_instance", 0, nil)
    return datastore.NewKey(c, "LocationEntityGroupType", "development_locationentitygroup", 0, nil)
}