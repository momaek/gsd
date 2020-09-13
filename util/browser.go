package util

import (
	"fmt"
	"log"
	"strings"

	"github.com/pkg/browser"
)

// OpenBrowser open browser
func OpenBrowser(url string) (err error) {

	slice := strings.Split(url, ":")

	switch len(slice) {
	case 1:
		return
	case 2:
		host := slice[0]
		port := slice[1]
		if len(host) == 0 {
			host = "localhost"
		}
		url = fmt.Sprintf("http://%s:%s", host, port)
	}

	log.Println("opening url", url)
	err = browser.OpenURL(url)
	return
}
