package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/exec"
	"strings"

	"github.com/ponzu-cms/ponzu/system/admin"
	"github.com/ponzu-cms/ponzu/system/api"
	"github.com/ponzu-cms/ponzu/system/api/analytics"
	"github.com/ponzu-cms/ponzu/system/db"
	"github.com/ponzu-cms/ponzu/system/tls"

	// import registers content types
	_ "github.com/ponzu-cms/ponzu/content"
)

var (
	usage = usageHeader + usageNew + usageGenerate + usageBuild + usageRun
	port  int
	https bool

	// for ponzu internal / core development
	dev   bool
	fork  string
	gocmd string
)

func init() {
	flag.Usage = func() {
		fmt.Println(usage)
	}
}

func main() {
	flag.IntVar(&port, "port", 8080, "port for ponzu to bind its listener")
	flag.BoolVar(&https, "https", false, "enable automatic TLS/SSL certificate management")
	flag.BoolVar(&dev, "dev", false, "modify environment for Ponzu core development")
	flag.StringVar(&fork, "fork", "", "modify repo source for Ponzu core development")
	flag.StringVar(&gocmd, "gocmd", "go", "custom go command if using beta or new release of Go")
	flag.Parse()

	args := flag.Args()

	if len(args) < 1 {
		fmt.Println(usage)
		os.Exit(0)
	}

	switch args[0] {
	case "help", "h":
		if len(args) < 2 {
			fmt.Println(usageHelp)
			fmt.Println(usage)
			os.Exit(0)
		}

		switch args[1] {
		case "new":
			fmt.Println(usageNew)
			os.Exit(0)

		case "generate", "gen", "g":
			fmt.Println(usageGenerate)
			os.Exit(0)

		case "build":
			fmt.Println(usageBuild)
			os.Exit(0)

		case "run":
			fmt.Println(usageRun)
			os.Exit(0)
		}

	case "new":
		if len(args) < 2 {
			fmt.Println(usage)
			os.Exit(0)
		}

		err := newProjectInDir(args[1])
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}

	case "generate", "gen", "g":
		if len(args) < 2 {
			flag.PrintDefaults()
			os.Exit(0)
		}

		err := generateContentType(args[1:])
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}

	case "build":
		err := buildPonzuServer(args)
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}

	case "run":
		var addTLS string
		if https {
			addTLS = "--https"
		} else {
			addTLS = "--https=false"
		}

		var services string
		if len(args) > 1 {
			services = args[1]
		} else {
			services = "admin,api"
		}

		serve := exec.Command("./ponzu-server",
			fmt.Sprintf("--port=%d", port),
			addTLS,
			"serve",
			services,
		)
		serve.Stderr = os.Stderr
		serve.Stdout = os.Stdout

		err := serve.Start()
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}

		err = serve.Wait()
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}

	case "serve", "s":
		db.Init()
		defer db.Close()

		analytics.Init()
		defer analytics.Close()

		if len(args) > 1 {
			services := strings.Split(args[1], ",")

			for i := range services {
				if services[i] == "api" {
					api.Run()
				} else if services[i] == "admin" {
					admin.Run()
				} else {
					fmt.Println("To execute 'ponzu serve', you must specify which service to run.")
					fmt.Println("$ ponzu --help")
					os.Exit(1)
				}
			}
		}

		if https {
			fmt.Println("Enabling HTTPS...")
			tls.Enable()
		}

		// save the port the system is listening on so internal system can make
		// HTTP api calls while in dev or production w/o adding more cli flags
		err := db.PutConfig("http_port", fmt.Sprintf("%d", port))
		if err != nil {
			log.Fatalln("System failed to save config. Please try to run again.")
		}

		log.Fatalln(http.ListenAndServe(fmt.Sprintf(":%d", port), nil))

	case "":
		fmt.Println(usage)
		fmt.Println(usageHelp)

	default:
		fmt.Println(usage)
		fmt.Println(usageHelp)
	}
}
