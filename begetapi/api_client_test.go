package begetapi_test

import (
	"context"
	"net/url"
	"testing"

	"github.com/boryashkin/cert-manager-webhook-beget/begetapi"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
)

type ApiClientTestSuite struct {
	suite.Suite
	begetApi *begetapi.BegetApiMock
	client   *begetapi.ApiClient
}

func (suite *ApiClientTestSuite) SetupTest() {
	suite.begetApi = begetapi.NewBegetApiMock("login", "password")
	go func() {
		suite.begetApi.Run(":12943")
	}()
	url, err := url.Parse("http://localhost:12943")
	suite.Require().NoError(err)

	suite.client = begetapi.NewApiClient(
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
	_, err := suite.client.GetData("api.example.com", begetapi.Credentials{Login: "login", Passwd: "password"})
	suite.Require().NoError(err, "getData returned an err %s", err)
}

func (suite *ApiClientTestSuite) TestApiClient_ChangeRecords() {
	err := suite.client.ChangeRecords("api.example.com", begetapi.Records{}, begetapi.Credentials{Login: "login", Passwd: "password"})
	suite.Require().NoError(err, "getData returned an err %s", err)
}

func (suite *ApiClientTestSuite) TestApiClient_GetChangedRecords() {
	records, err := suite.client.GetData("api.example.com", begetapi.Credentials{Login: "login", Passwd: "password"})
	suite.Require().NoError(err)
	suite.Require().Empty(records)

	begetapi.PushTXTRecord(records, "mydemo")

	err = suite.client.ChangeRecords("api.example.com", records, begetapi.Credentials{Login: "login", Passwd: "password"})
	suite.Require().NoError(err)

	newRecords, err := suite.client.GetData("api.example.com", begetapi.Credentials{Login: "login", Passwd: "password"})
	suite.Require().NoError(err)
	suite.Require().NotEmpty(newRecords)
}

func TestApiClient_PushTXTRecord(t *testing.T) {
	r := make(begetapi.Records)

	err := begetapi.PushTXTRecord(r, "test")

	assert.NoError(t, err)
	assert.Equal(t, "test", r[begetapi.TXTKey][0][begetapi.TXTDataKey])
}

func TestApiClient_PushTXTRecord_NotEmpty(t *testing.T) {
	r := make(begetapi.Records)
	r["SomeKey"] = make([]map[string]interface{}, 1)

	err := begetapi.PushTXTRecord(r, "test")

	assert.NoError(t, err)
	assert.Equal(t, "test", r[begetapi.TXTKey][0][begetapi.TXTDataKey])
}

func TestApiClient_PopTXTRecord(t *testing.T) {
	r := make(begetapi.Records)

	err := begetapi.PushTXTRecord(r, "test")
	cnt := begetapi.PopTXTRecordByValue(r, "test")

	assert.NoError(t, err)
	assert.Equal(t, 1, cnt)
}

func TestApiClient_PopTXTRecord_NotEmpty(t *testing.T) {
	r := make(begetapi.Records)

	r["SomeKey"] = make([]map[string]interface{}, 1)

	err := begetapi.PushTXTRecord(r, "test")
	cnt := begetapi.PopTXTRecordByValue(r, "test")

	assert.NoError(t, err)
	assert.Equal(t, 1, cnt)
}
