package begetapi

import (
	"fmt"

	"github.com/miekg/dns"
)

func (e *BegetApiMock) handleDNSRequest(w dns.ResponseWriter, req *dns.Msg) {
	fmt.Println("\n\nHandleDNS")
	msg := new(dns.Msg)
	fmt.Println("\n\n>" + msg.String())
	msg.SetReply(req)
	switch req.Opcode {
	case dns.OpcodeQuery:
		for _, q := range msg.Question {
			fmt.Println("\n\n>> for")
			if err := e.addDNSAnswer(q, msg, req); err != nil {
				msg.SetRcode(req, dns.RcodeServerFailure)
				break
			}
		}
	}
	w.WriteMsg(msg)
}

func (e *BegetApiMock) addDNSAnswer(q dns.Question, msg *dns.Msg, req *dns.Msg) error {
	switch q.Qtype {
	// Always return loopback for any A query
	case dns.TypeA:
		rr, err := dns.NewRR(fmt.Sprintf("%s 5 IN A 127.0.0.1", q.Name))
		if err != nil {
			return err
		}
		msg.Answer = append(msg.Answer, rr)
		return nil

	// TXT records are the only important record for ACME dns-01 challenges
	case dns.TypeTXT:
		fmt.Println("\n\n>> TypeTXT: " + q.Name)
		e.RLock()
		records, found := e.txtRecords[q.Name]
		e.RUnlock()
		if !found {
			fmt.Println("\n\n>> !FOUND")
			msg.SetRcode(req, dns.RcodeNameError)
			return nil
		}
		_, ok := records[TXTKey]
		if !ok || len(records[TXTKey]) == 0 {
			fmt.Println("\n\n>> !ok record")
			msg.SetRcode(req, dns.RcodeNameError)
			return nil
		}
		record, ok := records[TXTKey][0][TXTDataKey].(string)
		if !ok {
			fmt.Println("\n\n>> !ok record")
			msg.SetRcode(req, dns.RcodeNameError)
			return nil
		}

		fmt.Println("\n\n>> FOUND: " + fmt.Sprintf("%s 5 IN TXT %s", q.Name, record))
		rr, err := dns.NewRR(fmt.Sprintf("%s 5 IN TXT %s", q.Name, record))
		if err != nil {
			fmt.Println("\n\n>> rrErr " + err.Error())
			return err
		}
		fmt.Println("\n\n>> answer")
		msg.Answer = append(msg.Answer, rr)
		return nil

	// NS and SOA are for authoritative lookups, return obviously invalid data
	case dns.TypeNS:
		rr, err := dns.NewRR(fmt.Sprintf("%s 5 IN NS ns.example-acme-webook.invalid.", q.Name))
		if err != nil {
			return err
		}
		msg.Answer = append(msg.Answer, rr)
		return nil
	case dns.TypeSOA:
		rr, err := dns.NewRR(fmt.Sprintf("%s 5 IN SOA %s 20 5 5 5 5", "ns.example-acme-webook.invalid.", "ns.example-acme-webook.invalid."))
		if err != nil {
			return err
		}
		msg.Answer = append(msg.Answer, rr)
		return nil
	default:
		return fmt.Errorf("unimplemented record type %v", q.Qtype)
	}
}
