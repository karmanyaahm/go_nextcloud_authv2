package main

import (
	"context"
	"log"
	"os"

	auth "k.malhotra.cc/go/nextcloud_authv2"
)

func main() {
	if len(os.Args) < 2 {
		log.Fatalln("The first argument should be your server address")
	}
	//you can save the returned values somewhere
	log.Println(auth.Authenticate(context.TODO(), os.Args[1], "Golang_Example_Nextcloud_login/1.0", os.Stdout, os.Stdin))
}
