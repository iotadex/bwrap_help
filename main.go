package main

import (
	"bhelp/config"
	"bhelp/daemon"
	"bhelp/gl"
	"bhelp/model"
	"bhelp/server"
	"bhelp/tools"
	"fmt"
	"log"
	"os"

	"golang.org/x/term"
)

func main() {
	if os.Args[len(os.Args)-1] != "daemon" {
		input()
		os.Args = append(os.Args, "daemon")
	}
	daemon.Background("./out.log", true)

	config.Load(readRand())

	gl.CreateLogFiles()

	model.ConnectToMysql()

	fmt.Printf("Smpc Bridge Help %s is starting...\n", config.Version)

	go server.ListenTokens()

	daemon.WaitForKill()
}

func readRand() (string, [4]uint64) {
	data, err := os.ReadFile("rand.data")
	if err != nil {
		log.Panicf("read rand.data error. %v", err)
	}
	if err := os.WriteFile("rand.data", []byte("start the process successful! You are very great. Best to every one."), 0666); err != nil {
		log.Panicf("write rand.data error. %v", err)
	}
	os.Remove("rand.data")

	//generate seeds
	var seeds [4]uint64
	seeds[0] = tools.GenerateRandomSeed()
	seeds[1] = tools.GenerateRandomSeed()
	seeds[2] = tools.GenerateRandomSeed()
	seeds[3] = tools.GenerateRandomSeed()

	pwd := tools.GetEncryptString(string(data), seeds)
	return pwd, seeds
}

func input() {
	fmt.Printf("Input password \n:")
	pwd, err := term.ReadPassword(int(os.Stdin.Fd()))
	if err != nil {
		panic("read pwd error:" + err.Error())
	}
	if err := os.WriteFile("rand.data", []byte(pwd), 0666); err != nil {
		log.Panicf("write rand.data error. %v", err)
	}
}
