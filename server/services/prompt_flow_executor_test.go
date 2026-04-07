package services

import (
	"context"
	"testing"

	"open-nirmata/db"
	"open-nirmata/db/models"
	"open-nirmata/dto"

	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type fakePromptFlowExecutorDB struct {
	tools []models.Tool
}

func (f *fakePromptFlowExecutorDB) FindOne(ctx context.Context, col db.DbCollection, filter interface{}, opts ...*options.FindOneOptions) *mongo.SingleResult {
	return mongo.NewSingleResultFromDocument(nil, mongo.ErrNoDocuments, nil)
}

func (f *fakePromptFlowExecutorDB) Find(ctx context.Context, col db.DbCollection, filter interface{}, opts ...*options.FindOptions) (*mongo.Cursor, error) {
	documents := make([]interface{}, 0, len(f.tools))
	for _, tool := range f.tools {
		documents = append(documents, tool)
	}
	return mongo.NewCursorFromDocuments(documents, nil, nil)
}

func (f *fakePromptFlowExecutorDB) InsertOne(ctx context.Context, col db.DbCollection, document interface{}, opts ...*options.InsertOneOptions) (*mongo.InsertOneResult, error) {
	return nil, nil
}

func (f *fakePromptFlowExecutorDB) UpdateOne(ctx context.Context, col db.DbCollection, filter interface{}, update interface{}, opts ...*options.UpdateOptions) (*mongo.UpdateResult, error) {
	return nil, nil
}

func (f *fakePromptFlowExecutorDB) DeleteOne(ctx context.Context, col db.DbCollection, filter interface{}, opts ...*options.DeleteOptions) (*mongo.DeleteResult, error) {
	return nil, nil
}

func (f *fakePromptFlowExecutorDB) Aggregate(ctx context.Context, col db.DbCollection, filter interface{}, opts ...*options.AggregateOptions) (*mongo.Cursor, error) {
	return nil, nil
}

func (f *fakePromptFlowExecutorDB) CountDocuments(ctx context.Context, col db.DbCollection, filter interface{}, opts ...*options.CountOptions) (int64, error) {
	return 0, nil
}

func (f *fakePromptFlowExecutorDB) BulkWrite(ctx context.Context, col db.DbCollection, models []mongo.WriteModel, opts ...*options.BulkWriteOptions) (*mongo.BulkWriteResult, error) {
	return nil, nil
}

func (f *fakePromptFlowExecutorDB) UpdateMany(ctx context.Context, col db.DbCollection, filter interface{}, update interface{}, opts ...*options.UpdateOptions) (*mongo.UpdateResult, error) {
	return nil, nil
}

func (f *fakePromptFlowExecutorDB) SetDbInContext(ctx context.Context) context.Context {
	return ctx
}

func (f *fakePromptFlowExecutorDB) Disconnect(ctx context.Context) error {
	return nil
}

func (f *fakePromptFlowExecutorDB) GetCollection(collectionName string) *mongo.Collection {
	return nil
}

func TestLoadToolsUsesStoredInputSchema(t *testing.T) {
	svc := NewPromptFlowExecutorService(&fakePromptFlowExecutorDB{tools: []models.Tool{{
		Id:          "tool-1",
		Name:        "filesystem",
		Description: "Lists files",
		Config: &dto.ToolConfig{
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"path": map[string]interface{}{"type": "string"},
				},
				"required": []interface{}{"path"},
			},
		},
	}}})

	tools, err := svc.loadTools(context.Background(), []string{"tool-1"})
	if err != nil {
		t.Fatalf("expected loadTools to succeed, got error: %v", err)
	}
	if len(tools) != 1 {
		t.Fatalf("expected 1 tool, got %d", len(tools))
	}
	if tools[0].Function.Parameters["type"] != "object" {
		t.Fatalf("expected schema type to be preserved, got %#v", tools[0].Function.Parameters)
	}
	properties, ok := tools[0].Function.Parameters["properties"].(map[string]interface{})
	if !ok || len(properties) == 0 {
		t.Fatalf("expected schema properties to be populated, got %#v", tools[0].Function.Parameters)
	}
}

func TestBuildToolParametersSchemaDefaultsToObject(t *testing.T) {
	parameters := buildToolParametersSchema(models.Tool{}, nil)
	if parameters["type"] != "object" {
		t.Fatalf("expected default schema type object, got %#v", parameters)
	}
	if _, ok := parameters["properties"].(map[string]interface{}); !ok {
		t.Fatalf("expected default properties map, got %#v", parameters)
	}
}

func TestLoadToolsUsesDiscoveredMCPToolSchemas(t *testing.T) {
	svc := NewPromptFlowExecutorService(&fakePromptFlowExecutorDB{tools: []models.Tool{{
		Id:          "tool-2",
		Name:        "filesystem-server",
		Description: "Filesystem MCP server",
		Type:        string(dto.ToolTypeMCP),
		Tools: []dto.MCPDiscoveredTool{
			{
				Name:        "list_files",
				Description: "List files in a directory",
				InputSchema: map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"path": map[string]interface{}{"type": "string"},
					},
				},
			},
			{
				Name:        "read_file",
				Description: "Read a file",
				InputSchema: map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"file": map[string]interface{}{"type": "string"},
					},
				},
			},
		},
	}}})

	tools, err := svc.loadTools(context.Background(), []string{"tool-2"})
	if err != nil {
		t.Fatalf("expected loadTools to succeed, got error: %v", err)
	}
	if len(tools) != 2 {
		t.Fatalf("expected MCP discovered tools to expand to 2 chat tools, got %d", len(tools))
	}
	if tools[0].Function.Name != "list_files" {
		t.Fatalf("expected first discovered tool to be list_files, got %#v", tools[0].Function)
	}
	if tools[1].Function.Name != "read_file" {
		t.Fatalf("expected second discovered tool to be read_file, got %#v", tools[1].Function)
	}
}

func TestLoadToolsBuildsHTTPToolSchemaFromPayloadTemplate(t *testing.T) {
	svc := NewPromptFlowExecutorService(&fakePromptFlowExecutorDB{tools: []models.Tool{{
		Id:          "tool-3",
		Name:        "search_api",
		Description: "Searches a remote API",
		Type:        string(dto.ToolTypeHTTP),
		Config: &dto.ToolConfig{
			URL:             "https://example.com/search",
			Method:          "POST",
			PayloadTemplate: `{"query":"{{.query}}","limit":"{{.limit}}"}`,
		},
	}}})

	tools, err := svc.loadTools(context.Background(), []string{"tool-3"})
	if err != nil {
		t.Fatalf("expected loadTools to succeed, got error: %v", err)
	}
	if len(tools) != 1 {
		t.Fatalf("expected 1 HTTP chat tool, got %d", len(tools))
	}
	if tools[0].Function.Name != "search_api" {
		t.Fatalf("expected HTTP tool name to be preserved, got %#v", tools[0].Function)
	}
	properties, ok := tools[0].Function.Parameters["properties"].(map[string]interface{})
	if !ok {
		t.Fatalf("expected HTTP tool parameters to expose properties, got %#v", tools[0].Function.Parameters)
	}
	if _, ok := properties["query"]; !ok {
		t.Fatalf("expected inferred query property for HTTP tool, got %#v", properties)
	}
	if _, ok := properties["limit"]; !ok {
		t.Fatalf("expected inferred limit property for HTTP tool, got %#v", properties)
	}
}
