# AzureCosts-plugin for Grafana

This is an initial attempt for retrieving the Azure Costs for a subscription and supply data to Grafana. Very early stages, but DOES work. Need to determine signed vs unsigned plugins but that's probably it.


## Description

This Grafana datasource plugin reads Azure billing API.

## Configuration



## Installation

Currently this plugin is NOT signed (Grafana are still determining the best approach for having 3rd party plugins signed), so if you want to run this plugin there are couple of steps required

- Modify the Grafana defaults.ini file, in the [paths] section and include a line similar to: *plugins = "C:\temp\grafana-plugins"*  (or whichever path you install this plugin)
- Also modify defaults.ini (in [plugins] section) and modify to have a line similar to: *allow_loading_unsigned_plugins = kpfaulkner-azurecosts-plugin*  . This will allow a non-signed plugin to be executed. See [the docs](https://grafana.com/docs/grafana/latest/administration/configuration/#allow-loading-unsigned-plugins) for details. (note: you don't have to modify default.ini but can also choose other .ini files)
- For *nix systems, make sure the generated executable (gpx_azurecosts-plugin*) has the executable file flag set.
