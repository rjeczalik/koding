globals = require 'globals'
isProd  = globals.config.environment is 'production'
baseURL = globals.config.domains.base

module.exports = globals.config.providers =

  custom                   :
    name                   : 'Custom'
    link                   : "https://#{baseURL}"
    title                  : 'Custom Credential'
    color                  : '#b9c0b8'
    description            : '''Custom credentials can include meta
                               credentials for any service'''
    listText               : """
                              You're currently using these custom data in your
                              stack templates, you can change their contents
                              without touching stack templates.
                            """
    credentialFields       :
      credential           :
        label              : 'Credential'
        placeholder        : 'credential in JSON format'
        type               : 'textarea'

  aws                      :
    name                   : 'Amazon Web Services'
    link                   : 'https://aws.amazon.com'
    title                  : 'AWS'
    supported              : yes
    enabled                : yes
    color                  : '#F9A900'
    description            : 'Amazon Web Services'
    defaultTemplate        : require './templates/aws'
    advancedFields         : [
                              'subnet', 'sg', 'vpc',
                              'ami', 'acl', 'cidr_block',
                              'igw', 'rtb'
                             ]
    credentialFields       :
      access_key           :
        label              : 'Access Key ID'
        placeholder        : 'aws access key'
        attributes         :
          autocomplete     : if isProd then 'off' else 'on'
      secret_key           :
        label              : 'Secret Access Key'
        placeholder        : 'aws secret key'
        attributes         :
          autocomplete     : if isProd then 'off' else 'on'
      region               :
        label              : 'Region'
        type               : 'selection'
        placeholder        : 'Region'
        defaultValue       : 'us-east-1'
        values             : [
          { title: 'US East (N. Virginia) (us-east-1)',         value: 'us-east-1' }
          { title: 'US West (Oregon) (us-west-2)',              value: 'us-west-2' }
          { title: 'US West (N. California) (us-west-1)',       value: 'us-west-1' }
          { title: 'EU (Ireland) (eu-west-1)',                  value: 'eu-west-1' }
          { title: 'EU (Frankfurt) (eu-central-1)',             value: 'eu-central-1' }
          { title: 'Asia Pacific (Singapore) (ap-southeast-1)', value: 'ap-southeast-1' }
          { title: 'Asia Pacific (Sydney) (ap-southeast-2)',    value: 'ap-southeast-2' }
          { title: 'Asia Pacific (Tokyo) (ap-northeast-1)',     value: 'ap-northeast-1' }
          { title: 'South America (Sao Paulo) (sa-east-1)',     value: 'sa-east-1' }
        ]

  vagrant                  :
    name                   : 'Vagrant'
    link                   : 'http://www.vagrantup.com'
    title                  : 'Vagrant on Local'
    color                  : '#B52025'
    supported              : yes
    enabled                : 'beta'
    defaultTemplate        : require './templates/vagrant'
    description            : 'Local provisioning with Vagrant'
    credentialFields       :
      queryString          :
        label              : 'Kite ID'
        placeholder        : 'ID for my local machine kite'

  koding                   :
    name                   : 'Koding'
    link                   : 'https://koding.com'
    title                  : 'Koding'
    color                  : '#50c157'
    description            : 'Koding rulez.'
    credentialFields       : {}

  managed                  :
    name                   : 'Managed VMs'
    link                   : "https://#{baseURL}"
    title                  : 'Managed VM'
    color                  : '#6d119e'
    description            : 'Use your power.'
    credentialFields       : {}

  google                   :
    name                   : 'Google Compute Engine'
    link                   : 'https://cloud.google.com/compute/'
    title                  : 'GCE' # or Google Cloud or Google Compute Engine or ...
    color                  : '#357e99' # dunno
    supported              : yes
    enabled                : 'beta'
    defaultTemplate        : require './templates/google'
    description            : 'Google compute engine'
    advancedFields         : []
    credentialFields       :
      project              :
        label              : 'Project ID'
        placeholder        : 'ID of Project'
        attributes         :
          autocomplete     : if isProd then 'off' else 'on'
      credentials          :
        label              : 'Service account JSON key'
        placeholder        : 'Provide content of key in JSON format'
        type               : 'textarea'
      region               :
        label              : 'Region'
        type               : 'selection'
        placeholder        : '' # dunno
        defaultValue       : 'us-central1'
        values             : [
          { title: 'Western US (us-west1)',         value: 'us-west1' }
          { title: 'Central US (us-central1)',      value: 'us-central1' }
          { title: 'Eastern US (us-east1)',         value: 'us-east1' }
          { title: 'Western Europe (europe-west1)', value: 'europe-west1' }
          { title: 'Eastern Asia (asia-east1)',     value: 'asia-east1' }
        ]

  digitalocean             :
    name                   : 'Digital Ocean'
    link                   : 'https://digitalocean.com'
    title                  : 'Digitalocean'
    color                  : '#7abad7'
    supported              : yes
    slug                   : 'do'
    enabled                : 'beta'
    defaultTemplate        : require './templates/digitalocean'
    description            : 'Digital Ocean droplets'
    credentialFields       :
      access_token         :
        label              : 'Personal Access Token'
        placeholder        : 'Digital Ocean access token'

  azure                    :
    name                   : 'Azure'
    link                   : 'https://azure.microsoft.com/'
    title                  : 'Azure'
    color                  : '#ec06be'
    supported              : yes
    enabled                : no
    description            : 'Azure'
    credentialFields       :
      accountId            :
        label              : 'Account Id'
        placeholder        : 'account id in azure'
      secret               :
        label              : 'Secret'
        placeholder        : 'azure secret'
        type               : 'password'

  rackspace                :
    name                   : 'Rackspace'
    link                   : 'http://www.rackspace.com'
    title                  : 'Rackspace'
    color                  : '#d8deea'
    supported              : no
    enabled                : no
    description            : 'Rackspace machines'
    credentialFields       :
      username             :
        label              : 'Username'
        placeholder        : 'username for rackspace'
      apiKey               :
        label              : 'API Key'
        placeholder        : 'rackspace api key'

  softlayer                :
    name                   : 'Softlayer'
    link                   : 'http://www.softlayer.com'
    title                  : 'Softlayer'
    color                  : '#B52025'
    supported              : yes
    enabled                : no
    description            : 'Softlayer resources'
    credentialFields       :
      username             :
        label              : 'Username'
        placeholder        : 'username for softlayer'
      api_key              :
        label              : 'API Key'
        placeholder        : 'softlayer api key'

  userInput                :
    name                   : 'User Input'
    title                  : 'User Input'
    listText               : '''
                            Here you can change user input fields that you define
                            in your stack scripts. When you delete these,
                            make sure that you update the stack scripts that
                            these are used in. Otherwise you may experience
                            unwanted results while building your stacks.
                            '''
    credentialFields       : {}
