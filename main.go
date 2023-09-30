package main

import (
	"fmt"
	"net/url"
	"os"

	"github.com/boryashkin/cert-manager-webhook-beget/beget"
	"github.com/cert-manager/cert-manager/pkg/acme/webhook/cmd"
	"k8s.io/klog/v2"
)

const BegetProductionApiUrl = "https://api.beget.com"

var GroupName = os.Getenv("GROUP_NAME")
var BegetDnsApiUrl = os.Getenv("BEGET_DNS_API_URL")

func main() {
	if GroupName == "" {
		panic("GROUP_NAME must be specified")
	}

	if BegetDnsApiUrl == "" {
		BegetDnsApiUrl = BegetProductionApiUrl
	}
	klog.Infof("BEGET_DNS_API_URL=%s", BegetDnsApiUrl)

	begetUrl, err := url.Parse(BegetDnsApiUrl)
	if err != nil {
		panic(fmt.Sprintf("failed to parse begetUrl: %s", BegetDnsApiUrl))
	}

	cmd.RunWebhookServer(GroupName,
		beget.New(begetUrl),
	)
}
