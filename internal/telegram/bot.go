package telegram

import (
	"errors"
	"log"
	"strings"

	"bot/internal/database"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	_ "github.com/mattn/go-sqlite3"
	"github.com/skip2/go-qrcode"
)

type Bot struct {
	api *tgbotapi.BotAPI
	db  *database.Db
}

var numericKeyboard = tgbotapi.NewReplyKeyboard(
	tgbotapi.NewKeyboardButtonRow(
		tgbotapi.NewKeyboardButton("Пульт"),
	),
)

func NewBot(token string, db *database.Db) (*Bot, error) {
	bot, err := tgbotapi.NewBotAPI(token)

	if err != nil {
		panic(err)
	}
	_, err = bot.Request(tgbotapi.NewSetMyCommands(
		tgbotapi.BotCommand{
			Command:     "/save",
			Description: "Save list",
		},
		tgbotapi.BotCommand{
			Command:     "/getlist",
			Description: "Get list",
		},
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
			if update.Message.Text == "Пульт" {
				msg := tgbotapi.NewMessage(update.Message.Chat.ID, "Выберите команду")
				msg.ReplyMarkup = tgbotapi.NewInlineKeyboardMarkup(
					tgbotapi.NewInlineKeyboardRow(
						tgbotapi.NewInlineKeyboardButtonData(
							"Списки", "lists",
						),
						tgbotapi.NewInlineKeyboardButtonData(
							"Создать новый список", "create list",
						),
						tgbotapi.NewInlineKeyboardButtonData(
							"Удалить список", "delete list",
						),
					),
				)
				b.api.Send(msg)
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
			case "save":
				res := strings.Split(update.Message.Text, " ")
				if len(res) == 1 {
					b.sendMessage(update.Message.Chat.ID, "Нужно указать список и потом значение")
					continue
				}
				b.SaveList(update.Message.Chat.ID, res[0], res[1])
				b.db.ClearCommandById(update.Message.Chat.ID)
			case "getlist":
				res, _ := b.GetList(update.Message.Chat.ID, update.Message.Text)
				b.sendMessage(update.Message.Chat.ID, strings.Join(res, "\n"))
				b.db.ClearCommandById(update.Message.Chat.ID)
			case "qrcode":
				go b.sendQrCode(b.api, update)
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
		} else if update.CallbackQuery != nil {
			if update.CallbackQuery.Data == "lists" {
				res, err := b.db.GetListLists(update.CallbackQuery.From.ID)
				if err != nil {
					b.sendMessage(update.CallbackQuery.From.ID, err.Error())
					continue
				}
				if len(res) == 0 {
					b.sendMessage(update.CallbackQuery.From.ID, err.Error())
					continue
				}
				msg := tgbotapi.NewMessage(update.CallbackQuery.From.ID, "Выберите список:")
				var buttons []tgbotapi.InlineKeyboardButton
				for _, v := range res {
					buttons = append(buttons, tgbotapi.NewInlineKeyboardButtonData(v, "lists_button_"+v))
				}
				msg.ReplyMarkup = tgbotapi.NewInlineKeyboardMarkup(buttons)
				b.api.Send(msg)
				continue
			}
			if strings.HasPrefix(update.CallbackQuery.Data, "lists_button_") {
				res, err := b.GetList(
					update.CallbackQuery.From.ID,
					strings.Split(update.CallbackQuery.Data, "lists_button_")[1],
				)
				if err != nil {
					b.sendMessage(update.CallbackQuery.From.ID, err.Error())
					continue
				}
				b.sendMessage(update.CallbackQuery.From.ID, strings.Join(res, "\n"))
				continue
			}
			if update.CallbackQuery.Data == "create list" {
				b.db.SaveCommandById("save", update.CallbackQuery.From.ID)
				b.sendMessage(update.CallbackQuery.From.ID, "Введите список (пробел) значение:")
				continue
			}
			if update.CallbackQuery.Data == "delete list" {
				res, err := b.db.GetListLists(update.CallbackQuery.From.ID)
				if err != nil {
					b.sendMessage(update.CallbackQuery.From.ID, err.Error())
					continue
				}
				msg := tgbotapi.NewMessage(update.CallbackQuery.From.ID, "Выберите список для удаленя:")
				var buttons []tgbotapi.InlineKeyboardButton
				for _, v := range res {
					buttons = append(buttons, tgbotapi.NewInlineKeyboardButtonData(v, "delete_lists_button_"+v))
				}
				msg.ReplyMarkup = tgbotapi.NewInlineKeyboardMarkup(buttons)
				b.api.Send(msg)
				continue
			}
			if strings.HasPrefix(update.CallbackQuery.Data, "delete_lists_button_") {
				err := b.DeleteList(
					update.CallbackQuery.From.ID,
					strings.Split(update.CallbackQuery.Data, "delete_lists_button_")[1],
				)
				if err != nil {
					b.sendMessage(update.CallbackQuery.From.ID, err.Error())
					continue
				}
				b.sendMessage(update.CallbackQuery.From.ID, "Список '"+strings.Split(update.CallbackQuery.Data, "delete_lists_button_")[1]+"' успешно удален")
				continue
			}
		}
	}
}

func (b *Bot) sendMessage(id int64, text string) {
	repl := tgbotapi.NewMessage(id, text)
	repl.ReplyMarkup = numericKeyboard
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

func (b *Bot) SaveList(id int64, key, value string) error {
	return b.db.SaveList(id, key, value)
}

func (b *Bot) GetList(id int64, key string) ([]string, error) {
	res, err := b.db.GetList(id, key)
	if err != nil {
		return res, err
	}
	if len(res) == 0 {
		return res, errors.New("List is empty")
	}
	return res, err
}

func (b *Bot) DeleteList(id int64, key string) error {
	return b.db.DeleteList(id, key)
}
