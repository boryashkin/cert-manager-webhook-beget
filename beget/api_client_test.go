package beget_test

import (
	"context"
	"net/url"
	"testing"

	"github.com/boryashkin/cert-manager-webhook-beget/beget"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
)

type ApiClientTestSuite struct {
	suite.Suite
	begetApi *beget.BegetApiMock
	client   *beget.ApiClient
}

func (suite *ApiClientTestSuite) SetupTest() {
	suite.begetApi = beget.NewBegetApiMock("login", "password")
	go func() {
		suite.begetApi.Run(":12943")
	}()
	url, err := url.Parse("http://localhost:12943")
	suite.Require().NoError(err)

	suite.client = beget.NewApiClient(
		url,
	)
}

func (suite *ApiClientTestSuite) TearDownSuite() {
	suite.begetApi.Stop(context.TODO())
}

func TestApiClientTestSuiteSuite(t *testing.T) {
	suite.Run(t, new(ApiClientTestSuite))
}

func (suite *ApiClientTestSuite) TestApiClient_GetData() {
	_, err := suite.client.GetData("api.example.com", beget.Credentials{Login: "login", Passwd: "password"})
	suite.Require().NoError(err, "getData returned an err %s", err)
}

func (suite *ApiClientTestSuite) TestApiClient_ChangeRecords() {
	err := suite.client.ChangeRecords("api.example.com", beget.Records{}, beget.Credentials{Login: "login", Passwd: "password"})
	suite.Require().NoError(err, "getData returned an err %s", err)
}

func (suite *ApiClientTestSuite) TestApiClient_GetChangedRecords() {
	records, err := suite.client.GetData("api.example.com", beget.Credentials{Login: "login", Passwd: "password"})
	suite.Require().NoError(err)
	suite.Require().Empty(records)

	beget.PushTXTRecord(records, "mydemo")

	err = suite.client.ChangeRecords("api.example.com", records, beget.Credentials{Login: "login", Passwd: "password"})
	suite.Require().NoError(err)

	newRecords, err := suite.client.GetData("api.example.com", beget.Credentials{Login: "login", Passwd: "password"})
	suite.Require().NoError(err)
	suite.Require().NotEmpty(newRecords)
}

func TestApiClient_PushTXTRecord(t *testing.T) {
	r := make(beget.Records)

	err := beget.PushTXTRecord(r, "test")

	assert.NoError(t, err)
	assert.Equal(t, "test", r[beget.TXTKey][0][beget.TXTDataKey])
}

func TestApiClient_PushTXTRecord_NotEmpty(t *testing.T) {
	r := make(beget.Records)
	r["SomeKey"] = make([]map[string]interface{}, 1)

	err := beget.PushTXTRecord(r, "test")

	assert.NoError(t, err)
	assert.Equal(t, "test", r[beget.TXTKey][0][beget.TXTDataKey])
}

func TestApiClient_PopTXTRecord(t *testing.T) {
	r := make(beget.Records)

	err := beget.PushTXTRecord(r, "test")
	cnt := beget.PopTXTRecordByValue(r, "test")

	assert.NoError(t, err)
	assert.Equal(t, 1, cnt)
}

func TestApiClient_PopTXTRecord_NotEmpty(t *testing.T) {
	r := make(beget.Records)

	r["SomeKey"] = make([]map[string]interface{}, 1)

	err := beget.PushTXTRecord(r, "test")
	cnt := beget.PopTXTRecordByValue(r, "test")

	assert.NoError(t, err)
	assert.Equal(t, 1, cnt)
}
