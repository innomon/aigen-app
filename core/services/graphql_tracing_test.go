package services

import (
	"context"
	"testing"

	"github.com/graphql-go/graphql"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestApolloTracingExtension(t *testing.T) {
	queryType := graphql.NewObject(graphql.ObjectConfig{
		Name: "Query",
		Fields: graphql.Fields{
			"hello": &graphql.Field{
				Type: graphql.String,
				Resolve: func(p graphql.ResolveParams) (interface{}, error) {
					return "world", nil
				},
			},
		},
	})

	tracingExt := NewApolloTracingExtension()
	schema, err := graphql.NewSchema(graphql.SchemaConfig{
		Query: queryType,
		Extensions: []graphql.Extension{
			tracingExt,
		},
	})
	require.NoError(t, err)

	ctx := context.Background()
	params := graphql.Params{
		Schema:        schema,
		RequestString: "{ hello }",
		Context:       ctx,
	}

	result := graphql.Do(params)
	require.Empty(t, result.Errors)
	require.NotNil(t, result.Data)
	require.NotNil(t, result.Extensions)

	tracingRaw, exists := result.Extensions["tracing"]
	require.True(t, exists, "tracing extension must exist")

	report, ok := tracingRaw.(ApolloTracingReport)
	require.True(t, ok, "tracing extension must be ApolloTracingReport")

	assert.Equal(t, 1, report.Version)
	assert.NotEmpty(t, report.StartTime)
	assert.NotEmpty(t, report.EndTime)
	assert.Greater(t, report.Duration, int64(0))
	assert.NotEmpty(t, report.Execution.Resolvers)
	assert.Equal(t, "hello", report.Execution.Resolvers[0].FieldName)
	assert.Equal(t, "Query", report.Execution.Resolvers[0].ParentType)
	assert.Equal(t, "String", report.Execution.Resolvers[0].ReturnType)
}
