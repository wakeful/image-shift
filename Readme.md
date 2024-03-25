# image-shift

> an opinionated tool for swapping / updating container images in AWS ECS task definitions.

this tool speeds up the process of updating container images in ECS task definitions.
After specifying the target service and container, the tool will update the task definition and optionally deploy the new revision to the service.

```shell
image-shift -h
update container image(s) in given service

Usage:
  image-shift [flags]

Examples:
image-shift --cluster-name my-cluster --service api --container app=new-image-test:latest --container proxy=:bump-only-version

Flags:
  -r, --region string         region of your ECS cluster
  -n, --cluster-name string   name of your ECS cluster
  -s, --service string        select service in ECS cluster
  -c, --container strings     Name and version of the container
  -d, --deploy                update & deploy service to new task definition
  -h, --help                  help for image-shift
```

## Example

```shell
# get container image from task definition using aws cli
$ aws ecs describe-task-definition --task-definition my-task-definition --query 'taskDefinition.containerDefinitions[0].image'

"my-reg.dkr.ecr.eu-central-1.amazonaws.com/my-application/backend:v1.2.0"

# update `app` image container to `:v1.2.3` form task definition that is deployed in ECS cluster `ecs-fargate-cluster` under service `api`
$ image-shift -n ecs-fargate-cluster -s api -r eu-central-1 -c app=:v1.2.3
 level=INFO msg="updating container image" container=app old=...backend:v1.2.0 new=...backend:v1.2.3
 level=INFO msg="new task revision created" revision=arn:aws:ecs:...:task-definition/task-example:42
 level=INFO msg="task update / deployment skipped"

# get container image from task definition using aws cli
$ aws ecs describe-task-definition --task-definition my-task-definition --query 'taskDefinition.containerDefinitions[0].image'

"my-reg.dkr.ecr.eu-central-1.amazonaws.com/my-application/backend:v1.2.3"
```
