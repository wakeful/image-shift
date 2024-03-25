package ecs

import (
	"context"
	"fmt"
	"log/slog"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/ecs"
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
) (*string, error) {
	tags, err := c.client.ListTagsForResource(ctx, &ecs.ListTagsForResourceInput{
		ResourceArn: input.TaskDefinition.TaskDefinitionArn,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to list tags for resource, %w", err)
	}

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
	}

	output, err := c.client.RegisterTaskDefinition(ctx, &ecs.RegisterTaskDefinitionInput{
		Family:                  input.TaskDefinition.Family,
		TaskRoleArn:             input.TaskDefinition.TaskRoleArn,
		ExecutionRoleArn:        input.TaskDefinition.ExecutionRoleArn,
		NetworkMode:             input.TaskDefinition.NetworkMode,
		ContainerDefinitions:    newDefinition,
		RequiresCompatibilities: input.TaskDefinition.RequiresCompatibilities,
		Cpu:                     input.TaskDefinition.Cpu,
		Memory:                  input.TaskDefinition.Memory,
		Tags:                    tags.Tags,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to register task definition, %w", err)
	}

	return output.TaskDefinition.TaskDefinitionArn, nil
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
		return nil, fmt.Errorf("service: %s not found in cluster: %s", service, cluster)
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
