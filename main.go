/*
kubaru is a simple media file server.
It's designed for the sharing of media to friends and family at full quality, with media players being the indended clients.
This offers an alternative from highly compressed or resolution limited screen sharing solutions,
and is simpler and more lightweight that full featured media servers.

kubaru runs over HTTPS, enforces HTTP Basic authentication, and can generate a self-signed certificate for convenience.
*/
package main

import (
	"flag"
	"fmt"
	"log"
	"os"

	"github.com/pydronia/kubaru/internal/config"
	"github.com/pydronia/kubaru/internal/server"
	"github.com/pydronia/kubaru/internal/tlsutils"
)

// TODO:
// - Nice logging/bandwidth monitoring (colors!)
// - Rate limiting for authentication
// - Support changing directory contents
// - Make simple homepage with instructions (maybe)
// - support custom cert location
// - Playback syncing service (write vlc, mpv plugin)

func main() {
	// Flags
	// main flags
	var path, host, port, user, pass string
	flag.StringVar(&path, "path", "", "Path to directory to serve (required)")
	flag.StringVar(&host, "host", "::", "Host (interface) address to listen on")
	flag.StringVar(&port, "port", "443", "Port to listen on")
	flag.StringVar(&user, "user", "", "Username for basic auth. Also can be set with the KUBARU_USER environment variable. Defaults to \"user\".")
	flag.StringVar(&pass, "pass", "", "Password for basic auth. Also can be set with the KUBARU_PASS environment variable. Generate random password by default")

	// gen-cert flags
	var hosts string
	genCertFlags := flag.NewFlagSet("gen-cert", flag.ExitOnError)
	genCertFlags.StringVar(&hosts, "hosts", "localhost,127.0.0.1,::1", "Comma separated list of hosts to add to TLS certificate.")

	// Usage function
	flag.Usage = func() {
		fmt.Println("Usage for kubaru:")
		fmt.Println("  kubaru [flags]")
		fmt.Println("  kubaru gen-cert [flags]")
		fmt.Println("\nA simple https fileserver designed for easily sharing media files.")
		fmt.Println("Generate a self-signed TLS certificate with gen-cert, and start the server with `kubaru [flags]`.")
		fmt.Println("\nFlags:")
		flag.PrintDefaults()
		fmt.Println("\ngen-cert flags:")
		genCertFlags.PrintDefaults()
	}

	// gen-cert subcommand
	if len(os.Args) >= 2 && os.Args[1] == "gen-cert" {
		genCertFlags.Parse(os.Args[2:])
		if err := tlsutils.GenerateTLSCert(hosts); err != nil {
			log.Fatalln(err)
		}
		return
	}

	flag.Parse()

	kubaruCfg, err := config.NewConfig(user, pass, host, port, path)
	if err != nil {
		log.Fatalln(err)
	}
	log.Fatalln(server.StartServer(&kubaruCfg))
}
