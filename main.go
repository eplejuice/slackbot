package main

import (
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
	case "Hei":

		slackClient.PostMessage(ev.User, slack.MsgOptionText("Hei", true))
	default:
		slackClient.PostMessage(ev.User, slack.MsgOptionText("Sorry, i don't know", false))

	}
}
