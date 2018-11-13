package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/nlopes/slack"
	mgo "gopkg.in/mgo.v2"
	"gopkg.in/mgo.v2/bson"
)

var (
	slackClient *slack.Client
	db          *mgo.Database
)

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
	http.HandleFunc("/")
	port := "8080"
	if err := http.ListenAndServe(":"+port, nil); err != nil {
		log.Fatal(err)
	}

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
		log.Fatal(err)
	}
	db = connection.DB(m.Database)
}

func (m *AnimalShelter) Insert(dog Dog) error {
	err := db.C(COLLECTION).Insert(&dog)
	return err
}

func (m *AnimalShelter) Delete(id string) error {
	err := db.C(COLLECTION).Remove(bson.M{"_id": id})
	return err
}

func (m *AnimalShelter) FindCount() (int, error) {
	trackCount, err := db.C(COLLECTION).Count()
	return trackCount, err
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
		rtm.SendMessage(rtm.NewOutgoingMessage("Hey", ev.Channel))
		break
	case strings.Contains(text, "dog"):
		rtm.SendMessage(rtm.NewOutgoingMessage(GetDog(), ev.Channel))
		break
	case strings.Contains(text, "add"):
		rtm.SendMessage(rtmNewOutgoingMessage(AdoptDog(), ev.Channel))
	default:
		rtm.SendMessage(rtm.NewOutgoingMessage("Sorry, i don't know that command", ev.Channel))
	}
}

func GetDog() string {
	type dogs struct {
		Status  string `json:"status"`
		Message string `json:"message"`
	}

	resp, err := http.Get("https://dog.ceo/api/breeds/image/random")
	if err != nil {
		fmt.Printf("Got no joke")
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
