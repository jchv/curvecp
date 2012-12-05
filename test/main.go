package main

import (
	"crypto/rand"
	"encoding/hex"
	"io/ioutil"
	"log"
	"os"

	"code.google.com/p/curvecp"
	"code.google.com/p/go.crypto/nacl/box"
)

func main() {
	log.Println("Firing up")
	pub, priv, err := box.GenerateKey(rand.Reader)
	if err != nil {
		log.Fatalln(err)
	}

	log.Println("Writing key")
	if err = ioutil.WriteFile("server.pk", []byte(hex.EncodeToString(pub[:])), os.FileMode(0666)); err != nil {
		log.Fatalln(err)
	}

	log.Println("Starting listen")
	_, err = curvecp.Listen("127.0.0.1:4242", priv[:])
	if err != nil {
		log.Fatalln(err)
	}

	log.Println("Loop forever")
	select {
	}
}
