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

import (
    "fmt"
    //"html/template"
    "net/http"
	"encoding/json"
	"appengine"
    "appengine/datastore"
	"time"
)

type Location struct {
	X string
	Y string
	Clientid string
	Date time.Time
}

func init() {
	http.HandleFunc("/getlocation", GetLocation)
    http.HandleFunc("/setlocation", SetLocation)
    http.HandleFunc("/AddClient", AddClient)
    http.HandleFunc("/UpdateClient", UpdateClient)
}

func SetLocation(w http.ResponseWriter, r *http.Request) {
	
	var rx map[string]interface{}
	byt := []byte(`{"num":6.13,"strs":["a","b"]}`)
	
	if err := json.Unmarshal(byt, &rx); err != nil {
		fmt.Fprint(w, "Oops - something went wrong with the JSON. \n")
	} else {
		fmt.Fprint(w, "This is setlocation.\n", "The received location is:\n\n", 
			"x: \t\t", r.FormValue("x"), "\n",
			"y: \t\t", r.FormValue("y"), "\n",
			"clientid: \t", r.FormValue("clientid"))
	}
	
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
        return
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
	users := make([]Location, 0, 10)
	if _, err := query.GetAll(c, &users); err != nil {
        http.Error(w, err.Error(), http.StatusInternalServerError)
        return
    }
	fmt.Fprint(w, "Users:\n", users) //users[0].y)
	
	/*if err := json.Marshal(byt, &rx); err != nil {
		fmt.Fprint(w, "Oops - something went wrong with the JSON. \n")
	} else {
		fmt.Fprint(w, "This is getlocation.  Your device's loc:", rx)
	}*/
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