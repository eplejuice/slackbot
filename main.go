package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"

	"github.com/nlopes/slack"
)

var (
	slackClient *slack.Client
)

func main() {
	slackClient = slack.New(os.Getenv("SLACK_ACCESS_TOKEN"))
	rtm := slackClient.NewRTM()
	go rtm.ManageConnection()

	for msg := range rtm.IncomingEvents {
		switch ev := msg.Data.(type) {
		case *slack.MessageEvent:
			go respond(ev)
		}
	}
}

func respond(ev *slack.MessageEvent) {
	text := ev.Msg.Text

	switch text {
	case "Hello":
		slackClient.PostMessage(ev.User, slack.MsgOptionText("Hello", true))
	case "Show me a dog":
		slackClient.PostMessage(ev.Channel, slack.MsgOptionText(getDog(), true))
	default:
		slackClient.PostMessage(ev.User, slack.MsgOptionText("Sorry, i don't know that command", false))

	}
}

func getDog() string {
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
