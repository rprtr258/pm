package internal

import (
	"fmt"
	"html"
	"log"
	"net/http"
	"path"

	"github.com/sevlyar/go-daemon"
	"github.com/urfave/cli/v2"
)

var DaemonCmd = &cli.Command{
	Name:  "daemon",
	Usage: "run daemon",
	Action: func(*cli.Context) error {
		daemonCtx := &daemon.Context{
			PidFileName: path.Join(HomeDir, "pm.pid"),
			PidFilePerm: 0644,
			LogFileName: path.Join(HomeDir, "pm.log"),
			LogFilePerm: 0640,
			WorkDir:     "./",
			Umask:       027,
			Args:        []string{"pm", "daemon"},
		}
		d, err := daemonCtx.Reborn()
		if err != nil {
			return fmt.Errorf("unable to run: %w", err)
		}

		if d != nil {
			fmt.Println(d.Pid)
			return nil
		}

		defer daemonCtx.Release()

		log.Println("- - - - - - - - - - - - - - -")
		log.Println("daemon started")

		http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
			log.Printf("request from %s: %s %q", r.RemoteAddr, r.Method, r.URL)
			fmt.Fprintf(w, "go-daemon: %q", html.EscapeString(r.URL.Path))
		})
		http.ListenAndServe("127.0.0.1:8080", nil)

		return nil
	},
}
