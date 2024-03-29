# Scramd - Synchronous Concurrent Relay and Mailer Daemon

A Simple MTA for forwarding mail that uses synchronous channels to relay mail in realtime.

## Features:
 * Simple configuration in a single file
 * Forwards instantly, no need for queueing

TODO
 * [ ] Stats Interface
 * [ ] TLS Support
 * [ ] Blacklist Support
 * [ ] RBL
 * [ ] Greylisting
 * [ ] DB / key-value store lookups
 * [ ] RFC Compliant

## Why does this even exist ?
I had a requirement for a simple MTA that forwards email for multiple addresses to another address. Managing a full MTA can take a large amount of work and dealing with mail queues can be painful. This is mainly intended for personal use.

## Configuration Reference
All configuration is stored in `/etc/scramd.yaml`

See `example_config.yaml` for more inspiration

### forwards
The forwarding map is configured in YAML as a list of addresses bound to a forwarding address:
```
forwards:
    target:
     - address1
     - address2
     ...
```

Example configuration:
```
---
forwards:
  mailbox1@my.isp:
    - alias@other.dom
    - other.name@mydom.com
  realuser@other.isp:
    - first.last@vanity.domain
    - admin@vanity.domain
```

### hostname
The identifier sent in the SMTP connection banner. (Defaults to the system hostname)

### listen_address
The interface address to listen for SMTP requests. (Defaults to 0.0.0.0)

```
listen_address: 127.0.0.1
```

### listen_port
The TCP Port to listen for SMTP requests. (Defaults to 25)

```
listen_port: 25
```

### server_timeouts
Set timeouts for the receiving server (the one running at `listen_port`)

```
server_timeouts:
    read: 120s
    write: 90s
```

### client_timeouts
Set timeouts when we are sending mail to remote upstream servers

```
client_timeouts:
    connect: 90s
    read:    30s
    wite:    30s
```

# Contribute
Contributions are very welcome! Feel free to fork and raise a PR aginst this repo

# Credits
This is mostly built wtih the help of go-smtpd: https://github.com/bradfitz/go-smtpd

# Limitations
 * Remote errors cause a connection close with no message after DATA stream. This may cause issues with upstream hosts when the connection is closed unexpectedly.

# Licence
MIT © [Ian Bissett](https://github.com/bisscuitt)

