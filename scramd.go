package main

// TODO Next:
// X Read configuration in using `yaml` (Do we need cmdline options?)
// X Actually re-wite the `To:` Address
// D Do MX Record lookups with the `net` package (https://golang.org/pkg/net/#LookupMX)
// X Add Received header
//     X Generate an Mail ID per Message (should also be logged)
//     X Currently server IP is reported as listen address... needs to be IP that host client connected to
// i Split code up into logical packages
// - TLS Client Support
// - TLS Server Support							     | Need to fork go-smtpd and merge exsting PRs to add this functionality
// i Correct SMTP Responses for success and failure  | Fixed this call so it works.. just need to fix up all SMTPErrors
// - Daemonize or systemdify?
// - Commandline options
// i Add more error checking
// i Timeouts, Timeouts, Timeouts!
// - Tests
// - Fix up Logging (use standard log levels: CRIT,WARN,INFO,DEBUG)
// - Make sure we can never be an open relay!
// - Tidy up!
// - Publish initial release
// - Auto testing and package builds (travis?)

import (
	"log"

	"github.com/bisscuitt/scramd/pkg/config"
	"github.com/bisscuitt/scramd/pkg/smtp_server"
)

// TODO: Need to make path to config file portable
// Use buildtags? https://stackoverflow.com/questions/19847594/how-to-reliably-detect-os-platform-in-go
const CONFIG_FILE = "/etc/scramd.yaml"

var conf *config.Config

func init() {
	// TODO: Command line option for path to the config file
	// TODO: Setup logging here ?
	var err error
	conf, err = config.Read(CONFIG_FILE)

	if err != nil {
		log.Fatalf("ERROR: %v", err)
	}
}

func main() {
	smtp_server.New(conf)
}
