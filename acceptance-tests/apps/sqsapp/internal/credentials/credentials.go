package credentials

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/cloudfoundry-community/go-cfenv"
	"github.com/mitchellh/mapstructure"
)

type Credentials struct {
	AccessKeyID     string `mapstructure:"access_key_id"`
	SecretAccessKey string `mapstructure:"secret_access_key"`
	Region          string `mapstructure:"region"`
	ARN             string `mapstructure:"arn"`
	URL             string `mapstructure:"url"`
}

func Read() (Credentials, error) {
	app, err := cfenv.Current()
	if err != nil {
		return Credentials{}, fmt.Errorf("error reading app env: %w", err)
	}
	svs, err := app.Services.WithTag("sqs")
	if err != nil {
		return Credentials{}, fmt.Errorf("error reading Redis service details")
	}

	var r Credentials
	if err := mapstructure.Decode(svs[0].Credentials, &r); err != nil {
		return Credentials{}, fmt.Errorf("failed to decode credentials: %w", err)
	}

	if r.AccessKeyID == "" || r.SecretAccessKey == "" || r.Region == "" || r.ARN == "" || r.URL == "" {
		return Credentials{}, fmt.Errorf("parsed credentials are not valid")
	}

	return r, nil
}

func (c Credentials) Config() (aws.Config, error) {
	return config.LoadDefaultConfig(
		context.Background(),
		config.WithCredentialsProvider(aws.NewCredentialsCache(credentials.NewStaticCredentialsProvider(c.AccessKeyID, c.SecretAccessKey, ""))),
		config.WithRegion(c.Region),
	)
}
