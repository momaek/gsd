package util

import (
	"fmt"
	"log"
	"os/exec"
	"runtime"
	"strings"
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

	switch runtime.GOOS {
	case "linux":
		err = exec.Command("xdg-open", url).Start()
	case "windows":
		err = exec.Command("rundll32", "url.dll,FileProtocolHandler", url).Start()
	case "darwin":
		err = exec.Command("open", url).Start()
	default:
		err = fmt.Errorf("unsupported platform")
	}

	return
}
