package main

import (
	"encoding/json"
	"fmt"
	"math/rand"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/Wlademon/vkBot/file"
	"github.com/Wlademon/vkBot/file/cache"
	absTime "github.com/Wlademon/vkBot/time"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api"
	"github.com/goccy/go-yaml"
	"github.com/joho/godotenv"
)

var Location *time.Location

var TodayNames []string
var TodayPrediction []string

var Peoples map[int64]struct {
	Link  string
	Name  string
	Date  string
	Value string
}

func main() {
	cache.InitCache("cache")
	initEnv()
	absTime.InitTime(time.Hour * 3 / time.Second)
	setVars()
	getCacheStat()
	Location = time.FixedZone("Current", int(time.Hour*3/time.Second))
	Peoples = make(map[int64]struct {
		Link  string
		Name  string
		Date  string
		Value string
	})
	bot, err := initBot()
	if err != nil {
		panic(err)
	}
	u := tgbotapi.NewUpdate(0)
	u.Timeout = 60
	updates, _ := bot.GetUpdatesChan(u)
	observeCommands(updates, bot)
}

func setCacheStat() {
	marshal, err := json.Marshal(Peoples)
	if err != nil {
		return
	}
	cache.CreateForever("PEOPLES", string(marshal)).Set()
}

func getCacheStat() {
	ex, val := cache.Get("PEOPLES")
	if ex {
		err := json.Unmarshal([]byte(val), &Peoples)
		if err != nil {
			return
		}
	}
}

func setVars() {
	yamlFile := file.File{Name: "vars.yml"}
	read, err := yamlFile.Read()
	if err != nil {
		return
	}
	y := make(map[string][]string)

	err = yaml.Unmarshal(read, &y)
	if err != nil {
		return
	}
	TodayNames = y["variables"]
	TodayPrediction = y["prediction"]
}

func getTodayName(user int64) string {
	exist, value := cache.Get(strconv.FormatInt(user, 10))
	if exist {
		return value
	}
	nowDate := time.Now().In(Location)
	nextCurDay := nowDate.Add(time.Hour * 24)
	nextDate := time.Date(nextCurDay.Year(), nextCurDay.Month(), nextCurDay.Day(), 0, 0, 0, 0, nextCurDay.Location())

	value = randName(user)
	if value == "" {
		return "Вытяни другую"
	}
	cache.Create(strconv.FormatInt(user, 10), value, nextDate.Sub(nowDate)).Set()

	return value
}

func randName(user int64) string {
	rand.Seed(time.Now().UnixNano())
	i := rand.Intn(len(TodayNames) - 1)
	variable := TodayNames[i]
	if variable == "@" {
		pI := rand.Intn(len(TodayPrediction) - 1)
		return "Тебе выпало предсказание: " + TodayPrediction[pI]
	}
	if variable == "+" {
		return ""
	}

	return "Сегодня ты: " + TodayNames[i]
}

func initEnv() {
	err := godotenv.Load(".env")
	if err != nil {
		panic(err)
	}
}

func initBot() (*tgbotapi.BotAPI, error) {
	bot, err := tgbotapi.NewBotAPI(os.Getenv("TELEGRAM_ACCESS_TOKEN"))
	if err != nil {
		return nil, err
	}
	bot.Debug = true

	return bot, err
}

func observeCommands(updates tgbotapi.UpdatesChannel, bot *tgbotapi.BotAPI) {
	var isCommand bool
	for update := range updates {
		if update.Message == nil {
			continue
		}
		entities := update.Message.Entities
		isCommand = false
		if entities != nil {
			for _, entity := range *entities {
				if entity.Type == "bot_command" {
					isCommand = true
					break
				}
			}
		}
		if isCommand {
			returnMessage, reply := runCommand(update.Message.Text, update.Message)
			if returnMessage != "" {
				msg := tgbotapi.NewMessage(update.Message.Chat.ID, returnMessage)
				if reply {
					msg.ReplyToMessageID = update.Message.MessageID
				}
				_, _ = bot.Send(msg)
			}
		}
	}
}

func runCommand(command string, message *tgbotapi.Message) (string, bool) {
	user := int64(message.From.ID)
	arrCommand := strings.Split(strings.Trim(strings.ReplaceAll(command, "  ", " "), " "), " ")
	commandExec := arrCommand[0]
	commandExec = strings.Split(commandExec, "@")[0]
	switch commandExec {
	case "/state":
		result := ""
		for id, p := range Peoples {
			result +=
				"\n--------------------\n" +
					fmt.Sprintf("ID: %d\nName: %s\nLink: @%s\nValue: %s\nDate: %s", id, p.Name, p.Link, p.Value, p.Date)
		}
		return result, false
	case "/reset_names":
		setVars()
		return "OK", true
	case "/who_am_i_today":
		Peoples[int64(message.From.ID)] = struct {
			Link  string
			Name  string
			Date  string
			Value string
		}{Link: message.From.String(), Name: message.From.FirstName + " " + message.From.LastName, Date: time.Now().In(Location).Format("2006-01-02"), Value: getTodayName(user)}
		setCacheStat()
		return Peoples[int64(message.From.ID)].Value, true
	}

	return "", false
}
