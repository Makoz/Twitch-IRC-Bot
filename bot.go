package main

import (
	"bufio"
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"net"
	"net/http"
	"net/textproto"
	"os"
	"strings"
	"sync"
	"time"
)

type Bot struct {
	server         string
	port           string
	nick           string
	channel        string
	autoMSG1       string
	autoMSG1Count  int
	autoMSG2       string
	autoMSG2Count  int
	conn           net.Conn
	quotes         map[string]string
	mods           map[string]bool
	userLastMsg    map[string]int64
	lastmsg        int64
	maxMsgTime     int64
	userMaxLastMsg int
	lastfm         string
	timestamps     []time.Time
	limit          int
}

func NewBot() *Bot {
	return &Bot{
		server:         "irc.chat.twitch.tv",
		port:           "6667",
		nick:           "quanticbot", //Change to your Twitch username
		channel:        "#vaultpls",  //Change to your channel
		autoMSG1:       "Please follow if you like the stream!  Type !help to see my commands",
		autoMSG1Count:  10,
		autoMSG2:       "Fook yeah.  Follow dis gouy.",
		autoMSG2Count:  50,
		conn:           nil, //Don't change this
		quotes:         make(map[string]string),
		mods:           make(map[string]bool),
		lastmsg:        0,
		maxMsgTime:     5,
		userLastMsg:    make(map[string]int64),
		userMaxLastMsg: 2,
		lastfm:         "NExTliFE_",
		limit:          20, // TODO: add option if mod set to 100
		timestamps:     make([]time.Time, 0),
	}
}

func (bot *Bot) Connect() {
	var err error
	fmt.Printf("Attempting to connect to server...\n")
	bot.conn, err = net.Dial("tcp", bot.server+":"+bot.port)
	if err != nil {
		fmt.Printf("Unable to connect to Twitch IRC server! Reconnecting in 10 seconds...\n")
		time.Sleep(10 * time.Second)
		bot.Connect()
	}
	fmt.Printf("Connected to IRC server %s\n", bot.server)
}

// Need to add a lock here!
func (bot *Bot) Message(message string) {
	if message == "" {
		return
	}
	mtx.Lock()
	if bot.limit <= 5 {
		// clean up timestamps
		idx := 0
		for k, v := range bot.timestamps {
			expire := v.Add(time.Second * 30)
			if expire.Before(time.Now()) {
				idx++
				fmt.Println("INDEX ", k, " HAS EXPIRED WITH VAL: ", v)
				bot.limit++
			}
		}
		fmt.Println(bot.timestamps)
		bot.timestamps = bot.timestamps[idx:]
	}
	if bot.limit > 1 {
		fmt.Printf("Bot: " + message + "\n")
		fmt.Fprintf(bot.conn, "PRIVMSG "+bot.channel+" :"+message+"\r\n")
		bot.timestamps = append(bot.timestamps, time.Now())
		bot.limit--
	} else {
		fmt.Println("Attempted to spam message")

	}
	mtx.Unlock()
}

func (bot *Bot) AutoMessage() {
	for {
		time.Sleep(time.Duration(bot.autoMSG1Count) * time.Minute)
		bot.Message(bot.autoMSG1)
	}
}

func (bot *Bot) ConsoleInput() {
	reader := bufio.NewReader(os.Stdin)
	for {
		text, _ := reader.ReadString('\n')
		if text == "/quit" {
			bot.conn.Close()
			os.Exit(0)
		}
		if text != "" {
			bot.Message(text)
		}
	}
}

var mtx sync.Mutex

func main() {
	channel := flag.String("channel", "jayg_52", "Sets the channel for the bot to go into.")
	nick := flag.String("nickname", "makozdoto", "The username of the bot.")
	autoMSG1 := flag.String("timedmsg", "Welcome!  If you enjoy my stream, please follow!", "Set the automatic timed message.")
	autoMSG1Count := flag.Int("timedmsgcount", 10, "Set how often the timed message gets displayed.")
	autoMSG2 := flag.String("linemsg", "Follow me if you really enjoy the stream!  Thank you all!", "Set the automatic line message")
	autoMSG2Count := flag.Int("linemsgcount", 50, "Set the amount of lines until the line message gets displayed!")
	userMaxLastMsg := flag.Int("spamtime", 1, "Set a minimum time until the user can talk again(Gets timed out if talks before that).")
	lastfm := flag.String("lastfm", "NExTliFE_", "Set your Last.FM username to track your songs.")
	flag.Parse()
	fmt.Printf("Twitch IRC Bot made in Go! https://github.com/Vaultpls/Twitch-IRC-Bot\n")

	ircbot := NewBot()
	go ircbot.ConsoleInput()
	ircbot.Connect()
	messagesCount := 0

	pass1, err := ioutil.ReadFile("twitch_pass.txt")
	pass := strings.Replace(string(pass1), "\n", "", 0)
	fmt.Println(pass)
	if err != nil {
		fmt.Println("Error reading from twitch_pass.txt.  Maybe it isn't created?")
		os.Exit(1)
	}

	//Prep everything
	if !ircbot.readSettingsDB(*channel) {
		ircbot.nick = *nick
		ircbot.channel = "#" + *channel
		ircbot.autoMSG1 = *autoMSG1
		ircbot.autoMSG1Count = *autoMSG1Count
		ircbot.autoMSG2 = *autoMSG2
		ircbot.autoMSG2Count = *autoMSG2Count
		ircbot.userMaxLastMsg = *userMaxLastMsg
		ircbot.lastfm = *lastfm
		ircbot.writeSettingsDB()
	}
	//
	fmt.Fprintf(ircbot.conn, "USER %s 8 * :%s\r\n", ircbot.nick, ircbot.nick)
	fmt.Fprintf(ircbot.conn, "PASS %s\r\n", pass)
	fmt.Fprintf(ircbot.conn, "NICK %s\r\n", ircbot.nick)
	fmt.Fprintf(ircbot.conn, "JOIN %s\r\n", ircbot.channel)
	ircbot.readQuoteDB()
	fmt.Printf("Inserted information to server...\n")
	fmt.Printf("If you don't see the stream chat it probably means the Twitch oAuth password is wrong\n")
	fmt.Printf("Channel: " + ircbot.channel + "\n")

	defer ircbot.conn.Close()
	go ircbot.AutoMessage()
	reader := bufio.NewReader(ircbot.conn)
	tp := textproto.NewReader(reader)
	go ircbot.ConsoleInput()

	for {
		line, err := tp.ReadLine()
		if err != nil {
			fmt.Println(err)
			break // break loop on errors
		}
		fmt.Println(line)
		// fmt.Println(line)
		if strings.Contains(line, "PING") {
			pongdata := strings.Split(line, "PING ")
			fmt.Fprintf(ircbot.conn, "PONG %s\r\n", pongdata[1])
		} else if strings.Contains(line, ".tmi.twitch.tv PRIVMSG "+ircbot.channel) {
			messagesCount++
			if messagesCount == ircbot.autoMSG2Count {
				ircbot.Message(ircbot.autoMSG2)
			}
			userdata := strings.Split(line, ".tmi.twitch.tv PRIVMSG "+ircbot.channel)
			username := strings.Split(userdata[0], "@")
			usermessage := strings.Replace(userdata[1], " :", "", 1)

			fmt.Printf(username[1] + ": " + usermessage + "\n")
			if ircbot.userLastMsg[username[1]]+int64(ircbot.userMaxLastMsg) >= time.Now().Unix() {
				ircbot.timeout(username[1], "spam")
			}
			ircbot.userLastMsg[username[1]] = time.Now().Unix()
			go ircbot.CmdInterpreter(username[1], usermessage)

		} else if strings.Contains(line, ".tmi.twitch.tv JOIN "+ircbot.channel) {
			userjoindata := strings.Split(line, ".tmi.twitch.tv JOIN "+ircbot.channel)
			userjoined := strings.Split(userjoindata[0], "@")
			fmt.Printf(userjoined[1] + " has joined!\n") //TODO: Database for all people joined and to have it check to see if they have joined.  If not, greet them
		} else if strings.Contains(line, ".tmi.twitch.tv PART "+ircbot.channel) {
			userjoindata := strings.Split(line, ".tmi.twitch.tv PART "+ircbot.channel)
			userjoined := strings.Split(userjoindata[0], "@")
			fmt.Printf(userjoined[1] + " has left!\n")
		} else if strings.Contains(line, ":jtv MODE "+ircbot.channel+" +o ") {
			usermod := strings.Split(line, ":jtv MODE "+ircbot.channel+" +o ")
			ircbot.mods[usermod[1]] = true
			fmt.Printf(usermod[1] + " is a moderator!\n")
		} else if strings.Contains(line, ":jtv MODE "+ircbot.channel+" -o ") {
			usermod := strings.Split(line, ":jtv MODE "+ircbot.channel+" -o ")
			ircbot.mods[usermod[1]] = false
			fmt.Printf(usermod[1] + " isn't a moderator anymore!\n")
		}
	}
	fmt.Println("were exiting somehow?")
}

func getJson(url string, target interface{}) error {
	r, err := http.Get(url)
	if err != nil {
		return err
	}
	defer r.Body.Close()

	return json.NewDecoder(r.Body).Decode(target)
}
