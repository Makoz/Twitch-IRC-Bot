// commands
package main

import (
	"encoding/json"
	"fmt"
	"github.com/shkh/lastfm-go/lastfm"
	"io/ioutil"
	"net/http"
	"reflect"
	"strings"
	"time"
)

func (bot *Bot) CmdInterpreter(username string, usermessage string) {
	message := strings.ToLower(usermessage)
	tempstr := strings.Split(message, " ")

	for _, str := range tempstr {
		if strings.HasPrefix(str, "https://") || strings.HasPrefix(str, "http://") {
			go bot.Message("^ " + webTitle(str))
		} else if isWebsite(str) {
			go bot.Message("^ " + webTitle("http://"+str))
		}
	}

	if strings.HasPrefix(message, "!uptime") {
		uptime := bot.getUptime(bot.channel)
		bot.Message(uptime)
	}

	if strings.HasPrefix(message, "!help") {
		bot.Message("Sucks for you I have no idea what I'm doing")
	} else if strings.HasPrefix(message, "!quote") {
		bot.Message(bot.getQuote())
	} else if strings.HasPrefix(message, "!addquote ") {
		stringpls := strings.Replace(message, "!addquote ", "", 1)
		if bot.isMod(username) {
			bot.quotes[stringpls] = username
			bot.writeQuoteDB()
			bot.Message("Quote added!")
		} else {
			bot.Message(username + " you are not a mod!")
		}
	} else if strings.HasPrefix(message, "!timeout ") {
		stringpls := strings.Replace(message, "!timeout ", "", 1)
		temp1 := strings.Split(stringpls, " ")
		temp2 := strings.Replace(stringpls, temp1[0], "", 1)
		if temp2 == "" {
			temp2 = "no reason"
		}
		if bot.isMod(username) {
			bot.timeout(temp1[0], temp2)
		} else {
			bot.Message(username + " you are not a mod!")
		}
	} else if strings.HasPrefix(message, "!ban ") {
		stringpls := strings.Replace(message, "!ban ", "", 1)
		temp1 := strings.Split(stringpls, " ")
		temp2 := strings.Replace(stringpls, temp1[0], "", 1)
		if temp2 == "" {
			temp2 = "no reason"
		}
		if bot.isMod(username) {
			bot.ban(temp1[0], temp2)
		} else {
			bot.Message(username + " you are not a mod!")
		}
	} else if message == "!song" {
		api := lastfm.New("e6563970017df6d5966edfa836e12835", "dcc462ffd8a371fee5a5b49c248a2371")
		temp, _ := api.User.GetRecentTracks(lastfm.P{"user": bot.lastfm})
		var inserthere string
		if temp.Tracks[0].Date.Date != "" {
			inserthere = ". It was played on: " + temp.Tracks[0].Date.Date
		}
		bot.Message("Song: " + temp.Tracks[0].Artist.Name + " - " + temp.Tracks[0].Name + inserthere)
	}
}

//Website stuff
func webTitle(website string) string {
	response, err := http.Get(website)
	if err != nil {
		return "Error reading website"
	} else {
		defer response.Body.Close()
		contents, err := ioutil.ReadAll(response.Body)
		if err != nil {
			return "Error reading website"
		}
		if strings.Contains(string(contents), "<title>") && strings.Contains(string(contents), "</title>") {
			derp := strings.Split(string(contents), "<title>")
			derpz := strings.Split(derp[1], "</title>")
			return derpz[0]
		}
		return "No title"
	}
}

func isWebsite(website string) bool {
	domains := []string{".com", ".net", ".org", ".info", ".fm", ".gg", ".tv"}
	for _, domain := range domains {
		if strings.Contains(website, domain) {
			return true
		}
	}
	return false
}

//End website stuff

//Mod stuff

func (bot *Bot) getUptime(username string) string {

	url := "https://api.twitch.tv/kraken/streams/" + username
	resp, err := http.Get(url)
	if err != nil {
		fmt.Print(err)

	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	var f interface{}
	err = json.Unmarshal(body, &f)
	if err != nil {
		fmt.Println(err)
		return err.Error()
	}
	// fmt.Println(f, "EKLWJFKLE")
	m := f.(map[string]interface{})

	if stream, ok := m["stream"]; ok {
		streamMap := stream.(map[string]interface{})
		created_at := reflect.ValueOf(streamMap["created_at"]).String()
		fmt.Println(created_at)

		t, err := time.Parse("2006-01-02T15:04:05Z07:00", created_at)
		fmt.Println(t, err)
		fmt.Println(time.Since(t))
		str := time.Since(t).String()
		hours := str[0:strings.Index(str, "h")]
		mins := str[strings.Index(str, "h")+1 : strings.Index(str, "m")]
		fmt.Println(hours, mins)

		return "Stream has been up for " + hours + " hours and " + mins + " mins."
	} else {
		return "Stream is not online."
	}
}

func (bot *Bot) isMod(username string) bool {
	fmt.Println(bot.channel)
	temp := strings.Replace(bot.channel, "#", "", 1)
	fmt.Println("temp is: ", temp)
	if bot.mods[username] == true || temp == username || username == "vaultpls" {
		return true
	}

	// Look at chattesr tmi
	resp, err := http.Get("http://tmi.twitch.tv/group/user/" + bot.channel[1:] + "/chatters")

	if err != nil {
		// handle error
	}
	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)

	var f interface{}
	err = json.Unmarshal(body, &f)
	if err != nil {
		fmt.Println(err)
		return false
	}
	fmt.Println(f, "EKLWJFKLE")
	m := f.(map[string]interface{})
	s := m["chatters"]
	if s == nil {
		return false
	}
	fmt.Println(m, s)
	// fmt.Println(s)
	strings1 := s.(map[string]interface{})
	// fmt.Println(strings)
	powU := strings1["moderators"]
	mods := reflect.ValueOf(powU)

	for i := 0; i < mods.Len(); i++ {

		sTemp := mods.Index(i).Interface().(string)
		if username == sTemp {
			return true
		}
	}
	fmt.Println("ENDDD")
	return false
}

func (bot *Bot) timeout(username string, reason string) {
	if bot.isMod(username) {
		return
	}
	fmt.Fprintf(bot.conn, "PRIVMSG "+bot.channel+" :/timeout "+username+"\r\n")
	bot.Message(username + " was timed out(" + reason + ")!")
}

func (bot *Bot) ban(username string, reason string) {
	if bot.isMod(username) {
		return
	}
	fmt.Fprintf(bot.conn, "PRIVMSG "+bot.channel+" :/ban "+username+"\r\n")
	bot.Message(username + " was banned(" + reason + ")!")
}

//End mod stuff
