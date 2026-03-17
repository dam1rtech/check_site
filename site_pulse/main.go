package main

import (
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"net/url"
	"os"
	"strings"
	"sync"
	"time"
)

func SendToTelegram(message string) {
	token := os.Getenv("BOT_TOKEN")
	chatID := os.Getenv("BOT_CHAT_ID")

	if token == "" || chatID == "" {
		log.Printf("⚠️ ОШИБКА: Переменные BOT_TOKEN или BOT_CHAT_ID пусты! Проверь запуск программы.")
		return
	}

	encodedMsg := url.QueryEscape(message)
	apiURL := fmt.Sprintf("https://api.telegram.org/bot%s/sendMessage?chat_id=%s&text=%s", token, chatID, encodedMsg)

	resp, err := http.Get(apiURL)
	if err != nil {
		log.Printf("❌ СЕТЕВАЯ ОШИБКА: Не удалось достучаться до Telegram: %v", err)
		return
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)

	if resp.StatusCode != http.StatusOK {
		log.Printf("🚨 ТЕЛЕГРАМ ОТКАЗАЛ: Статус %d, Ответ: %s", resp.StatusCode, string(body))
	} else {
		log.Println("✅ ТЕЛЕГРАМ ПРИНЯЛ: Сообщение успешно отправлено!")
	}
}

func CheckURLSite(url string, ch chan string, wg *sync.WaitGroup) {
	defer wg.Done()
	resp, err := http.Get(url)
	if err != nil {
		ch <- fmt.Sprintf("❌ %s: ошибка (%v)", url, err)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusOK {
		ch <- fmt.Sprintf("✅ %s: работает! Status: %d", url, resp.StatusCode)
	} else {
		ch <- fmt.Sprintf("⚠️ %s: вернул статус %d", url, resp.StatusCode)
	}

}

func CheckConnection(target string, ch chan string, wg *sync.WaitGroup) {
	defer wg.Done()

	conn, err := net.DialTimeout("tcp", target, 3*time.Second)
	if err != nil {
		ch <- fmt.Sprintf("🚨 СЕТЬ: %s недоступен (%v)", target, err)
		return
	}
	conn.Close()

	ch <- fmt.Sprintf("✅ СЕТЬ: %s доступен", target)

}
func main() {
	f, err := os.OpenFile("app.log", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		log.Fatal(err)
	}
	defer f.Close()

	mv := io.MultiWriter(os.Stdout, f)
	log.SetOutput(mv)
	log.SetFlags(0)
	sites := []string{
		"https://www.googleee.com",
		"https://github.com",
		"https://yandex.ru",
		"https://attendance-app.mirea.ru",
	}
	dns_servers := []string{
		"8.8.8.8:53",
		"google.com:80",
	}
	ticker := time.NewTicker(10 * time.Second)
	fmt.Println("Запуск проверки сайтов... (каждые 10 секунд)")

	for range ticker.C {
		ch := make(chan string)
		var wg sync.WaitGroup

		for _, url := range sites {
			wg.Add(1)
			go CheckURLSite(url, ch, &wg)
		}
		for _, target := range dns_servers {
			wg.Add(1)
			go CheckConnection(target, ch, &wg)
		}

		go func() {
			wg.Wait()
			close(ch)
		}()
		log.Printf("\n--- Проверка от %s ---\n", time.Now().Format("15:04:05"))
		for mesage := range ch {
			log.Println(mesage)

			if strings.Contains(mesage, "❌") || strings.Contains(mesage, "🚨") || strings.Contains(mesage, "⚠️") {

				go SendToTelegram("📡 [MONITOR]: " + mesage)
			}
		}
		log.Println("---------------------------------")
	}
}
