package app

import (
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"

	"../spark"
	"github.com/kataras/iris"
	"github.com/prometheus/client_golang/prometheus"
)

type DataStruct struct {
	MessageID   string `json:"id"`
	RoomType    string `json:"roomType"`
	PersonID    string `json:"personId"`
	PersonEmail string `json:"personEmail"`
	Created     string `json:"created"`
}

type Message struct {
	ID        string     `json:"id"`
	Name      string     `json:"name"`
	TargetURL string     `json:"targetUrl"`
	Resource  string     `json:"resource"`
	Event     string     `json:"event"`
	Filter    string     `json:"filter"`
	OrgID     string     `json:"orgId"`
	CreatedBy string     `json:"createdBy"`
	AppID     string     `json:"appId"`
	OwnedBy   string     `json:"ownedBy"`
	Status    string     `json:"status"`
	Created   string     `json:"created"`
	ActorID   string     `json:"actorId"`
	Data      DataStruct `json:"data"`
}

func (a Application) addRoutes() {
	a.Server.Post("/callback", sparkbotCallback)
	a.Server.Get("/help", sparkbotHelp)
	a.Server.Get("/fallback", sparkbotFallback)
	a.Server.Get("/hello", sparkbotHello)
	a.Server.Get("/about", sparkbotAbout)
	a.Server.Get("/metrics", iris.FromStd(prometheus.Handler()))
}

func sparkbotCallback(ctx iris.Context) {
	body, err := ioutil.ReadAll(ctx.Request().Body)
	if err != nil {
		panic(err)
	}
	var mess Message
	err = json.Unmarshal(body, &mess)
	/*

		{"id":"Y2lzY29zcGFyazovL3VzL1dFQkhPT0svZWFhZWE1MjYtNzMxZi00OWIxLWIyYzEtNTFkOWY5NGQ1ZDY2",
			"name":"MyTestHook",
			"targetUrl":"https://roporter1234.localtunnel.me/callback",
			"resource":"messages",
			"event":"created",
			"filter":"roomId=Y2lzY29zcGFyazovL3VzL1JPT00vOGMyYWFkMTAtYTE0Mi0xMWU3LThmYzEtMWY5YWY0Y2EwOTNm",
			"orgId":"Y2lzY29zcGFyazovL3VzL09SR0FOSVpBVElPTi8xZWI2NWZkZi05NjQzLTQxN2YtOTk3NC1hZDcyY2FlMGUxMGY",
			"createdBy":"Y2lzY29zcGFyazovL3VzL1BFT1BMRS82YjdjY2ZmNy04ZWRhLTRkMTYtOWExZS0yNzQ0MDYwZDU0ODQ",
			"appId":"Y2lzY29zcGFyazovL3VzL0FQUExJQ0FUSU9OL0MyNzljYjMwYzAyOTE4MGJiNGJkYWViYjA2MWI3OTY1Y2RhMzliNjAyOTdjODUwM2YyNjZhYmY2NmM5OTllYzFm",
			"ownedBy":"creator",
			"status":"active",
			"created":"2017-09-25T10:13:14.729Z",
			"actorId":"Y2lzY29zcGFyazovL3VzL1BFT1BMRS82YjdjY2ZmNy04ZWRhLTRkMTYtOWExZS0yNzQ0MDYwZDU0ODQ",
			"data":{"id":"Y2lzY29zcGFyazovL3VzL01FU1NBR0UvMDhjNmIzMTAtYTFlMi0xMWU3LWFlN2QtY2Y4YzY2Nzg4NWU2",
				"roomId":"Y2lzY29zcGFyazovL3VzL1JPT00vOGMyYWFkMTAtYTE0Mi0xMWU3LThmYzEtMWY5YWY0Y2EwOTNm",
				"roomType":"group",
				"personId":"Y2lzY29zcGFyazovL3VzL1BFT1BMRS82YjdjY2ZmNy04ZWRhLTRkMTYtOWExZS0yNzQ0MDYwZDU0ODQ",
				"personEmail":"roporter@cisco.com",
				"created":"2017-09-25T11:09:44.001Z"}}

	*/
	message := getMessageContent(mess.Data.MessageID)
	fmt.Println(message)
	if message == "about" {
		sparkbotAbout(ctx)
	}
}

func getMessageContent(data string) string {
	a := New()
	tr := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}
	client := &http.Client{Transport: tr}
	sparkClient := ciscospark.NewClient(client)
	token := a.conf.GetString("spark.token")
	sparkClient.Authorization = "Bearer " + token
	htmlMessageGet, _, err := sparkClient.Messages.GetMessage(data)
	if err != nil {
		log.Fatal(err)
	}
	a.Log.Info("GET <ID>:", htmlMessageGet.ID, htmlMessageGet.Text, htmlMessageGet.Created)
	return htmlMessageGet.Text
}

func sparkbotHelp(ctx iris.Context) {
	sendSparkMessage("Hi, I am the Hello World bot !\n\nType /hello to see me in action.")
}

func sparkbotFallback(ctx iris.Context) {
	sendSparkMessage("Sorry, I did not understand.\n\nTry /help.")
}

func sparkbotHello(ctx iris.Context) {
	sendSparkMessage("Hello <@personEmail:roporter@cisco.com>")
}

func sparkbotAbout(ctx iris.Context) {
	sendSparkMessage("```\n{\n   'author':'Robert Porter <roporter@cisco.com>',\n   'code':'https://github.com/robjporter/go-sparkbot',\n   'description':'A handy tool to interact with Cisco Spark.',\n}```")
}

func deleteWebHooks() {
	a := New()
	tr := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}
	client := &http.Client{Transport: tr}
	sparkClient := ciscospark.NewClient(client)
	token := a.conf.GetString("spark.token")
	sparkClient.Authorization = "Bearer " + token
	webhooksQueryParams := &ciscospark.WebhookQueryParams{
		Max: 10,
	}
	webhooks, _, err := sparkClient.Webhooks.Get(webhooksQueryParams)
	if err != nil {
		log.Fatal(err)
	}
	for _, webhook := range webhooks {
		resp, err := sparkClient.Webhooks.DeleteWebhook(webhook.ID)
		if err != nil {
			log.Fatal(err)
		}
		fmt.Println("DELETE:", resp.StatusCode)
	}
}

func registerWebHook() {
	a := New()
	myRoomID := a.conf.GetString("spark.roomid")
	a.Log.Info("WEBHOOK: Registering a new WebHook for Room ID: ", myRoomID)
	tr := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}
	client := &http.Client{Transport: tr}
	sparkClient := ciscospark.NewClient(client)
	token := a.conf.GetString("spark.token")
	sparkClient.Authorization = "Bearer " + token
	webHookURL := "https://roporter1234.localtunnel.me"
	webhookRequest := &ciscospark.WebhookRequest{
		Name:      a.conf.GetString("spark.hookname"),
		TargetURL: webHookURL,
		Resource:  "messages",
		Event:     "created",
		Filter:    "roomId=" + myRoomID,
	}
	testWebhook, _, err := sparkClient.Webhooks.Post(webhookRequest)
	if err != nil {
		a.Log.Error(err)
	}
	a.Log.Info("POST:", testWebhook.ID, testWebhook.Name, testWebhook.TargetURL, testWebhook.Created)

}

func getSparkMessages(count int) {
	a := New()
	tr := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}
	client := &http.Client{Transport: tr}
	sparkClient := ciscospark.NewClient(client)
	token := a.conf.GetString("spark.token")
	sparkClient.Authorization = "Bearer " + token
	myRoomID := a.conf.GetString("spark.roomid")
	messageQueryParams := &ciscospark.MessageQueryParams{
		Max:    count,
		RoomID: myRoomID,
	}
	messages, _, err := sparkClient.Messages.Get(messageQueryParams)
	if err != nil {
		a.Log.Error(err)
	}
	for id, message := range messages {
		a.Log.Info("GET:", id, message.ID, message.Text, message.Created)
	}
}

func sendSparkMessage(mess string) {
	//getSparkMessages(1)
	a := New()
	tr := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}
	client := &http.Client{Transport: tr}
	sparkClient := ciscospark.NewClient(client)
	token := a.conf.GetString("spark.token")
	sparkClient.Authorization = "Bearer " + token
	myRoomID := a.conf.GetString("spark.roomid")
	htmlMessage := &ciscospark.MessageRequest{
		MarkDown: mess,
		RoomID:   myRoomID,
	}
	newHTMLMessage, _, err := sparkClient.Messages.Post(htmlMessage)
	if err != nil {
		a.Log.Error(err)
	}
	a.Log.Info("POST:", newHTMLMessage.ID, newHTMLMessage.MarkDown, newHTMLMessage.Created)
}
