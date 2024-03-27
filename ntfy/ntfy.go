package ntfy

import (
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"strings"

	"github.com/0ranki/hydroxide-push/auth"
	"github.com/0ranki/hydroxide-push/config"
	"github.com/emersion/go-imap"
	"github.com/emersion/go-imap/backend"
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
	cfg := NtfyConfig{}
	if err := cfg.Read(); err != nil {
		log.Printf("error reading notification: %v", err)
		return
	}
	req, _ := http.NewRequest("POST", cfg.String(), strings.NewReader("New message received"))
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

func AskToSaveBridgePw(cfg *NtfyConfig) (string, error) {
	scanner := bufio.NewScanner(os.Stdin)
	//fmt.Printf("Save bridge password to config?\nThe password is stored in plain text, but  (yes/n): ")
	//scanner.Scan()
	//if scanner.Text() == "yes" {
	if err := cfg.Save(); err != nil {
		return "", errors.New("failed to save notification config")
	}
	//}
	return scanner.Text(), nil
}

func LoginBridge(cfg *NtfyConfig) error {
	if cfg.BridgePw == "" {
		cfg.BridgePw = os.Getenv("HYDROXIDE_BRIDGE_PASSWORD")
	}
	if cfg.BridgePw == "" {
		scanner := bufio.NewScanner(os.Stdin)
		fmt.Printf("Bridge password: ")
		scanner.Scan()
		cfg.BridgePw = scanner.Text()
		_, err := AskToSaveBridgePw(cfg)
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
		err = LoginBridge(cfg)
		if err != nil {
			log.Fatal(err)
		}
	}
	_, err = be.Login(&conn, usernames[0], cfg.BridgePw)
	if err != nil {
		log.Fatal(err)
	}
}
