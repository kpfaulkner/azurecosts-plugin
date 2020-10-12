import { DataQuery, DataSourceJsonData } from '@grafana/data';

export interface MyQuery extends DataQuery {
  queryText?: string;
  constant: number;
}

export const defaultQuery: Partial<MyQuery> = {
  constant: 6.5,
};

export interface MyDataSourceOptions extends DataSourceJsonData {
  clientID?: string;
  clientSecret?: string;
  tenantID?: string;
  subscriptionID?: string;

}


