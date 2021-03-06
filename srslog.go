package srslog

import (
	"crypto/tls"
	"crypto/x509"
	"io/ioutil"
	"log"
	"os"
)

// This interface and the separate syslog_unix.go file exist for
// Solaris support as implemented by gccgo.  On Solaris you can not
// simply open a TCP connection to the syslog daemon.  The gccgo
// sources have a syslog_solaris.go file that implements unixSyslog to
// return a type that satisfies this interface and simply calls the C
// library syslog function.
type serverConn interface {
	writeString(p priority, hostname, tag, s string) error
	close() error
}

// New establishes a new connection to the system log daemon.  Each
// write to the returned writer sends a log message with the given
// priority and prefix.
func New(priority priority, tag string) (w *writer, err error) {
	return Dial("", "", priority, tag)
}

// Dial establishes a connection to a log daemon by connecting to
// address raddr on the specified network.  Each write to the returned
// writer sends a log message with the given facility, severity and
// tag.
// If network is empty, Dial will connect to the local syslog server.
func Dial(network, raddr string, priority priority, tag string) (*writer, error) {
	return DialWithTLSConfig(network, raddr, priority, tag, nil)
}

func DialWithTLSCertPath(network, raddr string, priority priority, tag, certPath string) (*writer, error) {
	pool := x509.NewCertPool()
	serverCert, err := ioutil.ReadFile(certPath)
	if err != nil {
		return nil, err
	}
	pool.AppendCertsFromPEM(serverCert)
	config := tls.Config{
		RootCAs: pool,
	}

	return DialWithTLSConfig(network, raddr, priority, tag, &config)
}

func DialWithTLSConfig(network, raddr string, priority priority, tag string, tlsConfig *tls.Config) (*writer, error) {
	if err := validatePriority(priority); err != nil {
		return nil, err
	}

	if tag == "" {
		tag = os.Args[0]
	}
	hostname, _ := os.Hostname()

	w := &writer{
		priority:  priority,
		tag:       tag,
		hostname:  hostname,
		network:   network,
		raddr:     raddr,
		tlsConfig: tlsConfig,
	}

	w.Lock()
	defer w.Unlock()

	err := w.connect()
	if err != nil {
		return nil, err
	}
	return w, err
}

// NewLogger creates a log.Logger whose output is written to
// the system log service with the specified priority. The logFlag
// argument is the flag set passed through to log.New to create
// the Logger.
func NewLogger(p priority, logFlag int) (*log.Logger, error) {
	s, err := New(p, "")
	if err != nil {
		return nil, err
	}
	return log.New(s, "", logFlag), nil
}
