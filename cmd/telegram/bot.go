package telegram

import (
	"log"

	"bot/cmd/database"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	_ "github.com/mattn/go-sqlite3"
	"github.com/skip2/go-qrcode"
)

type Bot struct {
	api *tgbotapi.BotAPI
	db  *database.Db
}

func NewBot(token string, db *database.Db) (*Bot, error) {
	bot, err := tgbotapi.NewBotAPI(token)
	if err != nil {
		panic(err)
	}
	_, err = bot.Request(tgbotapi.NewSetMyCommands(
		tgbotapi.BotCommand{
			Command:     "/qrcode",
			Description: "Generate qr-code",
		},
		tgbotapi.BotCommand{
			Command:     "/ping",
			Description: "ping bot",
		},
	))
	if err != nil {
		return &Bot{}, err
	}
	return &Bot{
		api: bot,
		db:  db,
	}, nil
}

func (b *Bot) Run() {
	u := tgbotapi.NewUpdate(0)
	u.Timeout = 60
	updates := b.api.GetUpdatesChan(u)
	for update := range updates {
		if update.Message != nil {
			if update.Message.Text == "start" {
				b.db.SaveCommandById(update.Message.Command(), update.Message.Chat.ID)
				continue
			}
			command, err := b.db.GetCommandById(update.Message.Chat.ID)
			if err != nil {
				b.sendMessage(update.Message.From.ID, err.Error())
				b.db.SaveCommandById(update.Message.Command(), update.Message.Chat.ID)
				continue
			}
			if command == update.Message.Text {
				continue
			}
			switch command {
			case "qrcode":
				b.sendQrCode(b.api, update)
				e := b.db.ClearCommandById(update.Message.Chat.ID)
				if e != nil {
					b.sendMessage(update.Message.From.ID, e.Error())
				}
			case "ping":
				b.sendMessage(update.Message.From.ID, "Pong "+update.Message.Text)
				e := b.db.ClearCommandById(update.Message.Chat.ID)
				if e != nil {
					b.sendMessage(update.Message.From.ID, e.Error())
				}
			default:
				b.db.SaveCommandById(update.Message.Command(), update.Message.Chat.ID)
			}
		}
	}
}

func (b *Bot) sendMessage(id int64, text string) {
	repl := tgbotapi.NewMessage(id, text)
	b.api.Send(repl)
}

func (b *Bot) sendQrCode(bot *tgbotapi.BotAPI, update tgbotapi.Update) {
	res, err := qrcode.Encode(update.Message.Text, qrcode.High, 512)
	if err != nil {
		log.Panic(err)
	}
	msg := tgbotapi.NewPhoto(update.Message.From.ID, tgbotapi.FileBytes{
		Name:  "qr.png",
		Bytes: res,
	})
	msg.ReplyToMessageID = update.Message.MessageID
	bot.Send(msg)
}
