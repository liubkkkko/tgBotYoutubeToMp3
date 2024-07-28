package main

import (
	"log"
	"os"
	"os/exec"
	"strings"
	"sync"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

func main() {
	// Створення нового бота з використанням вашого Telegram токена
	bot, err := tgbotapi.NewBotAPI("TOKEN_API")
	if err != nil {
		log.Fatalf("Failed to create bot: %v", err)
	}

	// Включення режиму відладки для отримання детальної інформації про роботу бота
	bot.Debug = true

	// Налаштування параметрів отримання оновлень (повідомлень) від Telegram
	u := tgbotapi.NewUpdate(0)
	u.Timeout = 60

	// Отримання каналу оновлень
	updates := bot.GetUpdatesChan(u)

	// Створення об'єкта WaitGroup для управління горутинами
	var wg sync.WaitGroup

	// Цикл обробки оновлень (повідомлень)
	for update := range updates {
		if update.Message == nil { // Пропуск оновлень, що не містять повідомлення
			continue
		}

		// Додавання нової горутини для обробки повідомлення
		if update.Message.Text != "" {
			wg.Add(1) // Збільшення лічильника горутин
			go func(update tgbotapi.Update) {
				defer wg.Done() // Зменшення лічильника горутин після завершення обробки
				handleUpdate(bot, update)
			}(update)
		}
	}

	// Очікування завершення всіх горутин
	wg.Wait()
}

// Функція для обробки кожного повідомлення
func handleUpdate(bot *tgbotapi.BotAPI, update tgbotapi.Update) {
	videoURL := update.Message.Text
	mp3File, err := downloadAndConvert(videoURL) // Завантаження та конвертація відео
	if err == nil {
		// Відправка конвертованого аудіо файлу користувачу
		audio := tgbotapi.NewAudio(update.Message.Chat.ID, tgbotapi.FilePath(mp3File))
		_, err = bot.Send(audio)
		if err != nil {
			log.Printf("Error sending audio: %v", err)
		}
		cleanupFiles(mp3File) // Видалення тимчасових файлів
	} else {
		// Відправка повідомлення про помилку користувачу
		msg := tgbotapi.NewMessage(update.Message.Chat.ID, "Sorry, I couldn't download the video.")
		_, err := bot.Send(msg)
		if err != nil {
			log.Printf("Error sending message: %v", err)
		}
	}
}

// Функція для завантаження та конвертації відео
func downloadAndConvert(url string) (string, error) {
	// Завантаження назви відео
	cmd := exec.Command("yt-dlp", "--get-title", url)
	titleBytes, err := cmd.Output()
	if err != nil {
		log.Printf("Error getting video title: %v", err)
		return "", err
	}

	title := strings.TrimSpace(string(titleBytes))
	title = sanitizeFileName(title) // Очистка назви файлу
	videoFile := title + ".webm"
	mp3File := title + ".mp3"

	// Завантаження відео
	cmd = exec.Command("yt-dlp", "-f", "bestaudio[ext=webm]", "-o", videoFile, url)
	output, err := cmd.CombinedOutput()
	if err != nil {
		log.Printf("Error downloading video: %s\nOutput: %s\n", err, output)
		return "", err
	}

	// Конвертація відео в аудіо
	cmd = exec.Command("ffmpeg", "-i", videoFile, "-q:a", "0", "-map", "a", mp3File)
	output, err = cmd.CombinedOutput()
	if err != nil {
		log.Printf("Error converting video: %s\nOutput: %s\n", err, output)
		return "", err
	}

	// Видалення відео файлу
	err = os.Remove(videoFile)
	if err != nil {
		log.Println("Error removing video file:", err)
	}

	return mp3File, nil
}

// Функція для очищення назви файлу
func sanitizeFileName(name string) string {
	invalidChars := []string{"/", "\\", ":", "*", "?", "\"", "<", ">", "|"}
	for _, char := range invalidChars {
		name = strings.ReplaceAll(name, char, "_")
	}
	return name
}

// Функція для видалення тимчасових файлів
func cleanupFiles(files ...string) {
	for _, file := range files {
		err := os.Remove(file)
		if err != nil {
			log.Println("Error removing file:", err)
		}
	}
}
