package main

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"os/signal"
	"time"

	_ "net/http/pprof"

	"github.com/gomods/athens/cmd/proxy/actions"
	"github.com/gomods/athens/pkg/build"
	"github.com/gomods/athens/pkg/config"
)

var (
	configFile = flag.String("config_file", "", "The path to the config file")
	version    = flag.Bool("version", false, "Print version information and exit")
)

func main() {
	flag.Parse()
	if *version {
		fmt.Println(build.String())
		os.Exit(0)
	}
	conf, err := config.Load(*configFile)
	if err != nil {
		log.Fatalf("could not load config file: %v", err)
	}

	handler, err := actions.App(conf)
	if err != nil {
		log.Fatal(err)
	}

	cert, key, mtlsCert, err := conf.TLSCertFiles()
	if err != nil {
		log.Fatal(err)
	}
	var srv *http.Server

	if mtlsCert != "" {
		clientCACert, err := ioutil.ReadFile(conf.TLSCACertFile)
		if err != nil {
			log.Fatal(err)
		}
		clientCertPool := x509.NewCertPool()
		clientCertPool.AppendCertsFromPEM(clientCACert)
		tlsConfig := &tls.Config{
			ClientAuth:               tls.RequireAndVerifyClientCert,
			ClientCAs:                clientCertPool,
			PreferServerCipherSuites: true,
			MinVersion:               tls.VersionTLS12,
		}
		srv = &http.Server{
			Addr:      conf.Port,
			Handler:   handler,
			TLSConfig: tlsConfig,
		}
	} else {
		srv = &http.Server{
			Addr:    conf.Port,
			Handler: handler,
		}
	}

	idleConnsClosed := make(chan struct{})

	go func() {
		sigint := make(chan os.Signal, 1)
		signal.Notify(sigint, os.Interrupt)
		<-sigint

		// We received an interrupt signal, shut down.
		ctx, cancel := context.WithTimeout(context.Background(), time.Second*10)
		defer cancel()
		if err := srv.Shutdown(ctx); err != nil {
			log.Fatal(err)
		}
		close(idleConnsClosed)
	}()

	if conf.EnablePprof {
		go func() {
			// pprof to be exposed on a different port than the application for security matters, not to expose profiling data and avoid DoS attacks (profiling slows down the service)
			// https://www.farsightsecurity.com/txt-record/2016/10/28/cmikk-go-remote-profiling/
			log.Printf("Starting `pprof` at port %v", conf.PprofPort)
			log.Fatal(http.ListenAndServe(conf.PprofPort, nil))
		}()
	}

	log.Printf("Starting application at port %v", conf.Port)
	if cert != "" && key != "" {
		err = srv.ListenAndServeTLS(conf.TLSCertFile, conf.TLSKeyFile)
	} else {
		err = srv.ListenAndServe()
	}

	if err != http.ErrServerClosed {
		log.Fatal(err)
	}

	<-idleConnsClosed
}
