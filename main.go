package main

import (
	"math/rand"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/Wlademon/vkBot/file/cache"
	absTime "github.com/Wlademon/vkBot/time"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api"
	"github.com/joho/godotenv"
)

var Location *time.Location
var TodayNames []string

func main() {
	cache.InitCache("cache")
	initEnv()
	setTodayNames()
	absTime.InitTime("Europe/Moscow")
	Location = time.FixedZone("Current", int(time.Hour*3/time.Second))
	bot, err := initBot()
	if err != nil {
		panic(err)
	}

	u := tgbotapi.NewUpdate(0)
	u.Timeout = 60
	updates, _ := bot.GetUpdatesChan(u)

	observeCommands(updates, bot)
}

func setTodayNames() {
	TodayNames = strings.Split(os.Getenv("VARIABLES"), "%%%")
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
	value = "Сегодня ты: " + value
	cache.Create(strconv.FormatInt(user, 10), value, nextDate.Sub(nowDate)).Set()

	return value
}

func randName(user int64) string {
	rand.Seed(time.Now().Unix())
	i := rand.Intn(len(TodayNames) + 1)
	if i > len(TodayNames)-1 {
		return ""
	}

	return TodayNames[i]
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
	case "/reset_names":
		setTodayNames()
		return "OK", true
	case "/who_am_i_today":
		return getTodayName(user), true
	}

	return "", false
}
