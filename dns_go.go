package main

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/miekg/dns"
	"gopkg.in/ini.v1"
	"log"
	"net"
	"os"
	"strings"
)

var records = map[string]string{}

const VER = "0.0.2"

func parseQuery(m *dns.Msg) {
	for _, q := range m.Question {
		switch q.Qtype {
		case dns.TypeA:
			log.Printf("Query for %s\n", q.Name)
			ip := records[q.Name]
			if ip != "" {
				rr, err := dns.NewRR(fmt.Sprintf("%s A %s", q.Name, ip))
				if err == nil {
					m.Answer = append(m.Answer, rr)
				}
			} else if os.Getenv("DNSGO_globr") == "true" {
				ips, err := net.LookupIP(q.Name)
				if err == nil {
					for _, ip := range ips {
						rr, err := dns.NewRR(fmt.Sprintf(q.Name + " IN A " + ip.String()))
						if err == nil {
							m.Answer = append(m.Answer, rr)
						}
					}
				}
			}
		}
	}
}

func handleDnsRequest(w dns.ResponseWriter, r *dns.Msg) {
	m := new(dns.Msg)
	m.SetReply(r)
	m.Compress = false

	switch r.Opcode {
	case dns.OpcodeQuery:
		parseQuery(m)
	}

	err := w.WriteMsg(m)
	if err != nil {
		log.Println("Failed to write response")
	}
}

func cmd() {
	log.Println("Type \"exit\" to exit")
	scanner := bufio.NewScanner(os.Stdin) // check input
	for scanner.Scan() {
		cmdInput := scanner.Text()               // scan input
		splitCmd := strings.Split(cmdInput, " ") // split args
		switch splitCmd[0] {                     // execute commands
		case "add":
			records[splitCmd[1]] = splitCmd[2]
			recStr, err := json.Marshal(records)
			if err != nil {
				log.Println("Failed to write record pool to file")
			}
			recFile, err := os.OpenFile("records.json", os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0755)
			if err != nil {
				log.Println("Failed to write record pool to file")
			}
			_, _ = recFile.Write(recStr)
			_ = recFile.Close()
			log.Printf("Successfully add a record: %s -> %s\n", splitCmd[1], splitCmd[2])
		case "del":
			ip := records[splitCmd[1]]
			if ip != "" {
				delete(records, splitCmd[1])
				recStr, err := json.Marshal(records)
				if err != nil {
					log.Println("Failed to write record pool to file")
				}
				recFile, err := os.OpenFile("records.json", os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0755)
				if err != nil {
					log.Println("Failed to write record pool to file")
				}
				_, _ = recFile.Write(recStr)
				_ = recFile.Close()
				log.Printf("Successfully delete a record: %s -> %s\n", splitCmd[1], ip)
			} else {
				log.Println("Record does not exist")
			}
		case "list":
			listStr := new(bytes.Buffer)
			for key, value := range records {
				_, _ = fmt.Fprintf(listStr, "record: %s -> %s \n", key, value)
			}
			log.Print("Record list: \n", listStr)
		case "exit":
			log.Println("Now exiting...")
			os.Exit(0)
		default:
			log.Printf("Unknown command: %s\n", cmdInput)
		}
	}
}

func readConfig() {
	log.Println("Loading config...")
	if _, errRead := os.Stat("config.ini"); os.IsNotExist(errRead) {
		// config fie does not exist
		log.Println("Config file does not exists. Now creating one...")
		newFile, errCreate := os.Create("config.ini") // create a new one
		if errCreate != nil {
			log.Panicf("Failed to create config.ini: %v \n", errCreate)
		}
		_, _ = newFile.WriteString("[Network]\naddress=127.0.0.1:53\nhandler_pattern=.\n\n[Lookup]\nallow_global_record=false\n")
		_ = newFile.Close()
	}
	cfg, errLoad := ini.Load("config.ini") // read config file
	if errLoad != nil {
		log.Panicf("Failed to read config.ini: %v \n", errLoad)
	}
	_ = os.Setenv("DNSGO_addr", cfg.Section("Network").Key("address").String())
	_ = os.Setenv("DNSGO_hpatt", cfg.Section("Network").Key("handler_pattern").String())
	_ = os.Setenv("DNSGO_globr", cfg.Section("Lookup").Key("allow_global_record").String())
}

func loadRecords() {
	log.Println("Loading records...")
	if _, errRead := os.Stat("records.json"); os.IsNotExist(errRead) {
		// config fie does not exist
		log.Println("Records file does not exists. Now creating one...")
		newFile, errCreate := os.Create("records.json") // create a new one
		if errCreate != nil {
			log.Panicf("Failed to create records.json: %v \n", errCreate)
		}
		_, _ = newFile.WriteString("{}")
		_ = newFile.Close()
	}
	recIo, errLoad := os.ReadFile("records.json")
	if errLoad != nil {
		if errLoad != nil {
			log.Panicf("Failed to read records.json: %v \n", errLoad)
		}
	}
	err := json.Unmarshal(recIo, &records)
	if err != nil {
		log.Panicf("Failed to load records: %v \n", err)
	}
}

func main() {
	log.Printf("Starting DNS-Go  ver: %s", VER)
	readConfig()
	loadRecords()

	// attach request handler func
	dns.HandleFunc(os.Getenv("DNSGO_hpatt"), handleDnsRequest)

	// start server
	go cmd()
	server := &dns.Server{Addr: os.Getenv("DNSGO_addr"), Net: "udp"}
	log.Printf("Binding at %s\n", os.Getenv("DNSGO_addr"))
	err := server.ListenAndServe()
	defer func(server *dns.Server) {
		err := server.Shutdown()
		if err != nil {
			log.Panicln("Failed to perform normal shutdown!")
		}
	}(server)
	if err != nil {
		log.Fatalf("Failed to start server: %s\n ", err.Error())
	}
}
