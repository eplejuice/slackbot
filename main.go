package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/nlopes/slack"
	mgo "gopkg.in/mgo.v2"
	"gopkg.in/mgo.v2/bson"
)

var slackClient *slack.Client
var db *mgo.Database

const (
	COLLECTION = "dogs"
)

var AS = AnimalShelter{
	Address:  os.Getenv("MONGO_ADDRESS"),
	Database: os.Getenv("MONGO_DATABASE"),
	Username: os.Getenv("MONGO_USER"),
	Password: os.Getenv("MONGO_PASSWORD"),
}

func main() {
	AS.Connect()
	slackClient = slack.New(os.Getenv("SLACK_ACCESS_TOKEN"))
	rtm := slackClient.NewRTM()
	go rtm.ManageConnection()

	for msg := range rtm.IncomingEvents {
		switch ev := msg.Data.(type) {
		case *slack.MessageEvent:
			go Respond(ev, rtm)
		}
	}
}

func (m *AnimalShelter) Connect() {
	session := &mgo.DialInfo{
		Addrs:    []string{m.Address},
		Timeout:  60 * time.Second,
		Database: m.Database,
		Username: m.Username,
		Password: m.Password,
	}

	connection, err := mgo.DialWithInfo(session)
	if err != nil {
		fmt.Println("Could not connect to DB")
		log.Fatal(err)
	}
	db = connection.DB(m.Database)
}

func (m *AnimalShelter) Insert(dog Dog) error {
	err := db.C(COLLECTION).Insert(&dog)
	return err
}

func (m *AnimalShelter) FindOldestDog() (Dog, error) {
	var dog Dog
	err := db.C(COLLECTION).Find(nil).Sort("_id").One(&dog)
	return dog, err
}

func (m *AnimalShelter) DeleteDogWithId(id string) (Dog, error) {
	var dog Dog
	err := db.C(COLLECTION).FindId(bson.ObjectIdHex(id)).One(&dog)
	err = db.C(COLLECTION).RemoveId(bson.ObjectIdHex(id))
	return dog, err
}

func (m *AnimalShelter) FindCount() (int, error) {
	trackCount, err := db.C(COLLECTION).Count()
	return trackCount, err
}

func (m *AnimalShelter) FindAll() ([]Dog, error) {
	fmt.Println("Trying to find all")
	var dogs []Dog
	// Using the nil parameter in find gets all tracks
	err := db.C(COLLECTION).Find(nil).All(&dogs)
	return dogs, err
}

func (m *AnimalShelter) DeleteAll() (*mgo.ChangeInfo, error) {
	rem, err := db.C(COLLECTION).RemoveAll(nil)
	return rem, err
}

func Respond(ev *slack.MessageEvent, rtm *slack.RTM) {
	text := ev.Msg.Text
	text = strings.ToLower(text)
	switch {
	case strings.Contains(text, "hey"):
		rtm.SendMessage(rtm.NewOutgoingMessage(HelloThere(ev, rtm), ev.Channel))
	case strings.Contains(text, "show me"):
		rtm.SendMessage(rtm.NewOutgoingMessage(ShowDog(), ev.Channel))
	case strings.Contains(text, "add"):
		rtm.SendMessage(rtm.NewOutgoingMessage(AddDog(), ev.Channel))
	case strings.Contains(text, "adopt"):
		rtm.SendMessage(rtm.NewOutgoingMessage(AdoptDog(), ev.Channel))
	case strings.Contains(text, "how many"):
		rtm.SendMessage(rtm.NewOutgoingMessage(getCount(), ev.Channel))
	case strings.Contains(text, "show all"):
		ShowAllDogs(ev, rtm)
	case strings.Contains(text, "help"):
		HelpFunc(ev, rtm)
		rtm.SendMessage(rtm.NewOutgoingMessage("Sorry, i don't know that command, try 'Help' for a list of commands", ev.Channel))
	}
}

func HelloThere(ev *slack.MessageEvent, etm *slack.RTM) string {
	returnString :=
		"https://media1.tenor.com/images/242ce12decbb1275829ec8e387990d17/tenor.gif?itemid=5312368"
	return returnString
}

func getCount() string {
	dogCount, err := AS.FindCount()
	if err != nil {
		panic(err)
	}
	returnString := strings.Join([]string{
		"There are ", strconv.Itoa(dogCount),
		" Dogs currently in the shelter"},
		"")
	return returnString
}

func ShowDog() string {
	type dogs struct {
		Status  string `json:"status"`
		Message string `json:"message"`
	}

	resp, err := http.Get("https://dog.ceo/api/breeds/image/random")
	if err != nil {
		fmt.Printf("Got no dog")
		panic(err)
	}

	//fmt.Println(resp.Body)
	defer resp.Body.Close()

	dog := &dogs{}
	err = json.NewDecoder(resp.Body).Decode(dog)
	if err != nil {
		fmt.Println("Error decoding json")
		panic(err)
	}
	return dog.Message
}

func AddDog() string {

	type dogs struct {
		Status  string `json:"status"`
		Message string `json:"message"`
	}

	resp, err := http.Get("https://dog.ceo/api/breeds/image/random")
	if err != nil {
		fmt.Printf("Got no dog")
		panic(err)
	}

	//fmt.Println(resp.Body)
	defer resp.Body.Close()

	dog := &dogs{}
	err = json.NewDecoder(resp.Body).Decode(dog)
	if err != nil {
		fmt.Println("Error decoding json")
		panic(err)
	}

	dogg := Dog{
		ID:      bson.NewObjectId(),
		Picture: dog.Message,
	}

	if err := AS.Insert(dogg); err != nil {
		fmt.Println("Error inserting new dog")
		panic(err)
	}

	returnString := strings.Join([]string{
		"New dog added ", dog.Message}, "")
	return returnString
}

func AdoptDog() string {
	dog, err := AS.FindOldestDog()
	if err != nil {
		fmt.Println("Could not find latest")
		return "Sorry, no dogs available for adoption"
	}
	dog, err = AS.DeleteDogWithId(dog.ID.Hex())
	if err != nil {
		fmt.Println("Coult not delete dog")
		panic(err)
	}

	returnString := strings.Join([]string{
		"Congratulations, you adopted a new dog ", dog.Picture}, "")
	return returnString
}

func ShowAllDogs(ev *slack.MessageEvent, rtm *slack.RTM) {
	dogs, err := AS.FindAll()
	if err != nil {
		rtm.SendMessage(rtm.NewOutgoingMessage("Error, please try again", ev.Channel))
	}

	for i := 0; i < len(dogs); i++ {
		rtm.SendMessage(rtm.NewOutgoingMessage(dogs[i].Picture, ev.Channel))
	}

}

func HelpFunc(ev *slack.MessageEvent, rtm *slack.RTM) {
	giveHelp := "Available commands: \n Hey - Say hello to the bot \n Show me - Shows a picture of a random dog \n Add dog - Adds a random dog to the shelter \n Adopt dog - Adopt a dog from the shelter \n How many - Gives a count of how many dogs are currently in the shelter \n Show all - Shows a picture of all dogs currently in the shelter \n "
	rtm.SendMessage(rtm.NewOutgoingMessage(giveHelp, ev.Channel))
}
