package main

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/grafana/grafana-plugin-sdk-go/backend"
	"github.com/grafana/grafana-plugin-sdk-go/backend/datasource"
	"github.com/grafana/grafana-plugin-sdk-go/backend/instancemgmt"
	"github.com/grafana/grafana-plugin-sdk-go/backend/log"
	"github.com/grafana/grafana-plugin-sdk-go/data"
	"github.com/kpfaulkner/azurecosts/pkg"
	"net/http"
	"strings"
	"time"
)

type AzureCostsQuery struct {
	Constant      float64 `json:"constant"`
	Datasource    string  `json:"datasource"`
	DatasourceID  int     `json:"datasourceId"`
	IntervalMs    int     `json:"intervalMs"`
	MaxDataPoints int     `json:"maxDataPoints"`
	OrgID         int     `json:"orgId"`
	QueryText     string  `json:"queryText"`
	RGSplit       string  `json:"rgText"`
	RefID         string  `json:"refId"`
}

type AzureCostsPluginConfig struct {
	ClientID       string `json:"clientID"`
	ClientSecret   string `json:"clientSecret"`
	TenantID       string `json:"tenantID"`
	SubscriptionID string `json:"SubscriptionID"`
}

type queryModel struct {
  Format string `json:"format"`
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
		cache: NewCache(),
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

	// cache at subscription level.
	cache *Cache
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
		res, err := td.query(q)
		if err != nil {
			return nil, err
		}

		// save the response in a hashmap
		// based on with RefID as identifier
		response.Responses[q.RefID] = *res
	}

	return response, nil
}

// executes query AND populates cache.
// return data that is in cache.
// SubscriptionEntryCache is for a single subscription
// and groups costs per resource group
func (td *AzureCostsDataSource) executeQueryAndPopulateCache(subscriptionID string, startTime time.Time, endTime time.Time) (*SubscriptionCacheEntry, error) {

	dailyBilling, err := td.azureCosts.GetAllBillingForSubscriptionID(subscriptionID, startTime, endTime)
	if err != nil {
		log.DefaultLogger.Error(fmt.Sprintf("ERROR getting costs %s", err.Error()))
		return nil, err
	}

	log.DefaultLogger.Info(fmt.Sprintf("Got data, len %d", len(dailyBilling)))
	cacheEntry := NewSubscriptionCacheEntry()
	cacheEntry.SubscriptionID = subscriptionID
	cacheEntry.StartDate = startTime
	cacheEntry.EndDate = endTime

	log.DefaultLogger.Info(fmt.Sprintf("first date result entry %s", dailyBilling[0].Properties.UsageStart))

	//return nil, errors.New("FAKE ERROR")

	for _, db := range dailyBilling {
		ce := convertDailyBillingDetailsToDailyCacheEntry(db)

		// resource group
		sp := strings.Split(db.Properties.InstanceID, "/")
		rg := strings.ToLower(sp[4]) // just deal with lowercase.

		var dailyCacheEntryCollection map[time.Time]DailyCacheEntry
		var ok bool
		dailyCacheEntryCollection, ok = cacheEntry.ResourceGroupCosts[rg]
		if !ok {
			dailyCacheEntryCollection = make(map[time.Time]DailyCacheEntry)
		}

		// assuming multiple instances of same RG are returned (different resources within same RG)
		// So just totalling up the amounts.
		existingEntryForDate, ok := dailyCacheEntryCollection[ce.StartDate]
		if ok {
			ce.Amount += existingEntryForDate.Amount
		}

		log.DefaultLogger.Info(fmt.Sprintf("Adding cache with key  %s", ce.StartDate))
		dailyCacheEntryCollection[ce.StartDate] = ce
		cacheEntry.ResourceGroupCosts[rg] = dailyCacheEntryCollection
	}

	td.cache.Set(subscriptionID, *cacheEntry)

	log.DefaultLogger.Info(fmt.Sprintf("returning cache entry, number of RG %d", len(cacheEntry.ResourceGroupCosts)))
	return cacheEntry, nil
}

func (td *AzureCostsDataSource) generateRGSplitFrame(roundedStartTime time.Time, cacheEntry *SubscriptionCacheEntry) (*data.Frame, error) {

  // create data frame response
  frame := data.NewFrame("response")

  // only set time once? I hope :)
  times := []time.Time{}

  currentTime := roundedStartTime

  // Generate times array
  for currentTime.Before(cacheEntry.EndDate) {
    times = append(times, currentTime)
    currentTime = currentTime.Add(24 * time.Hour)
  }

  log.DefaultLogger.Info(fmt.Sprintf("time count %d", len(times)))

  for rg, costs := range cacheEntry.ResourceGroupCosts {
    log.DefaultLogger.Info(fmt.Sprintf("roundedStartTime 3 %s ", roundedStartTime))
    log.DefaultLogger.Info(fmt.Sprintf("rg %s : costs size %d", rg, len(costs)))
    for _, c := range costs {
      log.DefaultLogger.Info(fmt.Sprintf("rg %s : cost key %s", rg, c.StartDate))
    }

    //log.DefaultLogger.Info(fmt.Sprintf("rg %s : costs data %v", rg, costs))
    amounts := []float64{}

    // loop through time, so we can provide empty entries
    // when we dont have data for that RG
    currentTime = roundedStartTime
    for currentTime.Before(cacheEntry.EndDate) {
      log.DefaultLogger.Info(fmt.Sprintf("RG %s ct %s", rg, currentTime))
      e, ok := costs[currentTime]
      if ok {
        amounts = append(amounts, e.Amount)
      } else {
        amounts = append(amounts, 0)
      }

      currentTime = currentTime.Add(24 * time.Hour)
    }

    frame.Fields = append(frame.Fields,
      data.NewField(rg, nil, amounts),
    )
  }

  frame.Fields = append(frame.Fields,
    data.NewField("time", nil, times),
  )

  return frame, nil
}

func (td *AzureCostsDataSource) generateSubscriptionFrame(roundedStartTime time.Time, cacheEntry *SubscriptionCacheEntry) (*data.Frame, error) {


  timeCosts := make(map[time.Time]float64)
  for _, costs := range cacheEntry.ResourceGroupCosts {
    for _,c := range costs {
      existing := timeCosts[c.StartDate]
      existing += c.Amount
      timeCosts[c.StartDate] = existing
    }
  }

  // only set time once? I hope :)
  times := []time.Time{}
  amounts := []float64{}

  // have all costs added up in timeCosts, now to map out to particular dates
  currentTime := roundedStartTime

  // Generate times array
  for currentTime.Before(cacheEntry.EndDate) {
    times = append(times, currentTime)
    cost:= timeCosts[currentTime]
    amounts = append(amounts, cost)
    currentTime = currentTime.Add(24 * time.Hour)
  }

  // create data frame response
  frame := data.NewFrame("response")
  frame.Fields = append(frame.Fields,
    data.NewField("time", nil, times),
  )

  frame.Fields = append(frame.Fields,
    data.NewField("subscription", nil, amounts),
  )
  return frame, nil
}

func (td *AzureCostsDataSource) query(query backend.DataQuery) (*backend.DataResponse, error) {
	// Unmarshal the json into our queryModel
	var qm queryModel

	var acQuery AzureCostsQuery
	queryBytes, _ := query.JSON.MarshalJSON()
	err := json.Unmarshal(queryBytes, &acQuery)
	if err != nil {
		// empty response? or real error? figure out later.
		log.DefaultLogger.Error(fmt.Sprintf("Unable to get query %s", err.Error()))
		return nil, err
	}

	response := backend.DataResponse{}
	response.Error = json.Unmarshal(query.JSON, &qm)
	if response.Error != nil {
		log.DefaultLogger.Error(fmt.Sprintf("Unable to get qm resp %s", err.Error()))
		return nil, err
	}

	// Log a warning if `Format` is empty.
	if qm.Format == "" {
		log.DefaultLogger.Warn("format is empty. defaulting to time series")
	}

	roundedStartTime := time.Date(query.TimeRange.From.Year(), query.TimeRange.From.Month(),
		query.TimeRange.From.Day(), 0, 0, 0, 0, time.UTC).UTC()

	roundedEndTime := time.Date(query.TimeRange.To.Year(), query.TimeRange.To.Month(),
		query.TimeRange.To.Day(), 0, 0, 0, 0, time.UTC).UTC()

	subscriptionID := acQuery.QueryText
	var cacheEntry *SubscriptionCacheEntry

	// check if we already have this in cache.
	cacheEntry = td.cache.Get(subscriptionID)

	// if no cache entry or cache dates dont match, then do real query.
	if cacheEntry == nil || !(cacheEntry.StartDate == roundedStartTime && cacheEntry.EndDate == roundedEndTime) {
		log.DefaultLogger.Info("Populating cache")
		cacheEntry, err = td.executeQueryAndPopulateCache(acQuery.QueryText, roundedStartTime, roundedEndTime)
		if err != nil {
			log.DefaultLogger.Error(fmt.Sprintf("query error %s", err.Error()))
			return nil, err
		}
	} else {
		log.DefaultLogger.Info("got data in cache")
	}

	// determine type of output (RG, total number, other etc).
  splitOnRG := acQuery.RGSplit=="Y"

  var frame *data.Frame
  if splitOnRG {
    frame, err = td.generateRGSplitFrame(roundedStartTime, cacheEntry)
    if err != nil {
      log.DefaultLogger.Error(fmt.Sprintf("error generating split frame %s ", err.Error))
      return nil, err
    }
  } else {
    frame, err = td.generateSubscriptionFrame(roundedStartTime, cacheEntry)
    if err != nil {
      log.DefaultLogger.Error(fmt.Sprintf("error generating subscription frame %s ", err.Error))
      return nil, err
    }
  }

	// add the frames to the response
	response.Frames = append(response.Frames, frame)
	return &response, nil
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
