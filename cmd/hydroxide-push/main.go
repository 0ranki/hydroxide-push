package main

import (
	"bufio"
	"crypto/tls"
	"flag"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/0ranki/hydroxide-push/auth"
	"github.com/0ranki/hydroxide-push/config"
	"github.com/0ranki/hydroxide-push/events"
	imapbackend "github.com/0ranki/hydroxide-push/imap"
	"github.com/0ranki/hydroxide-push/ntfy"
	"github.com/0ranki/hydroxide-push/protonmail"
	imapserver "github.com/emersion/go-imap/server"
	"golang.org/x/term"
)

const (
	defaultAPIEndpoint = "https://mail.proton.me/api"
	defaultAppVersion  = "Other"
)

var (
	debug       bool
	apiEndpoint string
	appVersion  string
	cfg         ntfy.NtfyConfig
)

func newClient() *protonmail.Client {
	return &protonmail.Client{
		RootURL:    apiEndpoint,
		AppVersion: appVersion,
		Debug:      debug,
	}
}

func askPass(prompt string) ([]byte, error) {
	f := os.Stdin
	if !term.IsTerminal(int(f.Fd())) {
		// This can happen if stdin is used for piping data
		// TODO: the following assumes Unix
		var err error
		if f, err = os.Open("/dev/tty"); err != nil {
			return nil, err
		}
		defer f.Close()
	}
	fmt.Fprintf(os.Stderr, "%v: ", prompt)
	b, err := term.ReadPassword(int(f.Fd()))
	if err == nil {
		fmt.Fprintf(os.Stderr, "\n")
	}
	return b, err
}

func listenEventsAndNotify(addr string, debug bool, authManager *auth.Manager, eventsManager *events.Manager, tlsConfig *tls.Config) {
	be := imapbackend.New(authManager, eventsManager)
	s := imapserver.New(be)
	s.Addr = addr
	s.AllowInsecureAuth = tlsConfig == nil
	s.TLSConfig = tlsConfig
	if debug {
		s.Debug = os.Stdout
	}
	ntfy.Login(&cfg, be)
	log.Println("Listening for events", s.Addr)
	for {
		time.Sleep(10 * time.Second)
	}
}

func authenticate(authCmd *flag.FlagSet) {
	var username string
	if os.Getenv("PROTON_ACCT") != "" {
		username = os.Getenv("PROTON_ACCT")
	} else {
		username = authCmd.Arg(0)
	}
	if username == "" {
		log.Fatal("usage: hydroxide auth <username>")
	}

	c := newClient()

	var a *protonmail.Auth
	/*if cachedAuth, ok := auths[username]; ok {
		var err error
		a, err = c.AuthRefresh(a)
		if err != nil {
			// TODO: handle expired token error
			log.Fatal(err)
		}
	}*/

	var loginPassword string
	if a == nil {
		if os.Getenv("PROTON_ACCT_PASSWORD") != "" {
			loginPassword = os.Getenv("PROTON_ACCT_PASSWORD")
		} else if pass, err := askPass("Password"); err != nil {
			log.Fatal(err)
		} else {
			loginPassword = string(pass)
		}

		authInfo, err := c.AuthInfo(username)
		if err != nil {
			log.Fatal(err)
		}

		a, err = c.Auth(username, loginPassword, authInfo)
		if err != nil {
			log.Fatal(err)
		}

		if a.TwoFactor.Enabled != 0 {
			if a.TwoFactor.TOTP != 1 {
				log.Fatal("Only TOTP is supported as a 2FA method")
			}

			scanner := bufio.NewScanner(os.Stdin)
			fmt.Printf("2FA TOTP code: ")
			scanner.Scan()
			code := scanner.Text()

			scope, err := c.AuthTOTP(code)
			if err != nil {
				log.Fatal(err)
			}
			a.Scope = scope
		}
	}

	var mailboxPassword string
	if a.PasswordMode == protonmail.PasswordSingle {
		mailboxPassword = loginPassword
	}
	if mailboxPassword == "" {
		prompt := "Password"
		if a.PasswordMode == protonmail.PasswordTwo {
			prompt = "Mailbox password"
		}
		if pass, err := askPass(prompt); err != nil {
			log.Fatal(err)
		} else {
			mailboxPassword = string(pass)
		}
	}

	keySalts, err := c.ListKeySalts()
	if err != nil {
		log.Fatal(err)
	}

	_, err = c.Unlock(a, keySalts, mailboxPassword)
	if err != nil {
		log.Fatal(err)
	}

	secretKey, bridgePassword, err := auth.GeneratePassword()
	if err != nil {
		log.Fatal(err)
	}

	err = auth.EncryptAndSave(&auth.CachedAuth{
		Auth:            *a,
		LoginPassword:   loginPassword,
		MailboxPassword: mailboxPassword,
		KeySalts:        keySalts,
	}, username, secretKey)
	if err != nil {
		log.Fatal(err)
	}
	cfg.BridgePw = bridgePassword
	cfg.Setup()
}

const usage = `usage: hydroxide-push [options...] <command>
Commands:
	auth <username>		Login to ProtonMail via hydroxide
	status				View hydroxide status
	notify				Start the notification daemon
	setup-ntfy          (Re)configure the push endpoint

Global options:
	-debug
		Enable debug logs
	-api-endpoint <url>
		ProtonMail API endpoint
	-app-version <version>
		ProtonMail application version

Environment variables:
	HYDROXIDE_BRIDGE_PASS	Don't prompt for the bridge password, use this variable instead
`

func main() {
	flag.BoolVar(&debug, "debug", false, "Enable debug logs")
	flag.StringVar(&apiEndpoint, "api-endpoint", defaultAPIEndpoint, "ProtonMail API endpoint")
	flag.StringVar(&appVersion, "app-version", defaultAppVersion, "ProtonMail app version")

	tlsCert := flag.String("tls-cert", "", "Path to the certificate to use for incoming connections")
	tlsCertKey := flag.String("tls-key", "", "Path to the certificate key to use for incoming connections")
	tlsClientCA := flag.String("tls-client-ca", "", "If set, clients must provide a certificate signed by the given CA")

	authCmd := flag.NewFlagSet("auth", flag.ExitOnError)

	flag.Usage = func() {
		fmt.Print(usage)
	}

	flag.Parse()

	tlsConfig, err := config.TLS(*tlsCert, *tlsCertKey, *tlsClientCA)
	if err != nil {
		log.Fatal(err)
	}

	err = cfg.Read()
	if err != nil {
		fmt.Println(err)
	}

	cmd := flag.Arg(0)
	switch cmd {
	case "auth":
		authCmd.Parse(flag.Args()[1:])
		authenticate(authCmd)

	case "status":
		usernames, err := auth.ListUsernames()
		if err != nil {
			log.Fatal(err)
		}

		if len(usernames) == 0 {
			fmt.Printf("No logged in user.\n")
		} else {
			fmt.Printf("%v logged in user(s):\n", len(usernames))
			for _, u := range usernames {
				fmt.Printf("- %v\n", u)
			}
		}

	case "setup-ntfy":
		cfg.Setup()

	case "notify":
		if os.Getenv("PROTON_ACCT_PASSWORD") != "" && os.Getenv("PROTON_ACCT") != "" && os.Getenv("PUSH_URL") != "" && os.Getenv("PUSH_TOPIC") != "" && cfg.BridgePw == "" {
			log.Println("Logging in to Proton account using values from environment")
			cfg.URL = os.Getenv("PUSH_URL")
			cfg.Topic = os.Getenv("PUSH_TOPIC")
			cfg.Save()
			authenticate(new(flag.FlagSet))
		}
		authManager := auth.NewManager(newClient)
		eventsManager := events.NewManager()
		listenEventsAndNotify("0", debug, authManager, eventsManager, tlsConfig)

	default:
		fmt.Print(usage)
		if cmd != "help" {
			log.Fatal("Unrecognized command")
		}
	}
}
