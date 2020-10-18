package main

import (
  "github.com/kpfaulkner/azurecosts/pkg"
  "time"
)

type DailyCacheEntry struct {
	StartDate     time.Time
	EndDate       time.Time
	ResourceGroup string
	Amount        float64
}

type SubscriptionCacheEntry struct {
	SubscriptionID string
	StartDate      time.Time
	EndDate        time.Time

	// resource group string, list of daily entries.
	ResourceGroupCosts map[string]map[time.Time]DailyCacheEntry
}

func NewSubscriptionCacheEntry() *SubscriptionCacheEntry {
	c := SubscriptionCacheEntry{}
	c.ResourceGroupCosts = make(map[string]map[time.Time]DailyCacheEntry)
	return &c
}

type Cache struct {
	cache map[string]SubscriptionCacheEntry
}

func NewCache() *Cache {
	c := Cache{}
	c.cache = make(map[string]SubscriptionCacheEntry)
	return &c
}

func (c *Cache) Get(subID string) *SubscriptionCacheEntry {
	entry, ok := c.cache[subID]
	if !ok {
		return nil
	}
	return &entry
}

func (c *Cache) Set(subID string, entry SubscriptionCacheEntry) {
	c.cache[subID] = entry
}

func convertDailyBillingDetailsToDailyCacheEntry(dbd pkg.DailyBillingDetails) DailyCacheEntry {
	dce := DailyCacheEntry{}
	dce.ResourceGroup = dbd.Properties.SubscriptionGUID
	dce.Amount = dbd.Properties.PretaxCost
	dce.StartDate = dbd.Properties.UsageStart
	dce.EndDate = dbd.Properties.UsageEnd

	return dce
}
