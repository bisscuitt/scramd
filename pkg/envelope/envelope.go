package envelope

import (
	"github.com/bisscuitt/go-smtpd/smtpd"
	"github.com/bisscuitt/scramd/pkg/smtp_client"
)

type Envelope struct {
	*smtpd.BasicEnvelope
	From           smtpd.MailAddress
	Rcpt           smtpd.MailAddress
	ForwardTo      string
	Connection     smtpd.Connection
	ClientHostname string
	ServerAddr     string
	MessageID      string
	ForwardClient  *smtp_client.ForwardClient
	DataChannel    chan string
	SyncChannel    chan bool
}
