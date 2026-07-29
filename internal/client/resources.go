package client

import (
	"context"
	"errors"
)

const resourcesQuery = `query ListResources($filter: JSON, $sort: [String!], $range: [Int!]) {
  resources(filter: $filter, sort: $sort, range: $range) {
    id
    name
    entityName
    description
    state
    status
    createdAt
    updatedAt
    labels
    template {
      id
      name
      cloudResourceTypes
    }
    workspace {
      id
      name
    }
    storage {
      id
      name
    }
    creator {
      id
      identifier
      email
      displayName
    }
    integrationIds {
      id
      name
      integrationProvider
      integrationType
    }
    secretIds {
      id
      name
    }
    revisionNumber
    storagePath
    sourceCodeVersion {
      id
      identifier
      sourceCodeFolder
      sourceCodeVersion
      sourceCodeBranch
      status
    }
    parents {
      id
      name
      state
      status
    }
    children {
      id
      name
      state
      status
    }
    variables
    outputs
    labels
    dependencyTags
    dependencyConfig
  }
  resourcesCount(filter: $filter)
}`

const templatesQuery = `query ListTemplates($filter: JSON, $sort: [String!], $range: [Int!]) {
  templates(filter: $filter, sort: $sort, range: $range) {
    id
    name
    description
    labels
    status
    abstract
    createdAt
    updatedAt
    cloudResourceTypes
    entityName
  }
  templatesCount(filter: $filter)
}`

const projectsQuery = `query ListProjects($filter: JSON, $sort: [String!], $range: [Int!]) {
  projects(filter: $filter, sort: $sort, range: $range) {
    id
    name
    description
    labels
    status
	    resourcesCount
    createdAt
    updatedAt
    entityName
  }
  projectsCount(filter: $filter)
}`

const resourceQuery = `query GetResource($id: UUID!) {
	resource(id: $id) {
		id
		name
		entityName
		description
		state
		status
		createdAt
    updatedAt
    revisionNumber
    abstract
    storagePath
    labels
    variables
    outputs
    dependencyTags
    dependencyConfig
    template {
      id
      name
      cloudResourceTypes
    }
    workspace {
      id
      name
    }
    storage {
      id
      name
    }
    creator {
      id
      identifier
      email
      displayName
    }
    integrationIds {
      id
      name
      integrationProvider
      integrationType
    }
    secretIds {
      id
      name
    }
    sourceCodeVersion {
      id
      identifier
      sourceCodeFolder
      sourceCodeVersion
      sourceCodeBranch
      status
    }
    parents {
      id
      name
      state
      status
    }
    children {
      id
      name
      state
      status
    }
  }
}`

const resourceActionsQuery = `query ResourceActions($id: UUID!) {
  resourceActions(id: $id)
}`

const resourceTempStateQuery = `query ResourceTempState($id: UUID!) {
  resourceTempStateByResource(resourceId: $id) {
    id
    resourceId
    value
    createdAt
    updatedAt
  }
}`

const templateQuery = `query GetTemplate($id: UUID!) {
  template(id: $id) {
    id
    name
    description
    documentation
    template
    createdAt
    updatedAt
    cloudResourceTypes
    labels
    status
    abstract
    configuration
    revisionNumber
    resourcesCount
    sourceCodeVersionsCount
    entityName
    creator {
      id
      identifier
      email
      displayName
    }
    parents {
      id
      name
      abstract
      cloudResourceTypes
      entityName
    }
    children {
      id
      name
      abstract
      cloudResourceTypes
      entityName
    }
  }
}`

const projectQuery = `query GetProject($id: UUID!) {
  project(id: $id) {
    id
    name
    description
	    workspaceId
	    configuration
	    dependencyTags
	    dependencyConfig
	    labels
	    status
	    revisionNumber
	    resourcesCount
    createdAt
    updatedAt
    entityName
    creator {
      id
      identifier
      email
      displayName
    }
	    owners {
      id
	      identifier
	      email
	      displayName
    }
	    workspace {
      id
      name
    }
  }
}`

const templateTreeQuery = `query TemplateTree($id: UUID!, $direction: String!) {
  templateTree(id: $id, direction: $direction) {
    id
    nodeId
    name
    status
    children {
      id
      nodeId
      name
      status
      children {
        id
        nodeId
        name
        status
        children {
          id
          nodeId
          name
          status
          children {
            id
            nodeId
            name
            status
          }
        }
      }
    }
  }
}`

const templateActionMutation = `mutation TemplateAction($id: UUID!, $input: TemplateActionInput!) {
  templateAction(id: $id, input: $input) {
    id
    entityName
    status
  }
}`

const projectActionMutation = `mutation ProjectAction($id: UUID!, $input: ProjectActionInput!) {
  projectAction(id: $id, input: $input) {
    id
    entityName
    status
  }
}`

const updateResourceMutation = `mutation UpdateResource($id: UUID!, $input: ResourceUpdateInput!) {
  updateResource(id: $id, input: $input) {
    id
    name
    entityName
  }
}`

const resourceActionMutation = `mutation ResourceAction($id: UUID!, $input: ResourceActionInput!) {
  resourceAction(id: $id, input: $input) {
    id
    entityName
    status
  }
}`

const deleteResourceMutation = `mutation DeleteResource($id: UUID!) {
  deleteResource(id: $id)
}`

const updateTemplateMutation = `mutation UpdateTemplate($id: UUID!, $input: TemplateUpdateInput!) {
  updateTemplate(id: $id, input: $input) {
    id
    name
    template
    entityName
  }
}`

const deleteTemplateMutation = `mutation DeleteTemplate($id: UUID!) {
  deleteTemplate(id: $id)
}`

const deleteProjectMutation = `mutation DeleteProject($id: UUID!) {
  deleteProject(id: $id)
}`

const resourceTreeQuery = `query ResourceTree($id: UUID!, $direction: String!) {
  resourceTree(id: $id, direction: $direction) {
    id
    nodeId
    name
    state
    status
    templateName
    children {
      id
      nodeId
      name
      state
      status
      templateName
      children {
        id
        nodeId
        name
        state
        status
        templateName
        children {
          id
          nodeId
          name
          state
          status
          templateName
          children {
            id
            nodeId
            name
            state
            status
            templateName
          }
        }
      }
    }
  }
}`

func (c *Client) Resources(ctx context.Context, filter map[string]any, sort []string, pageRange []int) (ResourcesResult, error) {
	variables := listVariables(filter, sort, pageRange)

	resp, err := query[resourcesQueryData](ctx, c.httpClient, c.endpoint, c.tokenProvider, graphqlRequest{
		Query:     resourcesQuery,
		Variables: variables,
	})
	if err != nil {
		return ResourcesResult{}, err
	}

	return ResourcesResult{Items: resp.Resources, Total: resp.ResourcesCount}, nil
}

func (c *Client) Resource(ctx context.Context, id string) (*Resource, error) {
	resp, err := query[resourceQueryData](ctx, c.httpClient, c.endpoint, c.tokenProvider, graphqlRequest{
		Query: resourceQuery,
		Variables: map[string]any{
			"id": id,
		},
	})
	if err != nil {
		return nil, err
	}
	return resp.Resource, nil
}

func (c *Client) Templates(ctx context.Context, filter map[string]any, sort []string, pageRange []int) (TemplatesResult, error) {
	variables := listVariables(filter, sort, pageRange)

	resp, err := query[templatesQueryData](ctx, c.httpClient, c.endpoint, c.tokenProvider, graphqlRequest{
		Query:     templatesQuery,
		Variables: variables,
	})
	if err != nil {
		return TemplatesResult{}, err
	}

	return TemplatesResult{Items: resp.Templates, Total: resp.TemplatesCount}, nil
}

func (c *Client) Projects(ctx context.Context, filter map[string]any, sort []string, pageRange []int) (ProjectsResult, error) {
	variables := listVariables(filter, sort, pageRange)

	resp, err := query[projectsQueryData](ctx, c.httpClient, c.endpoint, c.tokenProvider, graphqlRequest{
		Query:     projectsQuery,
		Variables: variables,
	})
	if err != nil {
		return ProjectsResult{}, err
	}

	return ProjectsResult{Items: resp.Projects, Total: resp.ProjectsCount}, nil
}

func (c *Client) Template(ctx context.Context, id string) (*Template, error) {
	resp, err := query[templateQueryData](ctx, c.httpClient, c.endpoint, c.tokenProvider, graphqlRequest{
		Query: templateQuery,
		Variables: map[string]any{
			"id": id,
		},
	})
	if err != nil {
		return nil, err
	}
	return resp.Template, nil
}

func (c *Client) Project(ctx context.Context, id string) (*Project, error) {
	resp, err := query[projectQueryData](ctx, c.httpClient, c.endpoint, c.tokenProvider, graphqlRequest{
		Query:     projectQuery,
		Variables: map[string]any{"id": id},
	})
	if err != nil {
		return nil, err
	}
	return resp.Project, nil
}

func (c *Client) TemplateTree(ctx context.Context, id string, direction string) (*TemplateTreeNode, error) {
	resp, err := query[templateTreeQueryData](ctx, c.httpClient, c.endpoint, c.tokenProvider, graphqlRequest{
		Query: templateTreeQuery,
		Variables: map[string]any{
			"id":        id,
			"direction": direction,
		},
	})
	if err != nil {
		return nil, err
	}
	return resp.TemplateTree, nil
}

func (c *Client) UpdateResource(ctx context.Context, id string, input map[string]any) error {
	_, err := query[updateResourceMutationData](ctx, c.httpClient, c.endpoint, c.tokenProvider, graphqlRequest{
		Query: updateResourceMutation,
		Variables: map[string]any{
			"id":    id,
			"input": input,
		},
	})
	return err
}

func (c *Client) ResourceActions(ctx context.Context, id string) ([]string, error) {
	resp, err := query[resourceActionsQueryData](ctx, c.httpClient, c.endpoint, c.tokenProvider, graphqlRequest{
		Query: resourceActionsQuery,
		Variables: map[string]any{
			"id": id,
		},
	})
	if err != nil {
		return nil, err
	}
	return resp.ResourceActions, nil
}

func (c *Client) ResourceTempState(ctx context.Context, id string) (*ResourceTempState, error) {
	resp, err := query[resourceTempStateQueryData](ctx, c.httpClient, c.endpoint, c.tokenProvider, graphqlRequest{
		Query: resourceTempStateQuery,
		Variables: map[string]any{
			"id": id,
		},
	})
	if err != nil {
		return nil, err
	}
	return resp.ResourceTempStateByResource, nil
}

func (c *Client) ApproveResource(ctx context.Context, id string) error {
	return c.resourceAction(ctx, id, "approve")
}

func (c *Client) RejectResource(ctx context.Context, id string) error {
	return c.resourceAction(ctx, id, "reject")
}

func (c *Client) ResourceAction(ctx context.Context, id string, action string) error {
	return c.resourceAction(ctx, id, action)
}

func (c *Client) DeleteResource(ctx context.Context, id string) error {
	resp, err := query[deleteResourceMutationData](ctx, c.httpClient, c.endpoint, c.tokenProvider, graphqlRequest{
		Query: deleteResourceMutation,
		Variables: map[string]any{
			"id": id,
		},
	})
	if err != nil {
		return err
	}
	if !resp.DeleteResource {
		return errors.New("resource delete failed")
	}
	return nil
}

func (c *Client) UpdateTemplate(ctx context.Context, id string, input map[string]any) error {
	_, err := query[updateTemplateMutationData](ctx, c.httpClient, c.endpoint, c.tokenProvider, graphqlRequest{
		Query: updateTemplateMutation,
		Variables: map[string]any{
			"id":    id,
			"input": input,
		},
	})
	return err
}

func (c *Client) EnableTemplate(ctx context.Context, id string) error {
	return c.templateAction(ctx, id, "enable")
}

func (c *Client) resourceAction(ctx context.Context, id string, action string) error {
	_, err := query[resourceActionMutationData](ctx, c.httpClient, c.endpoint, c.tokenProvider, graphqlRequest{
		Query: resourceActionMutation,
		Variables: map[string]any{
			"id": id,
			"input": map[string]any{
				"action": action,
			},
		},
	})
	return err
}

func (c *Client) DisableTemplate(ctx context.Context, id string) error {
	return c.templateAction(ctx, id, "disable")
}

func (c *Client) DeleteTemplate(ctx context.Context, id string) error {
	resp, err := query[deleteTemplateMutationData](ctx, c.httpClient, c.endpoint, c.tokenProvider, graphqlRequest{
		Query: deleteTemplateMutation,
		Variables: map[string]any{
			"id": id,
		},
	})
	if err != nil {
		return err
	}
	if !resp.DeleteTemplate {
		return errors.New("template delete failed")
	}
	return nil
}

func (c *Client) DeleteProject(ctx context.Context, id string) error {
	resp, err := query[deleteProjectMutationData](ctx, c.httpClient, c.endpoint, c.tokenProvider, graphqlRequest{
		Query: deleteProjectMutation,
		Variables: map[string]any{
			"id": id,
		},
	})
	if err != nil {
		return err
	}
	if !resp.DeleteProject {
		return errors.New("project delete failed")
	}
	return nil
}

func (c *Client) EnableProject(ctx context.Context, id string) error {
	return c.projectAction(ctx, id, "enable")
}

func (c *Client) DisableProject(ctx context.Context, id string) error {
	return c.projectAction(ctx, id, "disable")
}

func (c *Client) ResourceTree(ctx context.Context, id string, direction string) (*ResourceTreeNode, error) {
	resp, err := query[resourceTreeQueryData](ctx, c.httpClient, c.endpoint, c.tokenProvider, graphqlRequest{
		Query: resourceTreeQuery,
		Variables: map[string]any{
			"id":        id,
			"direction": direction,
		},
	})
	if err != nil {
		return nil, err
	}
	return resp.ResourceTree, nil
}

func (c *Client) templateAction(ctx context.Context, id string, action string) error {
	_, err := query[templateActionMutationData](ctx, c.httpClient, c.endpoint, c.tokenProvider, graphqlRequest{
		Query: templateActionMutation,
		Variables: map[string]any{
			"id": id,
			"input": map[string]any{
				"action": action,
			},
		},
	})
	return err
}

func (c *Client) projectAction(ctx context.Context, id string, action string) error {
	_, err := query[projectActionMutationData](ctx, c.httpClient, c.endpoint, c.tokenProvider, graphqlRequest{
		Query: projectActionMutation,
		Variables: map[string]any{
			"id": id,
			"input": map[string]any{
				"action": action,
			},
		},
	})
	return err
}

func listVariables(filter map[string]any, sort []string, pageRange []int) map[string]any {
	variables := map[string]any{}
	if filter != nil {
		variables["filter"] = filter
	}
	if len(sort) > 0 {
		variables["sort"] = sort
	}
	if len(pageRange) > 0 {
		variables["range"] = pageRange
	}
	return variables
}
