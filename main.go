package main

import (
	"bufio"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/exec"
	"regexp"
	"strings"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

var totalCounterVec = prometheus.NewCounterVec(
	prometheus.CounterOpts{
		Namespace: "worker",
		Subsystem: "logs",
		Name:      "groups_by_user",
		Help:      "number groups by user",
	},
	[]string{"service", "user"},
)

func reg(regexpTemplate string, text string) []string {
	return regexp.MustCompile(regexpTemplate).FindStringSubmatch(text)
}

func myParce(line string) {

	// Init new variables.
	var (
		regexpDate      = `^([A-Z][a-z]* [0-9]* [0-9]*:[0-9]*:[0-9]*)\s`
		regexHostname   = `([^\s]*)\s`
		regexGroupEvent = `([^\s]*):\s`
		regexAll        = regexpDate + regexHostname + regexGroupEvent

		regexExecutingCommand = regexAll + `((.*):.*USER=([^\]]*).*COMMAND=([^\]]*))`
		regexPamUnix          = regexAll + `(.*\((.*):(.*)\)):\s(.*\sfor\suser\s([^\s]*)).*`
		regexAccess           = regexAll + `(.*password\sfor\s([^\s]*)\sfrom\s([0-9.]*)\sport\s([0-9]*).*)`
		regexAuth             = regexAll + `.*\(((.*):(.*))\):.*user=([^\s]*)`
		regexPidEvent         = `([a-zA-Z-_0-9]*)\[([0-9]*)\].*`

		pamUnix    []string
		logLine    string
		GroupEvent string
		Username   string
	)

	// Parcing new line.
	logLine = line
	pamUnix = reg(regexPamUnix, logLine)

	if pamUnix == nil {
		executingCommand := reg(regexExecutingCommand, logLine)

		if executingCommand == nil {
			access := reg(regexAccess, logLine)

			if access == nil {
				auth := reg(regexAuth, logLine)

				if auth == nil {
					fmt.Println(logLine)

				} else {
					GroupEvent = auth[3]
					Username = auth[7]
				}

			} else {
				GroupEvent = access[3]
				Username = access[5]
			}

		} else {
			GroupEvent = executingCommand[3]
			Username = executingCommand[5]
		}
	} else {
		GroupEvent = pamUnix[3]
		Username = pamUnix[8]
	}

	// The parser is not perfect, we clear the variables from unnecessary
	Username = strings.TrimRight(strings.TrimLeft(Username, " "), " ")
	GroupEventTreatment := reg(regexPidEvent, GroupEvent)

	if GroupEventTreatment != nil {
		GroupEvent = GroupEventTreatment[1]
	}

	// Increase the value of the metric by 1
	totalCounterVec.WithLabelValues(GroupEvent, Username).Inc()
}

func tails() {
	cmd := exec.Command("tail", "-f", "/var/log/secure")

	// create a pipe for the output of the script
	cmdReader, err := cmd.StdoutPipe()
	if err != nil {
		fmt.Fprintln(os.Stderr, "Error creating StdoutPipe for Cmd", err)
		return
	}

	// Running myParse with new line from logfile.
	scanner := bufio.NewScanner(cmdReader)
	go func() {
		for scanner.Scan() {
			myParce(scanner.Text())
		}
	}()

	err = cmd.Start()
	if err != nil {
		fmt.Fprintln(os.Stderr, "Error starting Cmd", err)
		return
	}

	err = cmd.Wait()
	if err != nil {
		fmt.Fprintln(os.Stderr, "Error waiting for Cmd", err)
		return
	}
}
func main() {
	// Registered prometheus metrics
	prometheus.MustRegister(totalCounterVec)

	// Run tread with tails and myParce function.
	go func() {
		tails()
	}()

	// Launch http server to broadcast metrics
	flag.Parse()
	var addr = flag.String("127.0.0.1", ":8080",
		"The address to listen on for HTTP requests.")
	http.Handle("/metrics", promhttp.Handler())

	log.Printf("Starting web server at %s\n", *addr)
	err := http.ListenAndServe(*addr, nil)
	if err != nil {
		log.Printf("http.ListenAndServer: %v\n", err)
	}
}
