package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/mattermost/mattermost-server/model"
)

const (
	// HOST is the domain (and port) for the Mattermost Server
	HOST = "york.codesigned.co.uk"

	BOT_USERNAME = "york-22-bot"
	BOT_PASSWORD = "pwfy23"

	TEAM_NAME = "uni-of-york"

	// CHANNEL_NAME should be your username
	CHANNEL_NAME = "york-22"
)

var client *model.Client4
var webSocketClient *model.WebSocketClient
var channel *model.Channel
var bot *model.User

func main() {
	client = model.NewAPIv4Client("https://" + HOST)

	// Login as the bot user
	var resp *model.Response
	bot, resp = client.Login(BOT_USERNAME, BOT_PASSWORD)

	// Check if there was a login error
	if resp.Error != nil {
		fmt.Println("Login error:", resp.Error)
		os.Exit(1)
	}

	// Team
	team, resp := client.GetTeamByName(TEAM_NAME, "")
	if resp.Error != nil {
		fmt.Println("Error finding team:", resp.Error)
		os.Exit(1)
	}

	// Find the channel ID
	channel, resp = client.GetChannelByName(CHANNEL_NAME, team.Id, "")
	if resp.Error != nil {
		fmt.Println("Error finding channel:", resp.Error)
		os.Exit(1)
	}

	// Connect to Mattermost websocket
	var err *model.AppError
	webSocketClient, err = model.NewWebSocketClient("wss://"+HOST, client.AuthToken)

	// If there's an error, just exit
	if err != nil {
		fmt.Println("Web Socket Error:", err)
		os.Exit(1)
	}

	// Start the client listening
	webSocketClient.Listen()
	fmt.Println("Listening for messages on " + CHANNEL_NAME)

	// Forever loop waiting for messages on the EventChannel
	for {
		select {
		case resp := <-webSocketClient.EventChannel:
			HandleWebSocketResponse(resp)
		}
	}
}

// HandleWebSocketResponse receives all events from the web socket
func HandleWebSocketResponse(event *model.WebSocketEvent) {
	// Filter out all other channels
	if event.Broadcast.ChannelId != channel.Id {
		return
	}

	// Only respond to posted messages
	// More event types here:
	// https://github.com/mattermost/mattermost-server/blob/master/model/websocket_message.go#L12
	if event.Event != model.WEBSOCKET_EVENT_POSTED {
		return
	}

	post := model.PostFromJson(strings.NewReader(event.Data["post"].(string)))

	// If no issues, then continue
	if post != nil {
		// Ensure this isn't a bot message
		if post.UserId == bot.Id {
			return
		}

		fmt.Println("Received message, responding...")

		// Get the text message from the post
		msg := post.Message

		// Convert the message to slice of runes
		n := 0
		runes := make([]rune, len(msg))
		for _, r := range msg {
			runes[n] = r
			n++
		}
		runes = runes[0:n]

		// Reverse the runes
		for i := 0; i < n/2; i++ {
			runes[i], runes[n-1-i] = runes[n-1-i], runes[i]
		}

		// Convert back to a string
		output := string(runes)

		// Send the message to the channel as a reply to this post
		SendMessage(output, post.Id)
	}
}

// SendMessage creates a new post on the channel as a reply
func SendMessage(msg string, replyToId string) {
	// Create a post
	post := &model.Post{}
	post.ChannelId = channel.Id
	post.Message = msg

	// Setting root id makes this a reply
	post.RootId = replyToId

	if _, resp := client.CreatePost(post); resp.Error != nil {
		fmt.Println("Post error:", resp.Error)
	}
}
