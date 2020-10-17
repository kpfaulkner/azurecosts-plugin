import React, { ChangeEvent, PureComponent } from 'react';
import { LegacyForms } from '@grafana/ui';
import { DataSourcePluginOptionsEditorProps } from '@grafana/data';
import { MyDataSourceOptions } from './types';

const { FormField } = LegacyForms;

interface Props extends DataSourcePluginOptionsEditorProps<MyDataSourceOptions> {}

interface State {}

export class ConfigEditor extends PureComponent<Props, State> {
  onClientIDChangeChange = (event: ChangeEvent<HTMLInputElement>) => {
    const { onOptionsChange, options } = this.props;
    const jsonData = {
      ...options.jsonData,
      clientID: event.target.value,
    };
    onOptionsChange({ ...options, jsonData });
  };
  onClientSecretChangeChange = (event: ChangeEvent<HTMLInputElement>) => {
    const { onOptionsChange, options } = this.props;
    const jsonData = {
      ...options.jsonData,
      clientSecret: event.target.value,
    };
    onOptionsChange({ ...options, jsonData });
  };
  onTenantIDChangeChange = (event: ChangeEvent<HTMLInputElement>) => {
    const { onOptionsChange, options } = this.props;
    const jsonData = {
      ...options.jsonData,
      tenantID: event.target.value,
    };
    onOptionsChange({ ...options, jsonData });
  };
  onSubscriptionIDChangeChange = (event: ChangeEvent<HTMLInputElement>) => {
    const { onOptionsChange, options } = this.props;
    const jsonData = {
      ...options.jsonData,
      subscriptionID: event.target.value,
    };
    onOptionsChange({ ...options, jsonData });
  };

  onResetAPIKey = () => {
    const { onOptionsChange, options } = this.props;
    onOptionsChange({
      ...options,
      secureJsonFields: {
        ...options.secureJsonFields,
        apiKey: false,
      },
      secureJsonData: {
        ...options.secureJsonData,
        apiKey: '',
      },
    });
  };

  render() {
    const { options } = this.props;
    const { jsonData } = options;
    //const secureJsonData = (options.secureJsonData || {}) as MySecureJsonData;

    return (
      <div className="gf-form-group">
        <div className="gf-form">
          <FormField
            label="ClientID"
            labelWidth={6}
            inputWidth={20}
            onChange={this.onClientIDChangeChange}
            value={jsonData.clientID || ''}
            placeholder="Azure ClientID"
          />
        </div>
        <div className="gf-form">
          <FormField
            label="ClientSecret"
            labelWidth={6}
            inputWidth={20}
            onChange={this.onClientSecretChangeChange}
            value={jsonData.clientSecret || ''}
            placeholder="Azure ClientSecret"
          />
        </div>
        <div className="gf-form">
          <FormField
            label="TenantID"
            labelWidth={6}
            inputWidth={20}
            onChange={this.onTenantIDChangeChange}
            value={jsonData.tenantID || ''}
            placeholder="Azure TenantID"
          />
        </div>
        <div className="gf-form">
          <FormField
            label="Default SubscriptionID"
            labelWidth={6}
            inputWidth={20}
            onChange={this.onSubscriptionIDChangeChange}
            value={jsonData.subscriptionID || ''}
            placeholder="Azure SubscriptionID"
          />
        </div>

      </div>
    );
  }
}
