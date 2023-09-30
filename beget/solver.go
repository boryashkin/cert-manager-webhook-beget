package beget

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"strings"
	"sync"

	acme "github.com/cert-manager/cert-manager/pkg/acme/webhook/apis/acme/v1alpha1"
	certmgrv1 "github.com/cert-manager/cert-manager/pkg/apis/meta/v1"
	extapi "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/klog/v2"
)

type begetDNSProviderConfig struct {
	URL                string                      `json:"url"`
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

func (s *Solver) credentials(namespace string, login, password certmgrv1.SecretKeySelector) (Credentials, error) {
	klog.Info("solver.credentials")
	sec, err := s.k8sClient.CoreV1().
		Secrets(namespace).
		Get(context.TODO(), login.Name, v1.GetOptions{})
	if err != nil {
		klog.Errorf("solver.credentials: calling k8s: %v", err)

		return Credentials{}, err
	}
	loginBytes, ok := sec.Data[login.Key]
	if !ok {
		return Credentials{}, fmt.Errorf("key %q not found in secret \"%s/%s\"",
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
			return Credentials{}, err
		}

		passwordBytes, ok = passwdSec.Data[password.Key]
		if !ok {
			return Credentials{}, fmt.Errorf("key %q not found in secret \"%s/%s\"",
				password.Key,
				password.Name,
				namespace)
		}
	} else {
		passwordBytes, ok = sec.Data[password.Key]
		if !ok {
			return Credentials{}, fmt.Errorf("key %q not found in secret \"%s/%s\"",
				password.Key,
				password.Name,
				namespace)
		}
	}

	return Credentials{Login: string(loginBytes), Passwd: string(passwordBytes)}, nil
}

type Solver struct {
	name      string
	client    *ApiClient
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

	records := make(Records)

	PushTXTRecord(records, ch.Key)

	klog.Info("solver.present: before changeRecords")

	err = e.client.ChangeRecords(filteredFqdn(ch.ResolvedFQDN), records, creds)
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

	records := make(Records)

	err = e.client.ChangeRecords(filteredFqdn(ch.ResolvedFQDN), records, creds)
	if err != nil {
		return fmt.Errorf("changing DNS records via API: %w", err)
	}

	klog.Infof("solver.cleanUp client.changeRecords")

	return nil
}

func (e *Solver) Initialize(kubeClientConfig *rest.Config, stopCh <-chan struct{}) error {
	var kccString string
	if kubeClientConfig != nil {
		kccString = kubeClientConfig.String()
	}

	klog.Infof("solver.initialize kcc: %s", kccString)

	if e.k8sClient != nil {
		klog.Infof("solver.initialize noop: k8sClient is already initialized")

		return nil
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
		name:   "beget-unoficial",
		client: NewApiClient(begetURL),
	}
}

func filteredFqdn(fqdn string) string {
	return strings.Trim(fqdn, ".")
}
