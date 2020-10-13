package main

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/grafana/grafana-plugin-sdk-go/backend"
	"github.com/grafana/grafana-plugin-sdk-go/backend/datasource"
	"github.com/grafana/grafana-plugin-sdk-go/backend/instancemgmt"
	"github.com/grafana/grafana-plugin-sdk-go/backend/log"
	"github.com/kpfaulkner/azurecosts/pkg"
	"net/http"
)

type AzureCostsQuery struct {
	Constant      float64 `json:"constant"`
	Datasource    string  `json:"datasource"`
	DatasourceID  int     `json:"datasourceId"`
	IntervalMs    int     `json:"intervalMs"`
	MaxDataPoints int     `json:"maxDataPoints"`
	OrgID         int     `json:"orgId"`
	QueryText     string  `json:"queryText"`
	RefID         string  `json:"refId"`
}

type AzureCostsPluginConfig struct {
	ClientID       string `json:"clientID"`
	ClientSecret   string `json:"clientSecret"`
	TenantID       string `json:"tenantID"`
	SubscriptionID string `json:"SubscriptionID"`
}

// newAzureCostsDataSource returns datasource.ServeOpts.
func newAzureCostsDataSource() datasource.ServeOpts {
	// creates a instance manager for your plugin. The function passed
	// into `NewInstanceManger` is called when the instance is created
	// for the first time or when a datasource configuration changed.
	im := datasource.NewInstanceManager(newDataSourceInstance)

	ds := &AzureCostsDataSource{
		im: im,
		//azureCosts: pkg.NewAzureCost("","","",""),
	}

	return datasource.ServeOpts{
		QueryDataHandler:   ds,
		CheckHealthHandler: ds,
	}
}

// AzureCostsDataSource.... all things DD :)
type AzureCostsDataSource struct {
	// The instance manager can help with lifecycle management
	// of datasource instances in plugins. It's not a requirements
	// but a best practice that we recommend that you follow.
	im instancemgmt.InstanceManager

	config AzureCostsPluginConfig

	azureCosts *pkg.AzureCost
}

// QueryData handles multiple queries and returns multiple responses.
// req contains the queries []DataQuery (where each query contains RefID as a unique identifer).
// The QueryDataResponse contains a map of RefID to the response for each query, and each response
// contains Frames ([]*Frame).
func (td *AzureCostsDataSource) QueryData(ctx context.Context, req *backend.QueryDataRequest) (*backend.QueryDataResponse, error) {
	configBytes, _ := req.PluginContext.DataSourceInstanceSettings.JSONData.MarshalJSON()
	var config AzureCostsPluginConfig
	err := json.Unmarshal(configBytes, &config)
	if err != nil {
		return nil, err
	}

	td.config = config

	if td.azureCosts == nil {
		ac := pkg.NewAzureCost(td.config.SubscriptionID, td.config.TenantID, td.config.ClientID, td.config.ClientSecret)
		td.azureCosts = &ac
	}

	// create response struct
	response := backend.NewQueryDataResponse()

	// loop over queries and execute them individually.
	for _, q := range req.Queries {
		res, err := td.query(ctx, q)
		if err != nil {
			return nil, err
		}

		// save the response in a hashmap
		// based on with RefID as identifier
		response.Responses[q.RefID] = *res
	}

	return response, nil
}

type queryModel struct {
	Format string `json:"format"`
}

func (td *AzureCostsDataSource) query(ctx context.Context, query backend.DataQuery) (*backend.DataResponse, error) {
	// Unmarshal the json into our queryModel
	var qm queryModel

	var acQuery AzureCostsQuery
	queryBytes, _ := query.JSON.MarshalJSON()
	err := json.Unmarshal(queryBytes, &acQuery)
	if err != nil {
		// empty response? or real error? figure out later.
		return nil, err
	}

	response := backend.DataResponse{}
	response.Error = json.Unmarshal(query.JSON, &qm)
	if response.Error != nil {
		return nil, err
	}

	// Log a warning if `Format` is empty.
	if qm.Format == "" {
		log.DefaultLogger.Warn("format is empty. defaulting to time series")
	}

	sc, err := td.azureCosts.GenerateSubscriptionCostDetails([]string{td.config.SubscriptionID}, query.TimeRange.From, query.TimeRange.To)
	if err != nil {
		log.DefaultLogger.Error(fmt.Sprintf("ERROR getting costs %s", err.Error()))
		return nil, err
	}

	log.DefaultLogger.Info(fmt.Sprintf("azure costs res %v", sc))

	/*

	  // add the frames to the response
	  response.Frames = append(response.Frames, frame)
	  return &response, nil */

	return nil, nil
}

// CheckHealth handles health checks sent from Grafana to the plugin.
// The main use case for these health checks is the test button on the
// datasource configuration page which allows users to verify that
// a datasource is working as expected.
func (td *AzureCostsDataSource) CheckHealth(ctx context.Context, req *backend.CheckHealthRequest) (*backend.CheckHealthResult, error) {

	var status = backend.HealthStatusOk
	var message = "Data source is working"

	configBytes, _ := req.PluginContext.DataSourceInstanceSettings.JSONData.MarshalJSON()
	var config AzureCostsPluginConfig
	err := json.Unmarshal(configBytes, &config)
	if err != nil {
		status = backend.HealthStatusError
		message = "Unable to contact Azure"
	}

	td.config = config

	return &backend.CheckHealthResult{
		Status:  status,
		Message: message,
	}, nil
}

type instanceSettings struct {
	httpClient *http.Client
}

func newDataSourceInstance(setting backend.DataSourceInstanceSettings) (instancemgmt.Instance, error) {
	return &instanceSettings{
		httpClient: &http.Client{},
	}, nil
}

func (s *instanceSettings) Dispose() {
	// Called before creatinga a new instance to allow plugin authors
	// to cleanup.
}
