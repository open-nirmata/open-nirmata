package knowledgebases

import (
	"net/http"

	"open-nirmata/dto"
	"open-nirmata/utils/docs"

	"github.com/gofiber/fiber/v2"
)

var routeTags = []string{"knowledgebases"}

func RegisterRoutes(router fiber.Router, path string) {
	router.Get(path, ListKnowledgebases)
	router.Post(path, CreateKnowledgebase)
	router.Get(path+"/:id", GetKnowledgebase)
	router.Put(path+"/:id", UpdateKnowledgebase)
	router.Delete(path+"/:id", DeleteKnowledgebase)

	docs.RegisterApi(docs.ApiWrapper{
		Path:        path,
		Method:      http.MethodGet,
		Name:        "List Knowledgebases",
		Description: "List configured knowledgebase providers with optional provider, enabled, and search filters.",
		Response:    &docs.ApiResponse{Description: "Knowledgebases fetched successfully", Content: new(dto.KnowledgebaseListResponse)},
		Parameters: []docs.ApiParameter{
			{Name: "provider", In: "query", Description: "Filter by provider (milvus, mixedbread, zeroentropy, algolia, qdrant)"},
			{Name: "enabled", In: "query", Description: "Filter by enabled status"},
			{Name: "q", In: "query", Description: "Search by name, description, index, namespace, or embedding model"},
		},
		Tags:            routeTags,
		UnAuthenticated: true,
	})

	docs.RegisterApi(docs.ApiWrapper{
		Path:            path,
		Method:          http.MethodPost,
		Name:            "Create Knowledgebase",
		Description:     "Create a saved knowledgebase configuration for Milvus, Mixedbread, ZeroEntropy, Algolia, or Qdrant. Shilp can be added later.",
		RequestBody:     &docs.ApiRequestBody{Description: "Knowledgebase details", Content: new(dto.CreateKnowledgebaseRequest)},
		Response:        &docs.ApiResponse{Description: "Knowledgebase created successfully", Content: new(dto.KnowledgebaseResponse)},
		Tags:            routeTags,
		UnAuthenticated: true,
	})

	docs.RegisterApi(docs.ApiWrapper{
		Path:            path + "/:id",
		Method:          http.MethodGet,
		Name:            "Get Knowledgebase",
		Description:     "Fetch a single knowledgebase configuration by ID.",
		Response:        &docs.ApiResponse{Description: "Knowledgebase fetched successfully", Content: new(dto.KnowledgebaseResponse)},
		Parameters:      []docs.ApiParameter{{Name: "id", In: "path", Description: "Knowledgebase ID", Required: true}},
		Tags:            routeTags,
		UnAuthenticated: true,
	})

	docs.RegisterApi(docs.ApiWrapper{
		Path:            path + "/:id",
		Method:          http.MethodPut,
		Name:            "Update Knowledgebase",
		Description:     "Update a saved knowledgebase configuration by ID.",
		RequestBody:     &docs.ApiRequestBody{Description: "Knowledgebase fields to update", Content: new(dto.UpdateKnowledgebaseRequest)},
		Response:        &docs.ApiResponse{Description: "Knowledgebase updated successfully", Content: new(dto.KnowledgebaseResponse)},
		Parameters:      []docs.ApiParameter{{Name: "id", In: "path", Description: "Knowledgebase ID", Required: true}},
		Tags:            routeTags,
		UnAuthenticated: true,
	})

	docs.RegisterApi(docs.ApiWrapper{
		Path:            path + "/:id",
		Method:          http.MethodDelete,
		Name:            "Delete Knowledgebase",
		Description:     "Delete a saved knowledgebase configuration by ID.",
		Response:        &docs.ApiResponse{Description: "Knowledgebase deleted successfully", Content: new(dto.KnowledgebaseResponse)},
		Parameters:      []docs.ApiParameter{{Name: "id", In: "path", Description: "Knowledgebase ID", Required: true}},
		Tags:            routeTags,
		UnAuthenticated: true,
	})
}
