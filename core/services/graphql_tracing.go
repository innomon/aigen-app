package services

import (
	"context"
	"sync"
	"time"

	"github.com/graphql-go/graphql"
	"github.com/graphql-go/graphql/gqlerrors"
)

type tracingContextKey struct{}

type TracingResolverRecord struct {
	Path        []interface{} `json:"path"`
	ParentType  string        `json:"parentType"`
	FieldName   string        `json:"fieldName"`
	ReturnType  string        `json:"returnType"`
	StartOffset int64         `json:"startOffset"`
	Duration    int64         `json:"duration"`
}

type tracingState struct {
	mu                    sync.Mutex
	startTime             time.Time
	endTime               time.Time
	duration              time.Duration
	parsingStartOffset    time.Duration
	parsingDuration       time.Duration
	validationStartOffset time.Duration
	validationDuration    time.Duration
	resolvers             []TracingResolverRecord
}

type TracingParsing struct {
	StartOffset int64 `json:"startOffset"`
	Duration    int64 `json:"duration"`
}

type TracingValidation struct {
	StartOffset int64 `json:"startOffset"`
	Duration    int64 `json:"duration"`
}

type TracingExecution struct {
	Resolvers []TracingResolverRecord `json:"resolvers"`
}

type ApolloTracingReport struct {
	Version    int               `json:"version"`
	StartTime  string            `json:"startTime"`
	EndTime    string            `json:"endTime"`
	Duration   int64             `json:"duration"`
	Parsing    TracingParsing    `json:"parsing"`
	Validation TracingValidation `json:"validation"`
	Execution  TracingExecution  `json:"execution"`
}

type ApolloTracingExtension struct{}

func NewApolloTracingExtension() *ApolloTracingExtension {
	return &ApolloTracingExtension{}
}

func (e *ApolloTracingExtension) Name() string {
	return "tracing"
}

func (e *ApolloTracingExtension) HasResult() bool {
	return true
}

func (e *ApolloTracingExtension) Init(ctx context.Context, p *graphql.Params) context.Context {
	state := &tracingState{
		startTime: time.Now(),
		resolvers: make([]TracingResolverRecord, 0),
	}
	return context.WithValue(ctx, tracingContextKey{}, state)
}

func (e *ApolloTracingExtension) ParseDidStart(ctx context.Context) (context.Context, graphql.ParseFinishFunc) {
	state, _ := ctx.Value(tracingContextKey{}).(*tracingState)
	start := time.Now()
	return ctx, func(err error) {
		if state != nil {
			state.mu.Lock()
			state.parsingStartOffset = start.Sub(state.startTime)
			state.parsingDuration = time.Since(start)
			state.mu.Unlock()
		}
	}
}

func (e *ApolloTracingExtension) ValidationDidStart(ctx context.Context) (context.Context, graphql.ValidationFinishFunc) {
	state, _ := ctx.Value(tracingContextKey{}).(*tracingState)
	start := time.Now()
	return ctx, func(errs []gqlerrors.FormattedError) {
		if state != nil {
			state.mu.Lock()
			state.validationStartOffset = start.Sub(state.startTime)
			state.validationDuration = time.Since(start)
			state.mu.Unlock()
		}
	}
}

func (e *ApolloTracingExtension) ExecutionDidStart(ctx context.Context) (context.Context, graphql.ExecutionFinishFunc) {
	state, _ := ctx.Value(tracingContextKey{}).(*tracingState)
	return ctx, func(result *graphql.Result) {
		if state != nil {
			state.mu.Lock()
			state.endTime = time.Now()
			state.duration = state.endTime.Sub(state.startTime)
			state.mu.Unlock()
		}
	}
}

func (e *ApolloTracingExtension) ResolveFieldDidStart(ctx context.Context, info *graphql.ResolveInfo) (context.Context, graphql.ResolveFieldFinishFunc) {
	state, _ := ctx.Value(tracingContextKey{}).(*tracingState)
	start := time.Now()

	var path []interface{}
	if info != nil && info.Path != nil {
		path = info.Path.AsArray()
	}
	if len(path) == 0 && info != nil {
		path = []interface{}{info.FieldName}
	}

	var parentType string
	if info != nil && info.ParentType != nil {
		parentType = info.ParentType.Name()
	}

	var returnType string
	if info != nil && info.ReturnType != nil {
		returnType = info.ReturnType.String()
	}

	fieldName := ""
	if info != nil {
		fieldName = info.FieldName
	}

	return ctx, func(val interface{}, err error) {
		if state != nil {
			end := time.Now()
			rec := TracingResolverRecord{
				Path:        path,
				ParentType:  parentType,
				FieldName:   fieldName,
				ReturnType:  returnType,
				StartOffset: start.Sub(state.startTime).Nanoseconds(),
				Duration:    end.Sub(start).Nanoseconds(),
			}
			state.mu.Lock()
			state.resolvers = append(state.resolvers, rec)
			state.mu.Unlock()
		}
	}
}

func (e *ApolloTracingExtension) GetResult(ctx context.Context) interface{} {
	state, _ := ctx.Value(tracingContextKey{}).(*tracingState)
	if state == nil {
		return nil
	}

	state.mu.Lock()
	defer state.mu.Unlock()

	endTime := state.endTime
	if endTime.IsZero() {
		endTime = time.Now()
	}
	duration := state.duration
	if duration == 0 {
		duration = endTime.Sub(state.startTime)
	}

	resolvers := state.resolvers
	if resolvers == nil {
		resolvers = make([]TracingResolverRecord, 0)
	}

	return ApolloTracingReport{
		Version:   1,
		StartTime: state.startTime.UTC().Format(time.RFC3339Nano),
		EndTime:   endTime.UTC().Format(time.RFC3339Nano),
		Duration:  duration.Nanoseconds(),
		Parsing: TracingParsing{
			StartOffset: state.parsingStartOffset.Nanoseconds(),
			Duration:    state.parsingDuration.Nanoseconds(),
		},
		Validation: TracingValidation{
			StartOffset: state.validationStartOffset.Nanoseconds(),
			Duration:    state.validationDuration.Nanoseconds(),
		},
		Execution: TracingExecution{
			Resolvers: resolvers,
		},
	}
}
