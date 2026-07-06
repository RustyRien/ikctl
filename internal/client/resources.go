package client

import "context"

const resourcesQuery = `query ListResources($filter: JSON, $sort: [String!], $range: [Int!]) {
  resources(filter: $filter, sort: $sort, range: $range) {
    id
    name
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

const resourceQuery = `query GetResource($id: UUID!) {
  resource(id: $id) {
    id
    name
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
