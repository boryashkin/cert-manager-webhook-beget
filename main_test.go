package main

import (
	"context"
	"net/url"
	"os"
	"testing"

	"github.com/boryashkin/cert-manager-webhook-beget/begetapi"
	dns "github.com/cert-manager/cert-manager/test/acme"
)

var (
	zone = os.Getenv("TEST_ZONE_NAME")
)

func TestRunsSuite(t *testing.T) {
	// The manifest path should contain a file named config.json that is a
	// snippet of valid configuration that should be included on the
	// ChallengeRequest passed as part of the test cases.
	begerURL, err := url.Parse("http://localhost:8080")
	if err != nil {
		t.FailNow()
	}

	api := begetapi.NewBegetApiMock("login", "password")
	go func() {
		api.Run(":8080")
		t.Log("run")
	}()
	go func() {
		api.RunDns("59351")
		t.Log("run dns")
	}()
	defer func() {
		api.Stop(context.TODO())
		api.StopDns(context.TODO())
		t.Log("stopped servers")
	}()

	solver := New(begerURL)
	fixture := dns.NewFixture(solver,
		dns.SetResolvedZone("example.com."),
		dns.SetManifestPath("testdata/beget"),
		dns.SetDNSServer("127.0.0.1:59351"),
		dns.SetUseAuthoritative(false),
	)

	fixture.RunConformance(t)

}
