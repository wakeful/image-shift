package sub

import (
	"context"
	"errors"
	"fmt"
	"image-shift/pkg/ecs"
	"log/slog"
	"strings"

	"github.com/spf13/cobra"
)

type Shift struct {
	cluster    string
	service    string
	containers []string
	deploy     bool
	logger     *slog.Logger
	region     string
}

func (s *Shift) run(_ *cobra.Command, _ []string) error {
	ctx := context.Background()

	client, err := ecs.NewClient(ctx, s.logger, s.region)
	if err != nil {
		s.logger.Error("failed to create ECS client", "error", err)

		return fmt.Errorf("failed to create ECS client: %w", err)
	}

	task, err := client.GetTask(ctx, s.cluster, s.service)
	if err != nil {
		s.logger.Error("failed to get task definition", "error", err)

		return fmt.Errorf("failed to get task definition: %w", err)
	}

	remapContainers := parseUpdates(s.containers, s.logger)
	if len(remapContainers) == 0 {
		s.logger.Warn("no containers to update")

		return nil
	}

	revision, err := client.NewTaskRevision(ctx, task, remapContainers)
	if err != nil {
		s.logger.Error("failed to create new task revision", "error", err)

		return fmt.Errorf("failed to create new task revision: %w", err)
	}

	s.logger.Info("new task revision created", "revision", *revision)

	if !s.deploy {
		s.logger.Info("task update / deployment skipped")

		return nil
	}

	if errDeploy := client.DeployTaskARN(ctx, s.cluster, s.service, *revision); errDeploy != nil {
		s.logger.Error("failed to deploy new task revision", "error", errDeploy)

		return fmt.Errorf("failed to deploy new task revision: %w", errDeploy)
	}

	return nil
}

func parseUpdates(input []string, logger *slog.Logger) map[string]string {
	output := make(map[string]string)

	for _, mapping := range input {
		split := strings.Split(mapping, "=")
		if len(split) != 2 {
			logger.Warn("invalid container mapping, should be in format: container-name=image-name:tag or container-name=:tag")

			continue
		}

		if split[0] == "" || split[1] == "" {
			logger.Warn("invalid container mapping", "item", mapping, "reason", "missing container name or image name")
			logger.Warn("invalid container mapping, should be in format: container-name=image-name:tag or container-name=:tag")

			continue
		}

		output[split[0]] = split[1]
	}

	return output
}

var (
	errArgClusterNameRequired = errors.New("the --cluster-name flag is required")
	errArgServiceNameRequired = errors.New("the --service flag is required")
	errArgRegionRequired      = errors.New("the --region flag is required")
)

func NewShiftCmd(logger *slog.Logger) *cobra.Command {
	shiftCmd := &Shift{
		logger: logger,
	}

	serviceCmd := &cobra.Command{
		Use:   "image-shift",
		Short: "update container images in given service",
		CompletionOptions: cobra.CompletionOptions{
			DisableDefaultCmd: true,
		},
		SilenceUsage: true,
		RunE:         shiftCmd.run,
		Example:      "image-shift --cluster-name my-cluster --service api --container app=new-image-test:latest --container proxy=:bump-only-version",
		PreRunE: func(_ *cobra.Command, _ []string) error {
			if shiftCmd.region == "" {
				return errArgRegionRequired
			}

			if shiftCmd.cluster == "" {
				return errArgClusterNameRequired
			}

			if shiftCmd.service == "" {
				return errArgServiceNameRequired
			}

			return nil
		},
	}

	serviceCmd.Flags().SortFlags = false

	serviceCmd.Flags().StringVarP(
		&shiftCmd.region,
		"region",
		"r",
		"",
		"region of your ECS cluster",
	)
	serviceCmd.Flags().StringVarP(
		&shiftCmd.cluster,
		"cluster-name",
		"n",
		"",
		"name of your ECS cluster",
	)
	serviceCmd.Flags().StringVarP(
		&shiftCmd.service,
		"service",
		"s",
		"",
		"select service in ECS cluster",
	)
	serviceCmd.Flags().StringSliceVarP(
		&shiftCmd.containers,
		"container",
		"c",
		[]string{},
		"Name and version of the container",
	)
	serviceCmd.Flags().BoolVarP(
		&shiftCmd.deploy,
		"deploy",
		"d",
		false,
		"update & deploy service to new task definition",
	)

	return serviceCmd
}
