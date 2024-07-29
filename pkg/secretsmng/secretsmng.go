package secretsmng

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/secretsmanager"
)

func GetSecrets(ctx context.Context, region string, secrets []string) (map[string]string, error) {
	output := make(map[string]string)
	if len(secrets) == 0 {
		return output, nil
	}

	sConf, err := config.LoadDefaultConfig(ctx, config.WithRegion(region))
	if err != nil {
		return nil, fmt.Errorf("failed to load configuration, %w", err)
	}

	sClient := secretsmanager.NewFromConfig(sConf)

	value, err := sClient.BatchGetSecretValue(ctx, &secretsmanager.BatchGetSecretValueInput{
		SecretIdList: secrets,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to list secrets, %w", err)
	}

	for _, entry := range value.SecretValues {
		var res map[string]string

		if errJSON := json.Unmarshal([]byte(*entry.SecretString), &res); errJSON != nil {
			return nil, fmt.Errorf("failed to unmarshal secret, %w", errJSON)
		}

		for k := range res {
			output[k] = fmt.Sprintf("%s:%s::", *entry.ARN, k)
		}
	}

	return output, nil
}
