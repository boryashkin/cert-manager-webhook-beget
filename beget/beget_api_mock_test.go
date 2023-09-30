package beget_test

import (
	"context"
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/boryashkin/cert-manager-webhook-beget/beget"
	"github.com/stretchr/testify/suite"
)

type BegetApiMockTestSuite struct {
	suite.Suite
	begetApi *beget.BegetApiMock
}

func (suite *BegetApiMockTestSuite) SetupTest() {
	suite.begetApi = beget.NewBegetApiMock("testl", "testp")
	go func() {
		suite.begetApi.Run(":8488")
	}()
}

func (suite *BegetApiMockTestSuite) TearDownSuite() {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	suite.begetApi.Stop(ctx)
}

func TestBegetApiMockTestSuite(t *testing.T) {
	suite.Run(t, new(BegetApiMockTestSuite))
}

func (suite *BegetApiMockTestSuite) TestBegetApiMock_Run() {
	time.Sleep(time.Second)
	r, err := http.Get("http://localhost:8488/api/dns/getData?login=testl&passwd=testp&input_format=json&input_data={}")

	suite.Require().NoError(err, fmt.Sprintf("failed getData %s", err))
	suite.Require().Equal(200, r.StatusCode, fmt.Sprintf("getData responded %d", r.StatusCode))

	// r, err = http.Get("http://localhost:8488/api/dns/changeRecords?login=testl&passwd=testp")

	// suite.Require().NoError(err, fmt.Sprintf("failed changeRecords %s", err))
	// suite.Require().Equal(200, r.StatusCode, fmt.Sprintf("changeRecords responded %d", r.StatusCode))

	r, err = http.Get("http://localhost:8488/api/dns/getData?login=testl&passwd=testp&input_format=json&input_data={\"}")
	suite.Require().NoError(err, fmt.Sprintf("failed getData %s", err))
	suite.Require().Equal(500, r.StatusCode, fmt.Sprintf("changeRecords responded %d", r.StatusCode))
}

const inp = `{"fqdn":"example.com","records":{"A":[{"address":"85.12.197.93","ttl":600}],"DNS":[],"DNS_IP":[],"MX":[{"exchange":"mx1.beget.com.","preference":10,"ttl":600},{"exchange":"mx2.beget.com.","preference":20,"ttl":600}],"TXT":[{"ttl":600,"txtdata":"v=spf1 redirect=beget.com"},{"ttl":600,"txtdata":"1"}]}}`
