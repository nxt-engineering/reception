package dns

import (
	"fmt"
	"net"

	"github.com/miekg/dns"
	"github.com/ninech/reception/common"
)

// returns the appropriate answer for any incoming DNS query
type Handler struct {
	// maps Host (from request header) to destination Host
	Config *common.Config
}

func (handler Handler) ServeDns(response dns.ResponseWriter, request *dns.Msg) {
	request_domain := request.Question[0].Name

	fmt.Printf("Received a DNS request for '%v'.\n", request_domain)

	reply := new(dns.Msg)
	reply.SetReply(request)
	reply.Authoritative = true

	rr4 := new(dns.A)
	rr4.Hdr = dns.RR_Header{Name: request_domain, Rrtype: dns.TypeA, Class: dns.ClassINET, Ttl: 30}
	rr4.A = net.ParseIP("127.0.0.1")

	rr6 := new(dns.AAAA)
	rr6.Hdr = dns.RR_Header{Name: request_domain, Rrtype: dns.TypeAAAA, Class: dns.ClassINET, Ttl: 30}
	rr6.AAAA = net.ParseIP("::1")

	reply.Answer = []dns.RR{rr4, rr6}
	response.WriteMsg(reply)
}
