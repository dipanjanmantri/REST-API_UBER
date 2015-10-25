package main

import (

	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
    "strconv"
    "io/ioutil"
	"github.com/julienschmidt/httprouter"
	"gopkg.in/mgo.v2"
	"gopkg.in/mgo.v2/bson"
	
)

func main() {
	
	r := httprouter.New()

	
	c := NewLocationController(getSession())

	
	r.GET("/locations/:location_id", c.getlocation)

	
	r.POST("/locations", c.createlocation)

	
	r.PUT("/locations/:location_id", c.updatelocation)

	
	r.DELETE("/locations/:location_id", c.removelocation)

	
	http.ListenAndServe("localhost:8080", r)
}

type location_controller struct {
		session *mgo.Session
	}


type InputAddress struct {
		Name   string        `json:"name"`
		Address string 		`json:"address"`
		City string			`json:"city"`
		State string		`json:"state"`
		Zip string			`json:"zip"`
	}



type OutputAddress struct {

		Id     bson.ObjectId `json:"_id" bson:"_id,omitempty"`
		Name   string        `json:"name"`
		Address string 		`json:"address"`
		City string			`json:"city" `
		State string		`json:"state"`
		Zip string			`json:"zip"`

		Coordinate struct{
			Lat string 		`json:"lat"`
			Lang string 	`json:"lang"`
		}
	}


type GoogleResponse struct {
	Results []GoogleResult
}

type GoogleResult struct {

	Address      string               `json:"formatted_address"`
	AddressParts []GoogleAddressPart `json:"address_components"`
	Geometry     Geometry
	Types        []string
}

type GoogleAddressPart struct {

	Name      string `json:"long_name"`
	ShortName string `json:"short_name"`
	Types     []string
}

type Geometry struct {

	Bounds   Bounds
	Location Point
	Type     string
	Viewport Bounds
}
type Bounds struct {
	NorthEast, SouthWest Point
}

type Point struct {
	Lat float64
	Lng float64
}

func NewLocationController(s *mgo.Session) *location_controller {
	return &location_controller{s}
}

func getGoogLocation(address string) OutputAddress{
	client := &http.Client{}

	reqURL := "http://maps.google.com/maps/api/geocode/json?address="
	reqURL += url.QueryEscape(address)
	reqURL += "&sensor=false";
	fmt.Println("URL formed: "+ reqURL)
	req, err := http.NewRequest("GET", reqURL , nil)
	resp, err := client.Do(req)
	if err != nil {
		fmt.Println("error in sending req to google: ", err);	
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		fmt.Println("error in reading response: ", err);	
	}

	var res GoogleResponse
	err = json.Unmarshal(body, &res)
	if err != nil {
		fmt.Println("error in unmashalling response: ", err);	
	}

	var ret OutputAddress
	ret.Coordinate.Lat = strconv.FormatFloat(res.Results[0].Geometry.Location.Lat,'f',7,64)
	ret.Coordinate.Lang = strconv.FormatFloat(res.Results[0].Geometry.Location.Lng,'f',7,64)

	return ret;
}



func (c location_controller) getlocation(w http.ResponseWriter, r *http.Request, p httprouter.Params) {
	
	id := p.ByName("location_id")
	
	if !bson.IsObjectIdHex(id) {
        w.WriteHeader(404)
        return
    }

    
    oid := bson.ObjectIdHex(id)
	var o OutputAddress
	if err := c.session.DB("dipsjsu").C("locations").FindId(oid).One(&o); err != nil {
        w.WriteHeader(404)
        return
    }
	uj, _ := json.Marshal(o)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(200)
	fmt.Fprintf(w, "%s", uj)
}




func (c location_controller) createlocation(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	var u InputAddress
	var oA OutputAddress

	json.NewDecoder(r.Body).Decode(&u)	

	googResCoor := getGoogLocation(u.Address + "+" + u.City + "+" + u.State + "+" + u.Zip);
    fmt.Println("resp is: ", googResCoor.Coordinate.Lat, googResCoor.Coordinate.Lang);
	
	oA.Id = bson.NewObjectId()
	oA.Name = u.Name
	oA.Address = u.Address
	oA.City= u.City
	oA.State= u.State
	oA.Zip = u.Zip
	oA.Coordinate.Lat = googResCoor.Coordinate.Lat
	oA.Coordinate.Lang = googResCoor.Coordinate.Lang

	
	c.session.DB("dipsjsu").C("locations").Insert(oA)

	
	uj, _ := json.Marshal(oA)
	
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(201)
	fmt.Fprintf(w, "%s", uj)
}


func (c location_controller) removelocation(w http.ResponseWriter, r *http.Request, p httprouter.Params) {
	
	id := p.ByName("location_id")
	
	if !bson.IsObjectIdHex(id) {
		w.WriteHeader(404)
		return
	}
	
	oid := bson.ObjectIdHex(id)

	
	if err := c.session.DB("dipsjsu").C("locations").RemoveId(oid); err != nil {
		w.WriteHeader(404)
		return
	}

	w.WriteHeader(200)
}


func (c location_controller) updatelocation(w http.ResponseWriter, r *http.Request, p httprouter.Params) {
	var i InputAddress
	var o OutputAddress

	id := p.ByName("location_id")
	
	if !bson.IsObjectIdHex(id) {
        w.WriteHeader(404)
        return
    }
    oid := bson.ObjectIdHex(id)
	
	if err := c.session.DB("dipsjsu").C("locations").FindId(oid).One(&o); err != nil {
        w.WriteHeader(404)
        return
    }	

	json.NewDecoder(r.Body).Decode(&i)	
    googResCoor := getGoogLocation(i.Address + "+" + i.City + "+" + i.State + "+" + i.Zip);
    fmt.Println("resp is: ", googResCoor.Coordinate.Lat, googResCoor.Coordinate.Lang);

	
	o.Address = i.Address
	o.City = i.City
	o.State = i.State
	o.Zip = i.Zip
	o.Coordinate.Lat = googResCoor.Coordinate.Lat
	o.Coordinate.Lang = googResCoor.Coordinate.Lang

	
	c1 := c.session.DB("dipsjsu").C("locations")
	
	id2 := bson.M{"_id": oid}
	err := c1.Update(id2, o)
	if err != nil {
		panic(err)
	}
	
	
	uj, _ := json.Marshal(o)

	
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(201)
	fmt.Fprintf(w, "%s", uj)
}


func getSession() *mgo.Session {
	
	s, err := mgo.Dial("mongodb://dipanjansjsu:root@ds039504.mongolab.com:39504/dipsjsu")

	
	if err != nil {
		panic(err)
	}
	
	s.SetMode(mgo.Monotonic, true)

	return s
}