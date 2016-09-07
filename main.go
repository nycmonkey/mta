package main

import (
	"bytes"
	"crypto/sha256"
	"crypto/tls"
	"encoding/hex"
	"flag"
	"io"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"strconv"

	"bitbucket.org/chrj/smtpd"
)

var (
	tempDir  string
	dataRoot string
	port     int
)

func init() {
	flag.StringVar(&tempDir, "temp", "tmp", "temporary directory for inbound messages")
	flag.StringVar(&dataRoot, "dataRoot", "data", "parent folder for saved messages")
	flag.IntVar(&port, "p", 2525, "the port on which to listen")
}

func authenticate(peer smtpd.Peer, name string) error {
	// Receive email for any recipient
	return nil
}

func handleMessage(peer smtpd.Peer, env smtpd.Envelope) error {
	h := sha256.New()
	tf, err := ioutil.TempFile(tempDir, "inbound-email")
	if err != nil {
		return smtpd.Error{
			Code:    431,
			Message: "unable to write temp files",
		}
	}
	w := io.MultiWriter(h, tf)
	b := bytes.NewReader(env.Data)
	_, err = io.Copy(w, b)
	if err != nil {
		return smtpd.Error{
			Code:    431,
			Message: err.Error(),
		}
	}
	tf.Close()
	defer os.Remove(tf.Name())
	dgst := hex.EncodeToString(h.Sum(nil))
	dir := filepath.Join(dataRoot, dgst[0:1])
	err = os.MkdirAll(dir, 0755)
	if err != nil {
		return smtpd.Error{
			Code:    431,
			Message: err.Error(),
		}
	}
	err = os.Rename(tf.Name(), filepath.Join(dir, dgst+".eml"))
	if err != nil {
		return smtpd.Error{
			Code:    431,
			Message: err.Error(),
		}
	}
	os.Chmod(filepath.Join(dir, dgst+".eml"), 0755)
	log.Printf("%s received from %s\n", dgst, peer.Addr.String())
	return nil
}

func main() {
	flag.Parse()
	var server *smtpd.Server
	err := os.MkdirAll(tempDir, 0755)
	if err != nil {
		log.Fatalln(err)
	}
	cer, err := tls.LoadX509KeyPair("smtp.mnky.nyc.cer", "smtp.mnky.nyc.key")
	if err != nil {
		log.Fatalln(err)
	}
	config := &tls.Config{Certificates: []tls.Certificate{cer}}
	server = &smtpd.Server{
		ProtocolLogger: log.New(os.Stdout, "", log.Ldate|log.Ltime),
		Hostname:       "smtp.mnky.nyc",
		ForceTLS:       true,
		TLSConfig:      config,
		HeloChecker:    authenticate,
		Handler:        handleMessage,
	}
	err = server.ListenAndServe(":" + strconv.Itoa(port))
	if err != nil {
		log.Fatalln(err)
	}
}
