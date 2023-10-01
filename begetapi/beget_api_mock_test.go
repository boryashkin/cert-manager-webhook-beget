package begetapi_test

import (
	"context"
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/boryashkin/cert-manager-webhook-beget/begetapi"
	"github.com/stretchr/testify/suite"
)

type BegetApiMockTestSuite struct {
	suite.Suite
	begetApi *begetapi.BegetApiMock
}

func (suite *BegetApiMockTestSuite) SetupTest() {
	suite.begetApi = begetapi.NewBegetApiMock("testl", "testp")
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
	r, err := http.Get("http://localhost:8488/api/dns/getData?login=testl&passwd=testp&input_format=json&input_data={}")

	suite.Require().NoError(err, fmt.Sprintf("failed getData %s", err))
	suite.Require().Equal(200, r.StatusCode, fmt.Sprintf("getData responded %d", r.StatusCode))

	r, err = http.Get("http://localhost:8488/api/dns/getData?login=testl&passwd=testp&input_format=json&input_data={\"}")
	suite.Require().NoError(err, fmt.Sprintf("failed getData %s", err))
	suite.Require().Equal(500, r.StatusCode, fmt.Sprintf("changeRecords responded %d", r.StatusCode))
}
