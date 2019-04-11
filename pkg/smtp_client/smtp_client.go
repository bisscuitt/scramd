package smtp_client

import (
	"errors"
	"fmt"
	"io"
	"log"
	"net"
	"net/smtp"
	"strings"
	"time"

	"github.com/bisscuitt/scramd/pkg/config"
	"github.com/bisscuitt/scramd/pkg/dns"
)

type ForwardClient struct {
	sess   *smtp.Client
	conn   net.Conn
	from   string
	rcpt   string
	writer io.WriteCloser
}

var conf *config.Config

func Start(config_in *config.Config, f string, r string) (*ForwardClient, error) {
	// FIXME: This is really dumb... make it better
	conf = config_in

	// DNS Lookup to get MX Records
	d := r[strings.IndexByte(r, '@')+1:]  // Get the domain
	mx_hosts, err := dns.LookupMXHosts(d) // Request the MX records

	// TODO: server connection may time out before this client completes connection to remote servers

	c := &ForwardClient{
		from: f,
		rcpt: r,
	}

	// Try each MX record in order until we make a TCP connection
	for _, h := range mx_hosts {
		hp := net.JoinHostPort(h, "25")
		log.Printf("forwardConnectionStart(): attempting connection to %v (timeout: %v)", hp, conf.Timeouts.ClientConn)

		conn, err := net.DialTimeout("tcp", hp, conf.Timeouts.ClientConn)
		c.conn = conn
		if err != nil {
			log.Printf("forwardConnectionStart(): Failed to connect to %v: %v", hp, err)
			continue
		}

		// update the timeouts
		c.resetDeadlines()

		c.sess, err = smtp.NewClient(conn, hp)
		if err != nil {
			log.Printf("forwardConnectionStart(): Failed to setup SMTP session: %v", err)
		}

		log.Printf("forwardConnectionStart(): Sucessfuly connected to: %v", hp)

		break
	}

	if c.conn == nil {
		err = errors.New("Could not connect to any upstream servers")
		return nil, c.handleClientError("Could not forward mail to "+d, err)
	}

	// TODO: Start TLS if the server supports it and we have TLS config
	if ok, _ := c.sess.Extension("STARTTLS"); ok == true {
		// Need to setup with `tls.Config`
		//c.sess.StartTLS()
	}

	// Send on the `FROM` address
	err = c.sess.Mail(f)
	if err != nil {
		return nil, c.handleClientError("Failed to send Mail FROM", err)
	}
	c.resetDeadlines()

	// TODO: May be multiple rcpts
	// Send the `RCPT TO` to the forwarding address
	err = c.sess.Rcpt(r)
	if err != nil {
		return nil, c.handleClientError("Failed to send RCPT TO", err)
	}
	c.resetDeadlines()

	return c, nil
}

func (c *ForwardClient) DataEnd(sc chan bool, dc chan string) error {

	// Update timeouts for client
	c.resetDeadlines()

	// Close the data channel
	close(dc)

	// Wait for the `forwardDataStart` loop to complete
	log.Println("DataEnd() Waiting for SyncChannel to complete")
	ok := <-sc
	close(sc)
	if ok != true {
		return fmt.Errorf("Failed to complete data transfer")
	}

	// TODO: Before we complete the data transfer,
	// check that the inital connection from our client is still alive..
	// If not, bomb out here to avoid duplicate deliveries of mail (QUIT ForwardClient, Close Connection and Log)
	// We may need to expose the TCP session from go-smptd package for this to be a reality

	// TODO: Error check the response
	log.Println("DataEnd() returning success")
	return nil
}

// func DataStart creates channels for sending data to the smtp client
func (c *ForwardClient) DataStart(sc chan bool, dc chan string) error {
	log.Printf("forwardData() Seting up channel and waiting for data")

	var err error
	c.writer, err = c.sess.Data()
	if err != nil {
		// If we can't get the data session going log an error and
		// let those waiting for us that things went bad
		err = c.handleClientError("Could not start DATA transmission", err)
		sc <- false
		return err
	}

	// Tell others we are ready to go
	sc <- true

	for v := range dc {
		// TODO: Figure out how to flush this immediately
		// TODO: Error Checking ??
		// Otherwise it buffers until we close it
		log.Printf("On Channel: %s", v)
		_, err = io.WriteString(c.writer, v)
		if err != nil {
			log.Printf("Error writing to client: %v", err)
		}

		// Update the timeouts
		c.resetDeadlines()
	}

	// Update timeouts for client
	c.resetDeadlines()

	// Complete the Data transfer
	log.Println("Closing connection to 127.0.0.1:4444")
	err = c.writer.Close()
	if err != nil {
		log.Printf("ERROR: Failed to Close data session: %s", err)
		sc <- false
		return nil
	}
	log.Println("Data transfer to client complete!")

	// Close the connection to the remote server
	err = c.sess.Quit()
	if err != nil {
		log.Printf("WARNING: Failed to QUIT SMTP session: %s", err)
	}

	// Tell others that we have finished
	sc <- true

	return nil
}

func (c *ForwardClient) resetDeadlines() error {
	r := errors.New("Failed to update deadlines")

	err := c.conn.SetReadDeadline(time.Now().Add(conf.Timeouts.ClientRx))
	if err != nil {
		log.Printf("Error updating client read deadline: %s", err)
		return r
	}

	err = c.conn.SetWriteDeadline(time.Now().Add(conf.Timeouts.ClientWr))
	if err != nil {
		log.Printf("Error updating client write deadline: %s", err)
		return r
	}

	return nil
}

func (c *ForwardClient) handleClientError(msg string, err error) error {
	c.conn.Close()
	eout := fmt.Errorf("%v: %v", msg, err)
	log.Println(eout)
	return eout
}
