// Package server provides function for the scramd SMTP server
package smtp_server

import (
	"fmt"
	"log"
	"net"
	"strconv"
	"time"

	"github.com/bisscuitt/go-smtpd/smtpd"
	"github.com/kjk/betterguid"

	"github.com/bisscuitt/scramd/pkg/config"
	"github.com/bisscuitt/scramd/pkg/dns"
	"github.com/bisscuitt/scramd/pkg/envelope"
	"github.com/bisscuitt/scramd/pkg/smtp_client"
)

type Envelope envelope.Envelope

var server smtpd.Server
var conf *config.Config

func New(c *config.Config) {
	conf = c

	listen := net.JoinHostPort(conf.ListenAddr, strconv.Itoa(conf.ListenPort))
	server = smtpd.Server{
		Addr:            listen,
		OnNewMail:       onNewMail,
		OnNewConnection: onNewConnection,
		Hostname:        conf.Hostname,
		ReadTimeout:     conf.Timeouts.ServerRx,
		WriteTimeout:    conf.Timeouts.ServerWr,
		Software:        "Scramd - github.com/bisscuitt/scramd",
	}

	log.Printf("Starting SMTP server at %v (%v)", server.Addr, server.Hostname)

	err := server.ListenAndServe()
	if err != nil {
		log.Fatalln("SMTP Server Failed: ", err)
	}
}

func generateMessageID() string {
	id := "m" + betterguid.New()
	return id
}

func onNewConnection(c smtpd.Connection) error {
	log.Printf("Got a new connection on %v from: %v", c.LocalAddr(), c.Addr())
	return nil
}

func onNewMail(c smtpd.Connection, from smtpd.MailAddress) (smtpd.Envelope, error) {
	log.Println("Got a new mail:", from)

	// TODO: There must be a better way to do this
	e := &Envelope{
		new(smtpd.BasicEnvelope),
		from,
		nil,
		"",
		c,
		"unknown",
		"unknown",
		generateMessageID(),
		new(smtp_client.ForwardClient),
		nil,
		nil,
	}
	log.Printf("%T\n", e)

	lochost, _, err := net.SplitHostPort(c.LocalAddr().String())
	if err != nil {
		e.ServerAddr = lochost
	}

	// Lookup hostname of the IP of this new client
	var addr string
	remhost, _, _ := net.SplitHostPort(c.Addr().String())

	addr, err = dns.LookupRevRecord(remhost)
	if err != nil {
		log.Printf("Failed to complete reverse lookup of %v: %v", remhost, err)
		addr = "unknown"
	}

	e.ClientHostname = addr
	log.Printf("Client hostname: %v", addr)

	return e, nil
}

func (e *Envelope) generateReceivedHeader() string {
	return fmt.Sprintf("Received: from %v [%v]\n"+
		"        by %v [%v] id %v\n"+
		"        for <%v>\n"+
		"        %v\n",
		e.ClientHostname,
		e.Connection.Addr(),
		server.Hostname,
		e.ServerAddr,
		e.MessageID,
		e.Rcpt.Email(),
		time.Now().Format("Mon, 02 Jan 2006 15:04:05 -0700 (MST)"),
	)
}

func (e *Envelope) AddRecipient(rcpt smtpd.MailAddress) error {
	log.Println("Got Recipient:", rcpt.Email())
	e.Rcpt = rcpt

	// Deny relaying if we can't find anything in the forward map
	e.ForwardTo = conf.Forward[rcpt.Email()]
	if e.ForwardTo == "" {
		log.Printf("Relay denied to %v", rcpt)
		return smtpd.SMTPError("421 Relaying denied")
	}

	// Start the smtp connection to the remote server
	fc, err := smtp_client.Start(conf, e.From.Email(), e.ForwardTo)
	if err != nil {
		log.Printf("Error Making connection to remote mail server that handles %v", e.ForwardTo)
		return smtpd.SMTPError("441 remote server is currently unavailable. Try again later")
	}
	e.ForwardClient = fc

	return e.BasicEnvelope.AddRecipient(rcpt)
}

// This is when we have received the entire message
func (e *Envelope) Close() error {
	err := e.ForwardClient.DataEnd(e.SyncChannel, e.DataChannel)

	if err != nil {
		return smtpd.SMTPError("421 Remote server did not accept data")
	}

	return smtpd.SMTPError("250 Message relayed to remote server successfully")
}

// For each line received from the client, pass it to the remote mail server channel
func (e *Envelope) Write(line []byte) error {
	log.Println("GOT: ", string(line))
	e.DataChannel <- string(line)
	return nil
}

func (e *Envelope) BeginData() error {
	log.Println("Receiving Data")

	sc := make(chan bool)
	dc := make(chan string, 10)

	go e.ForwardClient.DataStart(sc, dc)

	// Wait until we have the data channel open to the remote server
	log.Printf("BeginData(): HERE1")
	ok := <-sc
	log.Printf("BeginData(): HERE2")
	if ok != true {
		return smtpd.SMTPError("441 Remote server did not accept data transmission. Try again later")
	}

	// Prepend a `Received:` Header before we send any other data
	dc <- e.generateReceivedHeader()

	e.SyncChannel = sc
	e.DataChannel = dc

	return nil
}
