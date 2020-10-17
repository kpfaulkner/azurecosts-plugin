
import defaults from 'lodash/defaults';
import React, { ChangeEvent, PureComponent } from 'react';
import {LegacyForms} from '@grafana/ui';

import {QueryEditorProps} from '@grafana/data';
import { DataSource } from './DataSource';
import {defaultQuery,  MyDataSourceOptions, MyQuery } from './types';


const {  FormField } = LegacyForms;

type Props = QueryEditorProps<DataSource, MyQuery, MyDataSourceOptions>;


export class QueryEditor extends PureComponent<Props> {
  onQueryTextChange = (event: ChangeEvent<HTMLInputElement>) => {
    const { onChange, query } = this.props;
    onChange({ ...query, queryText: event.target.value });
  };

  onRGTextChange = (event: ChangeEvent<HTMLInputElement>) => {
    const { onChange, query } = this.props;
    onChange({ ...query, rgText: event.target.value });
  };

  onConstantChange = (event: ChangeEvent<HTMLInputElement>) => {
    const { onChange, query, onRunQuery } = this.props;
    onChange({ ...query, constant: parseFloat(event.target.value) });
    // executes the query
    onRunQuery();
  };

  render() {
    const query = defaults(this.props.query, defaultQuery);
    const { queryText } = query;
    const { rgText } = query;
    return (
      <div className="gf-form">
        <FormField
          labelWidth={8}
          value={queryText || ''}
          onChange={this.onQueryTextChange}
          label="Subscription ID"
          tooltip="Subscription ID"
          inputWidth={100}
        />
        <FormField
          labelWidth={8}
          value={rgText || ''}
          onChange={this.onRGTextChange}
          label="Split on RG : Y/N"
          tooltip="Split on Resource Groups"
          inputWidth={100}
        />

      </div>
    );
  }
}
