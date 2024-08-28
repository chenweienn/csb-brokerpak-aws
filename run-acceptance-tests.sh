#!/usr/bin/env bash

set -ex


if [[ -z ${AWS_ACCESS_KEY_ID} ]]; then
  AWS_ACCESS_KEY_ID=$(vault kv get -field aws_access_key_id runway_concourse/service-enablement/aws_automation_admin_user ) 
  export AWS_ACCESS_KEY_ID
  AWS_SECRET_ACCESS_KEY=$(vault kv get -field aws_secret_access_key runway_concourse/service-enablement/aws_automation_admin_user ) 
  export AWS_SECRET_ACCESS_KEY
  AWS_DEFAULT_REGION=$(vault kv get -field default_region runway_concourse/service-enablement/aws_automation_admin_user ) 
  export AWS_DEFAULT_REGION
fi 
ORG=${ORG:-pivotal}
SPACE=${SPACE:-broker-cf-test}

ENVIRONMENT_LOCK_METADATA=${ENVIRONMENT_LOCK_METADATA:-environment/metadata}
ENV_NAME=$(jq .name -r < "${ENVIRONMENT_LOCK_METADATA}")
# shellcheck disable=1090
source <(smith om -l "${ENVIRONMENT_LOCK_METADATA}")
# shellcheck disable=1090
source <(smith bosh -l "${ENVIRONMENT_LOCK_METADATA}")
smith -l "${ENVIRONMENT_LOCK_METADATA}" cf-login <<< "${ORG}" &> /dev/null
DB_PASSWORD="$(echo "${ENV_NAME}" | sha256sum | cut -f1 -d' ')"
ENCRYPTION_PASSWORDS='[{"password": {"secret":"'${DB_PASSWORD}'"},"label":"first-encryption","primary":true}]'
if ! cf service-key  csb-sql csb-sql; then
  cf create-service-key csb-sql csb-sql
fi
CSB_DB_DATA_RAW=$( cf service-key  csb-sql csb-sql | tail -n+2)
CSB_DB_DATA=$( \
jq ".credentials | 
    { 
      host: .hostname, 
      encryption: { enabled: true, passwords: $ENCRYPTION_PASSWORDS }, 
      ca: { cert: .tls.cert.ca}, 
      name: .name, 
      user: .username, 
      password: .password,
      port: .port
    }" <<< "${CSB_DB_DATA_RAW}"  
)
PRODS="$(om -k curl -s -p /api/v0/staged/products)"
CF_DEPLOYMENT_ID="$(echo "$PRODS" | jq -r '.[] | select(.type == "cf") | .guid')"
UAA_CREDS="$(om -k curl -s -p "/api/v0/deployed/products/$CF_DEPLOYMENT_ID/credentials/.uaa.credhub_admin_client_client_credentials")"

CH_UAA_CLIENT_NAME="$(echo "${UAA_CREDS}" | jq -r .credential.value.identity)"
CH_UAA_CLIENT_SECRET="$(echo "${UAA_CREDS}" | jq -r .credential.value.password)"
CH_UAA_URL="https://uaa.service.cf.internal:8443"
CH_CRED_HUB_URL="https://credhub.service.cf.internal:8844"
AWS_PAS_VPC_ID="$(jq -r .pas_vpc_id "${ENVIRONMENT_LOCK_METADATA}")"
export AWS_PAS_VPC_ID
CF_API_PASS=$( credhub get --key password -n "/opsmgr/$CF_DEPLOYMENT_ID/uaa/admin_credentials" -j )


GSB_PROVISION_DEFAULTS='{"aws_vpc_id": "'${AWS_PAS_VPC_ID}'", "region": "'${AWS_DEFAULT_REGION}'"}'
GSB_SERVICE_CSB_AWS_S3_BUCKET_PLANS='[{"name":"default","id":"f64891b4-5021-4742-9871-dfe1a9051302","description":"Default S3 plan","display_name":"default"}]'
GSB_SERVICE_CSB_AWS_POSTGRESQL_PLANS='[{"name":"default","id":"de7dbcee-1c8d-11ed-9904-5f435c1e2316","description":"Default Postgres plan","display_name":"default","instance_class":"db.t3.micro","postgres_version":"14","storage_gb":100},{"name":"pg15","id":"eef1bd55-3eb7-4b01-ae3c-715cc64f4c05","description":"Postgres 15 plan","display_name":"pg15","instance_class":"db.t3.micro","postgres_version":"15","storage_gb":5, "storage_type":"standard"},{"name":"pg16","id":"a7b75f73-82d1-4c9e-a288-100e50154403","description":"Postgres 16 plan","display_name":"pg16","instance_class":"db.t3.micro","postgres_version":"16","storage_gb":5,"storage_type":"standard"}]'
GSB_SERVICE_CSB_AWS_AURORA_POSTGRESQL_PLANS='[{"name":"default","id":"d20c5cf2-29e1-11ed-93da-1f3a67a06903","description":"Default Aurora Postgres plan","display_name":"default"}]'
GSB_SERVICE_CSB_AWS_AURORA_MYSQL_PLANS='[{"name":"default","id":"10b2bd92-2a0b-11ed-b70f-c7c5cf3bb719","description":"Default Aurora MySQL plan","display_name":"default"}]'
GSB_SERVICE_CSB_AWS_MYSQL_PLANS='[{"name":"default","id":"0f3522b2-f040-443b-bc53-4aed25284840","description":"Default MySQL plan","display_name":"default","instance_class":"db.t3.micro","mysql_version":"8.0","storage_gb":100}]'
GSB_SERVICE_CSB_AWS_REDIS_PLANS='[{"name":"default", "id":"c7f64994-a1d9-4e1f-9491-9d8e56bbf146","description":"Default Redis plan","display_name":"default","node_type":"cache.t3.medium","redis_version": "6.0"},{"name" : "example-with-flexible-node-type","id" : "2deb6c13-7ea1-4bad-a519-0ac9600e9a29","description" : "An example of a Redis plan for which node_type can be specified at provision time. Replace with your own plan configuration.","redis_version" : "6.x","node_count" : 2}]'
GSB_SERVICE_CSB_AWS_MSSQL_PLANS='[{"name":"default","id":"7400cd8f-5f98-4457-8de0-03232ec12f62","description":"Default MSSQL plan","display_name":"default","engine":"sqlserver-se","mssql_version":"15.00","storage_gb":100, "instance_class":"db.r5.large" }]'
GSB_SERVICE_CSB_AWS_SQS_PLANS='[{"name":"standard","id":"c2fdfc84-bf86-11ee-a4f5-8b0d531ce7e2","description":"Default SQS standard queue plan","display_name":"standard"},{"name":"fifo","id":"093c1060-c1c0-11ee-8b97-ff07a1127dae","description":"Default SQS FIFO queue plan","display_name":"fifo","fifo":true}]'
GSB_BROKERPAK_CONFIG='{"global_labels":[{"key":"key1","value":"value1"},{"key":"key2","value":"value2"}]}'


cat << EOF > acceptance-tests/assets/vars.yml
env_name: $ENV_NAME
azs:
- us-west-2c

aws:
  access_key_id: $AWS_ACCESS_KEY_ID
  secret_access_key: $AWS_SECRET_ACCESS_KEY

api:
  password: broker-test-password
  user: broker-test

broker_bosh_dns_domain: api.cloud-service-broker.service.internal

broker:
  enable_global_access_to_plans: true
  name: cloud-service-broker-aws
  password: broker-test
  url: api.cloud-service-broker.service.internal


cf:
  admin_password: ${CF_API_PASS}
  admin_user: admin
  app_name: cloud-service-broker-aws
  org: system
  space: cloud-service-broker-space
  url: api.sys.berry-pirate.csb.cf-app.com

brokerpak:
  builtin:
    path: ./
  config: ${GSB_BROKERPAK_CONFIG}
  sources: |
    {}
  terraform:
    upgrades:
      enabled: true
  updates:
    enabled: true
compatibility:
  enable-beta-services: true
credhub:
  ca_cert_file: credhub_ca_cert.pem
  skip_ssl_validation: false
  uaa_client_name: ${CH_UAA_CLIENT_NAME}
  uaa_client_secret: ${CH_UAA_CLIENT_SECRET}
  uaa_url: ${CH_UAA_URL}
  url: ${CH_CRED_HUB_URL}
db: ${CSB_DB_DATA}

provision:
  defaults: |
    ${GSB_PROVISION_DEFAULTS}
request:
  property:
    validation:
      disabled: false
service:
  csb-aws-aurora-mysql:
    plans: |+
      ${GSB_SERVICE_CSB_AWS_AURORA_MYSQL_PLANS}
  csb-aws-aurora-postgresql:
    plans: |+
      ${GSB_SERVICE_CSB_AWS_AURORA_POSTGRESQL_PLANS}
  csb-aws-dynamodb-table: |+
    [
    ]

  csb-aws-mssql:
    plans: |+
      ${GSB_SERVICE_CSB_AWS_MSSQL_PLANS}
  csb-aws-mysql:
    plans: |+
      ${GSB_SERVICE_CSB_AWS_MYSQL_PLANS}
  csb-aws-postgresql:
    plans: |+
      ${GSB_SERVICE_CSB_AWS_POSTGRESQL_PLANS}
  csb-aws-redis:
    plans: |+
      ${GSB_SERVICE_CSB_AWS_REDIS_PLANS}
  csb-aws-s3-bucket:
    plans: |+
      ${GSB_SERVICE_CSB_AWS_S3_BUCKET_PLANS}
  csb-aws-sqs:
    plans: |+
      ${GSB_SERVICE_CSB_AWS_SQS_PLANS}

EOF
bosh -d cloud-service-broker-aws-d7d1fb61212b1a66aabe deploy ./acceptance-tests/assets/manifest.yml  -l ./acceptance-tests/assets/vars.yml -v name=cloud-service-broker-aws-d7d1fb61212b1a66aabe -v release_repo_path="$(pwd)/../csb-aws-release/" --no-redact -n

exit 0

#ginkgo -r acceptance-tests/

