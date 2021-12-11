package main

import (
	"flag"
	"fmt"
	"fredwangwang/go-winevent"
	"os"
	"syscall"
	"time"

	"golang.org/x/sys/windows"
)

const ns string = "test"
const evt1 string = "evt1"
const evt2 string = "evt2"
const evtExit string = "evtExit"

func send(pid int, event string) {
	err := winevent.SetEvent(ns, event, pid)
	if err != nil {
		if err == windows.ERROR_FILE_NOT_FOUND {
			fmt.Printf("server is not listenting to %s\n", event)
			os.Exit(1)
		}
		panic(err)
	}
}

func run() {
	var events = []string{evt1, evt2, evtExit}

	os.Getpid()
	fmt.Println("pid:", syscall.Getpid())
	fmt.Printf("try: send -pid %d evt1\n", syscall.Getpid())
	fmt.Printf("supported events: %v\n\n", events)

	winevent := winevent.NewWinEvent(ns)

	evt1Handler := func() { fmt.Printf("%s triggered\n", evt1) }
	evt2Handler := func() { fmt.Printf("%s triggered\n", evt2) }
	evtExitHandler := func() {
		fmt.Printf("%s triggered, stopping event loop\n", evtExit)
		winevent.Stop()
	}

	mustRegister(winevent, evt1, evt1Handler)
	mustRegister(winevent, evt2, evt2Handler)
	mustRegister(winevent, evtExit, evtExitHandler)

	go func() {
		var seconds time.Duration = 120
		fmt.Printf("self shutdown in %ds\n", seconds)
		time.Sleep(seconds * time.Second)
		fmt.Println("timer's up")
		winevent.Stop()
	}()

	winevent.Start()

	fmt.Println("bye!")
}

func main() {
	sendCmd := flag.NewFlagSet("send", flag.ExitOnError)
	sendPid := sendCmd.Int("pid", 0, "pid to send signal")

	if len(os.Args) < 2 {
		fmt.Println("expected 'run' or 'send' subcommands")
		os.Exit(1)
	}

	switch os.Args[1] {
	case "run":
		run()
	case "send":
		sendCmd.Parse(os.Args[2:])
		if *sendPid == 0 {
			fmt.Println("-pid requred")
			os.Exit(1)
		}
		if sendCmd.NArg() != 1 {
			fmt.Printf("usage: %s send -pid 2333 event\n", os.Args[0])
			os.Exit(1)
		}
		send(*sendPid, sendCmd.Arg(0))
	default:
		fmt.Println("expected 'run' or 'send' subcommands")
		os.Exit(1)
	}
}

func mustRegister(w *winevent.WinEvent, evt string, hdlr winevent.EventHandler) {
	if err := w.Register(evt, hdlr); err != nil {
		panic(err)
	}
}
