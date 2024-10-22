package ecs

import (
	"context"
	"fmt"
	"log/slog"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/ecs"
	"github.com/aws/aws-sdk-go-v2/service/ecs/types"
)

type Client struct {
	client *ecs.Client
	logger *slog.Logger
}

// NewClient creates a new ECS client.
func NewClient(ctx context.Context, logger *slog.Logger, region string) (*Client, error) {
	cfg, err := config.LoadDefaultConfig(ctx, config.WithRegion(region))
	if err != nil {
		return nil, fmt.Errorf("failed to load configuration, %w", err)
	}

	return &Client{
		client: ecs.NewFromConfig(cfg),
		logger: logger,
	}, nil
}

// NewTaskRevision creates a new task definition revision with updated container(s) image(s).
func (c *Client) NewTaskRevision(
	ctx context.Context,
	input *ecs.DescribeTaskDefinitionOutput,
	images map[string]string,
	secrets map[string]string,
) (*string, error) {
	tags, err := c.client.ListTagsForResource(ctx, &ecs.ListTagsForResourceInput{
		ResourceArn: input.TaskDefinition.TaskDefinitionArn,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to list tags for resource, %w", err)
	}

	secretMapping := map2secrets(secrets)

	newDefinition := input.TaskDefinition.ContainerDefinitions
	for position, container := range newDefinition {
		if image, ok := images[*container.Name]; ok {
			newImage := image

			if strings.HasPrefix(image, ":") {
				oldIMG := strings.Split(*container.Image, ":")
				newImage = oldIMG[0] + image
			}

			c.logger.Info(
				"updating container image",
				"container",
				*container.Name,
				"old",
				*container.Image,
				"new",
				newImage,
			)

			newDefinition[position].Image = aws.String(newImage)
		}

		if len(newDefinition[position].Environment) != 0 {
			var newEnv []types.KeyValuePair

			for _, n := range newDefinition[position].Environment {
				if _, ok := secrets[*n.Name]; ok {
					continue
				}

				newEnv = append(newEnv, n)
			}

			newDefinition[position].Environment = newEnv
		}

		if len(secretMapping) != 0 {
			newDefinition[position].Secrets = secretMapping
		}
	}

	output, err := c.client.RegisterTaskDefinition(ctx, &ecs.RegisterTaskDefinitionInput{
		ContainerDefinitions:    newDefinition,
		Family:                  input.TaskDefinition.Family,
		Cpu:                     input.TaskDefinition.Cpu,
		EphemeralStorage:        input.TaskDefinition.EphemeralStorage,
		ExecutionRoleArn:        input.TaskDefinition.ExecutionRoleArn,
		InferenceAccelerators:   input.TaskDefinition.InferenceAccelerators,
		IpcMode:                 input.TaskDefinition.IpcMode,
		Memory:                  input.TaskDefinition.Memory,
		NetworkMode:             input.TaskDefinition.NetworkMode,
		PidMode:                 input.TaskDefinition.PidMode,
		PlacementConstraints:    input.TaskDefinition.PlacementConstraints,
		ProxyConfiguration:      input.TaskDefinition.ProxyConfiguration,
		RequiresCompatibilities: input.TaskDefinition.RequiresCompatibilities,
		RuntimePlatform:         input.TaskDefinition.RuntimePlatform,
		Tags:                    tags.Tags,
		TaskRoleArn:             input.TaskDefinition.TaskRoleArn,
		Volumes:                 input.TaskDefinition.Volumes,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to register task definition, %w", err)
	}

	return output.TaskDefinition.TaskDefinitionArn, nil
}

// map2secrets change map[string]string into []types.Secret.
func map2secrets(secrets map[string]string) []types.Secret {
	output := make([]types.Secret, 0, len(secrets))
	for k, v := range secrets {
		output = append(output, types.Secret{
			Name:      aws.String(k),
			ValueFrom: aws.String(v),
		})
	}

	return output
}

type ServiceNotFoundError struct {
	Service string
	Cluster string
}

func (e *ServiceNotFoundError) Error() string {
	return fmt.Sprintf("service %s not found in cluster %s", e.Service, e.Cluster)
}

// GetTask return task definition for given service in ECS cluster.
func (c *Client) GetTask(ctx context.Context, cluster, service string) (*ecs.DescribeTaskDefinitionOutput, error) {
	result, err := c.client.DescribeServices(ctx, &ecs.DescribeServicesInput{
		Cluster:  aws.String(cluster),
		Services: []string{service},
	})
	if err != nil {
		return nil, fmt.Errorf("failed to describe service, %w", err)
	}

	if len(result.Services) == 0 {
		return nil, &ServiceNotFoundError{
			Service: service,
			Cluster: cluster,
		}
	}

	output, err := c.client.DescribeTaskDefinition(ctx, &ecs.DescribeTaskDefinitionInput{
		TaskDefinition: result.Services[0].TaskDefinition,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to describe task definition, %w", err)
	}

	return output, nil
}

// DeployTaskARN updates the service with the new task definition ARN.
func (c *Client) DeployTaskARN(ctx context.Context, cluster, service, taskARN string) error {
	_, err := c.client.UpdateService(ctx, &ecs.UpdateServiceInput{
		Cluster:        aws.String(cluster),
		Service:        aws.String(service),
		TaskDefinition: aws.String(taskARN),
	})
	if err != nil {
		return fmt.Errorf("failed to update service, %w", err)
	}

	return nil
}
