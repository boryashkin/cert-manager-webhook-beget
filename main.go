package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"os"
	"strings"
	"sync"

	"github.com/boryashkin/cert-manager-webhook-beget/begetapi"

	acme "github.com/cert-manager/cert-manager/pkg/acme/webhook/apis/acme/v1alpha1"
	"github.com/cert-manager/cert-manager/pkg/acme/webhook/cmd"
	certmgrv1 "github.com/cert-manager/cert-manager/pkg/apis/meta/v1"
	extapi "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/klog/v2"
)

const BegetProductionApiUrl = "https://api.beget.com"

var GroupName = os.Getenv("GROUP_NAME")
var BegetDnsApiUrl = os.Getenv("BEGET_DNS_API_URL")

// beget api doesn't support strict mode with retaining records
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
		New(begetUrl),
	)
}

type begetDNSProviderConfig struct {
	APILoginSecretRef  certmgrv1.SecretKeySelector `json:"apiLoginSecretRef"`
	APIPasswdSecretRef certmgrv1.SecretKeySelector `json:"apiPasswdSecretRef"`
}

func loadConfig(cfgJSON *extapi.JSON) (begetDNSProviderConfig, error) {
	klog.Info("solver.loadConfig")
	cfg := begetDNSProviderConfig{}
	if cfgJSON == nil {
		klog.Error("solver.loadConfig: empty cfg")
		return cfg, nil
	}

	if err := json.Unmarshal(cfgJSON.Raw, &cfg); err != nil {
		return cfg, fmt.Errorf("error decoding solver config: %v", err)
	}

	return cfg, nil
}

func (s *Solver) credentials(namespace string, login, password certmgrv1.SecretKeySelector) (begetapi.Credentials, error) {
	klog.Info("solver.credentials")
	sec, err := s.k8sClient.CoreV1().
		Secrets(namespace).
		Get(context.TODO(), login.Name, v1.GetOptions{})
	if err != nil {
		klog.Errorf("solver.credentials: calling k8s: %v", err)

		return begetapi.Credentials{}, err
	}

	loginBytes, ok := sec.Data[login.Key]
	if !ok {
		return begetapi.Credentials{}, fmt.Errorf("key %q not found in secret \"%s/%s\"",
			login.Key,
			login.Name,
			namespace)
	}

	var passwordBytes []byte

	if login.Name != password.Name {
		passwdSec, err := s.k8sClient.CoreV1().
			Secrets(namespace).
			Get(context.TODO(), password.Name, v1.GetOptions{})
		if err != nil {
			return begetapi.Credentials{}, err
		}

		passwordBytes, ok = passwdSec.Data[password.Key]
		if !ok {
			return begetapi.Credentials{}, fmt.Errorf("key %q not found in secret \"%s/%s\"",
				password.Key,
				password.Name,
				namespace)
		}
	} else {
		passwordBytes, ok = sec.Data[password.Key]
		if !ok {
			return begetapi.Credentials{}, fmt.Errorf("key %q not found in secret \"%s/%s\"",
				password.Key,
				password.Name,
				namespace)
		}
	}

	return begetapi.Credentials{Login: string(loginBytes), Passwd: string(passwordBytes)}, nil
}

type Solver struct {
	name      string
	client    *begetapi.ApiClient
	k8sClient *kubernetes.Clientset
	sync.RWMutex
}

func (e *Solver) Name() string {
	klog.Infof("solver.name: %s", e.name)
	return e.name
}

func (e *Solver) Present(ch *acme.ChallengeRequest) error {
	var chString string
	if ch != nil {
		chString = fmt.Sprintf("rn: %s, rz: %s, rfqdn: %s, dnsn: %s", ch.ResourceNamespace, ch.ResolvedZone, ch.ResolvedFQDN, ch.DNSName)
	}

	klog.Infof("solver.present: ch.: %s", chString)

	cfg, err := loadConfig(ch.Config)
	if err != nil {
		klog.Errorf("solver.present: loadConfig: %v", err)

		return err
	}

	klog.Info("solver.present: after loadConfig")

	creds, err := e.credentials(ch.ResourceNamespace, cfg.APILoginSecretRef, cfg.APIPasswdSecretRef)
	if err != nil {
		klog.Errorf("solver.present: credentials: %v", err)

		return err
	}

	klog.Info("solver.present: after credentials")

	records := make(begetapi.Records)

	begetapi.PushTXTRecord(records, ch.Key)

	klog.Info("solver.present: before changeRecords")

	err = e.client.ChangeRecords(trimFqdn(ch.ResolvedFQDN), records, creds)
	if err != nil {
		klog.Errorf("solver.present: changeRecords err: %v", err)

		return fmt.Errorf("changing DNS records via API: %w", err)
	}

	klog.Infof("solver.present client.changeRecords")

	return nil
}

func (e *Solver) CleanUp(ch *acme.ChallengeRequest) error {
	var chString string
	if ch != nil {
		chString = fmt.Sprintf("rn: %s, rz: %s, rfqdn: %s, dnsn: %s", ch.ResourceNamespace, ch.ResolvedZone, ch.ResolvedFQDN, ch.DNSName)
	}

	klog.Infof("solver.cleanUp ch.: %s", chString)

	cfg, err := loadConfig(ch.Config)
	if err != nil {
		return err
	}
	creds, err := e.credentials(ch.ResourceNamespace, cfg.APILoginSecretRef, cfg.APIPasswdSecretRef)
	if err != nil {
		return err
	}

	records := make(begetapi.Records)

	err = e.client.ChangeRecords(trimFqdn(ch.ResolvedFQDN), records, creds)
	if err != nil {
		return fmt.Errorf("changing DNS records via API: %w", err)
	}

	klog.Infof("solver.cleanUp client.changeRecords")

	return nil
}

func (e *Solver) Initialize(kubeClientConfig *rest.Config, stopCh <-chan struct{}) error {
	klog.Infof("solver.initialize kcc")

	if e.k8sClient != nil {
		klog.Info("solver.initialize: k8sClient is already initialized")
	}

	cl, err := kubernetes.NewForConfig(kubeClientConfig)
	if err != nil {
		return err
	}

	e.k8sClient = cl

	return nil
}

func New(begetURL *url.URL) *Solver {
	return &Solver{
		name:   "beget",
		client: begetapi.NewApiClient(begetURL),
	}
}

func trimFqdn(fqdn string) string {
	return strings.Trim(fqdn, ".")
}
