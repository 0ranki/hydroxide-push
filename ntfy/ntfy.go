package ntfy

import (
	"net/http"
	"strings"
)

func Notify() {
	req, _ := http.NewRequest("POST", "https://push.oranki.net/testing20240325", strings.NewReader("New message received"))
	req.Header.Set("Title", "ProtoMail")
	req.Header.Set("Click", "dismiss")
	req.Header.Set("Tags", "envelope")
	http.DefaultClient.Do(req)
}
