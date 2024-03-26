package ntfy

import (
	"bufio"
	"encoding/json"
	"fmt"
	"github.com/0ranki/hydroxide-push/auth"
	"github.com/0ranki/hydroxide-push/config"
	"github.com/emersion/go-imap"
	"github.com/emersion/go-imap/backend"
	"log"
	"net"
	"net/http"
	"os"
	"strings"
)

type NtfyConfig struct {
	URL      string `json:"url"`
	Topic    string `json:"topic"`
	BridgePw string `json:"bridgePw"`
}

func (cfg *NtfyConfig) String() string {
	return fmt.Sprintf("%s/%s", cfg.URL, cfg.Topic)
}

func (cfg *NtfyConfig) Save() error {
	b, err := json.Marshal(cfg)
	if err != nil {
		return err
	}
	path, err := ntfyConfigFile()
	if err != nil {
		return err
	}
	return os.WriteFile(path, b, 0600)
}

func ntfyConfigFile() (string, error) {
	return config.Path("notify.json")
}

func Notify() {
	req, _ := http.NewRequest("POST", "https://push.oranki.net/testing20240325", strings.NewReader("New message received"))
	req.Header.Set("Title", "ProtoMail")
	req.Header.Set("Click", "dismiss")
	req.Header.Set("Tags", "envelope")
	http.DefaultClient.Do(req)
}

func (cfg *NtfyConfig) Read() error {
	f, err := ntfyConfigFile()
	if err == nil {
		b, err := os.ReadFile(f)
		if err == nil {
			err = json.Unmarshal(b, &cfg)
		}
		if err != nil {
			return err
		}
	}
	return nil
}

func Login(cfg *NtfyConfig, be backend.Backend) {
	//time.Sleep(1 * time.Second)
	c, _ := net.ResolveIPAddr("ip", "127.0.0.1")
	conn := imap.ConnInfo{
		RemoteAddr: c,
		LocalAddr:  c,
		TLS:        nil,
	}
	usernames, err := auth.ListUsernames()
	if err != nil {
		log.Fatal(err)
	}
	if len(usernames) > 1 {
		log.Fatal("only one login supported for now")
	}
	err = cfg.Read()
	if err != nil {
		log.Println(err)
	}
	if len(usernames) == 0 || cfg.URL == "" || cfg.Topic == "" {
		executable, _ := os.Executable()
		log.Println("login first using " + executable + " auth <protonmail username>")
		log.Fatalln("then setup ntfy using " + executable + "setup-ntfy")
	}
	if cfg.BridgePw == "" {
		cfg.BridgePw = os.Getenv("HYDROXIDE_BRIDGE_PASSWORD")
	}
	if cfg.BridgePw == "" {
		scanner := bufio.NewScanner(os.Stdin)
		fmt.Printf("Bridge password: ")
		scanner.Scan()
		cfg.BridgePw = scanner.Text()
		scanner = bufio.NewScanner(os.Stdin)
		fmt.Printf("Save password to config? The password is stored in plain text! (yes/n): ")
		scanner.Scan()
		if scanner.Text() == "yes" {
			if err = cfg.Save(); err != nil {
				log.Fatal("failed to save notification config")
			}
		}
	}
	_, err = be.Login(&conn, usernames[0], cfg.BridgePw)
	if err != nil {
		log.Fatal(err)
	}
}
