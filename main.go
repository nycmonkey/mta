package main

import (
	"bytes"
	"encoding/hex"
	"flag"
	"io"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"

	"bitbucket.org/chrj/smtpd"
	"github.com/minio/sha256-simd"
)

var (
	tempDir  string
	dataRoot string
)

func init() {
	flag.StringVar(&tempDir, "temp", "tmp", "temporary directory for inbound messages")
	flag.StringVar(&dataRoot, "dataRoot", "data", "parent folder for saved messages")
}

func main() {
	var server *smtpd.Server
	err := os.MkdirAll(tempDir, 0755)
	if err != nil {
		log.Fatalln(err)
	}
	server = &smtpd.Server{
		// ProtocolLogger: log.New(os.Stdout, "", log.Ldate|log.Ltime),
		HeloChecker: func(peer smtpd.Peer, name string) error {
			return nil
		},

		Handler: func(peer smtpd.Peer, env smtpd.Envelope) error {
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
			// log.Printf("%s received from %s\n", dgst, peer.Addr.String())
			return nil
		},
	}
	err = server.ListenAndServe(":25")
	if err != nil {
		log.Fatalln(err)
	}
}
