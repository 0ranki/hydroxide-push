package ntfy

import (
	"bufio"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"golang.org/x/crypto/ssh/terminal"
	"log"
	"net"
	"net/http"
	"net/url"
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
	User     string `json:"user"`
	Password string `json:"password"`
}

func (cfg *NtfyConfig) Init() {
	if cfg.Topic == "" {
		r := make([]byte, 12)
		_, err := rand.Read(r)
		if err != nil {
			log.Fatal(err)
		}
		cfg.Topic = strings.Replace(base64.StdEncoding.EncodeToString(r), "/", "+", -1)

	}
	if cfg.URL == "" {
		cfg.URL = "http://ntfy.sh"
	}
}

func (cfg *NtfyConfig) URI() string {
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
		log.Printf("error reading configuration: %v\n", err)
		return
	}
	req, _ := http.NewRequest("POST", cfg.URI(), strings.NewReader("New message received"))
	if cfg.User != "" && cfg.Password != "" {
		pw, err := base64.StdEncoding.DecodeString(cfg.Password)
		if err != nil {
			log.Printf("Error decoding push endpoint password: %v\n", err)
			return
		}
		req.SetBasicAuth(cfg.User, string(pw))
	}
	req.Header.Set("Title", "ProtonMail")
	req.Header.Set("Click", "dismiss")
	req.Header.Set("Tags", "envelope")
	if _, err := http.DefaultClient.Do(req); err != nil {
		log.Printf("failed to publish to push topic: %v", err)
		return
	}
	log.Printf("Push event sent")

}

// Read reads the configuration from file. Creates the file
// if it does not exist
func (cfg *NtfyConfig) Read() error {
	f, err := ntfyConfigFile()
	if err == nil {
		b, err := os.ReadFile(f)
		if err == nil {
			err = json.Unmarshal(b, &cfg)
		} else if strings.HasSuffix(err.Error(), "no such file or directory") {
			cfg.Init()
			err = cfg.Save()
		}
		if err != nil {
			log.Fatal(err)
		}
		cfg.Init()
	}
	return nil
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

func (cfg *NtfyConfig) Setup() {

	// Configure using environment
	if os.Getenv("PUSH_URL") != "" && os.Getenv("PUSH_TOPIC") != "" {
		cfg.URL = os.Getenv("PUSH_URL")
		cfg.Topic = os.Getenv("PUSH_TOPIC")
		log.Printf("Current push endpoint: %s\n", cfg.URI())
		if os.Getenv("PUSH_USER") != "" && os.Getenv("PUSH_PASSWORD") != "" {
			cfg.User = os.Getenv("PUSH_USER")
			cfg.Password = base64.StdEncoding.EncodeToString([]byte(os.Getenv("PUSH_PASSWORD")))
			log.Println("Authentication for push endpoint configured using environment")
		} else {
			log.Println("Both PUSH_USER and PUSH_PASSWORD not set, assuming no authentication is necessary.")
		}
		err := cfg.Save()
		if err != nil {
			log.Fatal(err)
		}
		return
	}

	var n string
	if cfg.URL != "" && cfg.Topic != "" {
		fmt.Printf("Current push endpoint: %s\n", cfg.URI())
		n = "new "
	}
	if cfg.User != "" && cfg.Password != "" {
		fmt.Println("Push is currently configured for basic auth. You'll need to input credentials again")
	}

	// Read push base URL
	notValid := true
	scanner := bufio.NewScanner(os.Stdin)
	for notValid {
		tmpURL := cfg.URL
		fmt.Printf("Input %spush server URL ('%s') : ", n, cfg.URL)
		scanner.Scan()
		if len(scanner.Text()) > 0 {
			tmpURL = scanner.Text()
		}
		if _, err := url.ParseRequestURI(tmpURL); err != nil {
			fmt.Printf("Not a valid URL: %s\n", tmpURL)
		} else {
			notValid = false
			cfg.URL = tmpURL
		}
	}
	scanner = bufio.NewScanner(os.Stdin)
	// Read push topic
	fmt.Printf("Input push topic ('%s'): ", cfg.Topic)
	scanner.Scan()
	if len(scanner.Text()) > 0 {
		cfg.Topic = scanner.Text()
	}
	fmt.Printf("Using URL %s\n", cfg.URI())
	// Configure HTTP Basic Auth for push
	// This needs to be input each time the auth flow is done,
	// existing values are reset
	cfg.User = ""
	cfg.Password = ""
	fmt.Println("Configuring HTTP basic authentication for push endpoint.")
	fmt.Println("Previously set username and password have been cleared.")
	fmt.Println("Leave values blank to disable basic authentication.")
	scanner = bufio.NewScanner(os.Stdin)
	fmt.Printf("Username: ")
	scanner.Scan()
	if len(scanner.Text()) > 0 {
		cfg.User = scanner.Text()
	}
	fmt.Printf("Password: ")
	pwBytes, err := terminal.ReadPassword(0)
	if err != nil {
		fmt.Printf("Error reading password: %v\n", err)
		return
	}
	if len(pwBytes) > 0 {
		// Store the password in base64 for a little obfuscation
		cfg.Password = base64.StdEncoding.EncodeToString(pwBytes)
	}
	// Save bridge password
	if len(cfg.BridgePw) == 0 {
		err := LoginBridge(cfg)
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}
	} else {
		fmt.Println("Bridge password is set")
	}
	// Save configuration
	err = cfg.Save()
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	fmt.Println("Notification configuration saved")
}
