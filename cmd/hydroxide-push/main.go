package main

import (
	"bufio"
	"crypto/tls"
	"flag"
	"fmt"
	"github.com/emersion/go-imap"
	imapserver "github.com/emersion/go-imap/server"
	"github.com/emersion/hydroxide/auth"
	"github.com/emersion/hydroxide/config"
	"github.com/emersion/hydroxide/events"
	imapbackend "github.com/emersion/hydroxide/imap"
	"github.com/emersion/hydroxide/protonmail"
	"golang.org/x/term"
	"log"
	"net"
	"os"
	"time"
)

const (
	defaultAPIEndpoint = "https://mail.proton.me/api"
	defaultAppVersion  = "Other"
)

var (
	debug       bool
	apiEndpoint string
	appVersion  string

	//imapUser  *backend.User
	ntfyTopic string
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

func listenAndServeIMAP(addr string, debug bool, authManager *auth.Manager, eventsManager *events.Manager, tlsConfig *tls.Config) error {
	be := imapbackend.New(authManager, eventsManager)
	s := imapserver.New(be)
	s.Addr = addr
	s.AllowInsecureAuth = tlsConfig == nil
	s.TLSConfig = tlsConfig
	if debug {
		s.Debug = os.Stdout
	}

	if s.TLSConfig != nil {
		log.Println("IMAP server listening with TLS on", s.Addr)
		return s.ListenAndServeTLS()
	}
	go func() {
		time.Sleep(1 * time.Second)
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
		if len(usernames) == 0 {
			executable, _ := os.Executable()
			log.Fatal("login first using " + executable + " auth <protonmail username>")
		}
		// TODO: bridge password
		_, err = be.Login(&conn, usernames[0], os.Getenv("HYDROXIDE_BRIDGE_PASS"))
		if err != nil {
			log.Fatal(err)
		}
	}()

	log.Println("IMAP server listening on", s.Addr)
	return s.ListenAndServe()
}

const usage = `usage: hydroxide [options...] <command>
Commands:
	auth <username>		Login to ProtonMail via hydroxide
	carddav			Run hydroxide as a CardDAV server
	export-secret-keys <username> Export secret keys
	imap			Run hydroxide as an IMAP server
	import-messages <username> [file]	Import messages
	export-messages [options...] <username>	Export messages
	sendmail <username> -- <args...>	sendmail(1) interface
	serve			Run all servers
	smtp			Run hydroxide as an SMTP server
	status			View hydroxide status

Global options:
	-debug
		Enable debug logs
	-api-endpoint <url>
		ProtonMail API endpoint
	-app-version <version>
		ProtonMail application version
	-smtp-host example.com
		Allowed SMTP email hostname on which hydroxide listens, defaults to 127.0.0.1
	-imap-host example.com
		Allowed IMAP email hostname on which hydroxide listens, defaults to 127.0.0.1
	-carddav-host example.com
		Allowed SMTP email hostname on which hydroxide listens, defaults to 127.0.0.1
	-smtp-port example.com
		SMTP port on which hydroxide listens, defaults to 1025
	-imap-port example.com
		IMAP port on which hydroxide listens, defaults to 1143
	-carddav-port example.com
		CardDAV port on which hydroxide listens, defaults to 8080
	-disable-imap
		Disable IMAP for hydroxide serve
	-disable-smtp
		Disable SMTP for hydroxide serve
	-disable-carddav
		Disable CardDAV for hydroxide serve
	-tls-cert /path/to/cert.pem
		Path to the certificate to use for incoming connections (Optional)
	-tls-key /path/to/key.pem
		Path to the certificate key to use for incoming connections (Optional)
	-tls-client-ca /path/to/ca.pem
		If set, clients must provide a certificate signed by the given CA (Optional)

Environment variables:
	HYDROXIDE_BRIDGE_PASS	Don't prompt for the bridge password, use this variable instead
`

func main() {
	flag.BoolVar(&debug, "debug", false, "Enable debug logs")
	flag.StringVar(&apiEndpoint, "api-endpoint", defaultAPIEndpoint, "ProtonMail API endpoint")
	flag.StringVar(&appVersion, "app-version", defaultAppVersion, "ProtonMail app version")
	flag.StringVar(&ntfyTopic, "topic", "", "ntfy.sh/NextPush topic to push notifications to")

	imapHost := "127.0.0.1" // flag.String("imap-host", "127.0.0.1", "Allowed IMAP email hostname on which hydroxide listens, defaults to 127.0.0.1")
	imapPort := "1143"      // flag.String("imap-port", "1143", "IMAP port on which hydroxide listens, defaults to 1143")

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

	cmd := flag.Arg(0)
	switch cmd {
	case "auth":
		authCmd.Parse(flag.Args()[1:])
		username := authCmd.Arg(0)
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
			if pass, err := askPass("Password"); err != nil {
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

		fmt.Println("Bridge password:", bridgePassword)
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

	case "notify":
		if ntfyTopic == "" {
			log.Fatal("please set ntfy.sh topic using --topic")
		}
		addr := imapHost + ":" + imapPort
		authManager := auth.NewManager(newClient)
		eventsManager := events.NewManager()

		log.Fatal(listenAndServeIMAP(addr, debug, authManager, eventsManager, tlsConfig))

	default:
		fmt.Print(usage)
		if cmd != "help" {
			log.Fatal("Unrecognized command")
		}
	}
}
