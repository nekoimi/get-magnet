package main

import "log"

func init() {
	log.SetFlags(log.Lshortfile | log.Ldate | log.Lmicroseconds)
}

func main() {
}
