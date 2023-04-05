package main

import (
	"bufio"
	"bytes"
	"fmt"
	"log"
	"net"
	"os"
	"strconv"
	"strings"

	"github.com/miekg/dns"
)

var records = map[string]string{
	"test.service.": "192.168.0.2",
	"abc.com.":      "192.168.0.1",
}

const VER = "0.0.1"

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
			} else {
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
			log.Printf("Successfully add a record: %s -> %s\n", splitCmd[1], splitCmd[2])
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

func main() {
	log.Printf("Starting DNS-Go  ver: %s", VER)
	// attach request handler func
	dns.HandleFunc(".", handleDnsRequest)

	// start server
	go cmd()
	host := ""
	port := 53
	server := &dns.Server{Addr: host + ":" + strconv.Itoa(port), Net: "udp"}
	log.Printf("Binding at %s\n", host+":"+strconv.Itoa(port))
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
