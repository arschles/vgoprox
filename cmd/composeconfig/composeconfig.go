package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"sync"

	"github.com/BurntSushi/toml"
	"github.com/gomods/athens/pkg/config"
)

var description = `
composeconfig writes a config file based on the default config and addresses 
obtained from "docker-compose port". It is intended for use with integration 
tests.

example usage:
  composeconfig -p athensdev -config-file config.test.toml
`

func main() {
	var targetFile string
	var dockerComposeProject string
	flag.StringVar(&targetFile, "config-file", "", "*required* config file to write to")
	flag.StringVar(&dockerComposeProject, "p", "", "docker-compose project")
	usage := flag.Usage
	flag.Usage = func() {
		fmt.Print(strings.TrimSpace(description), "\n\n")
		usage()
	}
	flag.Parse()
	if targetFile == "" || len(flag.Args()) != 0 {
		flag.Usage()
		os.Exit(2)
	}
	cfg := config.DefaultConfig()

	svcAddrs, err := getServiceAddresses(dockerComposeProject,
		"mysql:3306",
		"postgres:5432",
		"redis:6379",
		"jaeger:14268",
		"etcd0:2379",
		"etcd1:2379",
		"etcd2:2379",
		"redis-sentinel:26379",
		"minio:9000",
		"mongo:27017",
		"azurite:10000",
	)
	if err != nil {
		log.Fatalf(err.Error())
	}

	mysql := cfg.Index.MySQL
	mysql.Host, mysql.Port, err = getAddrHostAndPort(svcAddrs["mysql:3306"])
	if err != nil {
		log.Fatalf("error getting host ane port for %q", "mysql:3306")
	}

	postgres := cfg.Index.Postgres
	postgres.Password = "postgres"
	postgres.Host, postgres.Port, err = getAddrHostAndPort(svcAddrs["postgres:5432"])
	if err != nil {
		log.Fatalf("error getting host ane port for %q", "postgres:5432")
	}

	cfg.SingleFlight.Redis.Endpoint = svcAddrs["redis:6379"]

	cfg.TraceExporterURL = fmt.Sprintf("http://%s", svcAddrs["jaeger:14268"])

	cfg.SingleFlight.Etcd.Endpoints = fmt.Sprintf("%s,%s,%s", svcAddrs["etcd0:2379"], svcAddrs["etcd1:2379"], svcAddrs["etcd2:2379"])

	sentinel := cfg.SingleFlight.RedisSentinel
	sentinel.Endpoints = []string{svcAddrs["redis-sentinel:26379"]}
	sentinel.SentinelPassword = "sekret"
	sentinel.MasterName = "redis-1"

	cfg.Storage = new(config.Storage)
	cfg.Storage.Minio = &config.MinioConfig{
		Endpoint:  svcAddrs["minio:9000"],
		Key:       "minio",
		Secret:    "minio123",
		Bucket:    "gomods",
		Region:    "",
		EnableSSL: false,
	}

	cfg.Storage.Mongo = &config.MongoConfig{
		URL: fmt.Sprintf("mongodb://%s", svcAddrs["mongo:27017"]),
	}

	cfg.Storage.AzureBlob = &config.AzureBlobConfig{
		AccountName: "devstoreaccount1",
		AccountKey:  "Eby8vdM02xNOcqFlqUwJPLlmEtlCDXJ1OUzFT50uSRZ6IFsuFq2UVErCz4I6tq/K1SZFPTOtr/KBHBeksoGMGw==",
		PortalURL:   fmt.Sprintf("http://%s/devstoreaccount1", svcAddrs["azurite:10000"]),
	}

	file, err := os.Create(targetFile)
	if err != nil {
		log.Fatalf("error creating %s", targetFile)
	}
	defer func() {
		cErr := file.Close()
		if cErr != nil {
			log.Fatalf("error closing %s", targetFile)
		}
	}()

	err = toml.NewEncoder(file).Encode(&cfg)
	if err != nil {
		log.Fatalf("error writing toml to %s", targetFile)
	}
}

func getAddrHostAndPort(addr string) (string, int, error) {
	parts := strings.Split(addr, ":")
	if len(parts) != 2 {
		return "", 0, fmt.Errorf("invalid address %q", addr)
	}
	port, err := strconv.Atoi(parts[1])
	if err != nil {
		return "", 0, err
	}
	return parts[0], port, nil
}

func getServiceAddresses(dockerComposeProject string, services ...string) (map[string]string, error) {
	result := map[string]string{}
	var resultMux sync.Mutex
	var wg sync.WaitGroup
	var hasErr bool
	var hasErrMux sync.Mutex
	for i := range services {
		svc := services[i]
		wg.Add(1)
		go func() {
			svcParts := strings.Split(svc, ":")
			if len(svcParts) != 2 {
				panic(`getServiceAddresses services must be in the form of "servicename:containerport"`)
			}
			var cmdArgs []string
			if dockerComposeProject != "" {
				cmdArgs = append(cmdArgs, "-p", dockerComposeProject)
			}
			cmdArgs = append(cmdArgs, "port", svcParts[0], svcParts[1])
			out, err := exec.Command("docker-compose", cmdArgs...).Output()
			if err != nil {
				hasErrMux.Lock()
				hasErr = true
				hasErrMux.Unlock()
				log.Printf("error getting service port for %s: %v", svc, err)
			}
			resultMux.Lock()
			result[svc] = strings.TrimSpace(string(out))
			resultMux.Unlock()
			wg.Done()
		}()
	}
	wg.Wait()
	if hasErr {
		return nil, fmt.Errorf("error getting one of more service addresses")
	}
	return result, nil
}
